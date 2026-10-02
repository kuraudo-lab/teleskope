# Frontend runtime and replay provenance

F03/F03a: build a typed browser state and data adapter, preserve same-origin endpoint overrides and offline boot payloads, control lifecycle with abort/dispose, reject inconsistent topology revisions, preserve namespace and selected identity. Bind AI responses to the original request revision/scope. Replay decoding must reject unsupported schemas and trailing documents while preserving schema-less legacy snapshots.

Validate pure protocol tests (delta replay/gaps/dangling edges, pause/disposal and stale analysis), current snapshot fixtures, malformed snapshots, full Go checks and embedded browser behavior. Snapshot remains original evidence; Advisor and topology are recomputed by the installed version. Previously generated AI artifacts are separate, not part of snapshot.json; no new history/archive format in this migration.
