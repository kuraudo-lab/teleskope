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
