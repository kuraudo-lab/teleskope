<script setup lang="ts">
import { computed, ref } from "vue";
import type { AnalysisRequest } from "../runtime/types";
const props = defineProps<{
  page: string;
  saved: { result: any; key: string };
  request: AnalysisRequest;
  available: boolean;
  busy: boolean;
  analyze: (request: AnalysisRequest) => Promise<any>;
}>();
const result = ref<any>(props.saved.result),
  lastKey = ref(props.saved.key),
  error = ref("");
const key = computed(() => JSON.stringify(props.request));
async function run(input: AnalysisRequest = props.request, redirected = false) {
  const original = JSON.stringify(input);
  error.value = "";
  try {
    const data = await props.analyze(JSON.parse(original));
    if (!redirected && original !== key.value) {
      error.value = "Scope changed during analysis. Run analysis again.";
      return;
    }
    result.value = data;
    lastKey.value = original;
    props.saved.result = data;
    props.saved.key = original;
  } catch (e) {
    error.value = String(e);
  }
}
defineExpose({ run });
const groups = computed(() => {
  const facts: any[] = [],
    inferences: any[] = [];
  for (const item of (result.value?.sections || []).flatMap(
    (section: any) => section.items || [],
  ))
    (["observed", "fact", "factual", "deterministic"].includes(
      String(item.basis).toLowerCase(),
    )
      ? facts
      : inferences
    ).push(item);
  return [
    ["Observed facts", facts],
    ["Inference", inferences],
  ] as const;
});
</script>
<template>
  <div v-if="available" class="panel page-analysis" :data-analysis-page="page">
    <div class="page-analysis-head">
      <h3>{{ page }} AI analysis</h3>
      <button
        class="export-button"
        :data-analyze-page="page"
        :disabled="busy || request.revision < 0 || lastKey === key"
        @click="run()"
      >
        {{
          busy ? "Analyzing…" : lastKey === key ? "Analyzed" : "Analyze " + page
        }}
      </button>
    </div>
    <div
      :id="page === 'advisor' ? 'analysisBody' : 'analysisBody-' + page"
      class="body analysis-body"
    >
      <p v-if="error" role="alert" class="error">{{ error }}</p>
      <template v-if="result"
        ><p class="analysis-meta">
          {{ result.model }} · {{ result.promptVersion }} ·
          {{ result.generatedAt }}
        </p>
        <p class="analysis-summary">{{ result.summary }}</p>
        <p v-if="lastKey !== key" class="muted">
          This result belongs to an earlier scope or snapshot.
        </p>
        <section v-for="[title, items] in groups" :key="title">
          <h4>{{ title }}</h4>
          <div v-for="(item, i) in items" :key="i" class="analysis-item">
            <strong>{{ item.summary || item.title || "Finding" }}</strong
            ><small>{{
              [item.severity, item.basis, item.confidence]
                .filter(Boolean)
                .join(" · ")
            }}</small>
            <p>{{ item.detail || item.description || item.summary }}</p>
            <p v-if="item.recommendation">
              <strong>Recommendation</strong> {{ item.recommendation }}
            </p>
            <ul v-if="item.evidence?.length">
              <li v-for="(e, j) in item.evidence" :key="j">{{ e }}</li>
            </ul>
            <span
              v-for="(resource, r) in item.resources || []"
              :key="r"
              class="chip"
              >{{ resource.kind }}/{{ resource.namespace }}/{{
                resource.name
              }}</span
            >
          </div>
        </section>
        <h4>Limitations</h4>
        <p v-for="(limit, i) in result.limitations || []" :key="i">
          {{ limit }}
        </p></template
      >
      <p v-else class="empty">No scoped analysis yet.</p>
    </div>
  </div>
</template>
