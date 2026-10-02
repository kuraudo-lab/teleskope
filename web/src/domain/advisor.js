import { esc, arr, ref, chips, list, kv, join } from "./format.js";
export const advisorDomainCatalog = [
  ["networking", "Networking"],
  ["storage", "Storage"],
  ["eks", "EKS lifecycle"],
  ["extensions", "Platform APIs"],
];

export function advisorDomain(capability) {
  const prefix = String(capability.key || "").split(".")[0];
  return advisorDomainCatalog.some(([value]) => value === prefix)
    ? prefix
    : "extensions";
}

export function advisorTitle(capability) {
  const labels = {
    "networking.ingress": "Ingress class",
    "networking.gateway": "Gateway API implementation",
    "storage.rwx": "ReadWriteMany storage",
    "storage.expansion": "Volume expansion",
    "extensions.api": "Custom API definition",
    "eks.insight": "EKS upgrade insight",
    "eks.addon": "EKS managed add-on",
    "eks.nodegroup": "EKS nodegroup readiness",
  };
  return labels[capability.key] || capability.key || "Capability";
}

export function advisorTone(capability) {
  if (capability.assessment === "unsupported") return "attention";
  if (capability.coverage !== "complete") return "gap";
  if (capability.assessment === "unknown") return "unknown";
  return "supported";
}

export function advisorToneLabel(capability) {
  const tone = advisorTone(capability);
  return tone === "attention"
    ? "Needs attention"
    : tone === "gap"
      ? "Coverage gap"
      : tone === "unknown"
        ? "Unknown"
        : "Supported";
}

export function advisorPriority(capability) {
  if (capability.assessment === "unsupported") return 0;
  if (capability.coverage !== "complete") return 1;
  if (capability.assessment === "unknown") return 2;
  return 3;
}

export function advisorSearchText(capability) {
  return [
    capability.key,
    capability.scope,
    capability.implementation,
    capability.assessment,
    capability.basis,
    capability.summary,
    capability.coverage,
    capability.freshness,
    capability.ruleId,
    ...arr(capability.constraints),
    ...arr(capability.collection).flatMap((item) => [
      item.area,
      item.resource,
      item.status,
      item.reason,
    ]),
    ...arr(capability.evidence).flatMap((item) => [
      ref(item.resource),
      item.field,
      item.value,
    ]),
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
}

export function formatAdvisorTime(value) {
  if (!value || String(value).startsWith("0001-01-01")) return "not recorded";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
}

export function advisorFilterButton(value, label, count, active, tone = "") {
  return `<button type="button" class="advisor-filter ${tone} ${active ? "active" : ""}" data-value="${esc(value)}" aria-pressed="${active}"><span class="advisor-filter-dot" aria-hidden="true"></span>${esc(label)}<span class="advisor-filter-count">${count}</span></button>`;
}

export function advisorEvidenceHTML(capability) {
  const evidence = arr(capability.evidence);
  if (!evidence.length)
    return '<p class="muted">No resource or field evidence was recorded.</p>';
  return `<ul class="advisor-evidence-list">${evidence
    .map((item) => {
      const collected = formatAdvisorTime(item.collectedAt);
      return `<li><span><strong>${esc(ref(item.resource))}</strong><span>${esc(item.field)} = ${esc(item.value)}</span></span><time datetime="${esc(item.collectedAt || "")}">${esc(collected)}</time></li>`;
    })
    .join("")}</ul>`;
}

export function advisorCollectionHTML(capability) {
  const gaps = arr(capability.collection).filter(
    (item) => item.status !== "complete",
  );
  if (!gaps.length) return "";
  return `<div class="advisor-detail-section"><h4>Collection gaps · ${gaps.length}</h4><ul class="advisor-collection-list">${gaps.map((item) => `<li><span><strong>${esc([item.area, item.resource].filter(Boolean).join("/") || "collection")}</strong><span>${esc(item.reason || "Collection did not complete.")}</span></span><em>${esc(item.status || "unknown")}</em></li>`).join("")}</ul></div>`;
}

export function advisorCapabilityHTML(capability, open) {
  const key =
    String(capability.key || "") + "/" + String(capability.scope || "");
  const tone = advisorTone(capability);
  const constraints = arr(capability.constraints);
  const meta = [
    capability.scope || "cluster",
    capability.implementation,
  ].filter(Boolean);
  return `<article class="advisor-finding ${tone}"><details data-key="${esc(key)}" ${open ? "open" : ""}><summary><span class="advisor-finding-title"><span class="advisor-title-row"><span class="advisor-assessment">${esc(advisorToneLabel(capability))}</span><strong>${esc(advisorTitle(capability))}</strong></span><span class="advisor-meta">${meta.map((value) => `<span>${esc(value)}</span>`).join("")}</span></span><span class="advisor-verdict"><span class="advisor-kicker">Deterministic conclusion</span><p>${esc(capability.summary || "No conclusion recorded.")}</p></span><span class="advisor-expand" aria-hidden="true"></span></summary><div class="advisor-finding-details"><div class="advisor-detail-main"><div class="advisor-detail-section"><h4>Interpretation boundary</h4>${constraints.length ? `<ul class="advisor-boundaries">${constraints.map((value) => `<li>${esc(value)}</li>`).join("")}</ul>` : "<p>No additional interpretation boundary was recorded.</p>"}</div>${advisorCollectionHTML(capability)}<div class="advisor-trace"><div><span>Assessment</span><strong>${esc(capability.assessment || "unknown")}</strong></div><div><span>Basis</span><strong>${esc(capability.basis || "unknown")}</strong></div><div><span>Coverage</span><strong>${esc(capability.coverage || "unknown")}</strong></div><div><span>Freshness</span><strong>${esc(capability.freshness || "unknown")}</strong></div><div><span>Rule</span><strong>${esc(capability.ruleId || "unknown")}</strong></div></div></div><div class="advisor-detail-side"><div class="advisor-detail-section"><h4>Evidence · ${arr(capability.evidence).length}</h4>${advisorEvidenceHTML(capability)}</div></div></div></details></article>`;
}
