local authorizationPolicy = import 'authorization-policy.jsonnet';
local clusterRoleBinding = import 'cluster-role-binding.jsonnet';
local clusterRole = import 'cluster-role.json';
local configMap = import 'config-map.jsonnet';
local customResourceDefinitionAccessPolicy = import 'custom-resource-definition-access-policy.json';
local customResourceDefinitionOpenIDProvider = import 'custom-resource-definition-openid-provider.json';
local deployment = import 'deployment.jsonnet';
local destinationRuleDiscovery = import 'destination-rule-discovery.jsonnet';
local namespace = import 'namespace.jsonnet';
local serviceAccount = import 'service-account.jsonnet';
local serviceDiscovery = import 'service-discovery.jsonnet';
local service = import 'service.jsonnet';

function(
  NAMESPACE,
  VERSION,
  REPLICAS=2,

  ANNOTATIONS={},
  AFFINITY={},
  TOLERATIONS=[],
) [
  namespace(NAMESPACE),

  customResourceDefinitionOpenIDProvider,
  customResourceDefinitionAccessPolicy,
  clusterRole,
  clusterRoleBinding(NAMESPACE),
  serviceAccount(NAMESPACE),

  service(NAMESPACE),
  serviceDiscovery(NAMESPACE),
  destinationRuleDiscovery(NAMESPACE),
  authorizationPolicy(NAMESPACE),

  deployment(NAMESPACE, VERSION, REPLICAS, ANNOTATIONS, AFFINITY, TOLERATIONS),
  configMap(NAMESPACE),
]
