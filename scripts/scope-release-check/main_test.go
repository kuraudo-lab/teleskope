package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseContract(t *testing.T) {
	c := loadTestContract(t)
	if err := validateContract(c); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := printContract(&output, c, ""); err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{
		"Scope reborn release contract: VALID",
		"Popeye replacement gate #40",
		"API graph baseline [required]",
		"Optional external evidence [optional]",
		"node probe=false",
	} {
		if !strings.Contains(output.String(), wanted) {
			t.Fatalf("output does not contain %q:\n%s", wanted, output.String())
		}
	}
}

func TestContractRejectsSecurityRegression(t *testing.T) {
	c := loadTestContract(t)
	c.SecurityBoundary.NodeProbe = true
	if err := validateContract(c); err == nil || !strings.Contains(err.Error(), "security capabilities") {
		t.Fatalf("validateContract() error = %v, want rejected security capabilities", err)
	}
}

func TestContractRejectsGraduationRegressions(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*contract)
		message string
	}{
		{
			name: "incomplete Popeye foundation",
			mutate: func(c *contract) {
				c.Product.PopeyeFoundationIssues = c.Product.PopeyeFoundationIssues[1:]
			},
			message: "foundation issues #33-#40",
		},
		{
			name: "wrong blocking issue",
			mutate: func(c *contract) {
				c.Product.BlockingGraduationIssue = 39
			},
			message: "graduation issue #40",
		},
		{
			name: "missing release contract gate",
			mutate: func(c *contract) {
				c.CrossCutting.Issues = []int{46}
			},
			message: "issues #46 and #47",
		},
		{
			name: "missing slice issue mapping",
			mutate: func(c *contract) {
				c.Slices[0].Issues = nil
			},
			message: "must map to issues [41 42]",
		},
		{
			name: "missing slice dependency",
			mutate: func(c *contract) {
				c.Slices[2].DependsOn = []string{"live-continuity"}
			},
			message: "must depend on [graph-baseline live-continuity]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := loadTestContract(t)
			test.mutate(&c)
			if err := validateContract(c); err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("validateContract() error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestLoadContractRejectsTrailingJSON(t *testing.T) {
	data, err := os.ReadFile(contractTestPath())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, append(data, []byte("\n{}\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadContract(path); err == nil || !strings.Contains(err.Error(), "exactly one JSON document") {
		t.Fatalf("loadContract() error = %v, want trailing document rejection", err)
	}
}

func TestPrintSingleSlice(t *testing.T) {
	c := loadTestContract(t)

	var output bytes.Buffer
	if err := printContract(&output, c, "external-evidence"); err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{"Optional: true", "Offline:", "Live:", "Hub:", "Acceptance:"} {
		if !strings.Contains(output.String(), wanted) {
			t.Fatalf("output does not contain %q:\n%s", wanted, output.String())
		}
	}
}

func loadTestContract(t *testing.T) contract {
	t.Helper()
	c, err := loadContract(contractTestPath())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func contractTestPath() string {
	return filepath.Join("..", "..", "docs", "scope-reborn-release-gates.json")
}
