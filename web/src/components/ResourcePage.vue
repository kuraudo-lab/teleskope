<script setup lang="ts">
import { computed, ref } from "vue";
import { inventoryModel } from "../domain/inventory.js";
import { pages } from "../domain/pages";
import { ref as resourceRef, arr } from "../domain/format.js";
import type { Snapshot, Resource } from "../runtime/types";
import ResourceTable from "./ResourceTable.vue";
const props = defineProps<{
  page: string;
  snapshot: Snapshot;
  projection: Record<string, any>;
  namespace: string;
}>();
defineEmits<{ select: [resource: Resource] }>();
const query = ref("");
const model = computed(() =>
  inventoryModel(
    props.snapshot,
    props.projection,
    props.namespace,
    panels.value.map((p) => p.id),
  ),
);
const panels = computed(
  () => pages.find((p) => p.id === props.page)?.panels || [],
);
const matches = computed(() =>
  model.value
    .filtered(props.snapshot.kubernetes?.workloads)
    .filter((w: Resource) => {
      const m = model.value;
      const text = [
        resourceRef(w),
        w.serviceAccountName,
        w.runtimeClassName,
        ...m.workloadImages(w),
        ...m.configRefsForWorkload(w).map(resourceRef),
        ...arr(w.volumes).map((v: Resource) => JSON.stringify(v)),
      ]
        .join(" ")
        .toLowerCase();
      return text.includes(query.value.trim().toLowerCase());
    })
    .slice(0, 18),
);
</script>
<template>
  <div class="stack">
    <div v-if="page === 'workloads'" class="panel">
      <h3>Workload lookup</h3>
      <div class="body workload-search">
        <input
          id="workloadSearch"
          v-model="query"
          type="search"
          aria-label="Search workloads"
          placeholder="Search workloads by name, namespace, image, service account, config, or volume"
        />
        <div
          id="workloadSearchResults"
          class="workload-results"
          aria-live="polite"
        >
          <button
            v-for="w in matches"
            :key="model.workloadKey(w)"
            class="workload-result"
            @click="$emit('select', w)"
          >
            <strong>{{ w.name }}</strong
            ><small
              >{{ w.kind }} · {{ w.namespace || "cluster" }}</small
            ></button
          ><span v-if="!matches.length" class="dependency-empty"
            >No matching workloads</span
          >
        </div>
      </div>
    </div>
    <div v-for="panel in panels" :key="panel.id" class="panel">
      <h3>{{ panel.title }}</h3>
      <ResourceTable
        v-if="model.tables[panel.id]"
        v-bind="model.tables[panel.id]"
        @select="$emit('select', $event)"
      />
    </div>
  </div>
</template>
