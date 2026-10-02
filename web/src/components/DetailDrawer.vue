<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { inventoryModel } from "../domain/inventory.js";
import { ref as resourceRef } from "../domain/format.js";
import type { Snapshot, Resource } from "../runtime/types";
const props = defineProps<{
  selected: Resource | null;
  snapshot: Snapshot;
  missing?: boolean;
}>();
const emit = defineEmits<{ close: [] }>();
const model = computed(() => inventoryModel(props.snapshot, {}, "all", []));
const button = ref<HTMLButtonElement>();
let previous: HTMLElement | null = null;
onMounted(() => {
  previous = document.activeElement as HTMLElement;
  button.value?.focus();
});
onUnmounted(() => previous?.focus());
</script>
<template>
  <div
    id="drawer"
    class="drawer open"
    role="dialog"
    aria-labelledby="drawerTitle"
    @keydown.esc="emit('close')"
  >
    <header>
      <h3 id="drawerTitle">
        {{
          selected && model.isWorkload(selected)
            ? resourceRef(selected)
            : selected?.name || selected?.image || selected?.kind || "Details"
        }}
      </h3>
      <button ref="button" type="button" @click="emit('close')">Close</button>
    </header>
    <div id="drawerBody" class="drawer-body">
      <p v-if="missing">
        Selection no longer in this view. This item was removed from the current
        snapshot.
      </p>
      <div
        v-else-if="selected && model.isWorkload(selected)"
        v-html="model.workloadDetailHTML(selected)"
      ></div>
      <pre v-else>{{ JSON.stringify(selected, null, 2) }}</pre>
    </div>
  </div>
</template>
