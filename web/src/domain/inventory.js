import {
  esc,
  ref,
  arr,
  list,
  chips,
  kv,
  shortDigest,
  join,
  resourceMap,
  nodeLabels,
  taints,
  gatewayListeners,
  gatewayParents,
  gatewayBackends,
  gatewayMatches,
  grantRefs,
  gatewayPolicyTargets,
  rbacRules,
  rbacSubjects,
  webhookClient,
  limitItems,
  workloadUpdated,
  workloadBatch,
} from "./format.js";

/** @param {string[] | null} requestedTables */
export function inventoryModel(
  snapshot,
  eksProjection = {},
  nsFilter = "all",
  requestedTables = null,
) {
  const k = snapshot.kubernetes || {};
  const eks = snapshot.eks || {};
  function roleItems() {
    return [...arr(k.rbac?.roleDetails), ...arr(k.rbac?.clusterRoleDetails)];
  }
  function bindingItems() {
    return [
      ...arr(k.rbac?.roleBindingDetails),
      ...arr(k.rbac?.clusterRoleBindingDetails),
    ];
  }
  function webhookRows() {
    const rows = [];
    arr(k.admissionWebhooks).forEach((c) =>
      arr(c.webhooks).forEach((w) =>
        rows.push({ ...w, configKind: c.kind, configName: ref(c) }),
      ),
    );
    return rows;
  }
  function namespacesOf(row) {
    const found = new Set();
    const visit = (value) => {
      if (!value) return;
      if (Array.isArray(value)) {
        value.forEach(visit);
        return;
      }
      if (typeof value !== "object") return;
      if (typeof value.namespace === "string" && value.namespace)
        found.add(value.namespace);
      if (Array.isArray(value.namespaces))
        value.namespaces.filter(Boolean).forEach((ns) => found.add(ns));
      for (const child of Object.values(value)) visit(child);
    };
    visit(row);
    return [...found];
  }
  function namespaceMatches(row) {
    const namespaces = namespacesOf(row);
    return (
      nsFilter === "all" ||
      namespaces.includes(nsFilter) ||
      namespaces.length === 0
    );
  }
  function filtered(rows) {
    return arr(rows).filter(namespaceMatches);
  }
  const workloadKinds = new Set([
    "deployment",
    "daemonset",
    "statefulset",
    "replicaset",
    "job",
    "cronjob",
  ]);
  const resourceKey = (r) =>
    [
      r?.apiVersion || "",
      (r?.kind || "").toLowerCase(),
      r?.namespace || "",
      r?.name || "",
    ].join("|");
  const workloadKey = (w) => resourceKey(w);
  const sameResource = (a, b) =>
    !!a &&
    !!b &&
    (a.kind || "").toLowerCase() === (b.kind || "").toLowerCase() &&
    (a.namespace || "") === (b.namespace || "") &&
    (a.name || "") === (b.name || "") &&
    (!a.apiVersion || !b.apiVersion || a.apiVersion === b.apiVersion);
  const isWorkload = (obj) =>
    workloadKinds.has((obj?.kind || "").toLowerCase()) && !!obj.name;
  function uniqueBy(values, keyFn = (x) => String(x)) {
    const seen = new Set();
    const out = [];
    arr(values).forEach((value) => {
      const key = keyFn(value);
      if (!key || seen.has(key)) return;
      seen.add(key);
      out.push(value);
    });
    return out;
  }
  function allContainers(w) {
    return [...arr(w.containers), ...arr(w.initContainers)];
  }
  function workloadImages(w) {
    return uniqueBy(
      allContainers(w)
        .map((c) => c.image)
        .filter(Boolean),
    );
  }
  function selectorMatches(selector, labels) {
    const entries = Object.entries(selector || {});
    return (
      entries.length > 0 &&
      entries.every(([key, value]) => (labels || {})[key] === value)
    );
  }
  function servicesForWorkload(w) {
    return arr(k.services).filter(
      (s) =>
        (!s.namespace || !w.namespace || s.namespace === w.namespace) &&
        selectorMatches(s.selector, w.selector),
    );
  }
  function runningContainersForWorkload(w) {
    return arr(k.runningContainers).filter((c) => sameResource(c.workload, w));
  }
  function podsForWorkload(w) {
    const podKeys = new Set(
      runningContainersForWorkload(w).map((c) => `${c.namespace}/${c.pod}`),
    );
    const ownerMatched = arr(k.pods).filter((p) =>
      arr(p.ownerReferences).some((o) => sameResource(o, w)),
    );
    return uniqueBy(
      [
        ...arr(k.pods).filter((p) => podKeys.has(`${p.namespace}/${p.name}`)),
        ...ownerMatched,
      ],
      (p) => `${p.namespace}/${p.name}`,
    );
  }
  function ingressForServices(services) {
    const serviceKeys = new Set(
      services.map((s) => `${s.namespace}/${s.name}`),
    );
    return arr(k.ingresses).filter(
      (i) =>
        arr(i.rules).some((rule) =>
          serviceKeys.has(`${i.namespace}/${rule.serviceName}`),
        ) ||
        arr(i.backends).some((b) =>
          serviceKeys.has(`${b.namespace || i.namespace}/${b.name}`),
        ),
    );
  }
  function gatewayRoutesForServices(services) {
    const serviceKeys = new Set(
      services.map((s) => `${s.namespace}/${s.name}`),
    );
    return arr(k.gatewayRoutes).filter((route) =>
      arr(route.rules).some((rule) =>
        arr(rule.backendRefs).some(
          (b) =>
            (b.kind || "Service") === "Service" &&
            serviceKeys.has(`${b.namespace || route.namespace}/${b.name}`),
        ),
      ),
    );
  }
  function gatewaysForRoutes(routes) {
    const gatewayKeys = new Set();
    routes.forEach((route) =>
      arr(route.parentRefs).forEach((parent) =>
        gatewayKeys.add(
          `${parent.namespace || route.namespace}/${parent.name}`,
        ),
      ),
    );
    return arr(k.gateways).filter((g) =>
      gatewayKeys.has(`${g.namespace}/${g.name}`),
    );
  }
  function pvcRefsForWorkload(w) {
    const refs = arr(w.volumes)
      .filter((v) => v.persistentVolumeClaim)
      .map((v) => ({
        apiVersion: "v1",
        kind: "PersistentVolumeClaim",
        namespace: w.namespace,
        name: v.persistentVolumeClaim,
        via: `volume/${v.name}`,
      }));
    arr(w.volumeClaimTemplates).forEach((t) =>
      refs.push({ ...t, via: "template" }),
    );
    return uniqueBy(refs, (r) => `${r.namespace}/${r.name}/${r.via || ""}`);
  }
  function configRefsForWorkload(w) {
    const refs = [
      ...arr(w.configRefs),
      ...arr(w.secretRefs),
      ...arr(w.imagePullSecretRefs),
    ];
    arr(w.volumes).forEach((v) => {
      if (v.configMap)
        refs.push({
          apiVersion: "v1",
          kind: "ConfigMap",
          namespace: w.namespace,
          name: v.configMap,
        });
      if (v.secret)
        refs.push({
          apiVersion: "v1",
          kind: "Secret",
          namespace: w.namespace,
          name: v.secret,
        });
      arr(v.projectedRefs).forEach((r) => refs.push(r));
    });
    allContainers(w).forEach((c) =>
      refs.push(...arr(c.envConfigRefs), ...arr(c.envSecretRefs)),
    );
    return uniqueBy(refs, resourceKey);
  }
  function detailItems(items, formatter) {
    return arr(items).length
      ? `<div class="dependency-items">${items.map((item) => `<span class="dependency-item">${formatter(item)}</span>`).join("")}</div>`
      : '<div class="dependency-empty">No observed relationship</div>';
  }
  function dependencyRow(label, items, formatter) {
    return `<div class="dependency-row"><div class="dependency-label">${esc(label)}</div>${detailItems(items, formatter)}</div>`;
  }
  function workloadDetailHTML(w) {
    const services = servicesForWorkload(w);
    const routes = gatewayRoutesForServices(services);
    const pods = podsForWorkload(w);
    const containers = runningContainersForWorkload(w);
    const nodes = uniqueBy(
      [
        ...containers.map((c) => c.nodeName),
        ...pods.map((p) => p.nodeName),
      ].filter(Boolean),
    );
    const podIdentity = arr(snapshot.eks?.podIdentityAssociations).filter(
      (i) =>
        i.namespace === w.namespace &&
        i.serviceAccount === w.serviceAccountName,
    );
    const serviceAccount = w.serviceAccountName
      ? [
          {
            kind: "ServiceAccount",
            namespace: w.namespace,
            name: w.serviceAccountName,
          },
        ]
      : [];
    const runtime = w.runtimeClassName
      ? [{ kind: "RuntimeClass", name: w.runtimeClassName }]
      : [];
    const compute = [
      `ready=${w.readyReplicas || 0}/${w.replicas ?? w.desiredScheduled ?? "-"}`,
      workloadUpdated(w),
      workloadBatch(w),
      kv(w.nodeSelector),
    ].filter((v) => v && v !== "-");
    return `<div class="detail-summary"><div><span>Workload</span><strong>${esc(ref(w))}</strong></div><div><span>Replicas</span><strong>${esc(w.readyReplicas || 0)}/${esc(w.replicas ?? w.desiredScheduled ?? "-")}</strong></div><div><span>Namespace</span><strong>${esc(w.namespace || "cluster")}</strong></div></div><div class="dependency-graph">${dependencyRow("Compute", compute, (x) => esc(x))}${dependencyRow("Pods", pods, (p) => esc(ref(p)))}${dependencyRow("Nodes", nodes, (n) => esc(n))}${dependencyRow("Network", services, (s) => `${esc(ref(s))} <span class="muted">${esc(s.type || "")}</span>`)}${dependencyRow("Ingress", ingressForServices(services), (i) => esc(ref(i)))}${dependencyRow("Gateway", [...gatewaysForRoutes(routes), ...routes], (r) => esc(ref(r)))}${dependencyRow("Storage", pvcRefsForWorkload(w), (p) => `${esc(ref(p))}${p.via ? ` <span class="muted">${esc(p.via)}</span>` : ""}`)}${dependencyRow("Identity", [...serviceAccount, ...podIdentity], (i) => (i.roleArn ? `${esc(i.serviceAccount)} <span class="muted">${esc(i.roleArn)}</span>` : esc(ref(i))))}${dependencyRow("Config", configRefsForWorkload(w), (c) => esc(ref(c)))}${dependencyRow("Images", workloadImages(w), (image) => esc(image))}${dependencyRow("Runtime", runtime, (r) => esc(ref(r)))}</div>`;
  }
  /** @type {Record<string, {id:string, headers:string[], rows:Array<{resource:any,cells:string[]}>}>} */
  const tables = {};
  function table(id, headers, rows, type, mapper) {
    if (requestedTables && !requestedTables.includes(id)) return;
    tables[id] = {
      id,
      headers,
      rows: filtered(rows).map((resource) => ({
        resource,
        cells: mapper(resource),
      })),
    };
  }
  function renderTables() {
    table(
      "eksOverviewTable",
      ["Field", "Value"],
      eksProjection.overview,
      "eks",
      (r) => [esc(r.field), esc(r.value)],
    );
    table(
      "eksInsightsTable",
      [
        "Name",
        "Category",
        "Kubernetes",
        "Status",
        "Reason",
        "Recommendation",
        "Affected",
      ],
      eksProjection.insights,
      "eks",
      (r) => [
        esc(r.name),
        esc(r.category),
        esc(r.kubernetesVersion),
        esc(r.status),
        esc(r.reason),
        esc(r.recommendation),
        esc(r.affectedResources),
      ],
    );
    table(
      "eksCapacityTable",
      [
        "Resource",
        "Capacity",
        "Allocatable",
        "Requested by specs",
        "Limits by specs",
      ],
      eksProjection.capacity,
      "eks",
      (r) => [
        esc(r.resource),
        esc(r.capacity),
        esc(r.allocatable),
        esc(r.requestedBySpecs),
        esc(r.limitsBySpecs),
      ],
    );
    table(
      "eksNetworkTable",
      ["Field", "Value"],
      eksProjection.network,
      "eks",
      (r) => [esc(r.field), esc(r.value)],
    );
    table(
      "eksNetworkDetailsTable",
      ["Area", "Source", "Association", "Evidence", "Coverage gap"],
      eksProjection.networkDetails,
      "eks",
      (r) => [
        esc(r.area),
        esc(r.source),
        esc(r.association),
        esc(r.evidence),
        esc(r.coverageGap),
      ],
    );
    table(
      "eksSecurityTable",
      ["Area", "Value"],
      eksProjection.security,
      "eks",
      (r) => [esc(r.field), esc(r.value)],
    );
    table(
      "accessEntriesTable",
      ["Principal", "Type", "User", "Groups", "Policies"],
      eksProjection.accessEntries,
      "eks",
      (r) => [
        esc(r.principal),
        esc(r.type),
        esc(r.user),
        esc(r.groups),
        esc(r.policies),
      ],
    );
    table(
      "podIdentityTable",
      ["Namespace", "ServiceAccount", "Role", "Target role", "Owner"],
      eksProjection.podIdentities,
      "eks",
      (r) => [
        esc(r.namespace),
        esc(r.serviceAccount),
        esc(r.role),
        esc(r.targetRole),
        esc(r.owner),
      ],
    );
    table(
      "addonsTable",
      [
        "Name",
        "Version",
        "Status",
        "Namespace",
        "Target Kubernetes",
        "Compatible versions",
        "Upgrade",
        "IAM",
        "Issues",
      ],
      eksProjection.addons,
      "eks",
      (r) => [
        esc(r.name),
        esc(r.version),
        esc(r.status),
        esc(r.namespace),
        esc(r.targetKubernetes),
        esc(r.compatibleVersions),
        esc(r.upgrade),
        esc(r.iam),
        esc(r.issues),
      ],
    );
    table(
      "nodegroupsTable",
      [
        "Name",
        "Version",
        "Release",
        "Status",
        "AMI",
        "Capacity",
        "Size",
        "Subnets",
        "ASGs",
        "Instances",
        "IAM",
        "Issues",
      ],
      eksProjection.nodegroups,
      "eks",
      (r) => [
        esc(r.name),
        esc(r.version),
        esc(r.release),
        esc(r.status),
        esc(r.ami),
        esc(r.capacity),
        esc(r.size),
        esc(r.subnets),
        esc(r.asgs),
        esc(r.instances),
        esc(r.iam),
        esc(r.issues),
      ],
    );
    table(
      "autoScalingGroupsTable",
      [
        "Name",
        "Nodegroup",
        "Size",
        "AZs",
        "Subnets",
        "Instances",
        "Launch template",
        "Health check",
      ],
      eksProjection.autoScalingGroups,
      "eks",
      (r) => [
        esc(r.name),
        esc(r.nodegroup),
        esc(r.size),
        esc(r.zones),
        esc(r.subnets),
        esc(r.instances),
        esc(r.launchTemplate),
        esc(r.healthCheckType),
      ],
    );
    table(
      "eksInstancesTable",
      [
        "Instance",
        "Name",
        "State",
        "Type",
        "AZ",
        "Private IP",
        "Subnet",
        "Security groups",
        "Nodegroup",
        "ASG",
        "Kubernetes node",
        "Launch template",
        "AMI",
      ],
      eksProjection.instances,
      "eks",
      (r) => [
        esc(r.instanceId),
        esc(r.name),
        esc(r.state),
        esc(r.instanceType),
        esc(r.zone),
        esc(r.privateIp),
        esc(r.subnetId),
        esc(r.securityGroups),
        esc(r.nodegroup),
        esc(r.autoScalingGroup),
        esc(r.kubernetesNode),
        esc(r.launchTemplate),
        esc(r.imageId),
      ],
    );
    table(
      "eksVPCsTable",
      [
        "VPC",
        "CIDR",
        "State",
        "Tenancy",
        "DNS support",
        "DNS hostnames",
        "DHCP options",
        "Tags",
      ],
      eksProjection.vpcs,
      "eks",
      (r) => [
        esc(r.vpcId),
        esc(r.cidr),
        esc(r.state),
        esc(r.tenancy),
        esc(r.dnsSupport),
        esc(r.dnsHostnames),
        esc(r.dhcpOptions),
        esc(r.tags),
      ],
    );
    table(
      "eksSubnetsTable",
      [
        "Subnet",
        "VPC",
        "CIDR",
        "AZ",
        "AZ ID",
        "State",
        "Available IPs",
        "Public IP on launch",
        "Route table",
        "NAT gateways",
        "Tags",
      ],
      eksProjection.subnets,
      "eks",
      (r) => [
        esc(r.subnetId),
        esc(r.vpcId),
        esc(r.cidr),
        esc(r.zone),
        esc(r.zoneId),
        esc(r.state),
        esc(r.availableAddresses),
        esc(r.publicIpOnLaunch),
        esc(r.routeTable),
        esc(r.natGateways),
        esc(r.tags),
      ],
    );
    table(
      "eksSecurityGroupsTable",
      [
        "Security group",
        "Name",
        "Description",
        "VPC",
        "Ingress rules",
        "Egress rules",
        "Tags",
      ],
      eksProjection.securityGroups,
      "eks",
      (r) => [
        esc(r.groupId),
        esc(r.name),
        esc(r.description),
        esc(r.vpcId),
        esc(r.ingress),
        esc(r.egress),
        esc(r.tags),
      ],
    );
    table(
      "nodegroupReadinessTable",
      [
        "Group",
        "Evidence",
        "AZ",
        "Expected version",
        "Expected release",
        "Expected AMI",
        "Launch template",
        "Nodes",
        "Kubelet",
        "OS image",
        "Runtime",
        "Instance types",
        "Readiness",
      ],
      eksProjection.nodegroupReadiness,
      "eks",
      (r) => [
        esc(r.group),
        esc(r.evidence),
        esc(r.zone),
        esc(r.expectedVersion),
        esc(r.expectedRelease),
        esc(r.expectedAmi),
        esc(r.launchTemplate),
        esc(r.nodes),
        esc(r.observedKubelet),
        esc(r.observedOs),
        esc(r.observedRuntime),
        esc(r.observedInstances),
        esc(r.readiness),
      ],
    );
    table(
      "nodesTable",
      [
        "Node",
        "ProviderID",
        "Ready",
        "Schedulable",
        "Kubelet",
        "Runtime",
        "OS",
        "Kernel",
        "Arch",
        "Capacity",
        "Allocatable",
        "Taints",
        "Key labels",
        "Blind spots",
      ],
      k.nodes,
      "node",
      (r) => [
        esc(r.name),
        esc(r.providerId),
        esc(r.ready),
        esc(!r.unschedulable),
        esc(r.kubeletVersion),
        esc(r.containerRuntime),
        esc(r.osImage),
        esc(r.kernelVersion),
        esc(r.architecture),
        esc(resourceMap(r.capacity)),
        esc(resourceMap(r.allocatable)),
        esc(taints(r.taints)),
        esc(nodeLabels(r.labels)),
        esc(join(r.nodeLocalBlindSpots)),
      ],
    );
    table(
      "imagesTable",
      [
        "Image",
        "Pods",
        "Containers",
        "Runtime",
        "Namespaces",
        "Workloads",
        "Image IDs",
      ],
      k.runningImages,
      "image",
      (r) => [
        `<b>${esc(r.image)} <span class="muted">(pods=${r.podCount || 0})</span></b>`,
        esc(r.podCount || 0),
        esc(r.containerCount || 0),
        chips(r.runtimes),
        chips(r.namespaces),
        chips(r.workloads),
        esc(arr(r.imageIds).map(shortDigest).join(", ") || "-"),
      ],
    );
    table(
      "containersTable",
      [
        "Namespace",
        "Pod",
        "Container",
        "Type",
        "Image",
        "Runtime",
        "Node",
        "Workload",
      ],
      k.runningContainers,
      "container",
      (r) => [
        esc(r.namespace),
        esc(r.pod),
        esc(r.container),
        esc(r.containerType),
        esc(r.image),
        esc(r.runtime),
        esc(r.nodeName),
        esc(ref(r.workload)),
      ],
    );
    table(
      "crdTypesTable",
      ["CRD type", "Instances", "Namespaces", "Scope", "Version", "Plural"],
      k.customResourceCounts,
      "crd",
      (r) => [
        `<b>${esc((r.group ? r.group + "/" : "") + (r.kind || "-"))}</b>`,
        esc(r.instanceCount || 0),
        esc(r.namespaceCount || 0),
        esc(r.scope),
        esc(r.version),
        esc(r.plural),
      ],
    );
    table(
      "crdInstancesTable",
      ["Instance", "CRD type", "Version", "CRD", "Owners"],
      k.customResourceInstances,
      "crd",
      (r) => [
        esc(ref(r)),
        esc((r.crdGroup ? r.crdGroup + "/" : "") + (r.crdKind || "-")),
        esc(r.crdVersion),
        esc(r.crdName),
        chips(r.ownerReferences),
      ],
    );
    table(
      "workloadsTable",
      [
        "Workload",
        "Ready",
        "Updated/Scheduled",
        "Batch",
        "Images",
        "ServiceAccount",
        "Runtime",
        "Volumes",
      ],
      k.workloads,
      "workload",
      (r) => [
        esc(ref(r)),
        `${r.readyReplicas || 0}/${r.replicas ?? "-"}`,
        esc(workloadUpdated(r)),
        esc(workloadBatch(r)),
        chips(
          [...arr(r.containers), ...arr(r.initContainers)]
            .map((c) => c.image)
            .filter(Boolean),
        ),
        esc(r.serviceAccountName),
        esc(r.runtimeClassName),
        chips(
          arr(r.volumes).map(
            (v) =>
              `${v.name}:${v.type}${v.persistentVolumeClaim ? ":" + v.persistentVolumeClaim : ""}${v.csi ? ":" + v.csi : ""}`,
          ),
        ),
      ],
    );
    table(
      "podsTable",
      [
        "Pod",
        "Phase",
        "Node",
        "ServiceAccount",
        "Runtime",
        "Images",
        "Volumes",
      ],
      k.pods,
      "pod",
      (r) => [
        esc(ref(r)),
        esc(r.phase),
        esc(r.nodeName),
        esc(r.serviceAccountName),
        esc(r.runtimeClassName),
        chips(
          [
            ...arr(r.containers),
            ...arr(r.initContainers),
            ...arr(r.ephemeralContainers),
          ]
            .map((c) => c.image)
            .filter(Boolean),
        ),
        chips(arr(r.volumes).map((v) => `${v.name}:${v.type}`)),
      ],
    );
    table(
      "servicesTable",
      ["Service", "Type", "Selector", "Ports"],
      k.services,
      "service",
      (r) => [
        esc(ref(r)),
        esc(r.type),
        esc(kv(r.selector)),
        esc(
          arr(r.ports)
            .map(
              (p) =>
                `${p.protocol || "TCP"}/${p.port}→${p.targetPort || p.port}`,
            )
            .join(", ") || "-",
        ),
      ],
    );
    table(
      "ingressesTable",
      ["Ingress", "Class", "Routes"],
      k.ingresses,
      "ingress",
      (r) => [
        esc(ref(r)),
        esc(r.className),
        arr(r.rules)
          .map(
            (x) =>
              `${esc(x.host || "*")}${esc(x.path || "/")} → ${esc(x.serviceName)}:${esc(x.servicePort)}`,
          )
          .join("<br>") || '<span class="muted">-</span>',
      ],
    );
    table(
      "gatewayClassesTable",
      ["GatewayClass", "Controller", "Parameters"],
      k.gatewayClasses,
      "gateway",
      (r) => [esc(r.name), esc(r.controllerName), esc(ref(r.parameters))],
    );
    table(
      "gatewaysTable",
      ["Gateway", "Class", "Listeners", "Addresses"],
      k.gateways,
      "gateway",
      (r) => [
        esc(ref(r)),
        esc(r.className),
        esc(gatewayListeners(r.listeners)),
        esc(join(r.addresses)),
      ],
    );
    table(
      "gatewayRoutesTable",
      ["Route", "Kind", "Matches", "Parents", "Backends"],
      k.gatewayRoutes,
      "gatewayRoute",
      (r) => [
        esc(ref(r)),
        esc(r.kind),
        esc(gatewayMatches(r.rules)),
        esc(gatewayParents(r.parentRefs)),
        esc(gatewayBackends(r.rules)),
      ],
    );
    table(
      "referenceGrantsTable",
      ["ReferenceGrant", "From", "To"],
      k.referenceGrants,
      "gatewayRoute",
      (r) => [esc(ref(r)), esc(grantRefs(r.from)), esc(grantRefs(r.to))],
    );
    table(
      "gatewayPoliciesTable",
      ["Policy", "Kind", "Targets", "Details"],
      k.gatewayPolicies,
      "gatewayRoute",
      (r) => [
        esc(ref(r)),
        esc(r.kind),
        esc(gatewayPolicyTargets(r.targetRefs)),
        esc(join(r.details)),
      ],
    );
    table(
      "rbacRolesTable",
      ["Role", "Kind", "Rules"],
      roleItems(),
      "crd",
      (r) => [esc(ref(r)), esc(r.kind), esc(rbacRules(r.rules))],
    );
    table(
      "rbacBindingsTable",
      ["Binding", "Kind", "RoleRef", "Subjects"],
      bindingItems(),
      "crd",
      (r) => [
        esc(ref(r)),
        esc(r.kind),
        esc(ref(r.roleRef)),
        esc(rbacSubjects(r.subjects)),
      ],
    );
    table(
      "admissionWebhooksTable",
      [
        "Configuration",
        "Kind",
        "Webhook",
        "Client",
        "Operations",
        "Resources",
        "Failure",
        "Timeout",
      ],
      webhookRows(),
      "crd",
      (r) => [
        esc(r.configName),
        esc(r.configKind),
        esc(r.name),
        esc(webhookClient(r)),
        esc(join(r.operations)),
        esc(join(r.resources)),
        esc(r.failurePolicy),
        esc(r.timeoutSeconds),
      ],
    );
    table(
      "pdbTable",
      ["PDB", "Selector", "Min", "Max", "Allowed", "Healthy"],
      k.policies?.podDisruptionBudgetDetails,
      "crd",
      (r) => [
        esc(ref(r)),
        esc(kv(r.selector)),
        esc(r.minAvailable),
        esc(r.maxUnavailable),
        esc(r.disruptionsAllowed),
        esc(
          `${r.currentHealthy || 0}/${r.desiredHealthy || 0} expected=${r.expectedPods || 0}`,
        ),
      ],
    );
    table(
      "networkPoliciesTable",
      ["NetworkPolicy", "Pod selector", "Types", "Rules", "Peers"],
      k.policies?.networkPolicyDetails,
      "crd",
      (r) => [
        esc(ref(r)),
        esc(kv(r.podSelector)),
        esc(join(r.policyTypes)),
        esc(`ingress=${r.ingressRules || 0} egress=${r.egressRules || 0}`),
        esc(`ingress=${r.ingressPeers || 0} egress=${r.egressPeers || 0}`),
      ],
    );
    table(
      "resourceQuotasTable",
      ["ResourceQuota", "Hard", "Used"],
      k.policies?.resourceQuotaDetails,
      "crd",
      (r) => [esc(ref(r)), esc(kv(r.hard)), esc(kv(r.used))],
    );
    table(
      "limitRangesTable",
      ["LimitRange", "Items"],
      k.policies?.limitRangeDetails,
      "crd",
      (r) => [esc(ref(r)), esc(limitItems(r.items))],
    );
    table(
      "endpointsTable",
      ["EndpointSlice", "Service", "Address type", "Ports", "Endpoints"],
      k.endpointSlices,
      "endpoint",
      (r) => [
        esc(ref(r)),
        esc(r.serviceName),
        esc(r.addressType),
        esc(
          arr(r.ports)
            .map(
              (p) => `${p.name || "-"}:${p.port || "-"}/${p.protocol || "-"}`,
            )
            .join(", ") || "-",
        ),
        esc(
          arr(r.endpoints)
            .map((e) => `${arr(e.addresses).join(",")} ${e.nodeName || ""}`)
            .join(" | ") || "-",
        ),
      ],
    );
    table(
      "pvcTable",
      ["PVC", "StorageClass", "Access modes", "Requested", "Volume", "Phase"],
      k.persistentVolumeClaims,
      "pvc",
      (r) => [
        esc(ref(r)),
        esc(r.storageClassName),
        chips(r.accessModes),
        esc(r.requestedStorage),
        esc(r.volumeName),
        esc(r.phase),
      ],
    );
    table(
      "storageTable",
      [
        "StorageClass",
        "Provisioner",
        "Binding",
        "Reclaim",
        "Expansion",
        "Parameters",
      ],
      k.storageClasses,
      "storage",
      (r) => [
        esc(r.name),
        esc(r.provisioner),
        esc(r.volumeBindingMode),
        esc(r.reclaimPolicy),
        esc(r.allowVolumeExpansion),
        esc(kv(r.parameters)),
      ],
    );
    table(
      "csiTable",
      ["CSI Driver", "Attach required", "Pod info", "Lifecycle"],
      k.csiDrivers,
      "csi",
      (r) => [
        esc(r.name),
        esc(r.attachRequired),
        esc(r.podInfoOnMount),
        chips(r.volumeLifecycleModes),
      ],
    );
    table(
      "runtimeTable",
      ["RuntimeClass", "Handler"],
      k.runtimeClasses,
      "runtime",
      (r) => [esc(ref(r)), esc(r.handler)],
    );
  }
  renderTables();
  return {
    tables,
    workloadDetailHTML,
    workloadImages,
    configRefsForWorkload,
    isWorkload,
    workloadKey,
    filtered,
  };
}
