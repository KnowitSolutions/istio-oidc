function(namespace) {
  apiVersion: 'v1',
  kind: 'Namespace',
  metadata: {
    name: namespace,
    labels: {
      'istio-injection': 'enabled',
    },
  },
}
