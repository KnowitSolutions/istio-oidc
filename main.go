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
	"net"
	"time"
)

// TODO: Forward tracing headers from gRPC when calling HTTP services
func main() {
	//log.SetHandler(logfmt.Default)
	log.SetHandler(cli.Default)

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

	err = srv.AddService(context.Background(), &config.Service{
		Name:  "test",
		Realm: "master",
		OIDC: config.OIDC{
			ClientID:     "test",
			ClientSecret: "be7457c0-a723-4375-967c-17eb7acc53fc",
			CallbackPath: "/oidc/callback",
		},
	})
	if err != nil {
		// TODO: Log more info about service
		// TODO: Not fatal when dynamic load from K8s
		log.WithError(err).Fatal("Unable to add service to server")
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
