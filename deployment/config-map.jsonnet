function(namespace) {
  apiVersion: 'v1',
  kind: 'ConfigMap',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc',
  },
  data: {
    'config.yaml': |||
      Controller:
        LeaderElection: true
      ExtAuthz:
        ClusterName: outbound|8080||istio-oidc.%(namespace)s.svc.cluster.local
      Replication:
        Mode: dns
        PeerAddress:
          Service: grpc
          Domain: istio-oidc-discovery
    ||| % {
      namespace: namespace,
    },
  },
}
