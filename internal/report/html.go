package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	_ "embed"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
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
	jsonData, err := SnapshotJSON(snapshot)
	if err != nil {
		return "", fmt.Errorf("encode snapshot for html: %w", err)
	}
	analysis, err := json.Marshal(advisor.Analyze(snapshot))
	if err != nil {
		return "", err
	}
	page := strings.Replace(reportHTMLTemplate, "__TELESKOPE_SNAPSHOT_JSON__", escapeScriptText(jsonData), 1)
	page = strings.Replace(page, "__TELESKOPE_ADVISOR_JSON__", escapeScriptText(string(analysis)), 1)
	return strings.Replace(page, "__TELESKOPE_MARKDOWN__", escapeScriptText(markdown(snapshot, target)), 1), nil
}

// LiveHTML renders the same embedded UI with a live data source and no credentials.
func LiveHTML() string {
	page := strings.Replace(reportHTMLTemplate, "__TELESKOPE_SNAPSHOT_JSON__", "{}", 1)
	page = strings.Replace(page, "__TELESKOPE_ADVISOR_JSON__", "{}", 1)
	page = strings.Replace(page, "__TELESKOPE_MARKDOWN__", "", 1)
	page = strings.Replace(page, "<body>", "<body data-live=\"true\" class=\"live-loading\">", 1)
	page = strings.Replace(page, "</style>", ".live-progress { display:flex; align-items:flex-start; gap:10px; margin-bottom:16px; padding:12px 14px; border:1px solid var(--line); border-radius:12px; background:var(--panel-2); color:var(--muted); overflow-wrap:anywhere; } .live-progress-icon { color:var(--cyan); font-weight:800; flex:none; display:inline-grid; place-items:center; width:16px; height:16px; line-height:16px; } .live-progress-icon.spinning { border:2px solid var(--line); border-top-color:var(--cyan); border-radius:999px; color:transparent; animation:teleskope-spin .8s linear infinite; } @keyframes teleskope-spin { to { transform:rotate(360deg); } } .live-event-list { display:grid; gap:4px; max-height:calc(100vh - 190px); overflow:auto; font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace; font-size:12px; } .live-event { display:grid; grid-template-columns:86px 92px 56px minmax(0,1fr); gap:8px; align-items:start; padding:5px 7px; border:1px solid var(--table-line); border-radius:8px; background:var(--live-event-bg); } .live-event-time, .live-event-source { color:var(--muted); } .live-event-level { text-transform:uppercase; font-weight:700; } .live-event-info .live-event-level, .live-event-debug .live-event-level { color:var(--cyan); } .live-event-warn .live-event-level { color:var(--amber); } .live-event-error .live-event-level { color:var(--red); } .live-event-message { min-width:0; overflow-wrap:anywhere; } @media (max-width:700px) { .live-event { grid-template-columns:74px 1fr 52px; } .live-event-message { grid-column:1 / -1; } } </style>", 1)
	return strings.Replace(page, "</body>", "<script>"+liveJS+"</script></body>", 1)
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
