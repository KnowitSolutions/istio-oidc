package main

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"google.golang.org/grpc"
	"istio-keycloak/auth"
	"istio-keycloak/config"
	"istio-keycloak/controller"
	"istio-keycloak/logging"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// TODO: Test OIDC with internal .svc k8s address from /etc/hosts
// TODO: Forward tracing headers from gRPC when calling HTTP services
func main() {
	//log.SetHandler(logfmt.Default)
	log.SetHandler(cli.Default)

	startCtrl()
	startExtAuthz()
}

func startCtrl() {
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
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.WithError(err).Fatal("Unable to start manager")
	}
}

func startExtAuthz() {
	srv := auth.NewServer()
	srv.KeycloakURL = "http://keycloak.localhost"
	srv.SessionCleaning.Interval = 30 * time.Second
	srv.SessionCleaning.GracePeriod = 30 * time.Second
	srv.Start()

	srv.Key = make([]byte, sha512.Size)
	_, err := rand.Read(srv.Key)
	if err != nil {
		log.WithError(err).Fatal("Unable to generate cryptographic key")
	}

	err = srv.AddAccessPolicy(context.Background(), &config.AccessPolicy{
		Name:  "jaeger",
		Realm: "master",
		OIDC: config.OIDC{
			ClientID:     "jaeger",
			ClientSecret: "742c63fc-1ead-43ea-87c4-ffd4d6a1550c",
			CallbackPath: "/oidc/callback",
		},
	})
	if err != nil {
		// TODO: Log more info about service
		// TODO: Not fatal when dynamic load from K8s
		log.WithError(err).Fatal("Unable to add access policy to server")
	}

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