local clusterRoleBinding = import 'cluster-role-binding.jsonnet';
local clusterRole = import 'cluster-role.json';
local configMap = import 'config-map.jsonnet';
local customResourceDefinition = import 'custom-resource-definition.json';
local deployment = import 'deployment.jsonnet';
local namespace = import 'namespace.jsonnet';
local serviceAccount = import 'service-account.jsonnet';
local service = import 'service.jsonnet';

function(
  NAMESPACE,
  VERSION,
  ANNOTATIONS={},
  KEYCLOAK_URL
) [
  namespace(NAMESPACE),
  customResourceDefinition,
  clusterRole,
  clusterRoleBinding(NAMESPACE),
  serviceAccount(NAMESPACE),
  service(NAMESPACE),
  configMap(NAMESPACE, KEYCLOAK_URL),
  deployment(NAMESPACE, VERSION, ANNOTATIONS),
]
