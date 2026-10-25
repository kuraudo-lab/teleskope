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
      "目标模板就绪",
      known.length
        ? `${match.length} / ${nodes.length}`
        : `— / ${nodes.length}`,
      `可核对 ${known.length} 节点 · 未知 ${nodes.length - known.length}`,
    ],
    [
      "Ready 节点",
      `${nodes.filter((n) => n.node.ready === "True").length} / ${nodes.length}`,
      "集群范围 · 含已 cordon 节点",
    ],
    [
      "Cordoned",
      nodes.filter((n) => n.node.unschedulable).length,
      "暂停接收新调度",
    ],
    [
      "non-DS / DS Pods",
      model.value.podsKnown
        ? `${nodes.reduce((s, n) => s + n.nonDS!, 0)} / ${nodes.reduce((s, n) => s + n.ds!, 0)}`
        : "— / —",
      "节点绑定 · 非终态 · 含 Terminating",
    ],
    [
      "Pending / Terminating",
      `${formatCount(model.value.pending)} / ${formatCount(model.value.terminating)}`,
      "集群范围 · 未绑定 Pending / 删除中",
    ],
  ];
});
const modeLabel = computed(() =>
  Object.values(props.sources).some((s) => s.mode === "recorded")
    ? "Recorded · 历史快照"
    : props.mode === "offline"
      ? "Offline · 离线报告"
      : "Live · 实时采集",
);
function showTooltip(event: Event, node: WatchNode) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
  tooltip.value = {
    node,
    left: Math.max(8, Math.min(window.innerWidth - 290, rect.right + 10)),
    top: Math.max(8, Math.min(window.innerHeight - 260, rect.top)),
  };
}
function hideTooltip() {
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
    ["AMI ID", node.instance?.imageId || "未采集"],
    ["实例 Launch Template", node.template],
    ["EC2 启动时间", displayTime(node.instance?.launchTime)],
    ["Node 注册时间", displayTime(node.node.createdAt)],
    [
      "Kubelet / Instance",
      `${node.node.kubeletVersion || "未知"} / ${node.instance?.instanceType || node.node.labels?.["node.kubernetes.io/instance-type"] || "未知"}`,
    ],
    ["Instance ID", node.instance?.instanceId || "未关联"],
    ["ASG", node.instance?.autoScalingGroupName || "未关联"],
    ["ASG 生命周期", node.instance?.lifecycleState || "未采集"],
    [
      "Cordon",
      node.node.unschedulable ? "spec.unschedulable = true" : "未 cordon",
    ],
    [
      "Taints",
      node.node.taints
        ?.map(
          (t: Resource) =>
            `${t.key}${t.value ? "=" + t.value : ""}:${t.effect}`,
        )
        .join(", ") || "无",
    ],
  ];
}
</script>
<template>
  <div class="watch" @keydown.esc="tooltip = null">
    <div class="intro">
      <div>
        <div class="eyebrow">EKS / UPGRADES / NODE WATCH</div>
        <h1>节点升级观察</h1>
        <p>
          观察节点替换、Pod 迁移与容量恢复。按节点实际配置核对 ASG 目标模板。
        </p>
      </div>
      <div class="source-summary">
        <span class="badge">{{ modeLabel }}</span
        ><span v-if="paused" class="badge warn">Paused · 已暂停更新</span
        ><span v-if="loading" class="badge">Updating</span>
        <p>快照 · {{ displayTime(snapshot.collectedAt) }}</p>
      </div>
    </div>
    <div class="row source-row">
      <span class="badge">只读观察 · 集群范围</span>
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
          >{{ name }} · {{ sourceLabel(source) }} · 发布于
          {{ displayTime(source.lastSuccess) }}</span
        >
      </div>
    </div>
    <p v-if="error" role="alert" class="notice">
      更新失败，显示最后可用快照：{{ error }}
    </p>
    <details v-if="coverage.length" class="coverage-note">
      <summary>
        数据覆盖待核对 ·
        {{
          coverage.map((c) => c.resource).join(" / ")
        }}（数字仅代表已观测对象）
      </summary>
      <p v-for="c in coverage" :key="c.resource">
        {{ c.resource }} · {{ c.status }} · {{ c.reason || "采集未完成" }} ·
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
          <h2>Node map <span class="badge">按 Node Group 分组</span></h2>
          <div class="controls">
            <select v-model="group" aria-label="筛选节点组">
              <option value="all">全部 Node Groups</option>
              <option
                v-if="group !== 'all' && !groups.includes(group)"
                :value="group"
              >
                {{ group }}（已无节点）
              </option>
              <option v-for="name in groups" :key="name">
                {{ name }}
              </option></select
            ><input
              v-model="search"
              aria-label="搜索节点"
              placeholder="搜索节点 / AZ / Instance"
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
          ><span>大数字 non-DS Pods / 小数字 DS Pods</span>
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
                      `${a.name} · desired ${a.desiredCapacity} · 目标 ${a.launchTemplateId || "未知"} / ${a.launchTemplateVersion || "未知"}`,
                  )
                  .join("；") || "未采集 ASG 目标配置"
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
                :aria-label="`${node.name}, ${formatCount(node.nonDS)} non-DS Pods, ${formatCount(node.ds)} DS Pods, ${nodeStates[node.state].label}, ${node.node.unschedulable ? 'Cordoned' : '未 cordon'}`"
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
                        : "LT 未知"
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
                ? "没有匹配的节点。"
                : "当前快照没有节点数据；请检查 Kubernetes 采集范围与权限。"
            }}
          </p>
          <button
            v-if="filtered.length > limit"
            class="control"
            @click="limit += 300"
          >
            显示更多（已显示 {{ limit }} / {{ filtered.length }}）
          </button>
        </div>
        <div class="map-foot">
          悬停 / 键盘聚焦查看节点信息，点击固定详情。Ready 且未 cordon
          不保证可调度，仍需核对 taint、资源与 Pod 约束。
        </div>
      </section>
      <section class="box inspector" aria-label="选中节点详情">
        <div class="section-head">
          <h2>节点详情</h2>
          <span class="badge">点击固定</span>
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
                ? "目标模板一致"
                : selected.target === "different"
                  ? "与目标模板不同"
                  : "目标模板待核对"
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
                ? `${formatCount(selected.nonDS)} 个 non-DS Pod 留在此节点，${formatCount(selected.terminating)} 个 Pod 正在删除。Cordon 本身不代表 drain 已开始或完成。`
                : "节点状态来自最近的 Kubernetes 观测；AWS 生命周期与节点 Readiness 分别展示。"
            }}
          </div>
          <details v-if="selected.pods.length" class="pod-details">
            <summary>已绑定 Pod（{{ selected.pods.length }}）</summary>
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
              ? "所选节点已离开当前快照，请选择另一个节点。"
              : "选择节点查看详情。"
          }}
        </p>
      </section>
    </div>
    <div v-if="model.unregistered.length" class="box unregistered">
      <h2>
        尚未关联 Kubernetes Node 的实例
        <span class="badge">{{ model.unregistered.length }}</span>
      </h2>
      <p>可能仍在启动、注册或已退出；是否缺失也取决于两侧采集时间。</p>
      <div class="instance-list">
        <span
          v-for="instance in model.unregistered"
          :key="instance.instanceId"
          class="badge"
          >{{ instance.instanceId }} ·
          {{ instance.nodegroupName || "未知节点组" }} ·
          {{ instance.state || "Unknown" }} ·
          {{ instance.lifecycleState || "生命周期未知" }}</span
        >
      </div>
    </div>
    <div class="lower">
      <section class="box">
        <div class="section-head">
          <h2>ASG 活动</h2>
          <span class="badge">{{
            group === "all" ? "全部节点组" : group
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
              "活动未采集。需要 autoscaling:DescribeScalingActivities 权限。"
            }}
          </p>
          <div class="activity-scroll">
            <div v-for="a in activities" :key="a.activityId" class="event">
              <time :title="displayTime(a.startTime)">{{
                a.startTime ? new Date(a.startTime).toLocaleTimeString() : "—"
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
                ? "本次采集没有匹配的 ASG 活动。"
                : "没有可展示的 ASG 活动证据。"
            }}
          </p>
        </div>
      </section>
      <section class="box">
        <div class="section-head">
          <h2>工作负载连续性</h2>
          <span class="badge">集群范围</span>
        </div>
        <div class="workload-scroll">
          <table>
            <thead>
              <tr>
                <th>工作负载</th>
                <th>Ready / Desired</th>
                <th>观察</th>
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
                      ? "期望数量未知"
                      : w.ready < w.desired
                        ? "就绪副本不足"
                        : "就绪数量满足"
                  }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="!model.workloads.length" class="empty">
          当前快照无 Deployment / StatefulSet / DaemonSet 数据。
        </p>
        <div class="pdb-note">
          <details v-if="model.pdbs.length">
            <summary class="warn">
              {{ model.pdbs.length }} 个 PDB 当前无可用驱逐预算
            </summary>
            <p v-for="pdb in model.pdbs" :key="`${pdb.namespace}/${pdb.name}`">
              {{ pdb.namespace }}/{{ pdb.name }} · healthy
              {{ pdb.currentHealthy ?? 0 }} / desired
              {{ pdb.desiredHealthy ?? 0 }}
            </p>
          </details>
          <p>
            PDB 预算为风险提示，不代表已发生驱逐阻塞。Pending Pod
            未绑定节点，单独统计。
          </p>
        </div>
      </section>
    </div>
    <p class="map-footnote">
      口径：节点数字为已绑定且非终态的 Pod，包含 Terminating；DS 按 DaemonSet
      ownerReference
      分类。目标模板仅比较实例实际模板与明确的数字版本，未知配置不会计为匹配。工作负载数量仅代表快照观测。
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
      {{ tooltip.node.instance?.imageId || "未采集" }}<br />Launch Template:
      {{ tooltip.node.template }}<br />EC2 启动:
      {{ displayTime(tooltip.node.instance?.launchTime) }}<br />{{
        tooltip.node.az
      }}
      · {{ nodeStates[tooltip.node.state].label }}<br /><span class="muted"
        >点击查看完整详情</span
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
