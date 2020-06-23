package main

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/logfmt"
	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
	"istio-keycloak/auth"
	"istio-keycloak/controller"
	"istio-keycloak/introspection"
	"istio-keycloak/logging"
	"istio-keycloak/state"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// TODO: Test OIDC with internal .svc k8s address from /etc/hosts
// TODO: Forward tracing headers from gRPC when calling HTTP services
func main() {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		log.SetHandler(cli.Default)
	} else {
		log.SetHandler(logfmt.Default)
	}

	keyStore := state.NewKeyStore()
	_, err := keyStore.MakeKey()
	if err != nil {
		log.WithError(err).Fatal("Unable to generate cryptographic key")
	}

	oidcCommStore := state.NewOidcCommunicatorStore()

	go startCtrl(oidcCommStore)
	go startGrpc(keyStore, oidcCommStore)
	go startIntrospection()
	select {}
}

func startCtrl(oidcCommStore state.OidcCommunicatorStore) {
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

	(&controller.Controller{
		Client:                mgr.GetClient(),
		OidcCommunicatorStore: oidcCommStore,
	}).SetupWithManager(mgr)

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.WithError(err).Fatal("Unable to start manager")
	}
}

func startGrpc(keyStore state.KeyStore, oidcCommStore state.OidcCommunicatorStore) {
	addr := ":8082"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.WithError(err).WithField("address", addr).
			Fatal("Unable to bind TCP socket")
	}

	srv := grpc.NewServer()
	startExtAuthz(srv, keyStore, oidcCommStore)

	err = srv.Serve(lis)
	if err != nil {
		log.WithError(err).Fatal("Unable to start gRPC server")
	}
}

func startExtAuthz(srv *grpc.Server, keyStore state.KeyStore, oidcCommStore state.OidcCommunicatorStore) {
	extAuth := auth.Server{
		KeyStore:              keyStore,
		OidcCommunicatorStore: oidcCommStore,
		SessionStore:          state.NewSessionStore(),
	}
	state.KeycloakURL = "http://keycloak.localhost"
	state.SessionCleaningInterval = 30 * time.Second
	state.SessionCleaningGracePeriod = 30 * time.Second
	extAuth.Start()

	authv2.RegisterAuthorizationServer(srv, extAuth.V2())
}

func startIntrospection() {
	mux := http.NewServeMux()
	srv := http.Server{Addr: ":8083", Handler: mux}

	introspection.RegisterProbes(mux)
	introspection.RegisterMetrics(mux)

	err := srv.ListenAndServe()
	if err != nil {
		log.WithError(err).Fatal("Unable to start HTTP server")
	}
}
