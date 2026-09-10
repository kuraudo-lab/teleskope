// Package compare derives deterministic migration-oriented differences between
// two Teleskope inventory snapshots.
package compare

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

const schemaVersion = "teleskope.io/comparison/v1alpha1"

// ClusterRole names a snapshot's role in a migration comparison.
type ClusterRole string

const (
	SourceRole ClusterRole = "source"
	TargetRole ClusterRole = "target"
)

// Report describes migration-relevant differences between two snapshots.
type Report struct {
	SchemaVersion    string            `json:"schemaVersion"`
	GeneratedAt      time.Time         `json:"generatedAt"`
	Source           ClusterSummary    `json:"source"`
	Target           ClusterSummary    `json:"target"`
	Summary          string            `json:"summary"`
	Findings         []Finding         `json:"findings"`
	CapabilityDiffs  []CapabilityDiff  `json:"capabilityDiffs,omitempty"`
	InventoryDiffs   []InventoryDiff   `json:"inventoryDiffs,omitempty"`
	CoverageWarnings []CoverageWarning `json:"coverageWarnings,omitempty"`
}

// ClusterSummary identifies one side of the comparison.
type ClusterSummary struct {
	Role        ClusterRole `json:"role"`
	Name        string      `json:"name"`
	Context     string      `json:"context,omitempty"`
	Server      string      `json:"server,omitempty"`
	Provider    string      `json:"provider,omitempty"`
	Region      string      `json:"region,omitempty"`
	Version     string      `json:"version,omitempty"`
	CollectedAt time.Time   `json:"collectedAt"`
}

// Finding is a migration planning item.
type Finding struct {
	Severity       string `json:"severity"`
	Category       string `json:"category"`
	Key            string `json:"key"`
	Summary        string `json:"summary"`
	Detail         string `json:"detail,omitempty"`
	Source         string `json:"source,omitempty"`
	Target         string `json:"target,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// CapabilityDiff compares advisor capability conclusions for the two clusters.
type CapabilityDiff struct {
	Key              string `json:"key"`
	Scope            string `json:"scope"`
	SourceAssessment string `json:"sourceAssessment"`
	TargetAssessment string `json:"targetAssessment"`
	SourceBasis      string `json:"sourceBasis,omitempty"`
	TargetBasis      string `json:"targetBasis,omitempty"`
	Summary          string `json:"summary"`
}

// InventoryDiff compares inventory sets relevant to migration planning.
type InventoryDiff struct {
	Category          string   `json:"category"`
	OnlyInSource      []string `json:"onlyInSource,omitempty"`
	OnlyInTarget      []string `json:"onlyInTarget,omitempty"`
	Changed           []string `json:"changed,omitempty"`
	MigrationRelevant bool     `json:"migrationRelevant"`
}

// CoverageWarning records incomplete collection that weakens comparison results.
type CoverageWarning struct {
	Role     ClusterRole `json:"role"`
	Area     string      `json:"area"`
	Resource string      `json:"resource"`
	Status   string      `json:"status"`
	Reason   string      `json:"reason,omitempty"`
}

// Analyze compares source and target snapshots without connecting to either cluster.
func Analyze(source, target *inventory.Snapshot) Report {
	r := Report{
		SchemaVersion: schemaVersion,
		GeneratedAt:   time.Now().UTC(),
		Source:        summarize(SourceRole, source),
		Target:        summarize(TargetRole, target),
	}
	if source == nil || target == nil {
		r.Findings = append(r.Findings, Finding{
			Severity:       "blocker",
			Category:       "input",
			Key:            "input.snapshot.missing",
			Summary:        "Both source and target snapshots are required.",
			Recommendation: "Run two scans and pass their snapshot.json files or report directories.",
		})
		r.Summary = summarizeFindings(r.Findings)
		return r
	}

	r.Findings = append(r.Findings, compareVersions(source, target)...)
	r.CapabilityDiffs, r.Findings = compareCapabilities(source, target, r.Findings)
	r.InventoryDiffs, r.Findings = compareInventory(source, target, r.Findings)
	r.Findings = append(r.Findings, compareWorkloadReferences(source, target)...)
	r.CoverageWarnings = coverageWarnings(SourceRole, source)
	r.CoverageWarnings = append(r.CoverageWarnings, coverageWarnings(TargetRole, target)...)
	for _, warning := range r.CoverageWarnings {
		r.Findings = append(r.Findings, Finding{
			Severity:       "info",
			Category:       "coverage",
			Key:            string(warning.Role) + "." + warning.Area + "." + warning.Resource,
			Summary:        fmt.Sprintf("%s %s coverage is %s.", warning.Role, warning.Resource, warning.Status),
			Detail:         warning.Reason,
			Recommendation: "Review collection coverage before treating absent inventory as absent capability.",
		})
	}
	sortFindings(r.Findings)
	sort.Slice(r.CapabilityDiffs, func(i, j int) bool {
		return r.CapabilityDiffs[i].Key+"/"+r.CapabilityDiffs[i].Scope < r.CapabilityDiffs[j].Key+"/"+r.CapabilityDiffs[j].Scope
	})
	sort.Slice(r.InventoryDiffs, func(i, j int) bool { return r.InventoryDiffs[i].Category < r.InventoryDiffs[j].Category })
	r.Summary = summarizeFindings(r.Findings)
	return r
}

// LoadSnapshot reads a snapshot from either a snapshot.json file or a scan report directory.
func LoadSnapshot(path string) (*inventory.Snapshot, error) {
	if path == "" {
		return nil, fmt.Errorf("snapshot path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", path, err)
	}
	if info.IsDir() {
		path = filepath.Join(path, "snapshot.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", path, err)
	}
	var snapshot inventory.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode snapshot %s: %w", path, err)
	}
	return &snapshot, nil
}

// WriteJSON writes a pretty JSON comparison report.
func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// WriteHuman writes a compact terminal comparison summary.
func WriteHuman(w io.Writer, report Report) error {
	fmt.Fprintf(w, "teleskope comparison\n")
	fmt.Fprintf(w, "source     %s  k8s=%s\n", value(report.Source.Name), value(report.Source.Version))
	fmt.Fprintf(w, "target     %s  k8s=%s\n", value(report.Target.Name), value(report.Target.Version))
	fmt.Fprintf(w, "summary    %s\n\n", report.Summary)
	if len(report.Findings) == 0 {
		fmt.Fprintln(w, "✓ no migration differences detected by current rules")
		return nil
	}
	for _, f := range report.Findings {
		fmt.Fprintf(w, "%s %s %s\n", severityMarker(f.Severity), f.Category, f.Summary)
		if f.Source != "" || f.Target != "" {
			fmt.Fprintf(w, "  source=%s target=%s\n", value(f.Source), value(f.Target))
		}
		if f.Recommendation != "" {
			fmt.Fprintf(w, "  next: %s\n", f.Recommendation)
		}
	}
	return nil
}

// Markdown renders a comparison report for migration review.
func Markdown(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Teleskope migration comparison\n\n")
	fmt.Fprintf(&b, "%s\n\n", report.Summary)
	fmt.Fprintf(&b, "## Clusters\n\n")
	fmt.Fprintf(&b, "| Role | Name | Kubernetes | Context | Region | Collected |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- | --- | --- |\n")
	fmt.Fprintf(&b, "| Source | %s | %s | %s | %s | %s |\n", mdCell(report.Source.Name), mdCell(report.Source.Version), mdCell(report.Source.Context), mdCell(report.Source.Region), mdCell(formatTime(report.Source.CollectedAt)))
	fmt.Fprintf(&b, "| Target | %s | %s | %s | %s | %s |\n\n", mdCell(report.Target.Name), mdCell(report.Target.Version), mdCell(report.Target.Context), mdCell(report.Target.Region), mdCell(formatTime(report.Target.CollectedAt)))

	fmt.Fprintf(&b, "## Findings\n\n")
	if len(report.Findings) == 0 {
		fmt.Fprintf(&b, "No migration differences were detected by the current deterministic rules.\n\n")
	} else {
		fmt.Fprintf(&b, "| Severity | Category | Finding | Source | Target | Recommendation |\n")
		fmt.Fprintf(&b, "| --- | --- | --- | --- | --- | --- |\n")
		for _, f := range report.Findings {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", mdCell(f.Severity), mdCell(f.Category), mdCell(f.Summary), mdCell(f.Source), mdCell(f.Target), mdCell(f.Recommendation))
		}
		fmt.Fprintln(&b)
	}

	if len(report.CapabilityDiffs) > 0 {
		fmt.Fprintf(&b, "## Capability differences\n\n")
		fmt.Fprintf(&b, "| Capability | Scope | Source | Target | Summary |\n")
		fmt.Fprintf(&b, "| --- | --- | --- | --- | --- |\n")
		for _, d := range report.CapabilityDiffs {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", mdCell(d.Key), mdCell(d.Scope), mdCell(d.SourceAssessment), mdCell(d.TargetAssessment), mdCell(d.Summary))
		}
		fmt.Fprintln(&b)
	}

	if len(report.InventoryDiffs) > 0 {
		fmt.Fprintf(&b, "## Inventory differences\n\n")
		for _, d := range report.InventoryDiffs {
			fmt.Fprintf(&b, "### %s\n\n", mdInline(d.Category))
			if len(d.OnlyInSource) > 0 {
				fmt.Fprintf(&b, "- Only in source: %s\n", mdInline(strings.Join(d.OnlyInSource, ", ")))
			}
			if len(d.OnlyInTarget) > 0 {
				fmt.Fprintf(&b, "- Only in target: %s\n", mdInline(strings.Join(d.OnlyInTarget, ", ")))
			}
			if len(d.Changed) > 0 {
				fmt.Fprintf(&b, "- Changed: %s\n", mdInline(strings.Join(d.Changed, ", ")))
			}
			fmt.Fprintln(&b)
		}
	}

	if len(report.CoverageWarnings) > 0 {
		fmt.Fprintf(&b, "## Coverage warnings\n\n")
		fmt.Fprintf(&b, "| Role | Area | Resource | Status | Reason |\n")
		fmt.Fprintf(&b, "| --- | --- | --- | --- | --- |\n")
		for _, w := range report.CoverageWarnings {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", mdCell(string(w.Role)), mdCell(w.Area), mdCell(w.Resource), mdCell(w.Status), mdCell(w.Reason))
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func summarize(role ClusterRole, s *inventory.Snapshot) ClusterSummary {
	if s == nil {
		return ClusterSummary{Role: role}
	}
	name := firstNonEmpty(s.EKS.Cluster.Name, s.Kubernetes.Context, s.Kubernetes.Server, "unknown")
	provider := ""
	if s.EKS.Cluster.Name != "" || s.EKS.Cluster.ARN != "" {
		provider = "eks"
	} else if s.Kubernetes.Server != "" || s.Kubernetes.Context != "" {
		provider = "kubernetes"
	}
	version := firstNonEmpty(s.Kubernetes.Version.GitVersion, s.EKS.Cluster.Version)
	return ClusterSummary{
		Role:        role,
		Name:        name,
		Context:     s.Kubernetes.Context,
		Server:      s.Kubernetes.Server,
		Provider:    provider,
		Region:      s.AWS.Region,
		Version:     version,
		CollectedAt: s.CollectedAt,
	}
}

func compareVersions(source, target *inventory.Snapshot) []Finding {
	src := firstNonEmpty(source.Kubernetes.Version.GitVersion, source.EKS.Cluster.Version)
	dst := firstNonEmpty(target.Kubernetes.Version.GitVersion, target.EKS.Cluster.Version)
	if src == "" || dst == "" || src == dst {
		return nil
	}
	severity := "info"
	recommendation := "Review API removals, controller compatibility, and upgrade notes before migration."
	if versionRank(dst) < versionRank(src) {
		severity = "warn"
		recommendation = "Target Kubernetes version appears older than source; verify workloads and APIs before planning migration."
	}
	return []Finding{{
		Severity:       severity,
		Category:       "version",
		Key:            "kubernetes.version",
		Summary:        "Kubernetes versions differ.",
		Source:         src,
		Target:         dst,
		Recommendation: recommendation,
	}}
}

func compareCapabilities(source, target *inventory.Snapshot, findings []Finding) ([]CapabilityDiff, []Finding) {
	sourceCaps := capabilityMap(advisor.Analyze(source))
	targetCaps := capabilityMap(advisor.Analyze(target))
	keys := unionKeys(sourceCaps, targetCaps)
	diffs := []CapabilityDiff{}
	for _, key := range keys {
		src, hasSource := sourceCaps[key]
		dst, hasTarget := targetCaps[key]
		if !hasSource || !hasTarget {
			summary := "Capability exists on only one side."
			diffs = append(diffs, CapabilityDiff{Key: capabilityKeyPart(key, 0), Scope: capabilityKeyPart(key, 1), SourceAssessment: assessmentOrMissing(src, hasSource), TargetAssessment: assessmentOrMissing(dst, hasTarget), Summary: summary})
			if hasSource && !hasTarget && src.Assessment == "supported" {
				findings = append(findings, Finding{Severity: "warn", Category: "capability", Key: key, Summary: "Source supported capability is missing from target.", Source: src.Summary, Recommendation: "Confirm the target has an equivalent implementation or update manifests before migration."})
			}
			continue
		}
		if src.Assessment == dst.Assessment && src.Implementation == dst.Implementation && src.Coverage == dst.Coverage {
			continue
		}
		summary := fmt.Sprintf("Source is %s; target is %s.", src.Assessment, dst.Assessment)
		diffs = append(diffs, CapabilityDiff{Key: src.Key, Scope: src.Scope, SourceAssessment: src.Assessment, TargetAssessment: dst.Assessment, SourceBasis: src.Basis, TargetBasis: dst.Basis, Summary: summary})
		if src.Assessment == "supported" && dst.Assessment != "supported" {
			findings = append(findings, Finding{Severity: "warn", Category: "capability", Key: key, Summary: "Target does not show a capability supported by source.", Source: src.Summary, Target: dst.Summary, Recommendation: "Add or verify target capability before moving dependent workloads."})
		}
	}
	return diffs, findings
}

func compareInventory(source, target *inventory.Snapshot, findings []Finding) ([]InventoryDiff, []Finding) {
	checks := []struct {
		category string
		source   map[string]string
		target   map[string]string
		relevant bool
	}{
		{"api resources", apiResourceSet(source.Kubernetes.APIResources), apiResourceSet(target.Kubernetes.APIResources), true},
		{"custom resource definitions", crdSet(source.Kubernetes.CustomResourceDefinitions), crdSet(target.Kubernetes.CustomResourceDefinitions), true},
		{"namespaces", namespaceSet(source.Kubernetes.Namespaces), namespaceSet(target.Kubernetes.Namespaces), true},
		{"ingress classes", ingressClassSet(source.Kubernetes.IngressClasses), ingressClassSet(target.Kubernetes.IngressClasses), true},
		{"gateway classes", gatewayClassSet(source.Kubernetes.GatewayClasses), gatewayClassSet(target.Kubernetes.GatewayClasses), true},
		{"storage classes", storageClassSet(source.Kubernetes.StorageClasses), storageClassSet(target.Kubernetes.StorageClasses), true},
		{"csi drivers", csiDriverSet(source.Kubernetes.CSIDrivers), csiDriverSet(target.Kubernetes.CSIDrivers), true},
		{"runtime classes", runtimeClassSet(source.Kubernetes.RuntimeClasses), runtimeClassSet(target.Kubernetes.RuntimeClasses), true},
		{"node platforms", nodePlatformSet(source.Kubernetes.Nodes), nodePlatformSet(target.Kubernetes.Nodes), true},
		{"service accounts", serviceAccountSet(source.Kubernetes.ServiceAccounts), serviceAccountSet(target.Kubernetes.ServiceAccounts), false},
		{"config maps", configObjectSet(source.Kubernetes.ConfigMaps), configObjectSet(target.Kubernetes.ConfigMaps), false},
		{"secrets", secretSet(source.Kubernetes.Secrets), secretSet(target.Kubernetes.Secrets), false},
	}
	diffs := []InventoryDiff{}
	for _, check := range checks {
		onlySource, onlyTarget, changed := diffMaps(check.source, check.target)
		if len(onlySource) == 0 && len(onlyTarget) == 0 && len(changed) == 0 {
			continue
		}
		diffs = append(diffs, InventoryDiff{Category: check.category, OnlyInSource: onlySource, OnlyInTarget: onlyTarget, Changed: changed, MigrationRelevant: check.relevant})
		if check.relevant && len(onlySource) > 0 {
			findings = append(findings, Finding{
				Severity:       severityForCategory(check.category),
				Category:       "inventory",
				Key:            "inventory." + strings.ReplaceAll(check.category, " ", "."),
				Summary:        fmt.Sprintf("%s used or declared in source are missing from target.", title(check.category)),
				Source:         strings.Join(limit(onlySource, 8), ", "),
				Recommendation: "Map source dependencies to target equivalents before migration.",
			})
		}
		if check.relevant && len(changed) > 0 {
			findings = append(findings, Finding{
				Severity:       "info",
				Category:       "inventory",
				Key:            "inventory." + strings.ReplaceAll(check.category, " ", ".") + ".changed",
				Summary:        fmt.Sprintf("%s with matching names have different configuration.", title(check.category)),
				Source:         strings.Join(limit(changed, 8), ", "),
				Recommendation: "Review whether name-compatible target objects behave equivalently.",
			})
		}
	}
	return diffs, findings
}

func compareWorkloadReferences(source, target *inventory.Snapshot) []Finding {
	targetNamespaces := namespaceSet(target.Kubernetes.Namespaces)
	targetServiceAccounts := serviceAccountSet(target.Kubernetes.ServiceAccounts)
	targetRuntimeClasses := runtimeClassSet(target.Kubernetes.RuntimeClasses)
	targetConfigMaps := configObjectSet(target.Kubernetes.ConfigMaps)
	targetSecrets := secretSet(target.Kubernetes.Secrets)
	targetPVCs := pvcSet(target.Kubernetes.PersistentVolumeClaims)

	missingNamespaces := map[string]string{}
	missingServiceAccounts := map[string]string{}
	missingRuntimeClasses := map[string]string{}
	missingConfigMaps := map[string]string{}
	missingSecrets := map[string]string{}
	missingPVCs := map[string]string{}

	for _, w := range source.Kubernetes.Workloads {
		if w.Namespace != "" && !hasKey(targetNamespaces, w.Namespace) {
			missingNamespaces[w.Namespace] = w.Namespace
		}
		if w.ServiceAccountName != "" {
			key := namespaced(w.Namespace, w.ServiceAccountName)
			if !hasKey(targetServiceAccounts, key) {
				missingServiceAccounts[key] = key
			}
		}
		if w.RuntimeClassName != "" && !hasKey(targetRuntimeClasses, w.RuntimeClassName) {
			missingRuntimeClasses[w.RuntimeClassName] = w.RuntimeClassName
		}
		for _, ref := range w.ConfigRefs {
			key := namespaced(firstNonEmpty(ref.Namespace, w.Namespace), ref.Name)
			if ref.Name != "" && !hasKey(targetConfigMaps, key) {
				missingConfigMaps[key] = key
			}
		}
		for _, ref := range w.SecretRefs {
			key := namespaced(firstNonEmpty(ref.Namespace, w.Namespace), ref.Name)
			if ref.Name != "" && !hasKey(targetSecrets, key) {
				missingSecrets[key] = key
			}
		}
		for _, volume := range w.Volumes {
			if volume.PersistentVolumeClaim == "" {
				continue
			}
			key := namespaced(w.Namespace, volume.PersistentVolumeClaim)
			if !hasKey(targetPVCs, key) {
				missingPVCs[key] = key
			}
		}
	}

	checks := []struct {
		key      string
		summary  string
		values   map[string]string
		severity string
	}{
		{"workload.namespaces", "Source workloads use namespaces missing from target.", missingNamespaces, "warn"},
		{"workload.serviceaccounts", "Source workloads reference ServiceAccounts missing from target.", missingServiceAccounts, "warn"},
		{"workload.runtimeclasses", "Source workloads reference RuntimeClasses missing from target.", missingRuntimeClasses, "warn"},
		{"workload.configmaps", "Source workloads reference ConfigMaps missing from target.", missingConfigMaps, "warn"},
		{"workload.secrets", "Source workloads reference Secrets missing from target.", missingSecrets, "warn"},
		{"workload.pvcs", "Source workloads reference PVCs missing from target.", missingPVCs, "info"},
	}
	findings := []Finding{}
	for _, check := range checks {
		values := sortedValues(check.values)
		if len(values) == 0 {
			continue
		}
		findings = append(findings, Finding{
			Severity:       check.severity,
			Category:       "workload",
			Key:            check.key,
			Summary:        check.summary,
			Source:         strings.Join(limit(values, 12), ", "),
			Recommendation: "Create target-side dependencies or adjust manifests during migration planning.",
		})
	}
	return findings
}

func coverageWarnings(role ClusterRole, s *inventory.Snapshot) []CoverageWarning {
	if s == nil {
		return nil
	}
	var out []CoverageWarning
	for _, item := range s.Coverage {
		if item.Status == "" || item.Status == "complete" {
			continue
		}
		out = append(out, CoverageWarning{Role: role, Area: item.Area, Resource: item.Resource, Status: item.Status, Reason: item.Reason})
	}
	sort.Slice(out, func(i, j int) bool {
		return string(out[i].Role)+out[i].Area+out[i].Resource < string(out[j].Role)+out[j].Area+out[j].Resource
	})
	return out
}

func capabilityMap(report advisor.Report) map[string]advisor.Capability {
	out := map[string]advisor.Capability{}
	for _, c := range report.Capabilities {
		out[c.Key+"\x00"+c.Scope] = c
	}
	return out
}

func capabilityKeyPart(key string, index int) string {
	parts := strings.Split(key, "\x00")
	if index < len(parts) {
		return parts[index]
	}
	return ""
}

func assessmentOrMissing(c advisor.Capability, ok bool) string {
	if !ok {
		return "missing"
	}
	return c.Assessment
}

func apiResourceSet(resources []inventory.APIResource) map[string]string {
	out := map[string]string{}
	for _, r := range resources {
		key := firstNonEmpty(r.Group, "core") + "/" + r.Version + "/" + r.Resource
		out[key] = strings.Join(sortedCopy(r.Verbs), ",")
	}
	return out
}

func crdSet(crds []inventory.CustomResourceDefinition) map[string]string {
	out := map[string]string{}
	for _, c := range crds {
		var versions []string
		for _, v := range c.Versions {
			if v.Served {
				versions = append(versions, v.Name)
			}
		}
		sort.Strings(versions)
		key := firstNonEmpty(c.Group, "unknown") + "/" + firstNonEmpty(c.Kind, c.Name)
		out[key] = strings.Join(versions, ",")
	}
	return out
}

func namespaceSet(namespaces []inventory.Namespace) map[string]string {
	out := map[string]string{}
	for _, n := range namespaces {
		if n.Name != "" {
			out[n.Name] = n.Phase
		}
	}
	return out
}

func ingressClassSet(classes []inventory.IngressClass) map[string]string {
	out := map[string]string{}
	for _, c := range classes {
		if c.Name != "" {
			out[c.Name] = c.Controller
		}
	}
	return out
}

func gatewayClassSet(classes []inventory.GatewayClass) map[string]string {
	out := map[string]string{}
	for _, c := range classes {
		if c.Name != "" {
			out[c.Name] = c.ControllerName
		}
	}
	return out
}

func storageClassSet(classes []inventory.StorageClass) map[string]string {
	out := map[string]string{}
	for _, c := range classes {
		if c.Name == "" {
			continue
		}
		expansion := "unknown"
		if c.AllowVolumeExpansion != nil {
			expansion = fmt.Sprint(*c.AllowVolumeExpansion)
		}
		out[c.Name] = c.Provisioner + "|binding=" + c.VolumeBindingMode + "|expansion=" + expansion
	}
	return out
}

func csiDriverSet(drivers []inventory.CSIDriver) map[string]string {
	out := map[string]string{}
	for _, d := range drivers {
		if d.Name != "" {
			out[d.Name] = strings.Join(sortedCopy(d.VolumeLifecycleModes), ",")
		}
	}
	return out
}

func runtimeClassSet(classes []inventory.RuntimeClass) map[string]string {
	out := map[string]string{}
	for _, c := range classes {
		if c.Name != "" {
			out[c.Name] = c.Handler
		}
	}
	return out
}

func nodePlatformSet(nodes []inventory.Node) map[string]string {
	out := map[string]string{}
	for _, n := range nodes {
		key := firstNonEmpty(n.OperatingSystem, "unknown-os") + "/" + firstNonEmpty(n.Architecture, "unknown-arch")
		out[key] = key
	}
	return out
}

func configObjectSet(items []inventory.ConfigObject) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[namespaced(item.Namespace, item.Name)] = strings.Join(sortedMapKeys(item.Data), ",")
	}
	return out
}

func serviceAccountSet(items []inventory.ServiceAccount) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[namespaced(item.Namespace, item.Name)] = strings.Join(sortedCopy(item.ImagePullSecrets), ",")
	}
	return out
}

func secretSet(items []inventory.Secret) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[namespaced(item.Namespace, item.Name)] = item.Type + "|" + strings.Join(sortedCopy(item.Keys), ",")
	}
	return out
}

func pvcSet(items []inventory.PersistentVolumeClaim) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[namespaced(item.Namespace, item.Name)] = item.StorageClassName + "|" + strings.Join(sortedCopy(item.AccessModes), ",")
	}
	return out
}

func diffMaps(source, target map[string]string) ([]string, []string, []string) {
	var onlySource, onlyTarget, changed []string
	for key, sourceValue := range source {
		targetValue, ok := target[key]
		if !ok {
			onlySource = append(onlySource, key)
			continue
		}
		if sourceValue != targetValue {
			changed = append(changed, key)
		}
	}
	for key := range target {
		if _, ok := source[key]; !ok {
			onlyTarget = append(onlyTarget, key)
		}
	}
	sort.Strings(onlySource)
	sort.Strings(onlyTarget)
	sort.Strings(changed)
	return onlySource, onlyTarget, changed
}

func unionKeys[A any](a, b map[string]A) []string {
	keys := make(map[string]struct{}, len(a)+len(b))
	for key := range a {
		keys[key] = struct{}{}
	}
	for key := range b {
		keys[key] = struct{}{}
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sortedValues(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedMapKeys(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func hasKey(values map[string]string, key string) bool {
	_, ok := values[key]
	return ok
}

func namespaced(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func versionRank(version string) int {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	rank := 0
	for i := 0; i < 3 && i < len(parts); i++ {
		n := 0
		for _, r := range parts[i] {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		rank = rank*1000 + n
	}
	return rank
}

func severityForCategory(category string) string {
	switch category {
	case "api resources", "custom resource definitions", "storage classes", "runtime classes":
		return "warn"
	default:
		return "info"
	}
}

func summarizeFindings(findings []Finding) string {
	if len(findings) == 0 {
		return "No migration differences were detected by the current deterministic rules."
	}
	counts := map[string]int{}
	for _, finding := range findings {
		counts[finding.Severity]++
	}
	var parts []string
	for _, severity := range []string{"blocker", "warn", "info"} {
		if counts[severity] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[severity], severity))
		}
	}
	return fmt.Sprintf("%d finding(s): %s.", len(findings), strings.Join(parts, ", "))
}

func sortFindings(findings []Finding) {
	order := map[string]int{"blocker": 0, "warn": 1, "info": 2}
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if order[a.Severity] != order[b.Severity] {
			return order[a.Severity] < order[b.Severity]
		}
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		return a.Key < b.Key
	})
}

func severityMarker(severity string) string {
	switch severity {
	case "blocker":
		return "✕"
	case "warn":
		return "!"
	default:
		return "·"
	}
}

func title(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func value(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func limit(values []string, n int) []string {
	if len(values) <= n {
		return values
	}
	out := append([]string(nil), values[:n]...)
	out = append(out, fmt.Sprintf("+%d more", len(values)-n))
	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func mdInline(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "`", "\\`")
	return value
}

func mdCell(value string) string {
	value = mdInline(value)
	value = strings.ReplaceAll(value, "\n", "<br>")
	value = strings.ReplaceAll(value, "|", "\\|")
	if value == "" {
		return "-"
	}
	return value
}
