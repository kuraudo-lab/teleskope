<script setup lang="ts">
import { computed, ref } from "vue";
import {
  advisorDomain,
  advisorDomainCatalog,
  advisorPriority,
  advisorSearchText,
  advisorCapabilityHTML,
} from "../domain/advisor.js";
const props = defineProps<{ report: Record<string, any> }>();
const query = ref(""),
  status = ref("all"),
  domain = ref("all");
const capabilities = computed(() => props.report.capabilities || []);
const statuses = [
  ["all", "All findings"],
  ["supported", "Supported"],
  ["attention", "Needs attention"],
  ["unknown", "Unknown"],
  ["gap", "Coverage gap"],
];
const matchesStatus = (c: any) =>
  status.value === "all" ||
  (status.value === "attention"
    ? c.assessment === "unsupported"
    : status.value === "gap"
      ? c.coverage !== "complete"
      : c.assessment === status.value);
const countStatus = (value: string) =>
  capabilities.value.filter(
    (c: any) =>
      value === "all" ||
      (value === "attention"
        ? c.assessment === "unsupported"
        : value === "gap"
          ? c.coverage !== "complete"
          : c.assessment === value),
  ).length;
const visible = computed(() =>
  capabilities.value
    .filter(
      (c: any) =>
        matchesStatus(c) &&
        (domain.value === "all" || advisorDomain(c) === domain.value) &&
        advisorSearchText(c).includes(query.value.toLowerCase()),
    )
    .sort(
      (a: any, b: any) =>
        advisorPriority(a) - advisorPriority(b) ||
        String(a.key).localeCompare(String(b.key)),
    ),
);
</script>
<template>
  <div class="panel advisor-workspace">
    <h3>Cluster capabilities</h3>
    <div class="body">
      <p id="advisor-summary">{{ report.summary }}</p>
      <p>
        Assessment and coverage counts may overlap. Runtime behavior is not
        verified.
      </p>
      <p>
        Deterministic evidence assessment. Unknown means evidence is missing,
        not that the capability is absent.
      </p>
      <input
        id="advisor-search"
        v-model="query"
        type="search"
        placeholder="Search capabilities and evidence"
        aria-label="Search capabilities"
      />
      <div id="advisor-status-filters">
        <button
          v-for="[s, label] in statuses"
          :key="s"
          class="chip"
          :aria-pressed="status === s"
          @click="status = s"
        >
          {{ label }} · {{ countStatus(s) }}
        </button>
      </div>
      <div id="advisor-domain-filters">
        <button
          class="chip"
          :aria-pressed="domain === 'all'"
          @click="domain = 'all'"
        >
          All domains</button
        ><button
          v-for="[value, label] in advisorDomainCatalog"
          :key="value"
          class="chip"
          :aria-pressed="domain === value"
          @click="domain = value"
        >
          {{ label }}
        </button>
      </div>
      <p id="advisor-result-count">{{ visible.length }} capabilities</p>
      <div id="advisor-capabilities">
        <div
          v-for="(capability, index) in visible"
          :key="capability.key + '/' + capability.scope"
          v-html="advisorCapabilityHTML(capability, index === 0)"
        ></div>
      </div>
      <p v-if="!visible.length" id="advisor-empty" class="empty">
        No capability matches these filters.
      </p>
    </div>
  </div>
</template>
