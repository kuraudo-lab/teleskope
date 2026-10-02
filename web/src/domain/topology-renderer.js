import {sourceLabel,sourceDetail} from "../runtime/source-status";
import {
  topologyCondensedHierarchy,
  topologyWeakComponents,
  topologyLayoutIsolates,
  topologyLayoutComponent,
  topologyOverviewLayout,
  topologyStraightEdges,
  topologyFocusLayout,
} from "./topology-layout.js";
import { esc, arr, ref, list, chips, kv } from "./format.js";

/** Imperative SVG island. All queries and event ownership are scoped to root. */
export function createTopologyRenderer(root, input, onSelect) {
  const abort = new AbortController();
  const document = {
    querySelectorAll: (selector) => root.querySelectorAll(selector),
    addEventListener: (type, listener) =>
      root.addEventListener(type, listener, { signal: abort.signal }),
  };
  const byId = (id) => root.querySelector("#" + id);
  let topologyGraph = input.graph || {},
    advisorReport = input.advisor || {};
  let snapshot = input.snapshot || {},
    k = snapshot.kubernetes || {};
  let disposed = false;
  function showText(text) {
    onSelect(JSON.parse(text));
  }
  const validTopologyParam = (value, allowed, fallback) =>
    allowed.includes(value) ? value : fallback;
  const topologyURL = new URLSearchParams(location.search);
  const topologyModeNames = ["graph", "table"];
  const topologyHealthNames = [
    "all",
    "healthy",
    "warning",
    "unknown",
    "stopped",
  ];
  const topologySystemNames = ["include", "exclude", "only"];
  const topologyStateNames = ["all", "connected", "disconnected", "stopped"];
  const topologyNumberParam = (name, fallback) => {
    const raw = topologyURL.get(name);
    if (raw === null || raw === "") return fallback;
    const value = Number(raw);
    return Number.isFinite(value) ? value : fallback;
  };
  let topologySearch = topologyURL.get("topologyQuery") || "";
  let topologySelectedNodeId = topologyURL.get("topologySelected") || "";
  let topologySelectedEdgeId = topologyURL.get("topologyEdge") || "";
  let topologyMode = validTopologyParam(
    topologyURL.get("topologyMode"),
    topologyModeNames,
    "graph",
  );
  let topologyHealthFilter = validTopologyParam(
    topologyURL.get("topologyHealth"),
    topologyHealthNames,
    "all",
  );
  let topologySystemFilter = validTopologyParam(
    topologyURL.get("topologySystem"),
    topologySystemNames,
    "include",
  );
  let topologyStateFilter = validTopologyParam(
    topologyURL.get("topologyState"),
    topologyStateNames,
    "all",
  );
  let topologyGroupReplicas = topologyURL.get("topologyGroup") !== "false";
  let topologySort = topologyURL.get("topologySort") || "kind";
  let topologySortDirection =
    topologyURL.get("topologyDirection") === "desc" ? "desc" : "asc";
  let topologyMetricKey = topologyURL.get("topologyMetric") || "";
  let topologyRelativeSort = "relation";
  let topologyRelativeDirection = "asc";
  let topologyRawExpanded = false;
  let topologyView = {
    scale: Math.max(0.35, Math.min(4, topologyNumberParam("topologyScale", 1))),
    x: topologyNumberParam("topologyX", 0),
    y: topologyNumberParam("topologyY", 0),
  };
  let topologyFocusView = {
    scale: Math.max(
      0.35,
      Math.min(4, topologyNumberParam("topologySelectedScale", 1)),
    ),
    x: topologyNumberParam("topologySelectedX", 0),
    y: topologyNumberParam("topologySelectedY", 0),
  };
  let topologyDrag = {
    active: false,
    moved: false,
    suppressClick: false,
    startX: 0,
    startY: 0,
    viewX: 0,
    viewY: 0,
    nodeId: "",
    edgeId: "",
  };
  let topologyRenderSample = 0;
  let topologyFitBounds = null;
  function topologySearchMatches(node) {
    if (!topologySearch) return true;
    const data = node.data || {};
    const haystack = [
      node.label,
      node.type,
      data.apiVersion,
      data.kind,
      data.namespace,
      data.name,
      data.image,
      data.nodeName,
      data.serviceAccountName,
      data.runtimeClassName,
      data.phase,
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    return haystack.includes(topologySearch);
  }
  const topologyWorkloadKinds = new Set([
    "pseudo-internet",
    "pseudo-external-endpoint",
    "ingress",
    "gateway",
    "gatewayroute",
    "service",
    "workload",
  ]);
  const topologyGraphPolicy = Object.freeze({
    mode: "unbounded",
    edges: "selection-only-straight",
  });
  function topologyKind(node) {
    return String(node?.kind || "").toLowerCase();
  }
  function topologyNodeHealth(node) {
    const metadata = node?.metadata || {},
      state = String(
        metadata.state || metadata.status || metadata.phase || "",
      ).toLowerCase();
    if (
      ["stopped", "terminated", "failed", "succeeded", "completed"].includes(
        state,
      )
    )
      return "stopped";
    if (
      arr(node?.findingRefs).length ||
      ["warning", "degraded", "error", "failed", "notready"].includes(state)
    )
      return "warning";
    if (topologyKind(node).startsWith("pseudo-") || state === "unknown")
      return "unknown";
    return "healthy";
  }
  function topologyIsSystem(node) {
    const namespace = String(node?.scope?.namespace || "");
    return namespace === "kube-system" || namespace.startsWith("kube-");
  }
  function topologyResourceMatches(node) {
    if (resourceFilter === "all") return true;
    const kind = topologyKind(node),
      aliases = {
        endpoint: ["endpoint-slice"],
        gatewayRoute: ["gatewayroute"],
        pvc: ["persistentvolumeclaim", "pvc"],
        storage: ["storageclass", "persistentvolume", "persistentvolumeclaim"],
        crd: ["customresourcedefinition", "customresource"],
        runtime: ["runtimeclass"],
      };
    return (aliases[resourceFilter] || [resourceFilter]).includes(kind);
  }
  function topologyCompare(actual, operator, expected) {
    if ([">", ">=", "<", "<="].includes(operator)) {
      const left = Number(actual),
        right = Number(expected);
      if (!Number.isFinite(left) || !Number.isFinite(right)) return false;
      return operator === ">"
        ? left > right
        : operator === ">="
          ? left >= right
          : operator === "<"
            ? left < right
            : left <= right;
    }
    const left = String(actual ?? "").toLowerCase(),
      right = String(expected ?? "").toLowerCase();
    return operator === "=" ? left === right : left.includes(right);
  }
  function topologySearchMatchesGraph(node) {
    const query = topologySearch.trim();
    if (!query) return true;
    const metadata = node.metadata || {},
      health = topologyNodeHealth(node),
      finding = arr(node.findingRefs).join(" ");
    const haystack = [
      node.id,
      node.identity,
      node.name,
      node.kind,
      node.apiVersion,
      node.scope?.provider,
      node.scope?.namespace,
      health,
      finding,
      ...Object.entries(metadata).flatMap(([key, value]) => [
        key,
        value,
        `${key}=${value}`,
      ]),
    ]
      .join(" ")
      .toLowerCase();
    const terms = query.match(/"[^"]*"|\S+/g) || [];
    return terms.every((raw) => {
      const term = raw.replace(/^"|"$/g, ""),
        match = term.match(/^([^:<>!=]+)(:|>=|<=|>|<|=)(.+)$/);
      if (!match) return haystack.includes(term.toLowerCase());
      const [, field, operator, expected] = match,
        key = field.toLowerCase();
      if (key === "kind") return topologyCompare(node.kind, operator, expected);
      if (key === "name" || key === "identity")
        return topologyCompare(node[key], operator, expected);
      if (key === "namespace" || key === "ns")
        return topologyCompare(node.scope?.namespace || "", operator, expected);
      if (key === "provider")
        return topologyCompare(node.scope?.provider || "", operator, expected);
      if (key === "health") return topologyCompare(health, operator, expected);
      if (key === "finding")
        return topologyCompare(finding, operator, expected);
      if (key === "label" || key === "metadata") {
        const [metadataKey, ...valueParts] = expected.split("=");
        return topologyCompare(
          metadata[metadataKey],
          valueParts.length ? "=" : ":",
          valueParts.join("="),
        );
      }
      return topologyCompare(metadata[field], operator, expected);
    });
  }
  function topologyReplicaGroup(node, edges, nodeById) {
    if (topologyKind(node) !== "pod") return "";
    const ownerEdge = edges.find(
      (edge) =>
        edge.kind === "ownership" &&
        edge.target === node.id &&
        topologyKind(nodeById.get(edge.source)) === "workload",
    );
    if (!ownerEdge) return "";
    const count = edges.filter(
      (edge) =>
        edge.kind === "ownership" &&
        edge.source === ownerEdge.source &&
        topologyKind(nodeById.get(edge.target)) === "pod",
    ).length;
    return `${nodeById.get(ownerEdge.source)?.name || ownerEdge.source} × ${count}`;
  }
  function topologyProjectedEdges(nodes, allNodes, allEdges) {
    const visibleIDs = new Set(nodes.map((node) => node.id)),
      nodeById = new Map(allNodes.map((node) => [node.id, node])),
      projected = new Map(),
      direct = allEdges.filter(
        (edge) => visibleIDs.has(edge.source) && visibleIDs.has(edge.target),
      );
    const add = (source, target, relation, path) => {
      if (
        !visibleIDs.has(source) ||
        !visibleIDs.has(target) ||
        source === target
      )
        return;
      const key = `projection/${source}/${target}/${relation}`,
        existing = projected.get(key),
        sourceEdges = path.map((edge) => edge.id),
        evidence = path.flatMap((edge) => arr(edge.evidence)),
        findingRefs = [
          ...new Set(path.flatMap((edge) => arr(edge.findingRefs))),
        ],
        providers = [
          ...new Set(path.map((edge) => edge.provider).filter(Boolean)),
        ];
      if (existing) {
        existing.metadata.sourceEdges = [
          ...new Set([
            ...String(existing.metadata.sourceEdges).split(","),
            ...sourceEdges,
          ]),
        ].join(",");
        existing.evidence.push(...evidence);
        existing.findingRefs = [
          ...new Set([...existing.findingRefs, ...findingRefs]),
        ];
        return;
      }
      projected.set(key, {
        id: key,
        source,
        target,
        kind: "resolved",
        provider: providers.join("+") || "projection",
        relation,
        directed: true,
        metadata: {
          projection: "evidence-backed workload projection",
          sourceEdges: sourceEdges.join(","),
        },
        evidence,
        findingRefs,
        projected: true,
      });
    };
    const serviceEndpoint = allEdges.filter(
      (edge) =>
        topologyKind(nodeById.get(edge.source)) === "service" &&
        topologyKind(nodeById.get(edge.target)) === "endpoint-slice",
    );
    serviceEndpoint.forEach((first) => {
      const service = first.source,
        endpoint = first.target;
      allEdges
        .filter((edge) => edge.source === endpoint)
        .forEach((second) => {
          const target = nodeById.get(second.target),
            targetKind = topologyKind(target);
          if (targetKind.startsWith("pseudo-"))
            add(service, second.target, "resolves external endpoint", [
              first,
              second,
            ]);
          if (targetKind !== "pod") return;
          allEdges
            .filter(
              (edge) =>
                edge.target === target.id &&
                edge.kind === "ownership" &&
                topologyKind(nodeById.get(edge.source)) === "workload",
            )
            .forEach((owner) =>
              add(service, owner.source, "backs workload", [
                first,
                second,
                owner,
              ]),
            );
        });
    });
    return [...direct, ...projected.values()];
  }
  function topologyFilteredGraph() {
    const allNodes = arr(topologyGraph?.nodes),
      allEdges = arr(topologyGraph?.edges),
      nodeById = new Map(allNodes.map((node) => [node.id, node]));
    const degree = new Map(allNodes.map((node) => [node.id, 0]));
    allEdges.forEach((edge) => {
      degree.set(edge.source, (degree.get(edge.source) || 0) + 1);
      degree.set(edge.target, (degree.get(edge.target) || 0) + 1);
    });
    const filtered = allNodes
      .filter(
        (node) => nsFilter === "all" || node.scope?.namespace === nsFilter,
      )
      .filter(
        (node) =>
          topologySystemFilter === "include" ||
          (topologySystemFilter === "only"
            ? topologyIsSystem(node)
            : !topologyIsSystem(node)),
      )
      .filter(
        (node) =>
          topologyHealthFilter === "all" ||
          topologyNodeHealth(node) === topologyHealthFilter,
      )
      .filter(
        (node) =>
          topologyStateFilter === "all" ||
          (topologyStateFilter === "connected"
            ? (degree.get(node.id) || 0) > 0
            : topologyStateFilter === "disconnected"
              ? (degree.get(node.id) || 0) === 0
              : topologyNodeHealth(node) === "stopped"),
      );
    let nodes = filtered
        .filter((node) => topologyWorkloadKinds.has(topologyKind(node)))
        .filter(topologyResourceMatches),
      edges = topologyProjectedEdges(nodes, allNodes, allEdges);
    const baseIDs = new Set(nodes.map((node) => node.id)),
      filteredIDs = new Set(filtered.map((node) => node.id));
    const graphReady =
      Boolean(topologyGraph?.schemaVersion) && allNodes.length > 0;
    if (
      topologySelectedNodeId &&
      graphReady &&
      !filteredIDs.has(topologySelectedNodeId)
    ) {
      topologySelectedNodeId = "";
      if (byId("drawer")?.classList.contains("open")) syncTopologyURL();
    }
    if (topologySelectedNodeId) {
      const focusIDs = new Set([topologySelectedNodeId]);
      allEdges.forEach((edge) => {
        if (edge.source === topologySelectedNodeId) focusIDs.add(edge.target);
        if (edge.target === topologySelectedNodeId) focusIDs.add(edge.source);
      });
      edges.forEach((edge) => {
        if (edge.source === topologySelectedNodeId) focusIDs.add(edge.target);
        if (edge.target === topologySelectedNodeId) focusIDs.add(edge.source);
      });
      nodes = [
        ...new Map(
          [...nodes, ...filtered.filter((node) => focusIDs.has(node.id))].map(
            (node) => [node.id, node],
          ),
        ).values(),
      ];
      const visibleIDs = new Set(nodes.map((node) => node.id)),
        rawIncident = allEdges.filter(
          (edge) =>
            (edge.source === topologySelectedNodeId ||
              edge.target === topologySelectedNodeId) &&
            visibleIDs.has(edge.source) &&
            visibleIDs.has(edge.target),
        ),
        projectedEdges = topologyProjectedEdges(
          nodes.filter((node) => baseIDs.has(node.id)),
          allNodes,
          allEdges,
        );
      edges = [
        ...new Map(
          [...projectedEdges, ...rawIncident].map((edge) => [edge.id, edge]),
        ).values(),
      ];
    }
    if (
      topologySelectedEdgeId &&
      graphReady &&
      !edges.some((edge) => edge.id === topologySelectedEdgeId)
    ) {
      topologySelectedEdgeId = "";
      syncTopologyURL();
    }
    return { nodes, edges, nodeById };
  }
  function topologyNodeDetail(node, incident) {
    const { x, y, w, h, ...resource } = node;
    return JSON.stringify({
      ...resource,
      relationships: incident.map((edge) => ({
        kind: edge.kind,
        relation: edge.relation,
        provider: edge.provider,
        source: edge.source,
        target: edge.target,
        evidence: edge.evidence,
      })),
    });
  }
  function topologyMetricCatalog() {
    const catalog = new Map();
    arr(topologyGraph?.nodes).forEach((node) =>
      arr(node.metrics).forEach((metric) => {
        if (!catalog.has(metric.key)) catalog.set(metric.key, metric);
      }),
    );
    return [...catalog.values()].sort(
      (a, b) => a.label.localeCompare(b.label) || a.key.localeCompare(b.key),
    );
  }
  function topologyMetricValue(metric) {
    const samples = arr(metric?.samples);
    return samples.length ? samples[samples.length - 1].value : null;
  }
  function topologyMetricSparkline(metric) {
    const samples = arr(metric?.samples);
    if (samples.length < 2)
      return '<p class="muted">Single snapshot point · history unavailable</p>';
    const values = samples.map((sample) => Number(sample.value)),
      min = Math.min(...values),
      max = Math.max(...values),
      span = max - min || 1,
      points = values
        .map(
          (value, index) =>
            `${(index / (values.length - 1)) * 120},${32 - ((value - min) / span) * 28}`,
        )
        .join(" ");
    return `<svg viewBox="0 0 120 36" role="img" aria-label="${esc(metric.label)} sparkline"><polyline class="metric-line" points="${points}"/>${values.map((value, index) => `<circle class="metric-dot" cx="${(index / (values.length - 1)) * 120}" cy="${32 - ((value - min) / span) * 28}" r="2"><title>${esc(String(value))} ${esc(metric.unit)} · ${esc(samples[index].at)}</title></circle>`).join("")}</svg>`;
  }
  function topologyFindingRef(item) {
    return `${item?.ruleId || ""}::${item?.scope || ""}`;
  }
  function topologyFinding(ref) {
    return (
      arr(advisorReport?.capabilities).find(
        (item) => topologyFindingRef(item) === ref,
      ) || null
    );
  }
  function topologyRelatives(node, nodeById, edges) {
    const rows = [],
      add = (relation, target, edge = null) => {
        if (!target || target.id === node.id) return;
        const key = relation + "\u0000" + target.id;
        if (rows.some((row) => row.key === key)) return;
        rows.push({ key, relation, node: target, edge });
      };
    edges.forEach((edge) => {
      if (edge.source !== node.id && edge.target !== node.id) return;
      const outgoing = edge.source === node.id,
        target = nodeById.get(outgoing ? edge.target : edge.source);
      if (!target) return;
      const direction = outgoing ? "downstream" : "upstream",
        relation =
          topologyLayer(target) === topologyLayer(node)
            ? direction
            : "cross-layer " + direction;
      add(relation, target, edge);
    });
    const value = (row, key) =>
      key === "name"
        ? row.node.name
        : key === "kind"
          ? row.node.kind
          : row.relation;
    rows.sort((a, b) => {
      const result = String(value(a, topologyRelativeSort)).localeCompare(
        String(value(b, topologyRelativeSort)),
      );
      return topologyRelativeDirection === "desc" ? -result : result;
    });
    return rows;
  }
  function topologyMetadataHTML(metadata) {
    const entries = Object.entries(metadata || {}).sort(([a], [b]) =>
      a.localeCompare(b),
    );
    return entries.length
      ? `<table><tbody>${entries.map(([key, value]) => `<tr><th scope="row">${esc(key)}</th><td>${esc(String(value))}</td></tr>`).join("")}</tbody></table>`
      : '<p class="muted topology-metric-control">No collected metadata for this object.</p>';
  }
  function topologyResolvedFindings(refs) {
    return arr(refs).map(
      (ref) =>
        topologyFinding(ref) || {
          ruleId: ref,
          summary: "Referenced finding is unavailable in this report revision.",
        },
    );
  }
  function topologyFindingsHTML(refs) {
    const findings = topologyResolvedFindings(refs);
    return (
      findings
        .map((finding) => {
          const tone =
            finding.assessment === "unsupported"
              ? "warning"
              : finding.coverage && finding.coverage !== "complete"
                ? "coverage gap"
                : finding.assessment || "unknown";
          const evidence =
            arr(finding.evidence)
              .map(
                (item) =>
                  `<li><strong>${esc(ref(item.resource))}</strong><br>${esc(item.field || "-")} = ${esc(item.value || "-")}<br><small>${esc(item.collectedAt || "time unknown")}</small></li>`,
              )
              .join("") || "<li>No resource evidence recorded.</li>";
          const collection =
            arr(finding.collection)
              .map(
                (item) =>
                  `<li><strong>${esc([item.area, item.resource].filter(Boolean).join("/") || "collection")}</strong> · ${esc(item.status || "unknown")}${item.reason ? `<br>${esc(item.reason)}` : ""}</li>`,
              )
              .join("") || "<li>No collection gaps recorded.</li>";
          return `<div class="topology-finding"><strong>${esc(finding.ruleId || "unknown rule")} · ${esc(tone)}</strong><small>${esc(finding.basis || "none")} · coverage ${esc(finding.coverage || "unknown")} · freshness ${esc(finding.freshness || "unknown")}</small><p>${esc(finding.summary || "")}</p>${arr(
            finding.constraints,
          )
            .map((item) => `<p class="muted">${esc(item)}</p>`)
            .join(
              "",
            )}<details><summary>Finding evidence · ${arr(finding.evidence).length}</summary><ul>${evidence}</ul></details><details><summary>Collection status · ${arr(finding.collection).length}</summary><ul>${collection}</ul></details></div>`;
        })
        .join("") ||
      '<p class="muted topology-metric-control">No deterministic findings reference this object.</p>'
    );
  }
  function topologyMetricHTML(node) {
    const catalog = topologyMetricCatalog(),
      pinnedDefinition = catalog.find(
        (metric) => metric.key === topologyMetricKey,
      ),
      options = [
        '<option value="">No canvas metric</option>',
        ...catalog.map(
          (metric) =>
            `<option value="${esc(metric.key)}" ${metric.key === topologyMetricKey ? "selected" : ""}>${esc(metric.label)} · ${esc(metric.semantic)}</option>`,
        ),
      ];
    if (topologyMetricKey && !pinnedDefinition)
      options.push(
        `<option value="${esc(topologyMetricKey)}" selected>${esc(topologyMetricKey)} · unavailable</option>`,
      );
    const selectedMetric = topologyMetricKey
        ? arr(node.metrics).find((metric) => metric.key === topologyMetricKey)
        : arr(node.metrics)[0],
      metricSamples = arr(selectedMetric?.samples),
      unavailable = topologyMetricKey
        ? `Pinned metric ${topologyMetricKey} is unavailable for this object in the current revision.`
        : "Metrics unavailable for this object. Capacity and declared values are shown only when collected; runtime usage requires an explicit external provider.";
    return `<div class="topology-metric-control"><label for="topologyMetric">Pin metric to canvas</label><select id="topologyMetric">${options.join("")}</select></div>${selectedMetric ? `<div class="topology-metric-card"><strong>${esc(selectedMetric.label)} · ${esc(formatTopologyMetric(selectedMetric))}</strong><p>${esc(selectedMetric.semantic)} · ${esc(selectedMetric.provider)} · ${esc(selectedMetric.sampling)} · ${esc(selectedMetric.freshness)}</p>${topologyMetricSparkline(selectedMetric)}<small>${esc(selectedMetric.windowStart || metricSamples[0]?.at || "time unknown")} → ${esc(selectedMetric.windowEnd || metricSamples[metricSamples.length - 1]?.at || "time unknown")}</small></div>` : `<p class="muted topology-metric-control">${esc(unavailable)}</p>`}`;
  }
  function topologyEdgeEvidenceHTML(edge) {
    const evidence = arr(edge?.evidence),
      nodeById = new Map(
        arr(topologyGraph?.nodes).map((node) => [node.id, node]),
      ),
      normalize = (value) =>
        String(value || "")
          .toLowerCase()
          .replace(/[^a-z0-9]/g, ""),
      kinds = [
        topologyKind(nodeById.get(edge?.source)),
        topologyKind(nodeById.get(edge?.target)),
      ]
        .filter(Boolean)
        .map(normalize),
      coverage = arr(topologyGraph?.coverage)
        .filter(
          (item) =>
            item.provider === edge?.provider &&
            kinds.some((kind) =>
              normalize(item.capability).includes(kind.replace(/^pseudo/, "")),
            ),
        )
        .slice(0, 6);
    const evidenceRows =
      evidence
        .map(
          (item) =>
            `<div class="topology-finding"><strong>${esc(item.basis || edge.kind || "unknown")} · ${esc(item.provider || edge.provider || "-")}</strong><small>freshness ${esc(item.freshness || "unknown")} · confidence ${esc(item.confidence || "unknown")}</small><p>${esc(item.summary || "No evidence summary recorded.")}</p></div>`,
        )
        .join("") ||
      '<p class="muted topology-metric-control">No relationship evidence was recorded.</p>';
    const coverageRows =
      coverage
        .map(
          (item) =>
            `<div class="topology-coverage-item"><strong>${esc(item.capability || "provider coverage")} · ${esc(item.provider || "-")}</strong><br>${esc(item.status || "unknown")}${item.reason ? ` · ${esc(item.reason)}` : ""}</div>`,
        )
        .join("") ||
      '<div class="topology-coverage-item"><strong>Coverage · unknown</strong><br>No matching provider coverage record.</div>';
    return `<div class="topology-finding"><small>Direction</small><strong>${esc(edge?.directed === false ? "undirected" : `${edge?.source || "-"} → ${edge?.target || "-"}`)}</strong><small>Relationship type</small><strong>${esc(edge?.kind || "unknown")} — ${esc(edge?.relation || "unspecified")}</strong></div>${evidenceRows}${coverageRows}`;
  }
  function topologyNodeEvidenceHTML(node) {
    const rows = arr(node?.evidence)
      .map(
        (item) =>
          `<div class="topology-coverage-item"><strong>${esc(item.basis || "unknown")} · ${esc(item.provider || node.scope?.provider || "-")}</strong><br>freshness ${esc(item.freshness || "unknown")} · ${esc(item.summary || "No evidence summary recorded.")}</div>`,
      )
      .join("");
    return (
      rows ||
      '<div class="topology-coverage-item"><strong>Source freshness · unknown</strong><br>No object evidence timestamp was recorded.</div>'
    );
  }
  function topologyLayer(node) {
    const kind = topologyKind(node);
    if (
      [
        "pseudo-internet",
        "pseudo-external-endpoint",
        "ingress",
        "gateway",
        "gatewayroute",
        "service",
      ].includes(kind)
    )
      return "application";
    if (["namespace", "workload"].includes(kind)) return "workload";
    if (["endpoint-slice", "pod", "node"].includes(kind)) return "runtime";
    return "infrastructure";
  }
  function topologyVisualKind(kind) {
    if (String(kind).startsWith("pseudo-")) return "external";
    if (["ingress", "gateway", "gatewayroute"].includes(kind)) return "gateway";
    if (kind === "service" || kind === "endpoint-slice") return "service";
    if (kind === "workload") return "workload";
    if (kind === "pod") return "pod";
    return "infrastructure";
  }
  function topologyIconID(kind) {
    return `topology-icon-${topologyVisualKind(kind)}`;
  }
  function topologyNodeShape(node) {
    const kind = topologyVisualKind(topologyKind(node)),
      cx = node.x + node.w / 2,
      cy = node.y + 30;
    if (kind === "pod")
      return `<rect class="node-shape" x="${cx - 20}" y="${cy - 20}" width="40" height="40" rx="12" transform="rotate(45 ${cx} ${cy})"/>`;
    if (kind === "infrastructure")
      return `<rect class="node-shape" x="${cx - 26}" y="${cy - 26}" width="52" height="52" rx="5"/>`;
    return `<circle class="node-shape" cx="${cx}" cy="${cy}" r="26"/>`;
  }
  function topologyNodeVisualBounds(node) {
    const metricOverflow = topologyMetricKey ? 32 : 0,
      pad = 12;
    return {
      minX: node.x - pad,
      minY: node.y - pad,
      maxX: node.x + node.w + metricOverflow + pad,
      maxY: node.y + node.h + pad,
    };
  }

  let activeSection = "overview";
  let nsFilter = topologyURL.get("namespace") || "all";
  let resourceFilter = topologyURL.get("resource") || "all";
  function currentTopologyView() {
    return topologySelectedNodeId ? topologyFocusView : topologyView;
  }
  function fitTopologyToContent() {
    const surface = byId("topology"),
      host = surface?.parentElement,
      bounds = topologyFitBounds;
    if (!surface || !host || !bounds) return;
    const width = Math.max(1, bounds.maxX - bounds.minX),
      height = Math.max(1, bounds.maxY - bounds.minY),
      viewportWidth = Math.max(1, host.clientWidth - 36),
      viewportHeight = Math.max(1, host.clientHeight - 36),
      scale = Math.max(
        0.12,
        Math.min(
          4,
          Math.min(viewportWidth / width, viewportHeight / height) * 0.64,
        ),
      ),
      view = currentTopologyView();
    view.scale = scale;
    view.x = (viewportWidth - width * scale) / 2 - bounds.minX * scale + 18;
    view.y = (viewportHeight - height * scale) / 2 - bounds.minY * scale + 18;
    syncTopologyURL();
    applyTopologyTransform();
  }
  function syncTopologyURL() {
    const url = new URL(location.href);
    const set = (key, value, defaultValue = "") => {
      if (value === defaultValue || value === "" || value === false)
        url.searchParams.delete(key);
      else url.searchParams.set(key, String(value));
    };
    set("namespace", nsFilter, "all");
    set("resource", resourceFilter, "all");
    set("topologyQuery", topologySearch);
    url.searchParams.delete("topologyView");
    url.searchParams.delete("topologyFocus");
    set("topologyMode", topologyMode, "graph");
    set("topologyHealth", topologyHealthFilter, "all");
    set("topologySystem", topologySystemFilter, "include");
    set("topologyState", topologyStateFilter, "all");
    set("topologySelected", topologySelectedNodeId);
    set("topologyEdge", topologySelectedEdgeId);
    set("topologyGroup", topologyGroupReplicas ? "" : "false");
    set("topologySort", topologySort, "kind");
    set("topologyDirection", topologySortDirection, "asc");
    set("topologyMetric", topologyMetricKey);
    set("topologyScale", Math.round(topologyView.scale * 1000) / 1000, 1);
    set("topologyX", Math.round(topologyView.x), 0);
    set("topologyY", Math.round(topologyView.y), 0);
    set(
      "topologySelectedScale",
      Math.round(topologyFocusView.scale * 1000) / 1000,
      1,
    );
    set("topologySelectedX", Math.round(topologyFocusView.x), 0);
    set("topologySelectedY", Math.round(topologyFocusView.y), 0);
    history.replaceState(null, "", url);
  }
  function currentTopologyResourceFilter() {
    return activeSection === "overview" ? resourceFilter : "all";
  }
  function selectTopologyNode(nodeId) {
    if (nodeId && nodeId !== topologySelectedNodeId)
      topologyFocusView = { scale: 1, x: 0, y: 0 };
    topologySelectedNodeId = nodeId || "";
    topologySelectedEdgeId = "";
    topologyRawExpanded = false;
    syncTopologyURL();
    renderTopology();
  }
  function selectTopologyEdge(edgeId) {
    topologySelectedEdgeId = edgeId || "";
    topologyRawExpanded = false;
    syncTopologyURL();
    renderTopology();
  }
  function clearTopologySelection() {
    if (!topologySelectedNodeId && !topologySelectedEdgeId) return;
    topologySelectedNodeId = "";
    topologySelectedEdgeId = "";
    topologyRawExpanded = false;
    syncTopologyURL();
    renderTopology();
  }
  function updateTopologySelectionSummary(
    node,
    incident,
    selectedEdge,
    visibleEdges = [],
  ) {
    const summary = byId("topologySelectedSummary"),
      context = byId("topologyContext"),
      coverage = byId("topologyCoverage"),
      inspectorTitle = byId("topologyInspectorTitle");
    if (!summary || !context || !coverage) return;
    summary.innerHTML = node
      ? `<span>Object</span><strong title="${esc(node.identity || node.id)}">${esc(node.name)}</strong><span>Identity</span><strong title="${esc(node.identity || node.id)}">${esc(node.identity || node.id)}</strong><span>Health</span><strong>${esc(topologyNodeHealth(node))}</strong><span>Relationships</span><strong>${incident.length}</strong><span>Findings</span><strong>${esc(arr(node.findingRefs).join(", ") || "none")}</strong>`
      : selectedEdge
        ? `<span>Relationship</span><strong>${esc(selectedEdge.relation || selectedEdge.kind)}</strong><span>Provider</span><strong>${esc(selectedEdge.provider)}</strong><span>Evidence tier</span><strong>${esc(selectedEdge.kind)}</strong><span>Findings</span><strong>${esc(arr(selectedEdge.findingRefs).join(", ") || "none")}</strong>`
        : "<span>State</span><strong>Select an object or relationship</strong><span>Layout</span><strong>Top-to-bottom relationships</strong><span>Projection</span><strong>Workloads</strong>";
    if (inspectorTitle)
      inspectorTitle.textContent = node
        ? node.name
        : selectedEdge
          ? `${selectedEdge.relation || selectedEdge.kind} relationship`
          : "Cluster summary";
    const evidence = node
      ? [
          ...arr(node.evidence),
          ...incident.flatMap((edge) => arr(edge.evidence)),
        ]
      : selectedEdge
        ? arr(selectedEdge.evidence)
        : [];
    const evidenceHTML = evidence
      .slice(0, 6)
      .map(
        (item) =>
          `<div class="topology-coverage-item"><strong>${esc(item.basis || "evidence")} · ${esc(item.provider || "-")}</strong><br>${esc(item.summary || item.freshness || "-")}</div>`,
      )
      .join("");
    const coverageHTML = arr(topologyGraph?.coverage)
      .slice(0, 8)
      .map(
        (item) =>
          `<div class="topology-coverage-item"><strong>${esc(item.provider)} · ${esc(item.capability)}</strong><br>${esc(item.status)}${item.reason ? ` · ${esc(item.reason)}` : ""}</div>`,
      )
      .join("");
    coverage.innerHTML = evidenceHTML + coverageHTML;
    const allNodes = arr(topologyGraph?.nodes),
      nodeById = new Map(allNodes.map((item) => [item.id, item])),
      allEdges = visibleEdges.length ? visibleEdges : arr(topologyGraph?.edges);
    if (selectedEdge) {
      const source = nodeById.get(selectedEdge.source),
        target = nodeById.get(selectedEdge.target),
        findings = topologyResolvedFindings(selectedEdge.findingRefs),
        endpoint = (label, item) =>
          `<div class="topology-finding"><small>${label}</small><button type="button" class="topology-relative-select" data-topology-select="${esc(item?.id || "")}">${esc(item?.name || "unavailable")}</button></div>`;
      const raw = {
        edge: selectedEdge,
        source,
        target,
        findings,
        coverage: arr(topologyGraph?.coverage),
      };
      context.innerHTML = `<details open><summary>Endpoints</summary>${endpoint("Source", source)}${endpoint("Target", target)}</details><details open><summary>Relationship evidence</summary>${topologyEdgeEvidenceHTML(selectedEdge)}</details><details open><summary>Relationship metadata</summary>${topologyMetadataHTML(selectedEdge.metadata)}</details><details open><summary>Deterministic findings · ${arr(selectedEdge.findingRefs).length}</summary>${topologyFindingsHTML(selectedEdge.findingRefs)}</details><details data-topology-raw ${topologyRawExpanded ? "open" : ""}><summary>Raw evidence</summary><pre class="topology-raw">${esc(JSON.stringify(raw, null, 2))}</pre></details>`;
      return;
    }
    if (!node) {
      context.innerHTML =
        '<p class="muted">Select an object or relationship to inspect context, findings, metrics, and raw evidence.</p>';
      return;
    }
    const relatives = topologyRelatives(node, nodeById, allEdges),
      headers = [
        ["relation", "Relation"],
        ["kind", "Kind"],
        ["name", "Object"],
      ];
    const relativeRows =
      relatives
        .map((row) => {
          const relatedIncident = allEdges.filter(
              (edge) =>
                edge.source === row.node.id || edge.target === row.node.id,
            ),
            evidence = arr(row.edge?.evidence)[0];
          return `<tr><td>${esc(row.relation)}</td><td>${esc(row.node.kind)}</td><td><button type="button" class="topology-relative-select" data-topology-select="${esc(row.node.id)}" data-detail="${esc(topologyNodeDetail(row.node, relatedIncident))}">${esc(row.node.name)}</button>${row.edge ? `<br><button type="button" class="topology-relative-select" data-topology-edge="${esc(row.edge.id)}"><small>${esc(row.edge.source === node.id ? "outgoing" : "incoming")} · ${esc(row.edge.kind)} · source ${esc(evidence?.provider || row.edge.provider || "-")} · freshness ${esc(evidence?.freshness || "unknown")}</small></button>` : ""}</td></tr>`;
        })
        .join("") ||
      '<tr><td colspan="3" class="muted">No collected relatives.</td></tr>';
    const findings = topologyResolvedFindings(node.findingRefs),
      raw = {
        node,
        relationships: incident,
        findings,
        coverage: arr(topologyGraph?.coverage),
        metrics: arr(node.metrics),
      };
    context.innerHTML = `<details open><summary>Context and source freshness</summary>${topologyNodeEvidenceHTML(node)}${topologyMetadataHTML(node.metadata)}</details><details open><summary>Direct relationships · ${relatives.length}</summary><table><thead><tr>${headers.map(([key, label]) => `<th><button type="button" data-relative-sort="${key}">${label}${topologyRelativeSort === key ? (topologyRelativeDirection === "asc" ? " ↑" : " ↓") : ""}</button></th>`).join("")}</tr></thead><tbody>${relativeRows}</tbody></table></details><details open><summary>Advisor findings · ${arr(node.findingRefs).length}</summary>${topologyFindingsHTML(node.findingRefs)}</details><details open><summary>Metrics and evidence type · ${arr(node.metrics).length}</summary>${topologyMetricHTML(node)}</details><details open><summary>Children and placement</summary>${topologyMetadataHTML({ parentId: node.parentId || "none", children: allNodes.filter((item) => item.parentId === node.id).length })}</details><details data-topology-raw ${topologyRawExpanded ? "open" : ""}><summary>Object evidence and raw data</summary><pre class="topology-raw">${esc(JSON.stringify(raw, null, 2))}</pre></details>`;
  }
  function formatTopologyMetric(metric) {
    const value = topologyMetricValue(metric);
    if (value === null) return "unavailable";
    if (metric.unit === "bytes") {
      const units = ["B", "KiB", "MiB", "GiB", "TiB"];
      let current = Number(value),
        i = 0;
      while (Math.abs(current) >= 1024 && i < units.length - 1) {
        current /= 1024;
        i++;
      }
      return `${current >= 10 ? current.toFixed(0) : current.toFixed(1)} ${units[i]}`;
    }
    if (metric.unit === "mCPU") return `${Number(value).toFixed(0)}m`;
    return `${Number(value).toLocaleString()} ${metric.unit || ""}`.trim();
  }
  function installTopologyHoverInteractions(edges) {
    const surface = byId("topology");
    if (!surface) return;
    surface.querySelectorAll(".node[data-node-id]").forEach((element) => {
      const reset = () =>
        surface
          .querySelectorAll(".node,.link")
          .forEach((item) =>
            item.classList.remove(
              "hover-dimmed",
              "hover-neighbor",
              "hover-incident",
            ),
          );
      element.addEventListener("mouseenter", () => {
        const id = element.dataset.nodeId,
          neighborIDs = new Set([id]),
          incidentIDs = new Set();
        edges.forEach((edge) => {
          if (edge.source === id || edge.target === id) {
            neighborIDs.add(edge.source);
            neighborIDs.add(edge.target);
            incidentIDs.add(edge.id);
          }
        });
        surface.querySelectorAll(".node[data-node-id]").forEach((node) => {
          node.classList.toggle(
            "hover-dimmed",
            !neighborIDs.has(node.dataset.nodeId),
          );
          node.classList.toggle(
            "hover-neighbor",
            neighborIDs.has(node.dataset.nodeId) && node.dataset.nodeId !== id,
          );
        });
        surface.querySelectorAll(".link[data-edge-id]").forEach((edge) => {
          edge.classList.toggle(
            "hover-dimmed",
            !incidentIDs.has(edge.dataset.edgeId),
          );
          edge.classList.toggle(
            "hover-incident",
            incidentIDs.has(edge.dataset.edgeId),
          );
        });
      });
      element.addEventListener("mouseleave", reset);
      element.addEventListener("focus", () =>
        element.dispatchEvent(new Event("mouseenter")),
      );
      element.addEventListener("blur", reset);
    });
  }
  function renderTopologyTable(nodes, edges, nodeById) {
    const allEdges = edges,
      incidentCount = (id) =>
        allEdges.filter((edge) => edge.source === id || edge.target === id)
          .length;
    const value = (node, key) =>
      key === "health"
        ? topologyNodeHealth(node)
        : key === "namespace"
          ? node.scope?.namespace || "cluster"
          : key === "provider"
            ? node.scope?.provider
            : key === "relationships"
              ? incidentCount(node.id)
              : node[key] || "";
    const sorted = [...nodes].sort((a, b) => {
      if (topologyGroupReplicas) {
        const groupOrder = topologyReplicaGroup(
          a,
          allEdges,
          nodeById,
        ).localeCompare(topologyReplicaGroup(b, allEdges, nodeById));
        if (groupOrder) return groupOrder;
      }
      const result = String(value(a, topologySort)).localeCompare(
        String(value(b, topologySort)),
        undefined,
        { numeric: true },
      );
      return topologySortDirection === "desc" ? -result : result;
    });
    const headers = [
      ["kind", "Kind"],
      ["name", "Name"],
      ["namespace", "Namespace"],
      ["provider", "Provider"],
      ["health", "Health"],
      ["relationships", "Relations"],
    ];
    const rows = sorted
      .map((node) => {
        const incident = allEdges.filter(
            (edge) => edge.source === node.id || edge.target === node.id,
          ),
          group = topologyReplicaGroup(node, allEdges, nodeById);
        return `<tr aria-selected="${node.id === topologySelectedNodeId}"><td>${esc(node.kind)}</td><td><button class="topology-table-select" type="button" data-topology-select="${esc(node.id)}" data-detail="${esc(topologyNodeDetail(node, incident))}">${esc(node.name)}</button>${group ? `<br><small class="muted">replica group ${esc(group)} · individual evidence retained</small>` : ""}</td><td>${esc(node.scope?.namespace || "cluster")}</td><td>${esc(node.scope?.provider || "-")}</td><td>${esc(topologyNodeHealth(node))}</td><td>${incident.length}</td></tr>`;
      })
      .join("");
    byId("topology").innerHTML =
      `<div class="topology-table-wrap"><table class="topology-table"><thead><tr>${headers.map(([key, label]) => `<th scope="col"><button type="button" data-topology-sort="${key}" aria-label="Sort by ${label}">${label}${topologySort === key ? (topologySortDirection === "asc" ? " ↑" : " ↓") : ""}</button></th>`).join("")}</tr></thead><tbody>${rows}</tbody></table></div>`;
    byId("topology")
      .querySelectorAll("[data-topology-sort]")
      .forEach((button) =>
        button.addEventListener("click", () => {
          const key = button.dataset.topologySort;
          topologySortDirection =
            topologySort === key && topologySortDirection === "asc"
              ? "desc"
              : "asc";
          topologySort = key;
          syncTopologyURL();
          renderTopology();
        }),
      );
  }
  function recordTopologyRender(start, nodes, edges, mode) {
    const sample = ++topologyRenderSample;
    requestAnimationFrame(() =>
      requestAnimationFrame(() => {
        if (sample !== topologyRenderSample) return;
        const surface = byId("topology");
        if (!surface) return;
        surface.dataset.renderMs = (performance.now() - start).toFixed(2);
        surface.dataset.renderNodes = String(nodes);
        surface.dataset.renderEdges = String(edges);
        surface.dataset.renderMode = mode;
      }),
    );
  }
  function renderTopology() {
    const renderStarted = performance.now();
    const graph = topologyFilteredGraph(),
      { nodes, edges, nodeById } = graph,
      selected = nodeById.get(topologySelectedNodeId),
      selectedEdge = edges.find((edge) => edge.id === topologySelectedEdgeId),
      incident = selected
        ? edges.filter(
            (edge) =>
              edge.source === selected.id || edge.target === selected.id,
          )
        : [];
    document
      .querySelectorAll("[data-topology-mode]")
      .forEach((button) =>
        button.setAttribute(
          "aria-pressed",
          String(button.dataset.topologyMode === topologyMode),
        ),
      );
    const tableNodes = topologySearch
        ? nodes.filter(topologySearchMatchesGraph)
        : nodes,
      warningCount = nodes.filter(
        (node) => topologyNodeHealth(node) === "warning",
      ).length,
      setText = (id, value) => {
        const element = byId(id);
        if (element) element.textContent = value;
      };
    setText(
      "topologyObjectStat",
      topologySearch
        ? nodes.filter(topologySearchMatchesGraph).length
        : nodes.length,
    );
    setText("topologyEdgeStat", edges.length);
    setText("topologyWarningStat", warningCount);
    setText(
      "topologyScope",
      `${nsFilter === "all" ? "All namespaces" : nsFilter} · Workloads`,
    );
    setText(
      "topologyTitle",
      topologyMode === "graph"
        ? "Workload relationship map"
        : "Relationship table",
    );
    setText(
      "topologyGatewayCount",
      nodes.filter((node) =>
        ["ingress", "gateway", "gatewayroute"].includes(topologyKind(node)),
      ).length,
    );
    setText(
      "topologyServiceCount",
      nodes.filter((node) => topologyKind(node) === "service").length,
    );
    setText(
      "topologyWorkloadCount",
      nodes.filter((node) => topologyKind(node) === "workload").length,
    );
    setText(
      "topologyPodCount",
      nodes.filter((node) => topologyKind(node) === "pod").length,
    );
    setText(
      "topologyStorageCount",
      nodes.filter((node) =>
        ["persistentvolume", "persistentvolumeclaim", "storageclass"].includes(
          topologyKind(node),
        ),
      ).length,
    );
    setText(
      "topologyImageCount",
      nodes.filter((node) => topologyKind(node) === "image").length,
    );
    const complexityNotice = byId("topologyComplexityNotice");
    if (complexityNotice) {
      complexityNotice.hidden = true;
      complexityNotice.textContent = "";
    }
    updateTopologySelectionSummary(selected, incident, selectedEdge, edges);
    if (!nodes.length) {
      byId("topology").innerHTML =
        '<div class="empty">No topology objects match the current view and filters.</div>';
      recordTopologyRender(renderStarted, 0, 0, "empty");
      return;
    }
    if (topologyMode === "table") {
      renderTopologyTable(tableNodes, edges, nodeById);
      recordTopologyRender(
        renderStarted,
        tableNodes.length,
        edges.length,
        "table",
      );
      return;
    }
    byId("topology").classList.toggle(
      "focused",
      Boolean(topologySelectedNodeId),
    );
    if (topologySelectedNodeId) {
      byId("topology").style.minWidth = "";
      byId("topology").style.minHeight = "";
    }
    const W = topologySelectedNodeId
        ? byId("topology").clientWidth || 600
        : Math.max(900, byId("topology").clientWidth || 900),
      H = Math.max(560, byId("topology").clientHeight || 560),
      nodeW = 120,
      nodeH = 92;
    const drawn = topologyOverviewLayout(nodes, edges, W, nodeW, nodeH);
    const overviewBounds = [...drawn.values()].map(topologyNodeVisualBounds),
      overviewW = Math.max(
        W,
        drawn.layoutBounds?.width || 0,
        ...overviewBounds.map((bounds) => bounds.maxX + 42),
      ),
      overviewH = Math.max(
        H,
        drawn.layoutBounds?.height || 0,
        ...overviewBounds.map((bounds) => bounds.maxY + 42),
      );
    let contentW = topologySelectedNodeId ? W : overviewW,
      contentH = topologySelectedNodeId ? H : overviewH;
    topologyFocusLayout(
      drawn,
      topologySelectedNodeId,
      edges,
      W,
      H,
      nodeW,
      nodeH,
    );
    const selectedNeighbors = new Set([topologySelectedNodeId]);
    incident.forEach((edge) => {
      selectedNeighbors.add(edge.source);
      selectedNeighbors.add(edge.target);
    });
    const fitNodes = [...drawn.values()].filter(
        (node) => !topologySelectedNodeId || selectedNeighbors.has(node.id),
      ),
      visualBounds = fitNodes.map(topologyNodeVisualBounds),
      visibleEdges = topologySelectedNodeId ? incident : [],
      routeDrawn = topologySelectedNodeId
        ? new Map([...drawn].filter(([id]) => selectedNeighbors.has(id)))
        : drawn,
      straightEdges = topologyStraightEdges(visibleEdges, routeDrawn);
    byId("topology").dataset.routeNodeHits = "0";
    byId("topology").dataset.routeStrategy = "straight-focus";
    byId("topology").dataset.routeCollisionChecked = "true";
    const routeBounds = [...straightEdges.values()].map(
        (segment) => segment.bounds,
      ),
      fitBounds = [...visualBounds, ...routeBounds];
    topologyFitBounds = fitBounds.length
      ? {
          minX: Math.min(...fitBounds.map((bounds) => bounds.minX)),
          minY: Math.min(...fitBounds.map((bounds) => bounds.minY)),
          maxX: Math.max(...fitBounds.map((bounds) => bounds.maxX)),
          maxY: Math.max(...fitBounds.map((bounds) => bounds.maxY)),
        }
      : null;
    const viewBoxX = Math.min(0, (topologyFitBounds?.minX || 0) - 24),
      viewBoxY = Math.min(0, (topologyFitBounds?.minY || 0) - 24),
      viewBoxMaxX = Math.max(
        contentW,
        (topologyFitBounds?.maxX || contentW) + 24,
      ),
      viewBoxMaxY = Math.max(
        contentH,
        (topologyFitBounds?.maxY || contentH) + 24,
      ),
      viewBoxW = viewBoxMaxX - viewBoxX,
      viewBoxH = viewBoxMaxY - viewBoxY;
    if (!topologySelectedNodeId) {
      contentW = Math.max(contentW, viewBoxW);
      contentH = Math.max(contentH, viewBoxH);
    }
    byId("topology").style.minWidth = contentW + 36 + "px";
    byId("topology").style.minHeight = contentH + 36 + "px";
    const edgeSVG = visibleEdges
      .map((edge) => {
        const A = drawn.get(edge.source),
          B = drawn.get(edge.target),
          segment = straightEdges.get(edge.id);
        if (!A || !B || !segment) return "";
        const selectedRelationship = edge.id === topologySelectedEdgeId,
          searchDim =
            topologySearch &&
            (!topologySearchMatchesGraph(A) || !topologySearchMatchesGraph(B)),
          showDirection = edge.directed !== false;
        return `<line class="link selectable ${esc(edge.kind)} hot${selectedRelationship ? " selected" : ""}${searchDim ? " dimmed" : ""}" data-edge-id="${esc(edge.id)}" role="button" tabindex="0" aria-pressed="${selectedRelationship}" aria-label="Select ${esc(edge.relation || edge.kind)} ${edge.directed === false ? "undirected" : "directed"} relationship" marker-end="${showDirection ? "url(#topology-arrow)" : ""}" x1="${segment.x1}" y1="${segment.y1}" x2="${segment.x2}" y2="${segment.y2}"><title>${esc(`${edge.kind} · ${edge.provider} · ${edge.relation || ""} · ${edge.source} → ${edge.target}`)}</title></line>`;
      })
      .join("");
    const nodeSVG = [...drawn.values()]
      .map((node) => {
        const selectedNode = node.id === topologySelectedNodeId,
          related =
            selectedNeighbors.has(node.id) ||
            (selectedEdge &&
              (selectedEdge.source === node.id ||
                selectedEdge.target === node.id)),
          nodeIncident = edges.filter(
            (edge) => edge.source === node.id || edge.target === node.id,
          ),
          health = topologyNodeHealth(node),
          statusColor =
            health === "warning"
              ? "var(--amber)"
              : health === "unknown"
                ? "var(--muted)"
                : health === "stopped"
                  ? "var(--red)"
                  : "var(--green)",
          group = topologyReplicaGroup(
            node,
            arr(topologyGraph?.edges),
            nodeById,
          ),
          metric = arr(node.metrics).find(
            (item) => item.key === topologyMetricKey,
          ),
          metricText = topologyMetricKey
            ? metric
              ? `${metric.label}: ${formatTopologyMetric(metric)}`
              : "metric unavailable"
            : "",
          metaText =
            group ||
            `${node.scope?.namespace || "cluster"} · ${node.scope?.provider || "-"}`,
          dimmed =
            (topologySelectedNodeId && !related) ||
            (topologySearch && !topologySearchMatchesGraph(node)),
          visualKind = topologyVisualKind(topologyKind(node)),
          cx = node.x + node.w / 2,
          cy = node.y + 30;
        return `<g class="node kind-${visualKind}${selectedNode ? " selected" : related ? " related" : ""}${dimmed ? " dimmed" : ""}" data-node-id="${esc(node.id)}" data-topology-depth="${node.rank}" data-topology-isolate="${Boolean(node.isolate)}" data-detail="${esc(topologyNodeDetail(node, nodeIncident))}" role="button" tabindex="0" aria-pressed="${selectedNode}" aria-label="Select ${esc(node.kind)} ${esc(node.name)}; ${esc(metaText)}${metricText ? `; ${esc(metricText)}` : ""}"><title>${esc(`${node.kind} · ${node.identity || node.id} · depth ${node.rank} · ${metaText}${metricText ? ` · ${metricText}` : ""}`)}</title>${topologyNodeShape(node)}<svg class="node-symbol" x="${cx - 11.5}" y="${cy - 11.5}" width="23" height="23"><use href="#${topologyIconID(topologyKind(node))}"/></svg><foreignObject x="${node.x + 6}" y="${node.y + 61}" width="108" height="31"><div xmlns="http://www.w3.org/1999/xhtml" class="node-copy label" title="${esc(node.name)}">${esc(node.name)}</div><div xmlns="http://www.w3.org/1999/xhtml" class="node-copy meta" title="${esc(metaText)}">${esc(metaText)}</div></foreignObject>${metricText ? `<foreignObject x="${cx + 24}" y="${node.y + 7}" width="64" height="18"><div xmlns="http://www.w3.org/1999/xhtml" class="node-metric-badge" title="${esc(metricText)}">${esc(metricText)}</div></foreignObject>` : ""}<circle class="node-status" cx="${cx + 23}" cy="${cy - 21}" r="5" fill="${statusColor}"/></g>`;
      })
      .join("");
    const activeView = currentTopologyView(),
      viewportTransform = `translate(${activeView.x} ${activeView.y}) scale(${activeView.scale})`;
    byId("topology").innerHTML =
      `<svg id="topologySvg" viewBox="${viewBoxX} ${viewBoxY} ${viewBoxW} ${viewBoxH}"><defs><marker id="topology-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="context-stroke"/></marker></defs><g id="topologyViewport" transform="${viewportTransform}">${edgeSVG}${nodeSVG}</g></svg><div class="legend"><span>Wheel to zoom</span><span>Drag to pan</span><span>${nodes.length} objects · ${edges.length} evidence-backed relationships</span></div>`;
    installTopologyHoverInteractions(edges);
    recordTopologyRender(renderStarted, nodes.length, edges.length, "graph");
  }
  function applyTopologyTransform() {
    const viewport = byId("topologyViewport"),
      view = currentTopologyView();
    if (viewport)
      viewport.setAttribute(
        "transform",
        `translate(${view.x} ${view.y}) scale(${view.scale})`,
      );
  }
  byId("topology-search")?.addEventListener("input", (event) => {
    topologySearch = event.target.value.trim();
    syncTopologyURL();
    renderTopology();
  });
  byId("topology-search").value = topologySearch;
  byId("topologyGroup").checked = topologyGroupReplicas;
  byId("topologyHealth").value = topologyHealthFilter;
  byId("topologySystem").value = topologySystemFilter;
  byId("topologyState").value = topologyStateFilter;
  byId("topologyModeOptions").addEventListener("click", (event) => {
    const button = event.target.closest("[data-topology-mode]");
    if (!button) return;
    topologyMode = button.dataset.topologyMode;
    syncTopologyURL();
    renderTopology();
  });
  byId("topologyHealth").addEventListener("change", (event) => {
    topologyHealthFilter = event.target.value;
    syncTopologyURL();
    renderTopology();
  });
  byId("topologySystem").addEventListener("change", (event) => {
    topologySystemFilter = event.target.value;
    syncTopologyURL();
    renderTopology();
  });
  byId("topologyState").addEventListener("change", (event) => {
    topologyStateFilter = event.target.value;
    syncTopologyURL();
    renderTopology();
  });
  byId("topologyGroup").addEventListener("change", (event) => {
    topologyGroupReplicas = event.target.checked;
    syncTopologyURL();
    renderTopology();
  });

  document.addEventListener("change", (e) => {
    if (e.target.id !== "topologyMetric") return;
    topologyMetricKey = e.target.value;
    syncTopologyURL();
    renderTopology();
  });
  byId("topology").addEventListener(
    "wheel",
    (e) => {
      const svg = byId("topologySvg");
      const viewport = byId("topologyViewport"),
        view = currentTopologyView();
      if (!svg || !viewport) return;
      e.preventDefault();
      const rect = svg.getBoundingClientRect();
      const px = e.clientX - rect.left,
        py = e.clientY - rect.top;
      const nextScale = Math.max(
        0.35,
        Math.min(4, view.scale * (e.deltaY < 0 ? 1.12 : 0.88)),
      );
      view.x = px - (px - view.x) * (nextScale / view.scale);
      view.y = py - (py - view.y) * (nextScale / view.scale);
      view.scale = nextScale;
      applyTopologyTransform();
      syncTopologyURL();
    },
    { passive: false },
  );
  byId("topology").addEventListener("pointerdown", (e) => {
    if (e.button !== 0 || !byId("topologyViewport")) return;
    const node = e.target.closest("[data-node-id]"),
      edge = e.target.closest(".link[data-edge-id]"),
      view = currentTopologyView();
    topologyDrag = {
      active: true,
      moved: false,
      suppressClick: false,
      startX: e.clientX,
      startY: e.clientY,
      viewX: view.x,
      viewY: view.y,
      nodeId: node?.dataset.nodeId || "",
      edgeId: edge?.dataset.edgeId || "",
    };
    byId("topology").classList.add("panning");
    if (!node && !edge) byId("topology").setPointerCapture(e.pointerId);
  });
  byId("topology").addEventListener("pointermove", (e) => {
    if (!topologyDrag.active) return;
    const dx = e.clientX - topologyDrag.startX,
      dy = e.clientY - topologyDrag.startY,
      view = currentTopologyView();
    topologyDrag.moved = topologyDrag.moved || Math.abs(dx) + Math.abs(dy) > 3;
    view.x = topologyDrag.viewX + dx;
    view.y = topologyDrag.viewY + dy;
    applyTopologyTransform();
  });
  byId("topology").addEventListener("pointerup", (e) => {
    if (!topologyDrag.active) return;
    topologyDrag.suppressClick = true;
    topologyDrag.active = false;
    byId("topology").classList.remove("panning");
    if (byId("topology").hasPointerCapture(e.pointerId))
      byId("topology").releasePointerCapture(e.pointerId);
    if (topologyDrag.moved) syncTopologyURL();
    else if (topologyDrag.nodeId) selectTopologyNode(topologyDrag.nodeId);
    else if (topologyDrag.edgeId) selectTopologyEdge(topologyDrag.edgeId);
    else clearTopologySelection();
  });
  byId("topology").addEventListener("pointercancel", (e) => {
    topologyDrag.active = false;
    byId("topology").classList.remove("panning");
    if (byId("topology").hasPointerCapture(e.pointerId))
      byId("topology").releasePointerCapture(e.pointerId);
  });
  byId("topology").addEventListener("click", (e) => {
    if (topologyDrag.suppressClick) {
      topologyDrag.suppressClick = false;
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    const nodeId =
        e.target.closest("[data-node-id]")?.dataset.nodeId ||
        topologyDrag.nodeId,
      edgeId =
        e.target.closest(".link[data-edge-id]")?.dataset.edgeId ||
        topologyDrag.edgeId;
    if (nodeId) selectTopologyNode(nodeId);
    else if (edgeId) selectTopologyEdge(edgeId);
    else clearTopologySelection();
    e.stopPropagation();
  });
  byId("topology").addEventListener("keydown", (e) => {
    if (e.key !== "Enter" && e.key !== " ") return;
    const node = e.target.closest(".node[data-node-id]"),
      edge = e.target.closest(".link[data-edge-id]");
    if (!node && !edge) return;
    e.preventDefault();
    if (node) selectTopologyNode(node.dataset.nodeId);
    else selectTopologyEdge(edge.dataset.edgeId);
  });

  byId("topologyFit").addEventListener("click", fitTopologyToContent);
  for (const [id, factor] of [
    ["topologyZoomOut", 0.8],
    ["topologyZoomIn", 1.25],
  ])
    byId(id).addEventListener("click", () => {
      const view = currentTopologyView();
      view.scale = Math.max(0.35, Math.min(4, view.scale * factor));
      syncTopologyURL();
      applyTopologyTransform();
    });
  document.addEventListener("click", (e) => {
    const target = e.target;
    const raw = target.closest("[data-topology-raw] summary");
    if (raw) {
      topologyRawExpanded = !raw.parentElement.open;
      return;
    }
    const sort = target.closest("[data-relative-sort]");
    if (sort) {
      const key = sort.dataset.relativeSort;
      topologyRelativeDirection =
        topologyRelativeSort === key && topologyRelativeDirection === "asc"
          ? "desc"
          : "asc";
      topologyRelativeSort = key;
      renderTopology();
      return;
    }
    const edge = target.closest("[data-topology-edge]");
    if (edge) {
      selectTopologyEdge(edge.dataset.topologyEdge);
      return;
    }
    const select = target.closest("[data-topology-select]");
    if (select) {
      selectTopologyNode(select.dataset.topologySelect);
      return;
    }
  });
  let pending = null;
  function update(next) {
    if (disposed) return;
    if (topologyDrag.active) {
      pending = next;
      return;
    }
    pending = null;
    topologyGraph = next.graph || {};
    advisorReport = next.advisor || {};
    snapshot = next.snapshot || {};
    k = snapshot.kubernetes || {};
    nsFilter = next.namespace || "all";
    resourceFilter = next.resource || "all";
    const counts = [
      ["Nodes", arr(k.nodes).length],
      ["Workloads", arr(k.workloads).length],
      ["Pods", arr(k.pods).length],
      ["Images", arr(k.runningImages).length],
      ["CRDs", arr(k.customResourceDefinitions).length],
      ["Routes", arr(k.ingresses).length + arr(k.gatewayRoutes).length],
    ];
    byId("cards").innerHTML = counts
      .map(
        ([label, count]) =>
          `<div class="card"><div class="label">${label}</div><div class="value">${count}</div></div>`,
      )
      .join("");
    byId("sourceFreshness").innerHTML = Object.entries(next.sources || {})
      .map(
        ([name, status]) =>
          `<span class="chip"><strong>${esc(name)}: ${esc(sourceLabel(status))}</strong><small>${esc(sourceDetail(status))}</small></span>`,
      )
      .join("");
    renderTopology();
  }
  root.addEventListener(
    "pointerup",
    () => {
      if (pending) queueMicrotask(() => update(pending));
    },
    { signal: abort.signal },
  );
  root.addEventListener(
    "pointercancel",
    () => {
      if (pending) queueMicrotask(() => update(pending));
    },
    { signal: abort.signal },
  );
  update(input);
  return {
    update,
    dispose() {
      disposed = true;
      pending = null;
      abort.abort();
      topologyRenderSample++;
    },
  };
}
