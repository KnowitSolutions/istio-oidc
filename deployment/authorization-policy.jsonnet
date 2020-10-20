function(namespace) {
  apiVersion: 'security.istio.io/v1beta1',
  kind: 'AuthorizationPolicy',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc',
  },
  spec: {
    selector: {
      matchLabels: {
        app: 'istio-oidc',
      },
    },
    action: 'ALLOW',
    rules: [
      {
        to: [
          {
            operation: {
              ports: ['8080'],
              paths: ['/envoy.service.auth.v2.Authorization/*'],
            },
          },
        ],
      },
      {
        from: [
          { source: { principals: ['*/ns/%s/sa/istio-oidc' % namespace] } },
        ],
        to: [
          {
            operation: {
              ports: ['8080'],
              paths: ['/github.com.KnowitSolutions.istio_oidc.api.Replication/*'],
            },
          },
        ],
      },
      {
        to: [
          { operation: { ports: ['8081'] } },
        ],
      },
    ],
  },
}
