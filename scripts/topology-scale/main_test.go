package main

import (
	"strings"
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/topology"
)

func TestRenderScalePageUsesSharedUIAndHarness(t *testing.T) {
	for _, profile := range topology.ScaleProfiles() {
		page, err := renderScalePage(profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{
			`id="topology"`,
			`window.__teleskopeScaleEvidence`,
			`topologySemanticView = 'runtime'`,
			`"clusterId":"scale-` + profile.Name + `"`,
			`nodes:` + itoa(profile.Nodes),
			`edges:` + itoa(profile.Edges),
		} {
			if !strings.Contains(page, want) {
				t.Errorf("%s page missing %q", profile.Name, want)
			}
		}
	}
}

func TestProfileNames(t *testing.T) {
	if got := profileNames(); got != "large,medium,small" {
		t.Fatalf("profileNames() = %q", got)
	}
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var result string
	for value > 0 {
		result = string(digits[value%10]) + result
		value /= 10
	}
	return result
}
