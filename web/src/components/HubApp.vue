<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { logo } from "../logo";
const fleet = ref<any>({ clusters: [], events: [], revision: 0 }),
  provider = ref("all"),
  region = ref("all"),
  state = ref("all"),
  page = ref("fleet"),
  error = ref(""),
  query = ref(""),
  results = ref<any[]>([]),
  searchStatus = ref("");
let disposed = false,
  etag = "",
  timer: ReturnType<typeof setTimeout>,
  searchGeneration = 0;
const abort = new AbortController();
const clusters = computed(() =>
  fleet.value.clusters.filter(
    (c: any) =>
      (provider.value === "all" || c.cluster.provider === provider.value) &&
      (region.value === "all" || c.cluster.region === region.value) &&
      (state.value === "all" || c.state === state.value),
  ),
);
const providers = computed(
  () =>
    [
      ...new Set(
        fleet.value.clusters.map((c: any) => c.cluster.provider || "-"),
      ),
    ] as string[],
);
const regions = computed(
  () =>
    [
      ...new Set(fleet.value.clusters.map((c: any) => c.cluster.region || "-")),
    ] as string[],
);
const totals = computed(() =>
  clusters.value.reduce(
    (t: any, c: any) => ({
      nodes: t.nodes + (c.nodes || 0),
      addons: t.addons + (c.addonRisks || 0),
      nodegroups: t.nodegroups + (c.nodegroupRisks || 0),
    }),
    { nodes: 0, addons: 0, nodegroups: 0 },
  ),
);
const versions = computed(() =>
  count(clusters.value.map((c: any) => c.kubernetesVersion || "unknown")),
);
const health = computed(() =>
  count(
    clusters.value.flatMap((c: any) =>
      Object.values(c.sources || {}).map((s: any) => s.state || "unknown"),
    ),
  ),
);
function count(items: string[]) {
  return items.reduce(
    (m: Record<string, number>, key) => ((m[key] = (m[key] || 0) + 1), m),
    {},
  );
}
async function http(url: string, init: RequestInit = {}) {
  const response = await fetch(url, {
    ...init,
    cache: "no-store",
    signal: AbortSignal.any([abort.signal, AbortSignal.timeout(10000)]),
  });
  if (!response.ok && response.status !== 304)
    throw Error(`HTTP ${response.status}`);
  return response;
}
async function refresh() {
  try {
    const response = await http("/api/clusters", {
      headers: etag ? { "If-None-Match": etag } : {},
    });
    if (response.status !== 304) {
      const data = await response.json();
      if (!disposed) {
        fleet.value = data;
        etag = response.headers.get("ETag") || "";
      }
    }
    error.value = "";
  } catch (e) {
    if (!disposed) {
      error.value = String(e);
      etag = "";
    }
  } finally {
    if (!disposed) timer = setTimeout(refresh, 5000);
  }
}
async function search() {
  const generation = ++searchGeneration;
  searchStatus.value = "Searching stored envelopes…";
  try {
    const data = await (
      await http("/api/search?q=" + encodeURIComponent(query.value))
    ).json();
    if (generation !== searchGeneration || disposed) return;
    results.value = data.results || [];
    searchStatus.value = data.truncated
      ? `${data.total} matches, showing first ${results.value.length}`
      : `${results.value.length} matches`;
  } catch (e) {
    if (generation === searchGeneration) searchStatus.value = String(e);
  }
}
async function download(format: string) {
  try {
    const response = await http(
      format === "json" ? "/api/export/fleet.json" : "/api/export/summary.md",
    );
    const url = URL.createObjectURL(await response.blob());
    const a = document.createElement("a");
    a.href = url;
    a.download =
      format === "json" ? "teleskope-fleet.json" : "teleskope-fleet-summary.md";
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  } catch (e) {
    error.value = String(e);
  }
}
function sourceLabel(status: any) {
  return status.mode === "watch"
    ? `watch ${status.state === "ready" ? "connected" : status.state === "stale" ? "reconnecting" : status.state}`
    : status.state;
}
function sourceDetail(status: any) {
  return `published ${status.lastSuccess || "never"} · event ${status.lastEventAt || "none"} · full resync ${status.lastFullSyncAt || "none"} · reconnects ${status.reconnects || 0}`;
}
onMounted(() => void refresh());
onUnmounted(() => {
  disposed = true;
  clearTimeout(timer);
  abort.abort();
});
</script>
<template>
  <div class="hub-ui">
    <header class="app-topbar">
      <div class="brand">
        <div class="logo"><img :src="logo" alt="" /></div>
        <div>
          <strong id="title">Teleskope hub</strong>
          <p>multi-cluster hub</p>
        </div>
      </div>
      <nav id="nav">
        <button
          v-for="id in ['fleet', 'events', 'search']"
          :key="id"
          class="tab"
          :data-target="id"
          @click="page = id"
        >
          {{ id === "fleet" ? "Fleet" : id === "events" ? "Events" : "Search" }}
        </button>
      </nav>
      <div class="global-actions">
        <select
          id="providerFilter"
          v-model="provider"
          aria-label="Provider filter"
        >
          <option value="all">All providers</option>
          <option v-for="p in providers" :key="p">{{ p }}</option></select
        ><select id="regionFilter" v-model="region" aria-label="Region filter">
          <option value="all">All regions</option>
          <option v-for="r in regions" :key="r">{{ r }}</option></select
        ><select id="stateFilter" v-model="state" aria-label="State filter">
          <option value="all">All states</option>
          <option v-for="s in ['ready', 'partial', 'stale', 'error']" :key="s">
            {{ s }}
          </option></select
        ><button id="exportJSON" @click="download('json')">JSON</button
        ><button id="exportMarkdown" @click="download('markdown')">
          Markdown
        </button>
      </div>
    </header>
    <main class="report-page stack">
      <p id="subtitle">
        {{ clusters.length }} of {{ fleet.clusters.length }} clusters shown ·
        hub revision {{ fleet.revision }}
      </p>
      <p v-if="error" role="alert">Hub API unavailable: {{ error }}</p>
      <template v-if="page === 'fleet'">
        <div id="cards" class="cards">
          <div
            v-for="[label, value] in [
              ['Clusters', clusters.length],
              ['Nodes', totals.nodes],
              ['Add-on risks', totals.addons],
              ['Nodegroup risks', totals.nodegroups],
            ]"
            :key="label"
            class="card"
          >
            <div class="label">{{ label }}</div>
            <div class="value">{{ value }}</div>
          </div>
        </div>
        <div class="hub-distributions">
          <div
            v-for="[id, title, counts] in [
              ['versionDist', 'Kubernetes versions', versions],
              [
                'riskCounts',
                'EKS risks',
                { addons: totals.addons, nodegroups: totals.nodegroups },
              ],
              ['sourceHealth', 'Source health', health],
            ]"
            :key="id as string"
            class="panel"
          >
            <h3>{{ title }}</h3>
            <table>
              <tbody :id="id as string">
                <tr v-for="(value, key) in counts" :key="key">
                  <td>{{ key }}</td>
                  <td>{{ value }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div class="panel">
          <h3>Clusters</h3>
          <div class="scroll">
            <table>
              <thead>
                <tr>
                  <th
                    v-for="h in [
                      'Cluster',
                      'Provider',
                      'Region',
                      'Version',
                      'State',
                      'Sources',
                      'EKS risk',
                      'Revision',
                      'Collected',
                      'Inventory',
                      'Advisor',
                    ]"
                    :key="h"
                  >
                    {{ h }}
                  </th>
                </tr>
              </thead>
              <tbody id="clusters">
                <tr v-for="c in clusters" :key="c.cluster.id">
                  <td>
                    <a :href="'/cluster?id=' + encodeURIComponent(c.cluster.id)"
                      ><strong>{{ c.cluster.name || c.cluster.id }}</strong></a
                    ><small>{{ c.cluster.id }}</small>
                  </td>
                  <td>{{ c.cluster.provider }}</td>
                  <td>{{ c.cluster.region }}</td>
                  <td>{{ c.kubernetesVersion }}</td>
                  <td>{{ c.state }}</td>
                  <td>
                    <div
                      v-for="(source, name) in c.sources"
                      :key="name"
                      class="chip"
                      :title="sourceDetail(source)"
                    >
                      {{ name }}: {{ sourceLabel(source)
                      }}<small>{{ sourceDetail(source) }}</small>
                    </div>
                  </td>
                  <td>
                    add-ons={{ c.addonRisks || 0 }} nodegroups={{
                      c.nodegroupRisks || 0
                    }}
                  </td>
                  <td>{{ c.revision }}</td>
                  <td>{{ c.collectedAt }}</td>
                  <td>
                    nodes={{ c.nodes || 0 }} workloads={{
                      c.workloads || 0
                    }}
                    pods={{ c.pods || 0 }} images={{ c.images || 0 }}
                  </td>
                  <td>{{ c.advisor }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
      <div v-else-if="page === 'events'" class="panel">
        <h3>Recent events</h3>
        <table>
          <tbody id="events">
            <tr v-for="(item, i) in fleet.events" :key="i">
              <td>{{ item.event.at }}</td>
              <td>{{ item.cluster.name || item.cluster.id }}</td>
              <td>{{ item.event.source }}</td>
              <td>{{ item.event.level }}</td>
              <td>{{ item.event.message }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="panel">
        <h3>Fleet resource search</h3>
        <div class="body">
          <input
            id="searchInput"
            v-model="query"
            type="search"
            aria-label="Search fleet resources"
            @keydown.enter="search"
          /><button id="runSearch" @click="search">Search</button>
          <p id="searchStatus" aria-live="polite">{{ searchStatus }}</p>
          <table>
            <thead>
              <tr>
                <th
                  v-for="h in [
                    'Cluster',
                    'Kind',
                    'Namespace',
                    'Name',
                    'Image',
                    'Detail',
                  ]"
                  :key="h"
                >
                  {{ h }}
                </th>
              </tr>
            </thead>
            <tbody id="searchResults">
              <tr v-for="(r, i) in results" :key="i">
                <td>
                  <a :href="r.clusterUrl">{{
                    r.cluster.name || r.cluster.id
                  }}</a>
                </td>
                <td>{{ r.kind }}</td>
                <td>{{ r.namespace }}</td>
                <td>{{ r.name }}</td>
                <td>{{ r.image }}</td>
                <td>{{ r.detail }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </main>
  </div>
</template>
<style>
.hub-ui .cards {
  grid-template-columns: repeat(4, 1fr);
}
.hub-distributions {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.hub-ui small {
  display: block;
}
.hub-ui .app-topbar {
  position: relative;
}
</style>
