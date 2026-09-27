package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultContractPath = "docs/scope-reborn-release-gates.json"

type contract struct {
	SchemaVersion int `json:"schemaVersion"`
	Product       struct {
		Lineage                 string `json:"lineage"`
		PreviewLabel            string `json:"previewLabel"`
		GraduatedLabel          string `json:"graduatedLabel"`
		BlockingGraduationIssue int    `json:"blockingGraduationIssue"`
		PopeyeFoundationIssues  []int  `json:"popeyeFoundationIssues"`
	} `json:"product"`
	ScopeConcepts map[string][]string `json:"scopeConcepts"`
	Slices        []releaseSlice      `json:"slices"`
	CrossCutting  struct {
		Issues           []int    `json:"issues"`
		TestCommands     []string `json:"testCommands"`
		BrowserScenarios []string `json:"browserScenarios"`
		Security         []string `json:"security"`
		Compatibility    []string `json:"compatibility"`
	} `json:"crossCuttingGates"`
	ExtensionPolicy struct {
		GeneralPluginSystem           bool   `json:"generalPluginSystem"`
		RequiresExplicitAuthorization bool   `json:"requiresExplicitAuthorization"`
		CurrentSeam                   string `json:"currentSeam"`
		RevisitWhen                   string `json:"revisitWhen"`
	} `json:"extensionPolicy"`
	SecurityBoundary struct {
		NodeProbe           bool `json:"nodeProbe"`
		PrivilegedDaemonSet bool `json:"privilegedDaemonSet"`
		BrowserCredentials  bool `json:"browserCredentials"`
		WriteControls       bool `json:"writeControls"`
	} `json:"securityBoundary"`
}

type releaseSlice struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Optional     bool              `json:"optional"`
	DependsOn    []string          `json:"dependsOn"`
	Issues       []int             `json:"issues"`
	Capabilities []string          `json:"capabilities"`
	Modes        map[string]string `json:"modes"`
	Acceptance   []string          `json:"acceptance"`
}

func main() {
	contractPath := flag.String("contract", defaultContractPath, "path to the release contract")
	sliceID := flag.String("slice", "", "print one release slice")
	asJSON := flag.Bool("json", false, "print the validated contract as JSON")
	flag.Parse()

	c, err := loadContract(*contractPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scope release contract:", err)
		os.Exit(1)
	}
	if err := validateContract(c); err != nil {
		fmt.Fprintln(os.Stderr, "scope release contract: INVALID:", err)
		os.Exit(1)
	}
	if *asJSON {
		if err := json.NewEncoder(os.Stdout).Encode(c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := printContract(os.Stdout, c, *sliceID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadContract(path string) (contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return contract{}, err
	}
	var c contract
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return contract{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return contract{}, errors.New("contract must contain exactly one JSON document")
	}
	return c, nil
}

func validateContract(c contract) error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("schemaVersion must be 1, got %d", c.SchemaVersion)
	}
	if c.Product.BlockingGraduationIssue != 40 || !equalInts(c.Product.PopeyeFoundationIssues, []int{33, 34, 35, 36, 37, 38, 39, 40}) {
		return errors.New("Popeye foundation issues #33-#40 and graduation issue #40 must block the product label")
	}
	if c.Product.PreviewLabel == "" || c.Product.GraduatedLabel == "" || c.Product.Lineage == "" {
		return errors.New("product lineage and labels are required")
	}
	for _, category := range []string{"core", "adapted", "deferred", "rejected"} {
		if len(c.ScopeConcepts[category]) == 0 {
			return fmt.Errorf("scope concept category %q is required", category)
		}
	}
	wantOrder := []string{"graph-baseline", "live-continuity", "contextual-investigation", "external-evidence"}
	wantIssues := [][]int{{41, 42}, {43}, {44}, {45}}
	wantDependencies := [][]string{{}, {"graph-baseline"}, {"graph-baseline", "live-continuity"}, {"contextual-investigation"}}
	if len(c.Slices) != len(wantOrder) {
		return fmt.Errorf("expected %d release slices, got %d", len(wantOrder), len(c.Slices))
	}
	seen := map[string]bool{}
	for i, slice := range c.Slices {
		if slice.ID != wantOrder[i] {
			return fmt.Errorf("slice %d must be %q, got %q", i+1, wantOrder[i], slice.ID)
		}
		if !equalInts(slice.Issues, wantIssues[i]) {
			return fmt.Errorf("slice %q must map to issues %v", slice.ID, wantIssues[i])
		}
		if !equalStrings(slice.DependsOn, wantDependencies[i]) {
			return fmt.Errorf("slice %q must depend on %v", slice.ID, wantDependencies[i])
		}
		for _, dependency := range slice.DependsOn {
			if !seen[dependency] {
				return fmt.Errorf("slice %q depends on unknown or later slice %q", slice.ID, dependency)
			}
		}
		seen[slice.ID] = true
		if slice.Name == "" || len(slice.Capabilities) == 0 || len(slice.Acceptance) == 0 {
			return fmt.Errorf("slice %q requires a name, capabilities, and acceptance scenarios", slice.ID)
		}
		for _, mode := range []string{"offline", "live", "hub"} {
			if strings.TrimSpace(slice.Modes[mode]) == "" {
				return fmt.Errorf("slice %q must define %s behavior", slice.ID, mode)
			}
		}
	}
	if c.Slices[0].Optional || c.Slices[1].Optional || c.Slices[2].Optional || !c.Slices[3].Optional {
		return errors.New("only external-evidence may be optional")
	}
	if !c.ExtensionPolicy.RequiresExplicitAuthorization || c.ExtensionPolicy.GeneralPluginSystem {
		return errors.New("external evidence must require authorization and a general plugin system must remain disabled")
	}
	if c.SecurityBoundary.NodeProbe || c.SecurityBoundary.PrivilegedDaemonSet || c.SecurityBoundary.BrowserCredentials || c.SecurityBoundary.WriteControls {
		return errors.New("rejected security capabilities must remain disabled")
	}
	if len(c.CrossCutting.TestCommands) == 0 || len(c.CrossCutting.BrowserScenarios) == 0 || len(c.CrossCutting.Security) == 0 || len(c.CrossCutting.Compatibility) == 0 {
		return errors.New("cross-cutting test, browser, security, and compatibility gates are required")
	}
	if !equalInts(c.CrossCutting.Issues, []int{46, 47}) {
		return errors.New("issues #46 and #47 must remain cross-cutting graduation gates")
	}
	return nil
}

func printContract(w io.Writer, c contract, selected string) error {
	if selected != "" {
		for _, slice := range c.Slices {
			if slice.ID == selected {
				fmt.Fprintf(w, "Scope release slice: %s (%s)\n", slice.Name, slice.ID)
				fmt.Fprintf(w, "Optional: %t\n", slice.Optional)
				fmt.Fprintf(w, "Dependencies: %s\n", listOrNone(slice.DependsOn))
				for _, mode := range []string{"offline", "live", "hub"} {
					fmt.Fprintf(w, "%s: %s\n", modeLabel(mode), slice.Modes[mode])
				}
				fmt.Fprintln(w, "Acceptance:")
				for _, item := range slice.Acceptance {
					fmt.Fprintln(w, "-", item)
				}
				return nil
			}
		}
		return fmt.Errorf("unknown slice %q", selected)
	}

	fmt.Fprintln(w, "Scope reborn release contract: VALID")
	fmt.Fprintf(w, "Product graduation requires Popeye replacement gate #%d.\n", c.Product.BlockingGraduationIssue)
	for i, slice := range c.Slices {
		kind := "required"
		if slice.Optional {
			kind = "optional"
		}
		fmt.Fprintf(w, "%d. %s [%s] — depends on: %s\n", i+1, slice.Name, kind, listOrNone(slice.DependsOn))
	}
	fmt.Fprintln(w, "Security boundary: node probe=false privileged DaemonSet=false browser credentials=false write controls=false")
	return nil
}

func equalInts(values, wanted []int) bool {
	if len(values) != len(wanted) {
		return false
	}
	for index := range values {
		if values[index] != wanted[index] {
			return false
		}
	}
	return true
}

func equalStrings(values, wanted []string) bool {
	if len(values) != len(wanted) {
		return false
	}
	for index := range values {
		if values[index] != wanted[index] {
			return false
		}
	}
	return true
}

func listOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func modeLabel(mode string) string {
	switch mode {
	case "offline":
		return "Offline"
	case "live":
		return "Live"
	case "hub":
		return "Hub"
	default:
		return mode
	}
}
