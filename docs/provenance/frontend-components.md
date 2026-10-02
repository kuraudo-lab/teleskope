# Component migration provenance

Implement F04–F09 as connected component boundaries: App shell/navigation/filter/export/drawer, resource pages with shared table and preserved cell definitions, workload template relationship model, EKS/Advisor/scoped analysis, topology Vue lifecycle adapter retaining current layout algorithm, and Hub Vue view. Original source is removed as components take ownership; retained string formatting is escaped domain presentation, not a parallel application.

Large immutable snapshot payloads use shallow reactive publication replacement, not deep proxying. Component-local state owns filters and selection. Topology imperative SVG renderer is scoped to its component root, receives explicit input/update/dispose, and may not mutate other pages. Large tables render only the active page. Shell CSS and branding are preserved without runtime DOM relocation.

Validation: zero-replica relationship and escaping unit tests; all navigation/resource pages; offline same-bundle HTML; recorded and live revision behavior; Hub routes; graph/table selection/zoom/filter/URL; full Go checks/build; production browser errors; generated artifacts synced. Source-fragment Go assertions are replaced by behavior/protocol tests, keeping Go payload/escaping contracts.
