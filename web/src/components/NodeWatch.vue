<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from "vue";
import type { Resource, Snapshot } from "../runtime/types";
import {
  collectionStatus,
  displayTime,
  nodeStates,
  nodeWatch,
  roundedNodePath,
  type WatchNode,
} from "../domain/node-watch";
import { sourceLabel, sourceDetail } from "../runtime/source-status";
const props = defineProps<{
  snapshot: Snapshot;
  sources: Record<string, Resource>;
  mode: string;
  paused: boolean;
  loading: boolean;
  error: string;
}>();
const model = computed(() => nodeWatch(props.snapshot));
const group = ref("all"),
  search = ref(""),
  selectedKey = ref(""),
  limit = ref(300);
const tooltip = ref<{ node: WatchNode; left: number; top: number } | null>(
  null,
);
watch(
  () => model.value.nodes,
  (nodes) => {
    if (!selectedKey.value && nodes.length)
      selectedKey.value = (
        nodes.find((n) => n.state === "cordon") || nodes[0]
      ).key;
    tooltip.value = null;
  },
  { immediate: true },
);
watch([group, search], () => {
  limit.value = 300;
  tooltip.value = null;
});
const groups = computed(() => [
  ...new Set(model.value.nodes.map((n) => n.group)),
]);
const filtered = computed(() =>
  model.value.nodes.filter(
    (n) =>
      (group.value === "all" || n.group === group.value) &&
      `${n.name} ${n.az} ${n.group} ${n.instance?.instanceId || ""}`
        .toLowerCase()
        .includes(search.value.toLowerCase().trim()),
  ),
);
const visibleGroups = computed(() =>
  groups.value
    .map((name) => ({
      name,
      nodes: filtered.value
        .slice(0, limit.value)
        .filter((n) => n.group === name),
      asgs: model.value.asgs.filter((g) => g.nodegroupName === name),
    }))
    .filter((g) => g.nodes.length),
);
const selected = computed(() =>
  model.value.nodes.find((n) => n.key === selectedKey.value),
);
const activities = computed(() =>
  model.value.activities.filter(
    (a) => group.value === "all" || a.nodegroupName === group.value,
  ),
);
const activityCoverage = computed(() =>
  collectionStatus(props.snapshot, "DescribeScalingActivities"),
);
const coverage = computed(() =>
  [
    "Nodes",
    "Pods",
    "Nodegroups",
    "DescribeInstances",
    "DescribeAutoScalingGroups",
    "Deployments",
    "StatefulSets",
    "DaemonSets",
    "PodDisruptionBudgets",
  ]
    .map((resource) => ({
      resource,
      ...collectionStatus(props.snapshot, resource),
    }))
    .filter((c) => c.status !== "complete"),
);
const formatCount = (n: number | null) => (n === null ? "—" : n);
const metrics = computed(() => {
  const nodes = model.value.nodes,
    known = nodes.filter((n) => n.target !== "unknown"),
    match = nodes.filter((n) => n.target === "match" && n.state === "ready");
  return [
    [
      "Ready on target template",
      known.length
        ? `${match.length} / ${nodes.length}`
        : `— / ${nodes.length}`,
      `${known.length} verifiable nodes · ${nodes.length - known.length} unknown`,
    ],
    [
      "Ready nodes",
      `${nodes.filter((n) => n.node.ready === "True").length} / ${nodes.length}`,
      "Cluster-wide · includes cordoned nodes",
    ],
    [
      "Cordoned",
      nodes.filter((n) => n.node.unschedulable).length,
      "Not accepting new scheduling",
    ],
    [
      "non-DS / DS Pods",
      model.value.podsKnown
        ? `${nodes.reduce((s, n) => s + n.nonDS!, 0)} / ${nodes.reduce((s, n) => s + n.ds!, 0)}`
        : "— / —",
      "Bound, non-terminal · includes terminating",
    ],
    [
      "Pending / Terminating",
      `${formatCount(model.value.pending)} / ${formatCount(model.value.terminating)}`,
      "Cluster-wide · unbound pending / terminating",
    ],
  ];
});
const modeLabel = computed(() =>
  Object.values(props.sources).some((s) => s.mode === "recorded")
    ? "Recorded · historical snapshot"
    : props.mode === "offline"
      ? "Offline · saved report"
      : "Live · collecting updates",
);
function showTooltip(event: Event, node: WatchNode) {
  positionTooltip(event.currentTarget as HTMLElement, node);
}
function positionTooltip(element: HTMLElement, node: WatchNode) {
  const rect = element.getBoundingClientRect();
  tooltip.value = {
    node,
    left: Math.max(8, Math.min(window.innerWidth - 290, rect.right + 10)),
    top: Math.max(8, Math.min(window.innerHeight - 260, rect.top)),
  };
}
function hideTooltip() {
  const active = document.activeElement;
  if (
    tooltip.value &&
    active instanceof HTMLElement &&
    active.matches("button.node") &&
    active.dataset.node === tooltip.value.node.name
  ) {
    positionTooltip(active, tooltip.value.node);
    return;
  }
  tooltip.value = null;
}
onMounted(() => {
  window.addEventListener("scroll", hideTooltip, true);
  window.addEventListener("resize", hideTooltip);
});
onUnmounted(() => {
  window.removeEventListener("scroll", hideTooltip, true);
  window.removeEventListener("resize", hideTooltip);
});
function selectNode(node: WatchNode) {
  selectedKey.value = node.key;
  tooltip.value = null;
}
function info(node: WatchNode) {
  return [
    ["Node Group", node.group],
    ["AMI ID", node.instance?.imageId || "Not collected"],
    ["Instance launch template", node.template],
    ["EC2 launch time", displayTime(node.instance?.launchTime)],
    ["Node registration time", displayTime(node.node.createdAt)],
    [
      "Kubelet / Instance",
      `${node.node.kubeletVersion || "Unknown"} / ${node.instance?.instanceType || node.node.labels?.["node.kubernetes.io/instance-type"] || "Unknown"}`,
    ],
    ["Instance ID", node.instance?.instanceId || "Not linked"],
    ["ASG", node.instance?.autoScalingGroupName || "Not linked"],
    ["ASG lifecycle", node.instance?.lifecycleState || "Not collected"],
    [
      "Cordon",
      node.node.unschedulable ? "spec.unschedulable = true" : "Uncordoned",
    ],
    [
      "Taints",
      node.node.taints
        ?.map(
          (t: Resource) =>
            `${t.key}${t.value ? "=" + t.value : ""}:${t.effect}`,
        )
        .join(", ") || "None",
    ],
  ];
}
</script>
<template>
  <div class="watch" @keydown.esc="tooltip = null">
    <div class="intro">
      <div>
        <div class="eyebrow">EKS / UPGRADES / NODE WATCH</div>
        <h1>Node upgrade watch</h1>
        <p>
          Monitor node replacement, Pod migration, and capacity recovery.
          Compare actual instance configuration with the ASG target template.
        </p>
      </div>
      <div class="source-summary">
        <span class="badge">{{ modeLabel }}</span
        ><span v-if="paused" class="badge warn">Paused · updates paused</span
        ><span v-if="loading" class="badge">Updating</span>
        <p>Snapshot · {{ displayTime(snapshot.collectedAt) }}</p>
      </div>
    </div>
    <div class="row source-row">
      <span class="badge">Read-only · cluster-wide</span>
      <div class="source-details">
        <span
          v-for="(source, name) in sources"
          :key="name"
          :class="{
            warn: ['error', 'partial', 'stale'].includes(source.state),
          }"
          :title="
            sourceDetail(source) + (source.error ? ` · ${source.error}` : '')
          "
          >{{ name }} · {{ sourceLabel(source) }} · published
          {{ displayTime(source.lastSuccess) }}</span
        >
      </div>
    </div>
    <p v-if="error" role="alert" class="notice">
      Update failed. Showing the last available snapshot:{{ error }}
    </p>
    <details v-if="coverage.length" class="coverage-note">
      <summary>
        Incomplete coverage ·
        {{ coverage.map((c) => c.resource).join(" / ") }} (counts reflect
        observed objects only)
      </summary>
      <p v-for="c in coverage" :key="c.resource">
        {{ c.resource }} · {{ c.status }} ·
        {{ c.reason || "Collection incomplete" }} ·
        {{ displayTime(c.collectedAt) }}
      </p>
    </details>
    <div class="stats">
      <div v-for="metric in metrics" :key="metric[0]" class="metric">
        <label>{{ metric[0] }}</label
        ><strong>{{ metric[1] }}</strong
        ><small>{{ metric[2] }}</small>
      </div>
    </div>
    <div class="main-grid">
      <section class="box" aria-label="Node map">
        <div class="section-head">
          <h2>Node map <span class="badge">Grouped by Node Group</span></h2>
          <div class="controls">
            <select v-model="group" aria-label="Filter node group">
              <option value="all">All Node Groups</option>
              <option
                v-if="group !== 'all' && !groups.includes(group)"
                :value="group"
              >
                {{ group }} (no nodes remaining)
              </option>
              <option v-for="name in groups" :key="name">
                {{ name }}
              </option></select
            ><input
              v-model="search"
              aria-label="Search nodes"
              placeholder="Search node / AZ / instance"
              size="22"
            />
          </div>
        </div>
        <div class="legend">
          <span
            v-for="(state, id) in nodeStates"
            :key="id"
            :style="{ '--s': state.color }"
            >{{ state.label }}</span
          ><span>Large: non-DS Pods / Small: DS Pods</span>
        </div>
        <div class="map-body">
          <div v-for="g in visibleGroups" :key="g.name" class="group">
            <div class="group-title">
              <b
                >{{ g.name }}
                <span class="badge">{{ g.nodes.length }} nodes</span></b
              ><span>{{
                g.asgs
                  .map(
                    (a) =>
                      `${a.name} · desired ${a.desiredCapacity} · target ${a.launchTemplateId || "Unknown"} / ${a.launchTemplateVersion || "Unknown"}`,
                  )
                  .join("; ") || "ASG target configuration not collected"
              }}</span>
            </div>
            <div class="honeycomb">
              <button
                v-for="node in g.nodes"
                :key="node.key"
                class="node"
                :class="{ selected: node.key === selectedKey }"
                :style="{ '--s': nodeStates[node.state].color }"
                :data-node="node.name"
                :aria-pressed="node.key === selectedKey"
                :aria-describedby="
                  tooltip?.node.key === node.key
                    ? 'node-watch-tooltip'
                    : undefined
                "
                :aria-label="`${node.name}, ${formatCount(node.nonDS)} non-DS Pods, ${formatCount(node.ds)} DS Pods, ${nodeStates[node.state].label}, ${node.node.unschedulable ? 'Cordoned' : 'Uncordoned'}`"
                @click="selectNode(node)"
                @mouseenter="showTooltip($event, node)"
                @mouseleave="tooltip = null"
                @focus="showTooltip($event, node)"
                @blur="tooltip = null"
              >
                <svg viewBox="0 0 125 140" aria-hidden="true">
                  <path :d="roundedNodePath" />
                </svg>
                <span class="node-content"
                  ><span class="node-id" :title="node.name">{{
                    node.name
                  }}</span
                  ><span class="pod-count"
                    ><strong>{{ formatCount(node.nonDS) }}</strong
                    ><span>/ {{ formatCount(node.ds) }}</span></span
                  ><span class="node-state">{{
                    nodeStates[node.state].label
                  }}</span
                  ><span class="node-version"
                    >{{
                      node.instance?.launchTemplateSource ===
                        "autoscaling-instance" &&
                      node.instance?.launchTemplateVersion
                        ? "LT v" + node.instance.launchTemplateVersion
                        : "LT unknown"
                    }}
                    · {{ node.az.split("-").at(-1) }}</span
                  ><span
                    v-if="
                      node.node.unschedulable &&
                      !['ready', 'cordon'].includes(node.state)
                    "
                    class="node-extra"
                    >Cordoned</span
                  ><span v-else-if="node.terminating" class="node-extra"
                    >{{ node.terminating }} Terminating</span
                  ><span v-else-if="node.node.taints?.length" class="node-extra"
                    >{{ node.node.taints.length }} taints</span
                  ></span
                >
              </button>
            </div>
          </div>
          <p v-if="!filtered.length" class="empty">
            {{
              model.nodes.length
                ? "No matching nodes."
                : "No node data in this snapshot. Check Kubernetes collection scope and permissions."
            }}
          </p>
          <button
            v-if="filtered.length > limit"
            class="control"
            @click="limit += 300"
          >
            Show more ({{ limit }} / {{ filtered.length }} shown)
          </button>
        </div>
        <div class="map-foot">
          Hover or focus a node for information; click to pin details. Ready and
          uncordoned does not guarantee scheduling: check taints, resources, and
          Pod constraints.
        </div>
      </section>
      <section class="box inspector" aria-label="Selected node details">
        <div class="section-head">
          <h2>Node details</h2>
          <span class="badge">Click to pin</span>
        </div>
        <div v-if="selected" class="detail">
          <strong class="node-name">{{ selected.name }}</strong>
          <p>{{ selected.az }}</p>
          <div class="status-line">
            <span
              class="badge"
              :style="{ color: nodeStates[selected.state].color }"
              >{{ nodeStates[selected.state].label }}</span
            ><span class="badge">{{
              selected.target === "match"
                ? "Matches target template"
                : selected.target === "different"
                  ? "Differs from target template"
                  : "Target template unknown"
            }}</span>
          </div>
          <dl>
            <div v-for="[label, value] in info(selected)" :key="label">
              <dt>{{ label }}</dt>
              <dd>{{ value }}</dd>
            </div>
          </dl>
          <div class="notice">
            {{
              selected.node.unschedulable
                ? `${formatCount(selected.nonDS)} non-DS Pods remain on this node; ${formatCount(selected.terminating)} Pods are terminating. Cordon alone does not indicate that draining has started or completed.`
                : "Node status comes from the latest Kubernetes observation. AWS lifecycle and node readiness are shown separately."
            }}
          </div>
          <details v-if="selected.pods.length" class="pod-details">
            <summary>Bound Pods ({{ selected.pods.length }})</summary>
            <p
              v-for="pod in selected.pods"
              :key="pod.uid || `${pod.namespace}/${pod.name}`"
            >
              {{ pod.namespace }}/{{ pod.name }} ·
              {{
                pod.deletionTimestamp ? "Terminating" : pod.phase || "Unknown"
              }}
            </p>
          </details>
        </div>
        <p v-else class="empty">
          {{
            selectedKey
              ? "The selected node is no longer in this snapshot. Select another node."
              : "Select a node to view details."
          }}
        </p>
      </section>
    </div>
    <div v-if="model.unregistered.length" class="box unregistered">
      <h2>
        Instances not linked to a Kubernetes Node
        <span class="badge">{{ model.unregistered.length }}</span>
      </h2>
      <p>
        Instances may be starting, registering, or leaving the cluster. AWS and
        Kubernetes collection times can also differ.
      </p>
      <div class="instance-list">
        <span
          v-for="instance in model.unregistered"
          :key="instance.instanceId"
          class="badge"
          >{{ instance.instanceId }} ·
          {{ instance.nodegroupName || "Unknown node group" }} ·
          {{ instance.state || "Unknown" }} ·
          {{ instance.lifecycleState || "Lifecycle unknown" }}</span
        >
      </div>
    </div>
    <div class="lower">
      <section class="box">
        <div class="section-head">
          <h2>ASG activity</h2>
          <span class="badge">{{
            group === "all" ? "All node groups" : group
          }}</span>
        </div>
        <div class="activity">
          <p class="collection-label">
            AWS · {{ displayTime(activityCoverage.collectedAt) }} ·
            {{ activityCoverage.status }}
          </p>
          <p v-if="activityCoverage.status !== 'complete'" class="notice">
            {{
              activityCoverage.reason ||
              "Activity not collected. Requires autoscaling:DescribeScalingActivities permission."
            }}
          </p>
          <div class="activity-scroll">
            <div v-for="a in activities" :key="a.activityId" class="event">
              <time :title="displayTime(a.startTime)">{{
                a.startTime
                  ? new Date(a.startTime).toLocaleTimeString("en-US")
                  : "—"
              }}</time>
              <div>
                {{ a.description || a.activityId
                }}<small>{{ a.autoScalingGroupName }}</small
                ><small>{{ a.statusMessage || a.cause }}</small>
              </div>
              <span
                class="event-status"
                :class="{
                  warn: ['Failed', 'Cancelled'].includes(a.statusCode),
                }"
                >{{ a.statusCode || "Unknown"
                }}<small v-if="a.progress != null"
                  >{{ a.progress }}%</small
                ></span
              >
            </div>
          </div>
          <p v-if="!activities.length" class="empty">
            {{
              activityCoverage.status === "complete"
                ? "No matching ASG activity in this collection."
                : "No ASG activity data available."
            }}
          </p>
        </div>
      </section>
      <section class="box">
        <div class="section-head">
          <h2>Workload continuity</h2>
          <span class="badge">Cluster-wide</span>
        </div>
        <div class="workload-scroll">
          <table>
            <thead>
              <tr>
                <th>Workload</th>
                <th>Ready / Desired</th>
                <th>Observation</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="w in model.workloads"
                :key="`${w.kind}/${w.namespace}/${w.name}`"
              >
                <td>
                  {{ w.name }}<small>{{ w.namespace }} · {{ w.kind }}</small>
                </td>
                <td>{{ w.ready }} / {{ w.desired ?? "—" }}</td>
                <td
                  :class="
                    w.desired !== null && w.ready < w.desired ? 'warn' : 'muted'
                  "
                >
                  {{
                    w.desired === null
                      ? "Desired count unknown"
                      : w.ready < w.desired
                        ? "Below desired readiness"
                        : "Ready count meets desired"
                  }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="!model.workloads.length" class="empty">
          No Deployment, StatefulSet, or DaemonSet data in this snapshot.
        </p>
        <div class="pdb-note">
          <details v-if="model.pdbs.length">
            <summary class="warn">
              PDBs with no available disruption budget: {{ model.pdbs.length }}
            </summary>
            <p v-for="pdb in model.pdbs" :key="`${pdb.namespace}/${pdb.name}`">
              {{ pdb.namespace }}/{{ pdb.name }} · healthy
              {{ pdb.currentHealthy ?? 0 }} / desired
              {{ pdb.desiredHealthy ?? 0 }}
            </p>
          </details>
          <p>
            PDB budgets indicate risk, not proof of a blocked eviction. Unbound
            Pending Pods are counted separately.
          </p>
        </div>
      </section>
    </div>
    <p class="map-footnote">
      Counts include bound, non-terminal Pods, including terminating Pods. DS
      ownership is identified by DaemonSet ownerReference. Target comparison
      uses actual instance templates and explicit numeric versions; unknown
      configurations do not count as matches. Workload counts reflect the
      snapshot only.
    </p>
    <div
      v-if="tooltip"
      id="node-watch-tooltip"
      class="tooltip"
      role="tooltip"
      :style="{ left: tooltip.left + 'px', top: tooltip.top + 'px' }"
    >
      <strong>{{ tooltip.node.name }}</strong
      ><br />Node Group: {{ tooltip.node.group }}<br />AMI:
      {{ tooltip.node.instance?.imageId || "Not collected" }}<br />Launch
      Template: {{ tooltip.node.template }}<br />EC2 launch:
      {{ displayTime(tooltip.node.instance?.launchTime) }}<br />{{
        tooltip.node.az
      }}
      · {{ nodeStates[tooltip.node.state].label }}<br /><span class="muted"
        >Click for full details</span
      >
    </div>
  </div>
</template>

<style scoped>
button,
select,
input {
  font: inherit;
}
button {
  cursor: pointer;
}
button:focus-visible,
select:focus-visible,
input:focus-visible {
  outline: 2px solid var(--cyan);
  outline-offset: 4px;
}
.watch {
  display: grid;
  gap: 16px;
  padding: 10px 8px;
}
.intro,
.row,
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.eyebrow {
  font-size: 11px;
  letter-spacing: 0.15em;
  color: var(--cyan);
  text-transform: uppercase;
}
h1 {
  font-size: 27px;
  margin: 8px 0;
}
h2 {
  font-size: 15px;
  margin: 0;
}
h3 {
  font-size: 13px;
  margin: 0;
}
p {
  color: var(--muted);
  font-size: 12px;
  line-height: 1.7;
  margin: 6px 0;
}
.badge {
  display: inline-block;
  padding: 4px 8px;
  border: 1px solid var(--line);
  border-radius: 5px;
  font-size: 11px;
  color: var(--muted);
}

.controls {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
button.control,
select,
input {
  background: var(--control-bg);
  border: 1px solid var(--line);
  border-radius: 5px;
  color: var(--text);
  padding: 9px 11px;
  font-size: 12px;
}

.stats {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px;
}
.metric {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 16px;
}
.metric label {
  font-size: 12px;
  color: var(--muted);
}
.metric strong {
  display: block;
  font-size: 28px;
  margin: 8px 0 4px;
  letter-spacing: -1px;
}
.metric small {
  color: var(--muted);
  font-size: 11px;
}
.main-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 300px;
  gap: 14px;
}
.box {
  border: 1px solid var(--line);
  background: var(--panel);
  border-radius: 8px;
  overflow: hidden;
}
.section-head {
  padding: 17px 19px;
  border-bottom: 1px solid var(--line);
}
.map-body {
  padding: 18px;
  background-image: radial-gradient(var(--line) 0.7px, transparent 0.7px);
  background-size: 16px 16px;
}
.legend {
  position: static;
  inset: auto;
  margin: 0;
  background: var(--panel);
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  font-size: 11px;
  color: var(--muted);
  padding: 12px 19px;
  border-bottom: 1px solid var(--line);
}
.legend span {
  padding: 0;
  border: 0;
  background: none;
  border-radius: 0;
}
.legend span:before {
  content: "";
  display: inline-block;
  width: 7px;
  height: 7px;
  background: var(--s);
  border-radius: 50%;
  margin-right: 6px;
}
.group {
  padding: 14px 4px 20px;
}
.group + .group {
  border-top: 1px solid var(--line);
}
.group-title {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  color: var(--muted);
  font-size: 11px;
  margin-bottom: 15px;
}
.group-title b {
  font-size: 13px;
  color: var(--text);
}
.honeycomb {
  display: flex;
  flex-wrap: wrap;
  gap: 9px;
}
.node {
  position: relative;
  width: 125px;
  height: 140px;
  border: 0;
  padding: 0;
  background: none;
  color: var(--text);
  flex-shrink: 0;
}
.node svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
}
.node path {
  fill: var(--panel-2);
  stroke: var(--s);
  stroke-width: 1.4;
  stroke-linejoin: round;
  transition:
    fill 160ms ease,
    stroke-width 160ms ease;
}
.node:hover path,
.node:focus-visible path,
.node.selected path {
  fill: var(--panel-3);
  stroke-width: 2.4;
}
.node-content {
  position: relative;
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 5px;
  font-size: 10px;
}
.node-id {
  color: var(--muted);
  font-family: monospace;
}
.pod-count {
  display: flex;
  align-items: baseline;
  gap: 7px;
}
.pod-count strong {
  font-size: 28px;
  line-height: 1.1;
}
.pod-count span {
  font-size: 14px;
  color: var(--muted);
}
.node-state {
  color: var(--s);
  font-size: 10px;
}
.node-version {
  font-size: 9px;
  color: var(--muted);
}
.map-foot {
  padding: 12px 19px;
  border-top: 1px solid var(--line);
  font-size: 11px;
  color: var(--muted);
}
.detail {
  padding: 18px;
}
.detail dl {
  margin: 16px 0;
  display: grid;
  gap: 14px;
}
.detail dt {
  font-size: 10px;
  color: var(--muted);
  margin-bottom: 4px;
}
.detail dd {
  margin: 0;
  font-size: 12px;
  overflow-wrap: anywhere;
}
.detail .status-line {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.notice {
  padding: 12px;
  border-left: 2px solid var(--amber);
  background: var(--panel-2);
  font-size: 12px;
  line-height: 1.7;
}
.lower {
  display: grid;
  grid-template-columns: 1.15fr 1fr;
  gap: 14px;
}
.activity {
  padding: 0 18px;
}
.event {
  display: grid;
  grid-template-columns: 55px 1fr auto;
  gap: 12px;
  padding: 14px 0;
  border-bottom: 1px solid var(--line);
  font-size: 12px;
}
.event:last-child {
  border: 0;
}
.event time {
  font-family: monospace;
  color: var(--muted);
  font-size: 11px;
}
.event small {
  display: block;
  color: var(--muted);
  margin-top: 5px;
  font-size: 11px;
}
.event-status {
  color: var(--cyan);
  font-size: 10px;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
td,
th {
  text-align: left;
  padding: 13px 18px;
  border-bottom: 1px solid var(--line);
}
th {
  font-size: 10px;
  color: var(--muted);
  font-weight: 500;
}
.tooltip {
  position: fixed;
  z-index: 10;
  pointer-events: none;
  width: 270px;
  padding: 14px;
  background: var(--drawer-bg);
  border: 1px solid var(--line-strong);
  border-radius: 7px;
  box-shadow: var(--shadow);
  font-size: 11px;
  line-height: 1.9;
}
.tooltip strong {
  color: var(--cyan);
}

.empty {
  padding: 35px;
  text-align: center;
}
.good {
  color: var(--green);
}
.warn {
  color: var(--warning-text);
}
.muted {
  color: var(--muted);
}
@media (min-width: 1600px) {
  .node {
    width: 140px;
    height: 153px;
  }
  .main-grid {
    grid-template-columns: minmax(0, 1fr) 330px;
  }
}
@media (max-width: 1000px) {
  .main-grid {
    grid-template-columns: 1fr;
  }
  .stats {
    grid-template-columns: repeat(3, 1fr);
  }
  .lower {
    grid-template-columns: 1fr;
  }
  .detail dl {
    grid-template-columns: repeat(3, 1fr);
  }
}
@media (max-width: 620px) {
  .intro,
  .row {
    align-items: flex-start;
    flex-direction: column;
  }
  .stats {
    grid-template-columns: repeat(2, 1fr);
  }
  .detail dl {
    grid-template-columns: 1fr 1fr;
  }

  .watch {
    padding: 0;
  }
  .node {
    width: 112px;
    height: 130px;
  }

  .section-head {
    flex-wrap: wrap;
  }
  .group-title {
    flex-wrap: wrap;
  }

  input {
    width: 100%;
  }
}

.source-summary {
  text-align: right;
}
.source-summary .badge {
  margin-left: 6px;
}
.source-row {
  min-height: 40px;
}
.source-details {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  color: var(--muted);
  font-size: 11px;
}
.coverage-note {
  padding: 10px 14px;
  border: 1px solid var(--line);
  border-radius: 8px;
  font-size: 12px;
  color: var(--warning-text);
}
summary {
  cursor: pointer;
  line-height: 1.6;
}
.group-title > span {
  max-width: 65%;
  overflow-wrap: anywhere;
  text-align: right;
}
.node-name {
  overflow-wrap: anywhere;
}
.node-id {
  max-width: 96px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-content {
  gap: 4px;
}
.node-extra {
  font-size: 9px;
  color: var(--muted);
}
.node-content .node-version {
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tooltip {
  width: 282px;
  max-width: calc(100vw - 16px);
  overflow-wrap: anywhere;
}
.inspector {
  align-self: start;
}
.detail dl {
  gap: 12px;
}
.activity-scroll {
  max-height: 360px;
  overflow: auto;
}
.workload-scroll {
  max-height: 360px;
  overflow: auto;
}
.event {
  grid-template-columns: 65px minmax(0, 1fr) auto;
}
.event div {
  overflow-wrap: anywhere;
}
.event-status {
  max-width: 105px;
  overflow-wrap: anywhere;
}
td small {
  display: block;
  color: var(--muted);
  font-size: 10px;
  margin-top: 4px;
}
td {
  overflow-wrap: anywhere;
}
.pdb-note {
  padding: 10px 18px;
}
.pod-details {
  margin-top: 14px;
  max-height: 260px;
  overflow: auto;
  font-size: 11px;
}
.pod-details p {
  overflow-wrap: anywhere;
}
.unregistered {
  padding: 16px 18px;
}
.instance-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.collection-label {
  padding-top: 10px;
}
.controls select {
  max-width: 220px;
}
.map-footnote {
  margin: 0;
}
.watch {
  min-width: 0;
}
.main-grid > * {
  min-width: 0;
}
.lower > * {
  min-width: 0;
}
.stats .metric {
  min-width: 0;
}
.stats .metric strong {
  font-size: clamp(20px, 2vw, 28px);
  overflow-wrap: anywhere;
}
.stats .metric label {
  display: block;
}
.watch h1 {
  margin: 8px 0;
}
.watch .legend {
  pointer-events: auto;
}
.watch .tooltip strong {
  color: var(--cyan);
}
@media (max-width: 620px) {
  .source-summary {
    text-align: left;
  }
  .group-title > span {
    max-width: 100%;
    text-align: left;
  }
  .detail dl {
    grid-template-columns: 1fr;
  }
  .controls {
    width: 100%;
  }
  .controls select {
    max-width: 100%;
  }
  table {
    min-width: 340px;
  }
  .event {
    grid-template-columns: 1fr;
  }
  .watch {
    padding: 0;
  }
  .watch h1 {
    font-size: 24px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .node path {
    transition: none;
  }
}
</style>
