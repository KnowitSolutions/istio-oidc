function(namespace) {
  apiVersion: 'v1',
  kind: 'Service',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc-discovery',
  },
  spec: {
    clusterIP: 'None',
    selector: { app: 'istio-oidc' },
    ports: [
      { name: 'tcp', port: 8080, targetPort: 'grpc' },
    ],
  },
}
