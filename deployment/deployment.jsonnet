function(namespace, version, annotations) {
  apiVersion: 'apps/v1',
  kind: 'Deployment',
  metadata: {
    namespace: namespace,
    name: 'istio-oidc',
    annotations: annotations,
  },
  spec: {
    selector: { matchLabels: { app: 'istio-oidc' } },
    template: {
      metadata: {
        labels: { app: 'istio-oidc' },
        annotations: annotations,
      },
      spec: {
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
            resources: {
              requests: { cpu: '10m', memory: '64Mi' },
              limits: { cpu: '10m', memory: '64Mi' },
            },
            securityContext: {
              runAsUser: 1000,
              runAsGroup: 1000,
              readOnlyRootFilesystem: true,
            },
          },
        ],
        volumes: [
          { name: 'config', configMap: { name: 'istio-oidc' } },
        ],
      },
    },
  },
}
