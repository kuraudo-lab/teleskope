package hub

import "github.com/kuraudo-lab/teleskope/internal/report"

// HubHTML shares the same embedded component bundle as cluster reports.
func HubHTML() string { return report.HubHTML() }
