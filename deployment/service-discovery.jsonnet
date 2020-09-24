function(namespace) {
  apiVersion: 'v1',
  kind: 'Service',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc-discovery',
  },
  spec: {
    clusterIP: 'None',
    publishNotReadyAddresses: true,
    selector: { app: 'istio-oidc' },
    ports: [
      { name: 'grpc', port: 8080, targetPort: 'grpc' },
    ],
  },
}
