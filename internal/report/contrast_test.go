package report

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestTopologyAccessibilityColorTokensMeetContrastGates(t *testing.T) {
	tests := []struct {
		name, foreground, background string
		minimum                      float64
	}{
		{name: "light text", foreground: "#0f172a", background: "#ffffff", minimum: 4.5},
		{name: "light muted text", foreground: "#64748b", background: "#ffffff", minimum: 4.5},
		{name: "light warning text", foreground: "#92400e", background: "#ffffff", minimum: 4.5},
		{name: "light disabled text", foreground: "#475569", background: "#ffffff", minimum: 4.5},
		{name: "dark text", foreground: "#e5eefb", background: "#101b2b", minimum: 4.5},
		{name: "dark muted text", foreground: "#95a7bd", background: "#101b2b", minimum: 4.5},
		{name: "dark warning text", foreground: "#fbbf24", background: "#101b2b", minimum: 4.5},
		{name: "dark disabled text", foreground: "#cbd5e1", background: "#101b2b", minimum: 4.5},
		{name: "light focus indicator", foreground: "#0891b2", background: "#ffffff", minimum: 3},
		{name: "light selected boundary", foreground: "#2563eb", background: "#ffffff", minimum: 3},
		{name: "dark focus indicator", foreground: "#22d3ee", background: "#101b2b", minimum: 3},
		{name: "dark selected boundary", foreground: "#60a5fa", background: "#101b2b", minimum: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ratio := contrastRatio(test.foreground, test.background)
			if ratio < test.minimum {
				t.Fatalf("contrast %s on %s = %.2f, want >= %.1f", test.foreground, test.background, ratio, test.minimum)
			}
		})
	}
	for _, token := range []string{"--warning-text:#92400e", "--warning-text:#fbbf24", "--disabled-text:#475569", "--disabled-text:#cbd5e1", "color:var(--warning-text)", "color:var(--disabled-text)"} {
		if !strings.Contains(uiCSS, token) {
			t.Errorf("report template missing tested accessibility token %q", token)
		}
	}
}

func contrastRatio(foreground, background string) float64 {
	light, dark := relativeLuminance(foreground), relativeLuminance(background)
	if dark > light {
		light, dark = dark, light
	}
	return (light + .05) / (dark + .05)
}

func relativeLuminance(value string) float64 {
	if len(value) != 7 || value[0] != '#' {
		panic(fmt.Sprintf("invalid color %q", value))
	}
	channels := make([]float64, 3)
	for i := range channels {
		encoded, err := strconv.ParseUint(value[1+i*2:3+i*2], 16, 8)
		if err != nil {
			panic(err)
		}
		channel := float64(encoded) / 255
		if channel <= .04045 {
			channels[i] = channel / 12.92
		} else {
			channels[i] = math.Pow((channel+.055)/1.055, 2.4)
		}
	}
	return .2126*channels[0] + .7152*channels[1] + .0722*channels[2]
}
