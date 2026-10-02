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

//go:embed assets/ui.js
var uiJS string

//go:embed assets/ui.css
var uiCSS string

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
	DisableAnalysis    bool
	SnapshotPath       string
	TopologyPath       string
	AnalyzePath        string
	ExportSnapshotPath string
	ExportSummaryPath  string
}

// LiveHTMLWithOptions renders the embedded UI with configurable same-origin APIs.
func LiveHTMLWithOptions(opts LiveHTMLOptions) string {
	page, err := RenderUI(UIRenderOptions{
		Mode:            UIModeLive,
		DisableAnalysis: opts.DisableAnalysis,
		Endpoints: UIEndpoints{
			Snapshot:       firstNonEmpty(opts.SnapshotPath, "/api/snapshot"),
			Topology:       firstNonEmpty(opts.TopologyPath, "/api/topology"),
			Analyze:        firstNonEmpty(opts.AnalyzePath, "/api/analyze"),
			ExportSnapshot: firstNonEmpty(opts.ExportSnapshotPath, "/api/export/snapshot.json"),
			ExportSummary:  firstNonEmpty(opts.ExportSummaryPath, "/api/export/summary.md"),
		},
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

// HubHTML renders the fleet entry using the shared frontend bundle.
func HubHTML() string {
	page, err := RenderUI(UIRenderOptions{Mode: UIModeOffline, Snapshot: &inventory.Snapshot{}})
	if err != nil {
		return ""
	}
	return strings.Replace(page, `<div id="ui-root">`, `<div id="ui-root" data-view="hub">`, 1)
}
