package main

import (
	"flag"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/auth"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/controller"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"github.com/KnowitSolutions/istio-oidc/replication"
	"github.com/KnowitSolutions/istio-oidc/state"
	"github.com/KnowitSolutions/istio-oidc/telemetry"
	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	log.Setup()

	cfg := flag.String("config", "config.yaml", "Configuration file to load")
	flag.Parse()
	config.Load(*cfg)

	id, err := replication.NewPeerId()
	if err != nil {
		log.Error(nil, err, "Failed creating peer ID")
		os.Exit(1)
	}
	vals := log.MakeValues("id", id)
	log.Info(nil, vals, "Assigned peer ID")

	keyStore := state.NewKeyStore()
	apStore := state.NewAccessPolicyStore()
	sessStore, err := state.NewSessionStore(id)
	if err != nil {
		log.Error(nil, err, "Failed creating stores")
		os.Exit(1)
	}

	self := replication.NewSelf(id, sessStore)
	peers := replication.NewPeers()

	init := make(chan struct{})

	go startCtrl(keyStore, apStore)
	go startGrpc(keyStore, apStore, sessStore, self, peers, init)
	go startTelemetry(init)
	select {}
}

func startCtrl(
	keyStore state.KeyStore,
	apStore state.AccessPolicyStore,
) {
	ctrl.SetLogger(log.Shim)
	klog.SetLogger(log.Shim.WithName("kubernetes"))

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Error(nil, err, "Unable to load Kubernetes config")
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	opts := ctrl.Options{
		Scheme:                  scheme,
		HealthProbeBindAddress:  "0",
		MetricsBindAddress:      "0",
		LeaderElection:          config.Controller.LeaderElection,
		LeaderElectionNamespace: config.Controller.LeaderElectionNamespace,
		LeaderElectionID:        config.Controller.LeaderElectionName,
	}
	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		log.Error(nil, err, "Unable to create manager")
		os.Exit(1)
	}

	err = controller.Register(mgr, keyStore, apStore)
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

func startGrpc(
	keyStore state.KeyStore,
	apStore state.AccessPolicyStore,
	sessStore state.SessionStore,
	self *replication.Self,
	peers *replication.Peers,
	init chan<- struct{},
) {
	lis, err := net.Listen("tcp", config.Service.Address)
	if err != nil {
		err = errors.Wrap(err, "", "address", config.Service.Address)
		log.Error(nil, err, "Unable to bind TCP socket")
		os.Exit(1)
	}

	srv := grpc.NewServer()
	startExtAuthz(srv, keyStore, apStore, sessStore, self, peers)
	startReplication(srv, self, peers, init)

	err = srv.Serve(lis)
	if err != nil {
		log.Error(nil, err, "Unable to start gRPC server")
		os.Exit(1)
	}
}

func startExtAuthz(
	srv *grpc.Server,
	keyStore state.KeyStore,
	apStore state.AccessPolicyStore,
	sessStore state.SessionStore,
	self *replication.Self,
	peers *replication.Peers,
) {
	extAuth := auth.Server{
		KeyStore:          keyStore,
		AccessPolicyStore: apStore,
		SessionStore:      sessStore,
		Client:            replication.Client{Self: self, Peers: peers},
	}
	authv2.RegisterAuthorizationServer(srv, extAuth.V2())
}

func startReplication(
	srv *grpc.Server,
	self *replication.Self,
	peers *replication.Peers,
	init chan<- struct{},
) {
	repl := replication.Server{Self: self, Peers: peers}
	api.RegisterReplicationServer(srv, &repl)

	replication.NewWorker(self, peers, init)
}

func startTelemetry(
	init <-chan struct{},
) {
	mux := http.NewServeMux()
	srv := http.Server{Addr: config.Telemetry.Address, Handler: mux}

	telemetry.RegisterProbes(mux, init)
	telemetry.RegisterMetrics(mux)

	err := srv.ListenAndServe()
	if err != nil {
		err = errors.Wrap(err, "", "address", config.Telemetry.Address)
		log.Error(nil, err, "Unable to start HTTP server")
		os.Exit(1)
	}
}
