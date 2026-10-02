import type { Resource } from "./types";
export function sourceLabel(status: Resource) {
  if (status.mode !== "watch") return status.state || "unknown";
  return `watch ${status.state === "ready" ? "connected" : status.state === "stale" ? "reconnecting" : status.state || "loading"}`;
}
export function sourceDetail(status: Resource) {
  const time = (v: unknown) =>
    v ? new Date(String(v)).toLocaleString() : "none";
  if (status.mode !== "watch") return `published ${time(status.lastSuccess)}`;
  return `event ${time(status.lastEventAt)} · full resync ${time(status.lastFullSyncAt)} · reconnects ${status.reconnects || 0}`;
}
export function publicationStatus(
  sources: Record<string, Resource>,
  loading: boolean,
  paused: boolean,
  error: string,
) {
  const statuses = Object.values(sources);
  if (error || statuses.some((s) => s.state === "error"))
    return { label: "Error", tone: "bad" };
  if (loading || statuses.some((s) => s.refreshing || s.state === "loading"))
    return { label: "Updating", tone: "loading" };
  if (paused) return { label: "Paused", tone: "warn" };
  if (statuses.some((s) => s.state === "partial" || s.state === "stale"))
    return { label: "Partial", tone: "bad" };
  return { label: "Ready", tone: "ready" };
}
export function eventKind(event: Resource) {
  const message = event.message || "";
  if (event.source === "analysis" || message.includes("analysis"))
    return "analysis";
  if (message.includes("watch ")) return "watch";
  if (
    message.includes("published") ||
    message.includes("retained previous data")
  )
    return "publication";
  if (message.includes("refresh starting") || message.includes("collect"))
    return "collection";
  return "event";
}
