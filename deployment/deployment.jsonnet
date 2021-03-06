function(namespace, version, replicas, annotations, affinity, tolerations) {
  apiVersion: 'apps/v1',
  kind: 'Deployment',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc',
    annotations: annotations,
    labels: { app: 'istio-oidc', version: version },
  },
  spec: {
    replicas: replicas,
    selector: { matchLabels: { app: 'istio-oidc' } },
    template: {
      metadata: {
        labels: { app: 'istio-oidc', version: version },
        annotations: annotations {
          'prometheus.io/scrape': 'true',
          'prometheus.io/port': '8081',
        },
      },
      spec: {
        affinity: affinity,
        containers: [
          {
            name: 'istio-oidc',
            image: 'knowitsolutions/istio-oidc:%s' % version,
            args: ['--config=/config/config.yaml'],
            ports: [
              { name: 'grpc', containerPort: 8080 },
              { name: 'http-telemetry', containerPort: 8081 },
            ],
            volumeMounts: [
              { name: 'config', mountPath: '/config' },
            ],
            livenessProbe: { httpGet: { port: 'http-telemetry', path: '/health' } },
            readinessProbe: { httpGet: { port: 'http-telemetry', path: '/ready' } },
            resources: { limits: { cpu: '50m', memory: '128Mi' } },
            securityContext: {
              runAsUser: 1000,
              runAsGroup: 1000,
              readOnlyRootFilesystem: true,
            },
          },
        ],
        serviceAccountName: 'istio-oidc',
        tolerations: tolerations,
        volumes: [
          { name: 'config', configMap: { name: 'istio-oidc' } },
        ],
      },
    },
  },
}
