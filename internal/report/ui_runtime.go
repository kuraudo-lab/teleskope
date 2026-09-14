package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

type UIMode string

const (
	UIModeOffline UIMode = "offline"
	UIModeLive    UIMode = "live"
)

type UIEndpoints struct {
	Snapshot       string `json:"snapshot,omitempty"`
	Analyze        string `json:"analyze,omitempty"`
	ExportSnapshot string `json:"exportSnapshot,omitempty"`
	ExportSummary  string `json:"exportSummary,omitempty"`
}

type UIRenderOptions struct {
	Mode      UIMode
	Snapshot  *inventory.Snapshot
	Target    string
	Endpoints UIEndpoints
	Styles    []string
	Scripts   []string
}

type uiBootConfig struct {
	Mode      UIMode      `json:"mode"`
	Endpoints UIEndpoints `json:"endpoints,omitempty"`
}

// RenderUI renders the embedded report shell for offline, live, and hub drilldown pages.
func RenderUI(opts UIRenderOptions) (string, error) {
	mode := opts.Mode
	if mode == "" {
		mode = UIModeOffline
	}
	payload, err := renderPayload(opts)
	if err != nil {
		return "", err
	}
	boot, err := json.Marshal(uiBootConfig{Mode: mode, Endpoints: opts.Endpoints})
	if err != nil {
		return "", fmt.Errorf("encode UI boot config: %w", err)
	}

	page := reportHTMLTemplate
	page = strings.Replace(page, "__TELESKOPE_BOOT_CONFIG__", escapeScriptText(string(boot)), 1)
	page = strings.Replace(page, "__TELESKOPE_ADVISOR_JSON__", escapeScriptText(payload.advisor), 1)
	page = strings.Replace(page, "__TELESKOPE_SNAPSHOT_JSON__", escapeScriptText(payload.snapshot), 1)
	page = strings.Replace(page, "__TELESKOPE_MARKDOWN__", escapeScriptText(payload.markdown), 1)
	page = strings.Replace(page, "<body>", bodyOpen(mode), 1)
	if len(opts.Styles) > 0 {
		page = strings.Replace(page, "</style>", strings.Join(opts.Styles, "\n")+"\n</style>", 1)
	}
	if len(opts.Scripts) > 0 {
		page = strings.Replace(page, "</body>", scriptTags(opts.Scripts)+"</body>", 1)
	}
	return page, nil
}

type uiPayload struct {
	advisor  string
	snapshot string
	markdown string
}

func renderPayload(opts UIRenderOptions) (uiPayload, error) {
	if opts.Mode == UIModeLive {
		return uiPayload{advisor: "{}", snapshot: "{}", markdown: ""}, nil
	}
	if opts.Snapshot == nil {
		return uiPayload{}, fmt.Errorf("snapshot is nil")
	}
	snapshot, err := SnapshotJSON(opts.Snapshot)
	if err != nil {
		return uiPayload{}, fmt.Errorf("encode snapshot for html: %w", err)
	}
	analysis, err := json.Marshal(advisor.Analyze(opts.Snapshot))
	if err != nil {
		return uiPayload{}, err
	}
	target := opts.Target
	if target == "" {
		target = targetName(opts.Snapshot)
	}
	return uiPayload{advisor: string(analysis), snapshot: snapshot, markdown: markdown(opts.Snapshot, target)}, nil
}

func bodyOpen(mode UIMode) string {
	if mode != UIModeLive {
		return "<body>"
	}
	return `<body data-live="true" class="live-loading">`
}

func scriptTags(scripts []string) string {
	var b strings.Builder
	for _, script := range scripts {
		b.WriteString("<script>")
		b.WriteString(escapeScriptText(script))
		b.WriteString("</script>")
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
