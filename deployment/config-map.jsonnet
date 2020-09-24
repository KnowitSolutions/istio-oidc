function(namespace, keycloak_url) {
  apiVersion: 'v1',
  kind: 'ConfigMap',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc',
  },
  data: {
    'config.yaml': |||
      ExtAuthz:
        ClusterName: istio-oidc
      Replication:
        Mode: dns
        PeerAddress:
          Service: grpc
          Domain: istio-oidc-discovery
      Keycloak:
        URL: %(keycloak_url)s
    ||| % {
      keycloak_url: keycloak_url,
    },
  },
}
