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

	keyStore := auth.NewKeyStore()
	_, err := keyStore.MakeKey()
	if err != nil {
		log.WithError(err).Fatal("Unable to generate cryptographic key")
	}

	polStore := auth.NewPolicyStore()

	startCtrl(polStore)
	startExtAuthz(keyStore, polStore)
}

func startCtrl(polStore auth.PolicyStore) {
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
		Client:      mgr.GetClient(),
		PolicyStore: polStore,
	}).SetupWithManager(mgr)

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.WithError(err).Fatal("Unable to start manager")
	}
}

func startExtAuthz(keyStore auth.KeyStore, polStore auth.PolicyStore) {
	srv := auth.NewServer()
	auth.KeycloakURL = "http://keycloak.localhost"
	auth.SessionCleaningInterval = 30 * time.Second
	auth.SessionCleaningGracePeriod = 30 * time.Second
	srv.KeyStore = keyStore
	srv.PolicyStore = polStore
	srv.Start()

	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		// TODO: Log bind parameters
		log.WithError(err).Fatal("Unable to bind TCP socket")
	}

	grpcServer := grpc.NewServer()
	authv2.RegisterAuthorizationServer(grpcServer, srv.V2())

	err = grpcServer.Serve(lis)
	if err != nil {
		log.WithError(err).Fatal("Unable to start gRPC server")
	}
}
