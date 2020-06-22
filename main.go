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
	"istio-keycloak/logging"
	"istio-keycloak/state"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
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
	select {}
}

func startCtrl(oidcCommStore state.OidcCommunicatorStore) {
	ctrl.SetLogger(logging.Log)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.WithError(err).Fatal("Unable to load Kubernetes config")
	}

	var scheme = runtime.NewScheme()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme})
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
