package main

import (
	"flag"
	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"google.golang.org/grpc"
	"istio-keycloak/auth"
	"istio-keycloak/config"
	"istio-keycloak/controller"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"istio-keycloak/state"
	"istio-keycloak/telemetry"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TODO: Leader election
func main() {
	log.Setup()

	cfg := flag.String("config", "config.yaml", "Configuration file to load")
	flag.Parse()
	config.Load(*cfg)

	keyStore := state.NewKeyStore()
	_, err := keyStore.MakeKey()
	if err != nil {
		log.Error(nil, err, "Unable to generate cryptographic key")
		os.Exit(1)
	}

	apStore := state.NewAccessPolicyStore()

	go startCtrl(apStore)
	go startGrpc(keyStore, apStore)
	go startTelemetry()
	select {}
}

func startCtrl(apStore state.AccessPolicyStore) {
	ctrl.SetLogger(log.Shim)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Error(nil, err, "Unable to load Kubernetes config")
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	opts := ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: "0",
		MetricsBindAddress:     "0",
	}
	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		log.Error(nil, err, "Unable to create manager")
		os.Exit(1)
	}

	err = controller.Register(mgr, apStore)
	if err != nil {
		log.Error(nil, err, "Unable to register controllers")
		os.Exit(1)
	}

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.Error(nil, err, "Unable to start manager")
		os.Exit(1)
	}
}

func startGrpc(keyStore state.KeyStore, apStore state.AccessPolicyStore) {
	lis, err := net.Listen("tcp", config.Service.Address)
	if err != nil {
		err = errors.Wrap(err, "", "address", config.Service.Address)
		log.Error(nil, err, "Unable to bind TCP socket")
		os.Exit(1)
	}

	srv := grpc.NewServer()
	startExtAuthz(srv, keyStore, apStore)

	err = srv.Serve(lis)
	if err != nil {
		log.Error(nil, err, "Unable to start gRPC server")
		os.Exit(1)
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
		err = errors.Wrap(err, "", "address", config.Telemetry.Address)
		log.Error(nil, err, "Unable to start HTTP server")
		os.Exit(1)
	}
}
