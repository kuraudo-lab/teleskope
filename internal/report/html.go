package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	_ "embed"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

//go:embed report.html
var reportHTMLTemplate string

//go:embed live.js
var liveJS string

// HTML renders a self-contained interactive static cluster report.
func HTML(snapshot *inventory.Snapshot) (string, error) {
	return html(snapshot, targetName(snapshot))
}

func html(snapshot *inventory.Snapshot, target string) (string, error) {
	if snapshot == nil {
		return "", fmt.Errorf("snapshot is nil")
	}
	return RenderUI(UIRenderOptions{Mode: UIModeOffline, Snapshot: snapshot, Target: target})
}

// LiveHTML renders the same embedded UI with a live data source and no credentials.
func LiveHTML() string {
	return LiveHTMLWithOptions(LiveHTMLOptions{})
}

// LiveHTMLOptions overrides API paths for live-style pages backed by another server.
type LiveHTMLOptions struct {
	SnapshotPath       string
	AnalyzePath        string
	ExportSnapshotPath string
	ExportSummaryPath  string
}

// LiveHTMLWithOptions renders the embedded UI with configurable same-origin APIs.
func LiveHTMLWithOptions(opts LiveHTMLOptions) string {
	page, err := RenderUI(UIRenderOptions{
		Mode: UIModeLive,
		Endpoints: UIEndpoints{
			Snapshot:       firstNonEmpty(opts.SnapshotPath, "/api/snapshot"),
			Analyze:        firstNonEmpty(opts.AnalyzePath, "/api/analyze"),
			ExportSnapshot: firstNonEmpty(opts.ExportSnapshotPath, "/api/export/snapshot.json"),
			ExportSummary:  firstNonEmpty(opts.ExportSummaryPath, "/api/export/summary.md"),
		},
		Styles:  []string{liveStyles},
		Scripts: []string{liveJS},
	})
	if err != nil {
		return ""
	}
	return page
}

// SnapshotJSON renders the complete snapshot JSON used by scan report artifacts.
func SnapshotJSON(snapshot *inventory.Snapshot) (string, error) {
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		return "", err
	}
	return data.String(), nil
}

func escapeScriptText(value string) string {
	return strings.ReplaceAll(value, "</", "<\\/")
}

const liveStyles = `.live-event-list { display:grid; gap:6px; max-height:calc(100vh - 190px); overflow:auto; font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace; font-size:11px; }
.live-event { display:grid; grid-template-columns:78px 78px 86px 50px minmax(0,1fr); gap:8px; align-items:start; padding:7px 8px; border:1px solid var(--table-line); border-radius:7px; background:var(--live-event-bg); }
.live-event-time, .live-event-kind, .live-event-source { color:var(--muted); }
.live-event-kind { text-transform:uppercase; font-size:10px; font-weight:800; letter-spacing:.04em; }
.live-event-level { text-transform:uppercase; font-weight:800; }
.live-event-info .live-event-level, .live-event-debug .live-event-level { color:var(--cyan); }
.live-event-warn .live-event-level { color:var(--amber); }
.live-event-error .live-event-level { color:var(--red); }
.live-event-message { min-width:0; overflow-wrap:anywhere; line-height:1.45; }
@media (max-width:700px) { .live-event { grid-template-columns:74px 1fr 52px; } .live-event-source, .live-event-message { grid-column:1 / -1; } }`
