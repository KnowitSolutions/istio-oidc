package main

import (
	"crypto/rand"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"
	"istio-keycloak/auth"
	"istio-keycloak/config"
	"net"
)

// TODO: Roles in Envoy metadata so that roles can be specified per path
// TODO: Forward tracing headers from gRPC when calling HTTP services
func main() {
	//log.SetHandler(logfmt.Default)
	log.SetHandler(cli.Default)

	srv := auth.NewServer()
	srv.TokenDuration = 30
	srv.KeycloakURL = "http://keycloak.localhost"

	srv.Key = make([]byte, 64)
	_, err := rand.Read(srv.Key)
	if err != nil {
		panic(err.Error())
	}

	err = srv.AddService(&config.Service{
		Name:  "test",
		Realm: "master",
		OIDC: config.OIDC{
			ClientID:     "test",
			ClientSecret: "5ca4509d-cf9b-47e9-9119-cf72cf7a5a44",
			CallbackPath: "/oidc/callback",
		},
	})
	if err != nil {
		panic(err.Error())
	}

	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		panic(err.Error())
	}

	grpcServer := grpc.NewServer()
	authv2.RegisterAuthorizationServer(grpcServer, srv.V2())
	authv3.RegisterAuthorizationServer(grpcServer, srv.V3())

	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err.Error())
	}
}
