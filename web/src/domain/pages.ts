export const pages = [
  {
    id: "overview",
    panels: [],
  },
  {
    id: "advisor",
    panels: [],
  },
  {
    id: "eks",
    panels: [
      {
        title: "EKS overview",
        id: "eksOverviewTable",
      },
    ],
  },
  {
    id: "eks-upgrades",
    panels: [
      {
        title: "Upgrade / rollback insights",
        id: "eksInsightsTable",
      },
    ],
  },
  {
    id: "eks-compute",
    panels: [
      {
        title: "Compute capacity and declared usage",
        id: "eksCapacityTable",
      },
      {
        title: "Managed nodegroups",
        id: "nodegroupsTable",
      },
      {
        title: "Auto Scaling groups",
        id: "autoScalingGroupsTable",
      },
      {
        title: "EC2 instances",
        id: "eksInstancesTable",
      },
      {
        title: "Nodegroup runtime readiness",
        id: "nodegroupReadinessTable",
      },
    ],
  },
  {
    id: "eks-network",
    panels: [
      {
        title: "Network",
        id: "eksNetworkTable",
      },
      {
        title: "VPC details",
        id: "eksVPCsTable",
      },
      {
        title: "Subnet details",
        id: "eksSubnetsTable",
      },
      {
        title: "Security groups",
        id: "eksSecurityGroupsTable",
      },
      {
        title: "EKS networking evidence",
        id: "eksNetworkDetailsTable",
      },
    ],
  },
  {
    id: "eks-security",
    panels: [
      {
        title: "Security and identity",
        id: "eksSecurityTable",
      },
      {
        title: "Access entries",
        id: "accessEntriesTable",
      },
      {
        title: "Pod Identity associations",
        id: "podIdentityTable",
      },
    ],
  },
  {
    id: "eks-addons",
    panels: [
      {
        title: "Managed add-ons",
        id: "addonsTable",
      },
    ],
  },
  {
    id: "nodes",
    panels: [
      {
        title: "Nodes",
        id: "nodesTable",
      },
    ],
  },
  {
    id: "images",
    panels: [
      {
        title: "Running images",
        id: "imagesTable",
      },
      {
        title: "Running containers",
        id: "containersTable",
      },
    ],
  },
  {
    id: "crds",
    panels: [
      {
        title: "CRD types",
        id: "crdTypesTable",
      },
      {
        title: "CRD instances",
        id: "crdInstancesTable",
      },
    ],
  },
  {
    id: "workloads",
    panels: [
      {
        title: "Workloads",
        id: "workloadsTable",
      },
      {
        title: "Pods",
        id: "podsTable",
      },
    ],
  },
  {
    id: "network",
    panels: [
      {
        title: "Services",
        id: "servicesTable",
      },
      {
        title: "Ingress routes",
        id: "ingressesTable",
      },
      {
        title: "Gateway classes",
        id: "gatewayClassesTable",
      },
      {
        title: "Gateways",
        id: "gatewaysTable",
      },
      {
        title: "Gateway routes",
        id: "gatewayRoutesTable",
      },
      {
        title: "Reference grants",
        id: "referenceGrantsTable",
      },
      {
        title: "Gateway policies",
        id: "gatewayPoliciesTable",
      },
      {
        title: "EndpointSlices",
        id: "endpointsTable",
      },
    ],
  },
  {
    id: "security",
    panels: [
      {
        title: "RBAC roles",
        id: "rbacRolesTable",
      },
      {
        title: "RBAC bindings",
        id: "rbacBindingsTable",
      },
      {
        title: "Admission webhooks",
        id: "admissionWebhooksTable",
      },
    ],
  },
  {
    id: "policies",
    panels: [
      {
        title: "PodDisruptionBudgets",
        id: "pdbTable",
      },
      {
        title: "NetworkPolicies",
        id: "networkPoliciesTable",
      },
      {
        title: "ResourceQuotas",
        id: "resourceQuotasTable",
      },
      {
        title: "LimitRanges",
        id: "limitRangesTable",
      },
    ],
  },
  {
    id: "storage",
    panels: [
      {
        title: "PersistentVolumeClaims",
        id: "pvcTable",
      },
      {
        title: "StorageClasses",
        id: "storageTable",
      },
      {
        title: "CSI drivers",
        id: "csiTable",
      },
      {
        title: "Runtime classes",
        id: "runtimeTable",
      },
    ],
  },
  {
    id: "events",
    panels: [],
  },
] as const;
