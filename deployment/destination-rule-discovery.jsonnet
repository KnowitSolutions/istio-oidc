function(namespace) {
  apiVersion: 'networking.istio.io/v1beta1',
  kind: 'DestinationRule',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc-discovery',
  },
  spec: {
    host: 'istio-oidc-discovery',
    trafficPolicy: {
      tls: { mode: 'ISTIO_MUTUAL' },
    },
  },
}
