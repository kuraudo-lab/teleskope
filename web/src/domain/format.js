export const esc = (v) =>
  String(v ?? "-").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );

export const ref = (r) =>
  !r || !r.name
    ? "-"
    : `${(r.kind || "").toLowerCase()}${r.namespace ? "/" + r.namespace : ""}/${r.name}`;

export const arr = (v) => (Array.isArray(v) ? v : []);

export const list = (v) =>
  Array.isArray(v) && v.length
    ? v.map((x) => (typeof x === "object" ? ref(x) : x)).join(", ")
    : "-";

export const chips = (v) =>
  Array.isArray(v) && v.length
    ? v
        .map(
          (x) =>
            `<span class="chip">${esc(typeof x === "object" ? ref(x) : x)}</span>`,
        )
        .join("")
    : '<span class="muted">-</span>';

export const kv = (v) =>
  v && Object.keys(v).length
    ? Object.entries(v)
        .map(([a, b]) => `${a}=${b}`)
        .join(", ")
    : "-";

export const shortDigest = (v) => {
  if (!v) return "-";
  v = String(v)
    .replace(/^docker-pullable:\/\//, "")
    .replace(/^docker:\/\//, "")
    .replace(/^containerd:\/\//, "");
  const i = v.lastIndexOf("sha256:");
  if (i >= 0) return "sha256:" + v.slice(i + 7, i + 19);
  return v.length > 36 ? v.slice(0, 36) : v;
};

export const join = (v) => (Array.isArray(v) && v.length ? v.join(", ") : "-");

export function resourceMap(v) {
  if (!v) return "-";
  const keys = ["cpu", "memory", "pods", "ephemeral-storage", "nvidia.com/gpu"];
  const used = new Set();
  const parts = keys
    .filter((key) => v[key])
    .map((key) => {
      used.add(key);
      return `${key}=${v[key]}`;
    });
  Object.keys(v)
    .sort()
    .filter((key) => !used.has(key))
    .forEach((key) => parts.push(`${key}=${v[key]}`));
  return parts.join(", ") || "-";
}

export function nodeLabels(v) {
  if (!v) return "-";
  const keys = [
    "eks.amazonaws.com/nodegroup",
    "eks.amazonaws.com/compute-type",
    "node.kubernetes.io/instance-type",
    "topology.kubernetes.io/region",
    "topology.kubernetes.io/zone",
    "kubernetes.io/arch",
    "kubernetes.io/os",
  ];
  const parts = keys.filter((key) => v[key]).map((key) => `${key}=${v[key]}`);
  return parts.join(", ") || kv(v);
}

export function taints(v) {
  return Array.isArray(v) && v.length
    ? v
        .map(
          (t) =>
            `${t.key || "-"}${t.value ? "=" + t.value : ""}${t.effect ? ":" + t.effect : ""}`,
        )
        .join(", ")
    : "-";
}

export function gatewayListeners(v) {
  return Array.isArray(v) && v.length
    ? v
        .map(
          (l) =>
            `${l.name || "-"}:${l.protocol || "-"}/${l.port || "-"} host=${l.hostname || "-"} allowed=${join(l.allowedRoutes)}`,
        )
        .join(" | ")
    : "-";
}

export function gatewayParents(v) {
  return Array.isArray(v) && v.length
    ? v
        .map(
          (p) =>
            `${p.kind || "Gateway"}${p.namespace ? "/" + p.namespace : ""}/${p.name || "-"}${p.sectionName ? "#" + p.sectionName : ""}`,
        )
        .join(", ")
    : "-";
}

export function gatewayBackends(rules) {
  const refs = [];
  arr(rules).forEach((rule) =>
    arr(rule.backendRefs).forEach((ref) => refs.push(ref)),
  );
  return refs.length ? refs.map(ref).join(", ") : "-";
}

export function gatewayMatches(rules) {
  const matches = [];
  arr(rules).forEach((rule) =>
    arr(rule.matches).forEach((match) => matches.push(match)),
  );
  return matches.length ? matches.join(", ") : `rules=${arr(rules).length}`;
}

export function grantRefs(v) {
  return Array.isArray(v) && v.length
    ? v
        .map(
          (r) =>
            `${r.group ? r.group + "/" : ""}${r.kind || "-"}${r.namespace ? " ns=" + r.namespace : ""}${r.name ? " name=" + r.name : ""}`,
        )
        .join(", ")
    : "-";
}

export function gatewayPolicyTargets(v) {
  return Array.isArray(v) && v.length ? v.map(ref).join(", ") : "-";
}

export function rbacRules(v) {
  return Array.isArray(v) && v.length
    ? v
        .map(
          (r) =>
            `${join(r.verbs)}:${join(r.resources) || join(r.nonResourceUrls)}`,
        )
        .join(" | ")
    : "-";
}

export function rbacSubjects(v) {
  return Array.isArray(v) && v.length
    ? v
        .map(
          (s) =>
            `${s.kind || "-"}${s.namespace ? "/" + s.namespace : ""}/${s.name || "-"}`,
        )
        .join(", ")
    : "-";
}

export function webhookClient(w) {
  return w.clientUrl || ref(w.clientService);
}

export function limitItems(v) {
  return Array.isArray(v) && v.length
    ? v
        .map(
          (i) =>
            `${i.type || "-"} default=${kv(i.default)} request=${kv(i.defaultRequest)} min=${kv(i.min)} max=${kv(i.max)}`,
        )
        .join(" | ")
    : "-";
}

export function workloadUpdated(v) {
  return (
    [
      v.updatedReplicas ? `updated=${v.updatedReplicas}` : "",
      v.unavailableReplicas ? `unavailable=${v.unavailableReplicas}` : "",
      v.desiredScheduled ? `desired=${v.desiredScheduled}` : "",
      v.currentScheduled ? `current=${v.currentScheduled}` : "",
      v.misscheduled ? `misscheduled=${v.misscheduled}` : "",
      v.strategy ? `strategy=${v.strategy}` : "",
      v.updateStrategy ? `update=${v.updateStrategy}` : "",
    ]
      .filter(Boolean)
      .join(" ") || "-"
  );
}

export function workloadBatch(v) {
  return (
    [
      v.schedule ? `schedule=${v.schedule}` : "",
      v.parallelism ? `parallelism=${v.parallelism}` : "",
      v.completions ? `completions=${v.completions}` : "",
      v.active ? `active=${v.active}` : "",
      v.succeeded ? `succeeded=${v.succeeded}` : "",
      v.failed ? `failed=${v.failed}` : "",
      v.suspend !== undefined ? `suspend=${v.suspend}` : "",
    ]
      .filter(Boolean)
      .join(" ") || "-"
  );
}
