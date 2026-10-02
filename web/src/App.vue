<script setup lang="ts">
import {
  computed,
  onMounted,
  onUnmounted,
  ref,
  watch,
  reactive,
  nextTick,
} from "vue";
import type { BootConfig, Resource } from "./runtime/types";
import { createReportStore } from "./runtime/report-store";
import ResourcePage from "./components/ResourcePage.vue";
import DetailDrawer from "./components/DetailDrawer.vue";
import AdvisorPage from "./components/AdvisorPage.vue";
import AnalysisPanel from "./components/AnalysisPanel.vue";
import TopologyView from "./components/TopologyView.vue";
import { publicationStatus, eventKind } from "./runtime/source-status";
import { logo } from "./logo";
import { symbols } from "./symbols";
import { pages } from "./domain/pages";
const read = (id: string, fallback: any = {}) =>
  JSON.parse(
    document.getElementById(id)?.textContent || JSON.stringify(fallback),
  );
const boot: BootConfig = read("boot-config");
const store = createReportStore(boot, {
  snapshot: read("snapshot-data"),
  graph: read("topology-data"),
  advisor: read("advisor-data"),
  eksProjection: read("eks-projection-data"),
});
const s = store.state;
const eventStatus = computed(() =>
  publicationStatus(s.sources, s.loading, s.paused, s.error),
);
const refreshEvent = () => void store.refresh();
watch(
  () => s.revision,
  (revision) => {
    document.getElementById("ui-root")!.dataset.revision = String(revision);
  },
  { immediate: true },
);

const url = new URL(location.href);
const page = ref(
  pages.some((p) => p.id === url.searchParams.get("page"))
    ? url.searchParams.get("page")!
    : "overview",
);
const namespace = ref(url.searchParams.get("namespace") || "all"),
  resource = ref(url.searchParams.get("resource") || "all");
const selected = ref<Resource | null>(null),
  menu = ref(""),
  exportOpen = ref(false),
  exportError = ref("");
const theme = ref(
  document.documentElement.dataset.theme ||
    (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"),
);
watch(
  () => s.loading,
  (value) => document.body.classList.toggle("live-loading", value),
  { immediate: true },
);
document.documentElement.dataset.theme = theme.value;
const k = computed(() => s.snapshot.kubernetes || {}),
  cluster = computed(() => s.snapshot.eks?.cluster || {});
const namespaces = computed(() =>
  [
    ...new Set(
      [
        namespace.value === "all" ? "" : namespace.value,
        ...(k.value.namespaces || []).map((n: Resource) => n.name),
        ...(k.value.workloads || []).map((n: Resource) => n.namespace),
        ...(k.value.pods || []).map((n: Resource) => n.namespace),
      ].filter(Boolean),
    ),
  ].sort(),
);
const groups = [
  {
    id: "eks",
    label: "EKS",
    items: [
      ["eks", "Overview"],
      ["eks-upgrades", "Upgrades"],
      ["eks-compute", "Compute"],
      ["eks-network", "Network"],
      ["eks-security", "Security"],
      ["eks-addons", "Add-ons"],
    ],
  },
  {
    id: "kubernetes",
    label: "Kubernetes",
    items: [
      ["nodes", "Nodes"],
      ["images", "Running images"],
      ["crds", "Custom resources"],
      ["workloads", "Workloads"],
      ["network", "Network"],
      ["security", "Security"],
      ["policies", "Policies"],
      ["storage", "Storage"],
    ],
  },
];
const resourceTypes = [
  ["all", "All resource types"],
  ["workload", "Workloads"],
  ["pod", "Pods"],
  ["service", "Services"],
  ["endpoint", "EndpointSlices"],
  ["ingress", "Ingresses"],
  ["gateway", "Gateways"],
  ["gatewayRoute", "Gateway routes"],
  ["image", "Running images"],
  ["container", "Running containers"],
  ["crd", "Custom resources"],
  ["pvc", "PVCs"],
  ["storage", "StorageClasses"],
  ["csi", "CSI drivers"],
  ["runtime", "RuntimeClasses"],
];
const recorded = computed(
  () =>
    Object.values(s.sources).length > 0 &&
    Object.values(s.sources).every((source: any) => source.mode === "recorded"),
);
const canAnalyze = computed(
  () =>
    boot.mode === "live" &&
    !!boot.endpoints?.analyze &&
    boot.analysisEnabled === true,
);
const analysisPage = computed(() =>
  page.value.startsWith("eks") ? "eks" : page.value,
);
const analysisCache = reactive<Record<string, { result: any; key: string }>>(
  {},
);
const savedAnalysis = computed(() => {
  if (!analysisCache[analysisPage.value])
    analysisCache[analysisPage.value] = { result: null, key: "" };
  return analysisCache[analysisPage.value];
});
const analysisPanel = ref<InstanceType<typeof AnalysisPanel>>();
async function toolbarAnalysis() {
  const input = analysisRequest.value;
  page.value = "advisor";
  await nextTick();
  await analysisPanel.value?.run(input, true);
}
const selectionMissing = ref(false);
function selectResource(value: Resource) {
  selectionMissing.value = false;
  selected.value = value;
}
const analysisRequest = computed(() => ({
  revision: s.revision,
  scope: {
    pageId: analysisPage.value,
    namespace: namespace.value === "all" ? "" : namespace.value,
    resourceType:
      page.value === "overview"
        ? resource.value === "all"
          ? ""
          : resource.value
        : analysisPage.value,
    selectedRefs:
      selected.value && !selectionMissing.value ? [selected.value] : [],
  },
}));
function menuKeys(event: KeyboardEvent) {
  const group = (event.target as Element).closest(".nav-menu")!;
  const trigger = group.querySelector<HTMLButtonElement>(".nav-menu-button")!;
  const items = [
    ...group.querySelectorAll<HTMLButtonElement>('[role="menuitem"]'),
  ];
  if (event.key === "Escape") {
    menu.value = "";
    trigger.focus();
    event.preventDefault();
    return;
  }
  if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
  const index = items.indexOf(event.target as HTMLButtonElement);
  if (index < 0) {
    trigger.click();
    requestAnimationFrame(() => items[0]?.focus());
  } else
    items[
      (index + (event.key === "ArrowDown" ? 1 : -1) + items.length) %
        items.length
    ]?.focus();
  event.preventDefault();
}
function selectPage(id: string) {
  page.value = id;
  menu.value = "";
}
function syncURL() {
  const next = new URL(location.href);
  next.searchParams.set("page", page.value);
  next.searchParams.set("namespace", namespace.value);
  next.searchParams.set("resource", resource.value);
  history.replaceState(null, "", next);
}
watch(
  [page, namespace, resource],
  () => {
    document.body.dataset.activeSection = page.value;
    syncURL();
  },
  { immediate: true },
);
function onPop() {
  const next = new URL(location.href);
  page.value = next.searchParams.get("page") || "overview";
  namespace.value = next.searchParams.get("namespace") || "all";
  resource.value = next.searchParams.get("resource") || "all";
}
function onOutside(e: MouseEvent) {
  const el = e.target as Element;
  if (!el.closest("#nav")) menu.value = "";
  if (!el.closest("#exportMenu")) exportOpen.value = false;
}
function toggleTheme() {
  theme.value = theme.value === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = theme.value;
  try {
    localStorage.setItem("teleskope.theme", theme.value);
  } catch {}
}
async function exportReport(format: string) {
  exportOpen.value = false;
  exportError.value = "";
  try {
    let text = "";
    if (boot.mode === "live") {
      const path =
        format === "json"
          ? boot.endpoints?.exportSnapshot
          : boot.endpoints?.exportSummary;
      if (!path) throw Error("Export unavailable");
      const response = await fetch(path);
      if (!response.ok) throw Error(`HTTP ${response.status}`);
      text = await response.text();
    } else
      text =
        format === "json"
          ? JSON.stringify(s.snapshot, null, 2)
          : document.getElementById("markdown-data")?.textContent || "";
    const href = URL.createObjectURL(
      new Blob([text], {
        type: format === "json" ? "application/json" : "text/markdown",
      }),
    );
    const link = document.createElement("a");
    link.href = href;
    link.download = `${cluster.value.name || k.value.context || "teleskope"}.${format === "json" ? "json" : "md"}`;
    link.click();
    setTimeout(() => URL.revokeObjectURL(href), 1000);
  } catch (e) {
    exportError.value = String(e);
  }
}
watch(
  () => s.snapshot,
  () => {
    if (!selected.value) return;
    const old = selected.value;
    const candidates = Object.values(k.value)
      .filter(Array.isArray)
      .flat() as Resource[];
    const replacement = candidates.find((r) =>
      old.uid
        ? r.uid === old.uid
        : r.name === old.name &&
          r.namespace === old.namespace &&
          r.kind === old.kind,
    );
    if (replacement) selected.value = replacement;
    selectionMissing.value = !replacement && !!old.kind && !!old.name;
  },
);
onMounted(() => {
  store.start();
  document.addEventListener("teleskope:refresh", refreshEvent);
  window.addEventListener("popstate", onPop);
  document.addEventListener("click", onOutside);
});
onUnmounted(() => {
  store.dispose();
  document.removeEventListener("teleskope:refresh", refreshEvent);
  window.removeEventListener("popstate", onPop);
  document.removeEventListener("click", onOutside);
});
</script>
<template>
  <div v-html="symbols" aria-hidden="true"></div>
  <div class="shell">
    <header class="app-topbar">
      <div class="brand">
        <div class="logo" aria-hidden="true"><img :src="logo" alt="" /></div>
        <div>
          <strong id="title">{{
            s.loading
              ? "Waiting for first snapshot"
              : cluster.name || k.context || "Kubernetes cluster"
          }}</strong>
          <p id="subtitle">
            {{ k.version?.gitVersion || cluster.version || "unknown k8s" }} ·
            collected {{ s.snapshot.collectedAt || "-" }} ·
            {{
              recorded
                ? "recorded snapshot"
                : boot.mode === "offline"
                  ? "archived report"
                  : "live cluster inventory"
            }}
          </p>
        </div>
      </div>
      <nav id="nav" class="app-tabs" aria-label="Report pages">
        <button
          v-for="[id, label] in [
            ['overview', 'Overview'],
            ['advisor', 'Advisory'],
          ]"
          :key="id"
          class="tab"
          :class="{ active: page === id }"
          :data-target="id"
          :aria-current="page === id ? 'page' : undefined"
          @click="selectPage(id)"
        >
          {{ label }}
        </button>
        <div
          v-for="group in groups"
          :key="group.id"
          class="nav-menu"
          :class="{ open: menu === group.id }"
          @keydown="menuKeys"
        >
          <button
            class="tab nav-menu-button"
            aria-haspopup="menu"
            :aria-expanded="menu === group.id"
            @click="menu = menu === group.id ? '' : group.id"
          >
            {{ group.label }}
          </button>
          <div class="nav-menu-items" role="menu" :aria-label="group.label">
            <button
              v-for="[id, label] in group.items"
              :key="id"
              role="menuitem"
              class="tab"
              :class="{ active: page === id }"
              :data-target="id"
              @click="selectPage(id)"
            >
              {{ label }}
            </button>
          </div>
        </div>
        <button
          v-if="boot.mode === 'live'"
          class="tab"
          data-target="events"
          @click="selectPage('events')"
        >
          Events
          <span
            id="eventsNavStatus"
            class="nav-status"
            :class="eventStatus.tone"
            >{{ eventStatus.label }}</span
          >
        </button>
      </nav>
      <div class="global-actions">
        <select
          id="focus"
          v-model="namespace"
          class="select"
          aria-label="Namespace"
          :disabled="['advisor', 'events'].includes(page)"
        >
          <option value="all">All namespaces</option>
          <option v-for="ns in namespaces" :key="ns" :value="ns">
            {{ ns }}
          </option>
        </select>
        <select
          id="resourceType"
          v-model="resource"
          class="select"
          aria-label="Resource type"
          :disabled="page !== 'overview'"
        >
          <option v-for="[id, label] in resourceTypes" :key="id" :value="id">
            {{ label }}
          </option>
        </select>
        <button
          v-if="canAnalyze"
          id="analyzeSnapshot"
          class="action-button ai-button"
          :disabled="s.loading || s.analyzing"
          @click="toolbarAnalysis"
        >
          Analyze with AI
        </button>
        <div id="exportMenu" class="export" :class="{ open: exportOpen }">
          <button
            id="exportReport"
            class="action-button"
            aria-haspopup="menu"
            :aria-expanded="exportOpen"
            @click="exportOpen = !exportOpen"
          >
            Export
          </button>
          <div class="export-menu" role="menu">
            <button
              role="menuitem"
              data-export-format="json"
              @click="exportReport('json')"
            >
              JSON</button
            ><button
              role="menuitem"
              data-export-format="markdown"
              @click="exportReport('markdown')"
            >
              Markdown
            </button>
          </div>
        </div>
        <button
          id="themeToggle"
          class="action-button"
          :aria-label="`Switch to ${theme === 'dark' ? 'light' : 'dark'} theme`"
          @click="toggleTheme"
        >
          {{ theme === "dark" ? "Light" : "Dark" }}
        </button>
        <button
          v-if="boot.mode === 'live' && !recorded"
          id="pause-updates"
          class="action-button"
          :aria-pressed="s.paused"
          @click="store.pause(!s.paused)"
        >
          {{ s.paused ? "Resume page updates" : "Pause updates" }}
        </button>
      </div>
    </header>
    <p v-if="s.error || exportError" role="alert" class="error">
      {{ s.error || exportError }}
    </p>
    <TopologyView
      v-if="page === 'overview'"
      :graph="s.graph"
      :snapshot="s.snapshot"
      :advisor="s.advisor"
      :sources="s.sources"
      :namespace="namespace"
      :resource="resource"
      @select="selectResource"
    />
    <main v-else class="workspace-main report-page">
      <section class="section active stack" :data-section="page">
        <AnalysisPanel
          ref="analysisPanel"
          :key="analysisPage"
          :saved="savedAnalysis"
          :page="analysisPage"
          :request="analysisRequest"
          :available="
            canAnalyze &&
            [
              'advisor',
              'eks',
              'nodes',
              'workloads',
              'network',
              'storage',
              'security',
            ].includes(analysisPage)
          "
          :busy="s.analyzing"
          :analyze="store.analyze"
        />
        <AdvisorPage v-if="page === 'advisor'" :report="s.advisor" />
        <div v-else-if="page === 'events'" class="panel">
          <h3>Recent events</h3>
          <div id="live-event-list" class="body live-event-list">
            <div
              v-for="(event, index) in [...s.events].reverse()"
              :key="index"
              class="live-event"
            >
              <time>{{ event.at }}</time
              ><span>{{ eventKind(event) }}</span
              ><strong>{{ event.level }}</strong
              ><span>{{ event.source }}</span
              ><span>{{ event.message }}</span>
            </div>
            <p v-if="!s.events.length">No events yet</p>
          </div>
        </div>
        <ResourcePage
          v-else
          :key="page"
          :page="page"
          :snapshot="s.snapshot"
          :projection="s.eksProjection"
          :namespace="namespace"
          @select="selectResource"
        />
      </section>
    </main>
  </div>
  <DetailDrawer
    v-if="selected"
    :selected="selected"
    :missing="selectionMissing"
    :snapshot="s.snapshot"
    @close="selected = null"
  />
</template>
<style>
#ui-root {
  min-height: 100vh;
}
.report-page {
  display: block;
  margin: 18px;
  overflow: auto;
}
.shell > .report-page {
  height: auto;
  margin: 0;
  overflow: auto;
}
.report-page > .section.active {
  display: block;
  height: auto;
}
.report-page .stack {
  display: grid;
  align-content: start;
}
.report-page .panel {
  overflow: visible;
}
.report-page .page-analysis {
  margin-bottom: 14px;
}
.shell .app-topbar .nav-menu-items {
  position: absolute;
  top: 100%;
  left: 0;
}
.shell .app-topbar .nav-menu {
  position: relative;
}
.live-event-list {
  display: grid;
  gap: 8px;
}
.live-event {
  display: grid;
  grid-template-columns: 180px 90px 60px 110px 1fr;
  gap: 10px;
}
</style>
