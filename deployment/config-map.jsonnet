function(namespace, keycloak_url) {
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
          Service: tcp
          Domain: istio-oidc-discovery
      Keycloak:
        URL: %(keycloak_url)s
    ||| % {
      namespace: namespace,
      keycloak_url: keycloak_url,
    },
  },
}
