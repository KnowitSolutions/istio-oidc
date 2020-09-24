function(namespace) {
  apiVersion: 'v1',
  kind: 'Service',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc',
  },
  spec: {
    type: 'None',
    selector: { app: 'istio-oidc' },
    ports: [
      { name: 'grpc', port: 8080, targetPort: 'grpc' },
      { name: 'http-telemetry', port: 8081, targetPort: 'http-telemetry' },
    ],
  },
}
