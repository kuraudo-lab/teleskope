package report

import (
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
	if snapshot == nil {
		return "", fmt.Errorf("snapshot is nil")
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode snapshot for html: %w", err)
	}
	jsonData := strings.ReplaceAll(string(data), "</", "<\\/")
	return strings.Replace(reportHTMLTemplate, "__TELESKOPE_SNAPSHOT_JSON__", jsonData, 1), nil
}

// LiveHTML renders the same embedded UI with a live data source and no credentials.
func LiveHTML() string {
	page := strings.Replace(reportHTMLTemplate, "__TELESKOPE_SNAPSHOT_JSON__", "{}", 1)
	page = strings.Replace(page, "<body>", "<body data-live=\"true\" class=\"live-loading\">", 1)
	page = strings.Replace(page, "</style>", ".live-loading .section { display:none !important; } .live-status { margin-bottom:18px; padding:14px 18px; border:1px solid var(--line); border-radius:12px; overflow-wrap:anywhere; } .live-status p { margin:6px 0; } .live-status summary { cursor:pointer; margin-top:8px; } .live-status button { margin-top:10px; background:var(--chip); color:var(--text); border:1px solid var(--line); border-radius:8px; padding:8px 12px; cursor:pointer; } .live-status button:focus-visible { outline:2px solid var(--cyan); outline-offset:2px; } </style>", 1)
	page = strings.Replace(page, "<section class=\"section active\"", "<div id=\"live-status\" class=\"live-status\"><p role=\"status\">Waiting for first snapshot…</p></div><section class=\"section active\"", 1)
	return strings.Replace(page, "</body>", "<script>"+liveJS+"</script></body>", 1)
}
