function(namespace) {
  apiVersion: 'v1',
  kind: 'ServiceAccount',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc',
  },
}
