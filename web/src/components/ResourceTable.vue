<script setup lang="ts">
import type { Resource } from "../runtime/types";
defineProps<{
  id: string;
  headers: string[];
  rows: { resource: Resource; cells: string[] }[];
}>();
defineEmits<{ select: [resource: Resource] }>();
</script>
<template>
  <div class="scroll">
    <table :id="id">
      <thead>
        <tr>
          <th v-for="header in headers" :key="header" scope="col">
            {{ header }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, index) in rows"
          :key="row.resource.uid || index"
          tabindex="0"
          @click="$emit('select', row.resource)"
          @keydown.enter="$emit('select', row.resource)"
        >
          <td v-for="(cell, i) in row.cells" :key="i" v-html="cell"></td>
        </tr>
        <tr v-if="!rows.length">
          <td class="empty" :colspan="headers.length">No matching data</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
