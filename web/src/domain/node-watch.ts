import type { Resource, Snapshot } from "../runtime/types";

export const roundedNodePath =
  "M 71.2 8 Q 62.5 3 53.8 8 L 13.7 31 Q 5 36 5 46 L 5 93 Q 5 103 13.7 108 L 53.8 131 Q 62.5 136 71.2 131 L 111.3 108 Q 120 103 120 93 L 120 46 Q 120 36 111.3 31 Z";
export const nodeStates = {
  ready: { label: "Ready · 未 cordon", color: "var(--green)" },
  cordon: { label: "Ready · Cordoned", color: "var(--amber)" },
  notready: { label: "NotReady", color: "var(--red)" },
  unknown: { label: "Unknown", color: "var(--muted)" },
};
export type NodeState = keyof typeof nodeStates;
export interface WatchNode {
  key: string;
  name: string;
  node: Resource;
  instance?: Resource;
  group: string;
  az: string;
  state: NodeState;
  pods: Resource[];
  nonDS: number | null;
  ds: number | null;
  terminating: number | null;
  template: string;
  target: "match" | "different" | "unknown";
}
export function displayTime(value: unknown) {
  if (!value) return "未采集";
  const date = new Date(String(value));
  return Number.isNaN(date.getTime()) ? "未知时间" : date.toLocaleString();
}
export function collectionStatus(snapshot: Snapshot, resource: string) {
  const records = (snapshot.coverage || []).filter(
    (c) => c.resource === resource,
  );
  if (!records.length)
    return { status: "unknown", reason: "无采集覆盖记录", collectedAt: "" };
  const record = records.find((r) => r.status !== "complete") || records[0];
  return {
    status: String(record.status || "unknown"),
    reason: String(record.reason || ""),
    collectedAt: String(record.collectedAt || ""),
  };
}
function state(node: Resource): NodeState {
  if (node.ready === "False") return "notready";
  if (node.ready !== "True") return "unknown";
  return node.unschedulable ? "cordon" : "ready";
}
function actualTemplate(instance?: Resource) {
  if (
    instance?.launchTemplateSource !== "autoscaling-instance" ||
    !instance.launchTemplateId ||
    !instance.launchTemplateVersion
  )
    return "未采集实例模板";
  return `${instance.launchTemplateId} / v${instance.launchTemplateVersion}`;
}
function targetMatch(
  instance: Resource | undefined,
  asg: Resource | undefined,
): WatchNode["target"] {
  if (
    !instance ||
    !asg ||
    instance.launchTemplateSource !== "autoscaling-instance" ||
    !instance.launchTemplateId ||
    !asg.launchTemplateId ||
    !/^\d+$/.test(instance.launchTemplateVersion || "") ||
    !/^\d+$/.test(asg.launchTemplateVersion || "")
  )
    return "unknown";
  return instance.launchTemplateId === asg.launchTemplateId &&
    instance.launchTemplateVersion === asg.launchTemplateVersion
    ? "match"
    : "different";
}
export function nodeWatch(snapshot: Snapshot) {
  const k = snapshot.kubernetes || {},
    infra = snapshot.eks?.infrastructure || {};
  const instances: Resource[] = infra.instances || [],
    asgs: Resource[] = infra.autoScalingGroups || [];
  const instanceById = new Map(instances.map((i) => [i.instanceId, i]));
  const instanceByName = new Map(
    instances
      .filter((i) => i.kubernetesNodeName)
      .map((i) => [i.kubernetesNodeName, i]),
  );
  const asgByName = new Map(asgs.map((g) => [g.name, g]));
  const podCoverage = collectionStatus(snapshot, "Pods").status;
  const podsKnown =
    podCoverage === "complete" ||
    (Array.isArray(k.pods) && (k.pods.length > 0 || podCoverage === "unknown"));
  const pods: Resource[] = (k.pods || []).filter(
    (p: Resource) => !["Succeeded", "Failed"].includes(p.phase),
  );
  const podsByNode = new Map<string, Resource[]>();
  for (const pod of pods) {
    if (pod.nodeName) {
      const list = podsByNode.get(pod.nodeName) || [];
      list.push(pod);
      podsByNode.set(pod.nodeName, list);
    }
  }
  const joined = new Set<string>();
  const nodes: WatchNode[] = (k.nodes || [])
    .map((node: Resource) => {
      // A mismatched provider ID must not silently join by recycled DNS name.
      const id = node.providerId?.split("/").at(-1);
      const instance = id
        ? instanceById.get(id)
        : instanceByName.get(node.name);
      if (instance) joined.add(instance.instanceId);
      const bound = podsByNode.get(node.name || "") || [];
      const ds = bound.filter((p) =>
        p.ownerReferences?.some((o: Resource) => o.kind === "DaemonSet"),
      ).length;
      return {
        key: node.uid || node.name,
        name: node.name || "Unnamed node",
        node,
        instance,
        group:
          node.labels?.["eks.amazonaws.com/nodegroup"] ||
          instance?.nodegroupName ||
          "未归属 Node Group",
        az:
          node.labels?.["topology.kubernetes.io/zone"] ||
          instance?.availabilityZone ||
          "AZ 未知",
        state: state(node),
        pods: bound,
        nonDS: podsKnown ? bound.length - ds : null,
        ds: podsKnown ? ds : null,
        terminating: podsKnown
          ? bound.filter((p) => p.deletionTimestamp).length
          : null,
        template: actualTemplate(instance),
        target: targetMatch(
          instance,
          asgByName.get(instance?.autoScalingGroupName),
        ),
      };
    })
    .sort(
      (a: WatchNode, b: WatchNode) =>
        a.group.localeCompare(b.group) || a.name.localeCompare(b.name),
    );
  const workloads: Resource[] = (k.workloads || [])
    .filter((w: Resource) =>
      ["Deployment", "StatefulSet", "DaemonSet"].includes(w.kind || ""),
    )
    .map((w: Resource) => ({
      ...w,
      desired:
        w.kind === "DaemonSet"
          ? (w.desiredScheduled ?? 0)
          : (w.replicas ?? null),
      ready: w.readyReplicas ?? 0,
    }))
    .sort(
      (a: Resource, b: Resource) =>
        Number(b.desired !== null && b.ready < b.desired) -
          Number(a.desired !== null && a.ready < a.desired) ||
        `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`),
    );
  return {
    nodes,
    asgs,
    podsKnown,
    pending: podsKnown
      ? pods.filter((p) => p.phase === "Pending" && !p.nodeName).length
      : null,
    terminating: podsKnown
      ? pods.filter((p) => p.deletionTimestamp).length
      : null,
    workloads,
    pdbs: ((k.policies?.podDisruptionBudgetDetails || []) as Resource[]).filter(
      (p: Resource) =>
        (p.disruptionsAllowed ?? 0) === 0 && (p.expectedPods ?? 0) > 0,
    ),
    unregistered: instances.filter(
      (i) =>
        !joined.has(i.instanceId) &&
        !["terminated", "shutting-down"].includes(i.state),
    ),
    activities: [...(infra.scalingActivities || [])].sort(
      (a, b) =>
        Number(!b.endTime) - Number(!a.endTime) ||
        String(b.startTime || "").localeCompare(String(a.startTime || "")),
    ),
  };
}
