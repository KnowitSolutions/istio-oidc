package main

import (
	"flag"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/logfmt"
	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
	"istio-keycloak/auth"
	"istio-keycloak/config"
	"istio-keycloak/controller"
	"istio-keycloak/logging"
	"istio-keycloak/state"
	"istio-keycloak/telemetry"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TODO: Switch to context logging
func main() {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		log.SetHandler(cli.Default)
	} else {
		log.SetHandler(logfmt.Default)
	}

	cfg := flag.String("config", "config.yaml", "Configuration file to load")
	flag.Parse()
	config.Load(*cfg)

	keyStore := state.NewKeyStore()
	_, err := keyStore.MakeKey()
	if err != nil {
		log.WithError(err).Fatal("Unable to generate cryptographic key")
	}

	apStore := state.NewAccessPolicyStore()

	go startCtrl(apStore)
	go startGrpc(keyStore, apStore)
	go startTelemetry()
	select {}
}

func startCtrl(apStore state.AccessPolicyStore) {
	ctrl.SetLogger(logging.Log)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.WithError(err).Fatal("Unable to load Kubernetes config")
	}

	scheme := runtime.NewScheme()
	opts := ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: "0",
		MetricsBindAddress:     "0",
	}
	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		log.WithError(err).Fatal("Unable to create manager")
	}

	err = controller.Register(mgr, apStore)
	if err != nil {
		log.WithError(err).Fatal("Unable to register controllers")
	}

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.WithError(err).Fatal("Unable to start manager")
	}
}

func startGrpc(keyStore state.KeyStore, apStore state.AccessPolicyStore) {
	lis, err := net.Listen("tcp", config.Service.Address)
	if err != nil {
		log.WithError(err).WithField("address", config.Service.Address).
			Fatal("Unable to bind TCP socket")
	}

	srv := grpc.NewServer()
	startExtAuthz(srv, keyStore, apStore)

	err = srv.Serve(lis)
	if err != nil {
		log.WithError(err).Fatal("Unable to start gRPC server")
	}
}

func startExtAuthz(srv *grpc.Server, keyStore state.KeyStore, apStore state.AccessPolicyStore) {
	extAuth := auth.Server{
		KeyStore:          keyStore,
		AccessPolicyStore: apStore,
		SessionStore:      state.NewSessionStore(),
	}
	extAuth.Start()

	authv2.RegisterAuthorizationServer(srv, extAuth.V2())
}

func startTelemetry() {
	mux := http.NewServeMux()
	srv := http.Server{Addr: config.Telemetry.Address, Handler: mux}

	telemetry.RegisterProbes(mux)
	telemetry.RegisterMetrics(mux)

	err := srv.ListenAndServe()
	if err != nil {
		log.WithError(err).WithField("address", config.Telemetry.Address).
			Fatal("Unable to start HTTP server")
	}
}
