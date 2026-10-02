<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue";
import { createTopologyRenderer } from "../domain/topology-renderer.js";
import type { Graph, Snapshot, Resource } from "../runtime/types";
const props = defineProps<{
  graph?: Graph;
  snapshot: Snapshot;
  advisor: Record<string, any>;
  sources: Record<string, Resource>;
  namespace: string;
  resource: string;
}>();
const emit = defineEmits<{ select: [resource: Resource] }>();
const root = ref<HTMLElement>();
let renderer: ReturnType<typeof createTopologyRenderer> | undefined;
const rerender = () => renderer?.update(props);
onMounted(() => {
  renderer = createTopologyRenderer(root.value!, props, (resource: Resource) =>
    emit("select", resource),
  );
  document.addEventListener("teleskope:render-topology", rerender);
});
watch(
  () => [
    props.graph,
    props.snapshot,
    props.advisor,
    props.namespace,
    props.resource,
    props.sources,
  ],
  () => renderer?.update(props),
  { flush: "post" },
);
onUnmounted(() => {
  renderer?.dispose();
  document.removeEventListener("teleskope:render-topology", rerender);
});
</script>
<template>
  <div ref="root" class="app">
    <aside class="panel sidebar">
      <div class="pane-head">
        <p class="eyebrow">Topology controls</p>
        <h1>Investigate by relationship and evidence</h1>
      </div>
      <div class="controls">
        <div class="control-group">
          <label class="control-title" for="topology-search">Search</label
          ><input
            id="topology-search"
            class="search"
            type="search"
            placeholder="kind:service health:warning"
            aria-label="Search topology objects"
          />
        </div>
        <div class="control-group">
          <div class="control-title">Projection</div>
          <div
            class="topology-segmented views"
            role="tablist"
            aria-label="Topology projection"
          >
            <button
              type="button"
              role="tab"
              aria-selected="true"
              aria-disabled="false"
            >
              Workloads
            </button>
          </div>
        </div>
        <div class="control-group">
          <div class="control-title">Presentation</div>
          <div class="topology-segmented" id="topologyModeOptions">
            <button
              type="button"
              data-topology-mode="graph"
              aria-pressed="true"
            >
              Graph</button
            ><button
              type="button"
              data-topology-mode="table"
              aria-pressed="false"
            >
              Table
            </button>
          </div>
        </div>
        <div class="control-group topology-control-row">
          <label class="control-title" for="topologyHealth">Health</label
          ><select id="topologyHealth">
            <option value="all">All health</option>
            <option value="healthy">Healthy</option>
            <option value="warning">Warning</option>
            <option value="unknown">Unknown</option>
            <option value="stopped">Stopped</option>
          </select>
        </div>
        <div class="control-group topology-control-row">
          <label class="control-title" for="topologySystem"
            >System objects</label
          ><select id="topologySystem">
            <option value="include">Include system</option>
            <option value="exclude">Hide system</option>
            <option value="only">Only system</option>
          </select>
        </div>
        <div class="control-group topology-control-row">
          <label class="control-title" for="topologyState"
            >Relationship state</label
          ><select id="topologyState">
            <option value="all">All objects</option>
            <option value="connected">Connected</option>
            <option value="disconnected">Disconnected</option>
            <option value="stopped">Stopped</option>
          </select>
        </div>
        <label class="check"
          ><span>Group replicas by owner</span
          ><input id="topologyGroup" type="checkbox" checked
        /></label>
        <div class="control-group">
          <div class="control-title">Object types</div>
          <label class="check"
            ><span
              ><svg class="icon"><use href="#icon-gateway"></use></svg>Ingress /
              Gateway</span
            ><span class="count" id="topologyGatewayCount">0</span></label
          >
          <label class="check"
            ><span
              ><svg class="icon"><use href="#icon-service"></use></svg
              >Services</span
            ><span class="count" id="topologyServiceCount">3</span></label
          >
          <label class="check"
            ><span
              ><svg class="icon"><use href="#icon-workload"></use></svg
              >Workloads</span
            ><span class="count" id="topologyWorkloadCount">5</span></label
          >
          <label class="check"
            ><span
              ><svg class="icon"><use href="#icon-pod"></use></svg>Pods</span
            ><span class="count" id="topologyPodCount">0</span></label
          >
          <label class="check"
            ><span
              ><svg class="icon"><use href="#icon-storage"></use></svg
              >Storage</span
            ><span class="count" id="topologyStorageCount">0</span></label
          >
          <label class="check"
            ><span
              ><svg class="icon"><use href="#icon-image"></use></svg
              >Images</span
            ><span class="count" id="topologyImageCount">0</span></label
          >
        </div>
        <div class="control-group">
          <div class="control-title">Relationship evidence</div>
          <div class="topology-evidence-legend">
            <span><i></i>Declared</span
            ><span class="resolved"><i></i>Resolved</span
            ><span class="ownership"><i></i>Ownership</span
            ><span class="inferred"><i></i>Inferred</span
            ><span class="observed"><i></i>Observed</span>
          </div>
        </div>
        <div class="control-group">
          <div class="control-title">Legend</div>
          <div class="legend" aria-label="Topology icon legend">
            <div class="legend-item gateway">
              <svg class="icon"><use href="#icon-gateway"></use></svg>Gateway
            </div>
            <div class="legend-item service">
              <svg class="icon"><use href="#icon-service"></use></svg>Service
            </div>
            <div class="legend-item workload">
              <svg class="icon"><use href="#icon-workload"></use></svg>Workload
            </div>
            <div class="legend-item pod">
              <svg class="icon"><use href="#icon-pod"></use></svg>Pod
            </div>
            <div class="legend-item node">
              <svg class="icon"><use href="#icon-node"></use></svg>Node
            </div>
            <div class="legend-item image">
              <svg class="icon"><use href="#icon-image"></use></svg>Image
            </div>
          </div>
        </div>
      </div>
    </aside>
    <main class="workspace-main">
      <section
        class="section active panel topology-workspace"
        data-section="overview"
      >
        <div class="toolbar">
          <div class="toolbar-title">
            <p class="eyebrow" id="topologyScope">All namespaces · Workloads</p>
            <h1 id="topologyTitle">Workload relationship map</h1>
          </div>
          <div class="stat">
            <strong id="topologyObjectStat">11</strong><span>objects</span>
          </div>
          <div class="stat">
            <strong id="topologyEdgeStat">4</strong><span>edges</span>
          </div>
          <div class="stat">
            <strong id="topologyWarningStat">0</strong><span>warnings</span>
          </div>
          <button
            type="button"
            class="icon-button"
            id="topologyZoomOut"
            aria-label="Zoom topology out"
          >
            −</button
          ><button
            type="button"
            class="icon-button"
            id="topologyZoomIn"
            aria-label="Zoom topology in"
          >
            +</button
          ><button
            type="button"
            class="icon-button"
            id="topologyFit"
            aria-label="Reset topology zoom and position"
          >
            <svg class="icon"><use href="#icon-fit"></use></svg>
          </button>
          <p
            class="topology-complexity-notice"
            id="topologyComplexityNotice"
            role="status"
            aria-live="polite"
            hidden=""
          ></p>
        </div>
        <div class="map">
          <div
            id="topology"
            class="map-inner"
            data-route-node-hits="0"
            data-route-strategy="straight-focus"
            data-route-collision-checked="true"
          ></div>
        </div>
      </section>
    </main>
    <aside class="panel inspector">
      <div class="pane-head">
        <p class="eyebrow">Inspector</p>
        <h1 id="topologyInspectorTitle">Cluster summary</h1>
      </div>
      <div class="inspector-body">
        <div class="summary-card">
          <div class="summary-title">
            <div class="glyph workload">
              <svg class="icon"><use href="#icon-workload"></use></svg>
            </div>
            <strong>Inventory health</strong>
          </div>
          <div id="cards" class="grid cards"></div>
        </div>
        <div class="summary-card">
          <div class="summary-title">
            <div class="glyph service">
              <svg class="icon"><use href="#icon-event"></use></svg>
            </div>
            <strong>Source freshness</strong>
          </div>
          <div
            id="sourceFreshness"
            class="source-freshness"
            aria-label="Source freshness"
          ></div>
        </div>
        <div class="summary-card">
          <div class="summary-title">
            <div class="glyph gateway">
              <svg class="icon"><use href="#icon-gateway"></use></svg>
            </div>
            <strong>Selected object</strong>
          </div>
          <div class="kv" id="topologySelectedSummary"></div>
          <div id="topologyContext" class="topology-context"></div>
          <div id="topologyCoverage" class="topology-coverage"></div>
        </div>
      </div>
    </aside>
  </div>
</template>
