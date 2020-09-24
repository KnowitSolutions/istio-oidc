function(namespace) {
  apiVersion: 'rbac.authorization.k8s.io/v1',
  kind: 'ClusterRoleBinding',
  metadata: {
    name: 'istio-oidc',
  },
  roleRef: {
    apiGroup: 'rbac.authorization.k8s.io',
    kind: 'ClusterRole',
    name: 'istio-oidc',
  },
  subjects: [
    {
      kind: 'ServiceAccount',
      namespace: namespace,
      name: 'istio-oidc',
    },
  ],
}
