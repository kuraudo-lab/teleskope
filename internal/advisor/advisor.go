// Package advisor derives conservative, evidence-backed capability summaries.
// It never connects to a cluster or mutates its input.
package advisor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// Analyze summarizes declared configuration and observed bindings, not runtime guarantees.
func Analyze(s *inventory.Snapshot) Report {
	r := Report{SchemaVersion: "teleskope.io/advisor/v1alpha1", RuleVersion: "1", Capabilities: []Capability{}}
	if s == nil {
		r.Summary = "No snapshot available; cluster capabilities are unknown."
		return r
	}
	coverage := func(resources ...string) (string, time.Time) {
		state := "complete"
		var at time.Time
		for _, resource := range resources {
			found := false
			for _, c := range s.Coverage {
				if c.Area != "kubernetes" || c.Resource != resource {
					continue
				}
				found = true
				if c.Status != "complete" {
					state = "partial"
				}
				if at.IsZero() || c.CollectedAt.Before(at) {
					at = c.CollectedAt
				}
			}
			if !found {
				state = "partial"
			}
		}
		return state, at
	}
	add := func(key, scope, impl, assessment, basis, summary, constraint string, ref inventory.ObjectRef, field, value string, resources ...string) {
		cov, _ := coverage(resources...)
		var at time.Time
		if len(resources) > 0 {
			_, at = coverage(resources[0])
		}
		c := Capability{Key: key, Scope: scope, Implementation: impl, Assessment: assessment, Basis: basis, Summary: summary, Coverage: cov, Freshness: "snapshot", RuleID: key + "/v1"}
		for _, resource := range resources {
			found := false
			for _, item := range s.Coverage {
				if item.Area == "kubernetes" && item.Resource == resource {
					c.Collection = append(c.Collection, item)
					found = true
				}
			}
			if !found {
				c.Collection = append(c.Collection, inventory.CoverageItem{Area: "kubernetes", Resource: resource, Status: "unavailable", Reason: "Collection coverage not recorded in this snapshot."})
			}
		}
		if constraint != "" {
			c.Constraints = []string{constraint}
		}
		if ref.Name != "" {
			c.Evidence = []Evidence{{Resource: ref, Field: field, Value: value, CollectedAt: at}}
		}
		r.Capabilities = append(r.Capabilities, c)
	}
	k := s.Kubernetes
	volumes := make(map[string]inventory.PersistentVolume, len(k.PersistentVolumes))
	for _, pv := range k.PersistentVolumes {
		volumes[pv.Name] = pv
	}
	claims := make(map[string][]inventory.PersistentVolumeClaim)
	for _, pvc := range k.PersistentVolumeClaims {
		claims[pvc.StorageClassName] = append(claims[pvc.StorageClassName], pvc)
	}
	for _, c := range k.IngressClasses {
		add("networking.ingress", c.Name, c.Controller, "supported", "declared", "Ingress class configured.", "Controller health and end-to-end routing are not verified.", c.ObjectRef, "spec.controller", c.Controller, "IngressClasses")
	}
	if len(k.IngressClasses) == 0 {
		add("networking.ingress", "cluster", "", "unknown", "none", "No Ingress class observed.", "Legacy or classless controllers may exist; missing classes do not prove lack of support.", inventory.ObjectRef{}, "", "", "IngressClasses")
	}
	for _, c := range k.GatewayClasses {
		add("networking.gateway", c.Name, c.ControllerName, "supported", "declared", "Gateway class configured.", "Route kinds, controller acceptance and data-plane readiness are not verified.", c.ObjectRef, "spec.controllerName", c.ControllerName, "GatewayClasses")
	}
	if len(k.GatewayClasses) == 0 {
		add("networking.gateway", "cluster", "", "unknown", "none", "No Gateway class observed.", "API registration alone does not establish a usable Gateway implementation.", inventory.ObjectRef{}, "", "", "GatewayClasses", "APIResources")
	}
	for _, c := range k.StorageClasses {
		assessment, basis, summary := "unknown", "none", "RWX support is unknown."
		var bound []inventory.PersistentVolumeClaim
		for _, pvc := range claims[c.Name] {
			if pvc.StorageClassName != c.Name || pvc.Phase != "Bound" || !has(pvc.AccessModes, "ReadWriteMany") {
				continue
			}
			if pv, exists := volumes[pvc.VolumeName]; exists {
				if pv.Name == pvc.VolumeName && pv.StorageClassName == c.Name && pv.Phase == "Bound" && pv.ClaimRef.Name == pvc.Name && pv.ClaimRef.Namespace == pvc.Namespace && (pv.ClaimRef.UID == "" || pvc.UID == "" || pv.ClaimRef.UID == pvc.UID) && has(pv.AccessModes, "ReadWriteMany") {
					bound = append(bound, pvc)
				}
			}
		}
		if len(bound) > 0 {
			assessment, basis, summary = "supported", "observed", fmt.Sprintf("%d RWX PVC/PV binding(s) observed.", len(bound))
		}
		add("storage.rwx", c.Name, c.Provisioner, assessment, basis, summary, "Binding is not a multi-node read/write test or a guarantee for new volumes. Check volume mode and backend configuration.", c.ObjectRef, "provisioner", c.Provisioner, "StorageClasses", "PersistentVolumes", "PersistentVolumeClaims")
		current := &r.Capabilities[len(r.Capabilities)-1]
		for _, pvc := range bound {
			_, at := coverage("PersistentVolumeClaims")
			current.Evidence = append(current.Evidence, Evidence{pvc.ObjectRef, "spec.accessModes / spec.volumeName / status.phase", "ReadWriteMany / " + pvc.VolumeName + " / Bound", at})
			if pv, exists := volumes[pvc.VolumeName]; exists {
				if pv.Name == pvc.VolumeName {
					_, at = coverage("PersistentVolumes")
					current.Evidence = append(current.Evidence, Evidence{pv.ObjectRef, "spec.accessModes / spec.volumeMode / status.phase", strings.Join(pv.AccessModes, ",") + " / " + pv.VolumeMode + " / " + pv.Phase, at})
				}
			}
		}
		assessment, summary = "unknown", "Volume expansion setting was not recorded."
		if c.AllowVolumeExpansion != nil {
			assessment = "unsupported"
			summary = "StorageClass does not allow volume expansion."
			if *c.AllowVolumeExpansion {
				assessment = "supported"
				summary = "StorageClass allows volume expansion."
			}
		}
		add("storage.expansion", c.Name, c.Provisioner, assessment, "declared", summary, "Actual expansion depends on the driver, backend and volume.", c.ObjectRef, "allowVolumeExpansion", expansionValue(c.AllowVolumeExpansion), "StorageClasses")
	}
	if len(k.StorageClasses) == 0 {
		add("storage.rwx", "cluster", "", "unknown", "none", "No StorageClass observed.", "Static volumes may exist; this does not prove RWX is unavailable.", inventory.ObjectRef{}, "", "", "StorageClasses")
	}
	for _, c := range k.CustomResourceDefinitions {
		var served []string
		for _, v := range c.Versions {
			if v.Served {
				served = append(served, v.Name)
			}
		}
		sort.Strings(served)
		assessment := "unknown"
		if len(served) > 0 {
			assessment = "supported"
		}
		add("extensions.api", c.Name, c.Group, assessment, "declared", "Custom API declared: "+c.Kind+" ("+strings.Join(served, ", ")+").", "CRD declarations do not verify API establishment or operator health.", c.ObjectRef, "spec.versions[served=true]", strings.Join(served, ", "), "CustomResourceDefinitions")
	}
	if len(k.CustomResourceDefinitions) == 0 {
		add("extensions.api", "cluster", "", "unknown", "none", "No custom API definitions observed.", "Review collection coverage before drawing conclusions about extensions.", inventory.ObjectRef{}, "", "", "CustomResourceDefinitions")
	}
	for _, insight := range sortedEKSInsights(s.EKS.Insights) {
		r.Capabilities = append(r.Capabilities, eksInsightCapability(insight, s.Coverage))
	}
	if len(s.EKS.Insights) == 0 {
		cov, _ := coverageArea(s.Coverage, "eks", "Insights")
		r.Capabilities = append(r.Capabilities, Capability{
			Key:         "eks.insight",
			Scope:       "cluster",
			Assessment:  "unknown",
			Basis:       "none",
			Summary:     "No EKS upgrade or readiness insights were observed in the snapshot.",
			Constraints: []string{"Missing insight evidence is not proof that the cluster is upgrade-ready."},
			Collection:  collectionFor(s.Coverage, "eks", "Insights"),
			Coverage:    cov,
			Freshness:   "snapshot",
			RuleID:      "eks.insight/v1",
		})
	}
	addonCompatibility := addonCompatibilityByName(s.EKS.Insights)
	for _, addon := range s.EKS.Addons {
		assessment, basis := "supported", "observed"
		summary := "Managed add-on is active and has no recorded health issues."
		constraint := "Add-on status and EKS insights do not verify controller runtime health inside the cluster."
		if addon.Status != "" && addon.Status != "ACTIVE" {
			assessment = "unsupported"
			summary = "Managed add-on is not ACTIVE."
		}
		if len(addon.Issues) > 0 {
			assessment = "unsupported"
			summary = fmt.Sprintf("Managed add-on reports %d health issue(s).", len(addon.Issues))
		}
		compat := addonCompatibility[addon.Name]
		if len(compat.CompatibleVersions) > 0 {
			basis = "aws-reported"
			if assessment == "supported" && !has(compat.CompatibleVersions, addon.Version) {
				assessment = "unknown"
				summary = "EKS reports compatible add-on versions for a target Kubernetes release; current version is not listed."
			}
		}
		cov, at := coverageArea(s.Coverage, "eks", "Addons")
		capability := Capability{
			Key:            "eks.addon",
			Scope:          addon.Name,
			Implementation: addon.Version,
			Assessment:     assessment,
			Basis:          basis,
			Summary:        summary,
			Constraints:    []string{constraint},
			Collection:     collectionFor(s.Coverage, "eks", "Addons", "Insights"),
			Coverage:       cov,
			Freshness:      "snapshot",
			RuleID:         "eks.addon/v1",
		}
		ref := inventory.ObjectRef{Kind: "Addon", Name: addon.Name}
		capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "addonVersion/status", Value: addon.Version + "/" + addon.Status, CollectedAt: at})
		for _, issue := range addon.Issues {
			capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "health.issue", Value: strings.TrimSpace(issue.Code + ":" + issue.Message), CollectedAt: at})
		}
		if len(compat.TargetKubernetes) > 0 || len(compat.CompatibleVersions) > 0 {
			_, insightAt := coverageArea(s.Coverage, "eks", "Insights")
			capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "insight.addonCompatibility", Value: fmt.Sprintf("target=%s compatible=%s status=%s", strings.Join(compat.TargetKubernetes, ","), strings.Join(compat.CompatibleVersions, ","), strings.Join(compat.Statuses, ",")), CollectedAt: insightAt})
		}
		r.Capabilities = append(r.Capabilities, capability)
	}
	nodegroups := map[string]inventory.Nodegroup{}
	for _, nodegroup := range s.EKS.Nodegroups {
		if nodegroup.Name != "" {
			nodegroups[nodegroup.Name] = nodegroup
		}
	}
	for _, nodegroup := range sortedNodegroups(s.EKS.Nodegroups) {
		nodes := nodesForNodegroup(s.Kubernetes.Nodes, nodegroup.Name)
		assessment, basis := "supported", "observed"
		summary := fmt.Sprintf("Managed nodegroup has %d observed Kubernetes node(s).", len(nodes))
		constraint := "EKS nodegroup configuration and Kubernetes node status do not prove AMI patch freshness or workload safety."
		if len(nodes) == 0 {
			assessment, basis = "unknown", "declared"
			summary = "Managed nodegroup has no matching Kubernetes node evidence."
		}
		if strings.Contains(strings.ToUpper(nodegroup.AMIType), "CUSTOM") {
			assessment = "unknown"
			summary = "Managed nodegroup uses a custom AMI; runtime compatibility cannot be inferred from EKS AMI type."
		}
		if nodegroup.Status != "" && nodegroup.Status != "ACTIVE" {
			assessment = "unsupported"
			summary = "Managed nodegroup is not ACTIVE."
		}
		_, at := coverageArea(s.Coverage, "eks", "Nodegroups")
		capability := Capability{
			Key:            "eks.nodegroup",
			Scope:          nodegroup.Name,
			Implementation: nodegroup.ReleaseVersion,
			Assessment:     assessment,
			Basis:          basis,
			Summary:        summary,
			Constraints:    []string{constraint},
			Collection:     nodegroupCollection(s.Coverage),
			Coverage:       combinedCoverage(s.Coverage, [][2]string{{"eks", "Nodegroups"}, {"kubernetes", "Nodes"}}),
			Freshness:      "snapshot",
			RuleID:         "eks.nodegroup/v1",
		}
		ref := inventory.ObjectRef{Kind: "Nodegroup", Name: nodegroup.Name}
		capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "version/release/ami/status", Value: strings.Join([]string{nodegroup.Version, nodegroup.ReleaseVersion, nodegroup.AMIType, nodegroup.Status}, "/"), CollectedAt: at})
		if lt := launchTemplateValue(nodegroup); lt != "" {
			capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "launchTemplate", Value: lt, CollectedAt: at})
		}
		_, nodeAt := coverageArea(s.Coverage, "kubernetes", "Nodes")
		for _, node := range nodes {
			capability.Evidence = append(capability.Evidence, Evidence{Resource: node.ObjectRef, Field: "node.runtime", Value: fmt.Sprintf("zone=%s kubelet=%s os=%s runtime=%s instance=%s", node.Labels["topology.kubernetes.io/zone"], node.KubeletVersion, node.OSImage, node.ContainerRuntime, node.Labels["node.kubernetes.io/instance-type"]), CollectedAt: nodeAt})
		}
		r.Capabilities = append(r.Capabilities, capability)
	}
	for _, group := range unmanagedNodeGroups(s.Kubernetes.Nodes, nodegroups) {
		capability := Capability{
			Key:            "eks.nodegroup",
			Scope:          group.Scope,
			Implementation: "",
			Assessment:     "unknown",
			Basis:          "observed",
			Summary:        group.Summary,
			Constraints:    []string{"Node ownership is observed from Kubernetes labels only; EKS managed nodegroup configuration was not available for this group."},
			Collection:     nodegroupCollection(s.Coverage),
			Coverage:       "partial",
			Freshness:      "snapshot",
			RuleID:         "eks.nodegroup/v1",
		}
		_, nodeAt := coverageArea(s.Coverage, "kubernetes", "Nodes")
		for _, node := range group.Nodes {
			capability.Evidence = append(capability.Evidence, Evidence{Resource: node.ObjectRef, Field: "node.runtime", Value: fmt.Sprintf("zone=%s kubelet=%s os=%s runtime=%s instance=%s", node.Labels["topology.kubernetes.io/zone"], node.KubeletVersion, node.OSImage, node.ContainerRuntime, node.Labels["node.kubernetes.io/instance-type"]), CollectedAt: nodeAt})
		}
		r.Capabilities = append(r.Capabilities, capability)
	}
	sort.Slice(r.Capabilities, func(i, j int) bool {
		a, b := r.Capabilities[i], r.Capabilities[j]
		return a.Key+"/"+a.Scope < b.Key+"/"+b.Scope
	})
	for i := range r.Capabilities {
		sort.Slice(r.Capabilities[i].Evidence, func(a, b int) bool {
			x, y := r.Capabilities[i].Evidence[a], r.Capabilities[i].Evidence[b]
			return x.Resource.Kind+"/"+x.Resource.Namespace+"/"+x.Resource.Name < y.Resource.Kind+"/"+y.Resource.Namespace+"/"+y.Resource.Name
		})
	}
	r.Summary = fmt.Sprintf("%d Ingress classes, %d Gateway classes, %d StorageClasses, %d custom API definitions, %d EKS insights, %d EKS managed add-ons and %d EKS managed nodegroups observed. Conclusions describe configuration and collected evidence; runtime behavior is not verified.", len(k.IngressClasses), len(k.GatewayClasses), len(k.StorageClasses), len(k.CustomResourceDefinitions), len(s.EKS.Insights), len(s.EKS.Addons), len(s.EKS.Nodegroups))
	unknown, incomplete, rwx := 0, 0, 0
	for _, c := range r.Capabilities {
		if c.Assessment == "unknown" {
			unknown++
		}
		if c.Coverage != "complete" {
			incomplete++
		}
		if c.Key == "storage.rwx" && c.Basis == "observed" {
			rwx++
		}
	}
	r.Summary += fmt.Sprintf(" RWX bindings observed in %d StorageClasses; %d assessments remain unknown and %d have incomplete coverage.", rwx, unknown, incomplete)
	return r
}

type addonCompatibility struct {
	TargetKubernetes   []string
	CompatibleVersions []string
	Statuses           []string
}

func eksInsightCapability(insight inventory.EKSInsight, coverage []inventory.CoverageItem) Capability {
	cov, at := coverageArea(coverage, "eks", "Insights")
	assessment, summary := insightAssessment(insight)
	ref := inventory.ObjectRef{Kind: "EKSInsight", Name: insightScope(insight)}
	capability := Capability{
		Key:            "eks.insight",
		Scope:          insightScope(insight),
		Implementation: insight.KubernetesVersion,
		Assessment:     assessment,
		Basis:          "aws-reported",
		Summary:        summary,
		Constraints:    []string{"EKS insights are AWS-reported point-in-time readiness signals; verify workload behavior and remediation before upgrading."},
		Collection:     collectionFor(coverage, "eks", "Insights"),
		Coverage:       cov,
		Freshness:      "snapshot",
		RuleID:         "eks.insight/v1",
	}
	capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "status/category/target", Value: strings.Join([]string{displayInsightValue(insight.Status), displayInsightValue(insight.Category), displayInsightValue(insight.KubernetesVersion)}, "/"), CollectedAt: at})
	if insight.Reason != "" {
		capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "reason", Value: insight.Reason, CollectedAt: at})
	}
	if insight.Recommendation != "" {
		capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "recommendation", Value: strings.Join(strings.Fields(insight.Recommendation), " "), CollectedAt: at})
	}
	for _, detail := range insight.DeprecationDetails {
		capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "deprecation.detail", Value: deprecationDetailValue(detail), CollectedAt: at})
	}
	for _, resource := range insight.Resources {
		capability.Evidence = append(capability.Evidence, Evidence{Resource: insightResourceRef(resource), Field: "insight.resource", Value: insightResourceValue(resource), CollectedAt: at})
	}
	for _, item := range additionalInfoValues(insight.AdditionalInfo) {
		capability.Evidence = append(capability.Evidence, Evidence{Resource: ref, Field: "additionalInfo", Value: item, CollectedAt: at})
	}
	return capability
}

func insightAssessment(insight inventory.EKSInsight) (string, string) {
	status := strings.ToUpper(strings.TrimSpace(insight.Status))
	name := insightScope(insight)
	switch status {
	case "PASSING", "PASSED", "OK", "RESOLVED":
		return "supported", fmt.Sprintf("EKS reports insight %q as %s.", name, insight.Status)
	case "WARNING", "ERROR", "FAILING", "FAILED":
		if insight.Reason != "" {
			return "unsupported", fmt.Sprintf("EKS reports insight %q as %s: %s", name, insight.Status, insight.Reason)
		}
		return "unsupported", fmt.Sprintf("EKS reports insight %q as %s.", name, insight.Status)
	default:
		return "unknown", fmt.Sprintf("EKS insight %q has unknown status evidence.", name)
	}
}

func insightScope(insight inventory.EKSInsight) string {
	parts := nonEmpty(insight.Category, insight.Name, insight.KubernetesVersion)
	if len(parts) == 0 {
		return "cluster"
	}
	return strings.Join(parts, "/")
}

func nonEmpty(values ...string) []string {
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func displayInsightValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func deprecationDetailValue(detail inventory.DeprecationDetail) string {
	parts := []string{}
	if detail.Usage != "" {
		parts = append(parts, "usage="+detail.Usage)
	}
	if detail.ReplacedWith != "" {
		parts = append(parts, "replacedWith="+detail.ReplacedWith)
	}
	if detail.StartServingReplacementVersion != "" {
		parts = append(parts, "replacementStarts="+detail.StartServingReplacementVersion)
	}
	if detail.StopServingVersion != "" {
		parts = append(parts, "stopsServing="+detail.StopServingVersion)
	}
	if len(detail.UserAgents) > 0 {
		agents := append([]string(nil), detail.UserAgents...)
		sort.Strings(agents)
		parts = append(parts, "userAgents="+strings.Join(agents, ","))
	}
	return strings.Join(parts, " ")
}

func insightResourceRef(resource inventory.EKSInsightResource) inventory.ObjectRef {
	name := resource.KubernetesResourceURI
	if name == "" {
		name = resource.ARN
	}
	if name == "" {
		name = "unknown"
	}
	return inventory.ObjectRef{Kind: "EKSInsightResource", Name: name}
}

func insightResourceValue(resource inventory.EKSInsightResource) string {
	parts := []string{}
	if resource.Status != "" {
		parts = append(parts, "status="+resource.Status)
	}
	if resource.Reason != "" {
		parts = append(parts, "reason="+resource.Reason)
	}
	if resource.ARN != "" {
		parts = append(parts, "arn="+resource.ARN)
	}
	if resource.KubernetesResourceURI != "" {
		parts = append(parts, "uri="+resource.KubernetesResourceURI)
	}
	return strings.Join(parts, " ")
}

func additionalInfoValues(items map[string]string) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+items[key])
	}
	return out
}

func sortedEKSInsights(items []inventory.EKSInsight) []inventory.EKSInsight {
	out := append([]inventory.EKSInsight(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Category, out[i].Name, out[i].KubernetesVersion, out[i].ID}, "\x00")
		right := strings.Join([]string{out[j].Category, out[j].Name, out[j].KubernetesVersion, out[j].ID}, "\x00")
		return left < right
	})
	return out
}

func addonCompatibilityByName(insights []inventory.EKSInsight) map[string]addonCompatibility {
	out := map[string]addonCompatibility{}
	for _, insight := range insights {
		for _, item := range insight.AddonCompatibility {
			if item.Name == "" {
				continue
			}
			current := out[item.Name]
			current.TargetKubernetes = appendUnique(current.TargetKubernetes, insight.KubernetesVersion)
			current.CompatibleVersions = appendUnique(current.CompatibleVersions, item.CompatibleVersions...)
			current.Statuses = appendUnique(current.Statuses, insight.Status)
			out[item.Name] = current
		}
	}
	for name, current := range out {
		sort.Strings(current.TargetKubernetes)
		sort.Strings(current.CompatibleVersions)
		sort.Strings(current.Statuses)
		out[name] = current
	}
	return out
}

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			seen[value] = struct{}{}
		}
	}
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		values = append(values, value)
		seen[value] = struct{}{}
	}
	return values
}

type unmanagedNodeGroup struct {
	Scope   string
	Summary string
	Nodes   []inventory.Node
}

func sortedNodegroups(items []inventory.Nodegroup) []inventory.Nodegroup {
	out := append([]inventory.Nodegroup(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedNodes(items []inventory.Node) []inventory.Node {
	out := append([]inventory.Node(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func nodesForNodegroup(nodes []inventory.Node, name string) []inventory.Node {
	var out []inventory.Node
	for _, node := range sortedNodes(nodes) {
		if node.Labels["eks.amazonaws.com/nodegroup"] == name {
			out = append(out, node)
		}
	}
	return out
}

func unmanagedNodeGroups(nodes []inventory.Node, managed map[string]inventory.Nodegroup) []unmanagedNodeGroup {
	groups := map[string]*unmanagedNodeGroup{}
	for _, node := range sortedNodes(nodes) {
		scope, summary := unmanagedNodeScope(node, managed)
		if scope == "" {
			continue
		}
		group := groups[scope]
		if group == nil {
			group = &unmanagedNodeGroup{Scope: scope, Summary: summary}
			groups[scope] = group
		}
		group.Nodes = append(group.Nodes, node)
	}
	out := make([]unmanagedNodeGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, *group)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scope < out[j].Scope })
	return out
}

func unmanagedNodeScope(node inventory.Node, managed map[string]inventory.Nodegroup) (string, string) {
	if name := strings.TrimSpace(node.Labels["eks.amazonaws.com/nodegroup"]); name != "" {
		if _, ok := managed[name]; ok {
			return "", ""
		}
		return name, "Node has a nodegroup label, but no matching EKS managed nodegroup was collected."
	}
	if nodepool := strings.TrimSpace(node.Labels["karpenter.sh/nodepool"]); nodepool != "" {
		return "karpenter/" + nodepool, "Karpenter nodepool observed; EKS managed nodegroup readiness is unknown."
	}
	if provisioner := strings.TrimSpace(node.Labels["karpenter.sh/provisioner-name"]); provisioner != "" {
		return "karpenter/" + provisioner, "Karpenter provisioner observed; EKS managed nodegroup readiness is unknown."
	}
	if computeType := strings.TrimSpace(node.Labels["eks.amazonaws.com/compute-type"]); computeType != "" {
		return computeType, "EKS compute type observed without managed nodegroup evidence."
	}
	return "self-managed/unknown", "Node has no EKS managed nodegroup evidence; self-managed readiness is unknown."
}

func launchTemplateValue(nodegroup inventory.Nodegroup) string {
	parts := []string{}
	if nodegroup.LaunchTemplateName != "" {
		parts = append(parts, nodegroup.LaunchTemplateName)
	} else if nodegroup.LaunchTemplateID != "" {
		parts = append(parts, nodegroup.LaunchTemplateID)
	}
	if nodegroup.LaunchTemplateVersion != "" {
		parts = append(parts, "version="+nodegroup.LaunchTemplateVersion)
	}
	return strings.Join(parts, " ")
}

func coverageArea(items []inventory.CoverageItem, area, resource string) (string, time.Time) {
	state := "partial"
	var at time.Time
	for _, item := range items {
		if item.Area != area || item.Resource != resource {
			continue
		}
		state = item.Status
		at = item.CollectedAt
		break
	}
	return state, at
}

func combinedCoverage(items []inventory.CoverageItem, refs [][2]string) string {
	state := "complete"
	for _, ref := range refs {
		found := false
		for _, item := range items {
			if item.Area != ref[0] || item.Resource != ref[1] {
				continue
			}
			found = true
			if item.Status != "complete" {
				state = "partial"
			}
		}
		if !found {
			state = "partial"
		}
	}
	return state
}

func nodegroupCollection(items []inventory.CoverageItem) []inventory.CoverageItem {
	out := collectionFor(items, "eks", "Nodegroups")
	out = append(out, collectionFor(items, "kubernetes", "Nodes")...)
	return out
}

func collectionFor(items []inventory.CoverageItem, area string, resources ...string) []inventory.CoverageItem {
	var out []inventory.CoverageItem
	for _, resource := range resources {
		found := false
		for _, item := range items {
			if item.Area == area && item.Resource == resource {
				out = append(out, item)
				found = true
			}
		}
		if !found {
			out = append(out, inventory.CoverageItem{Area: area, Resource: resource, Status: "unavailable", Reason: "Collection coverage not recorded in this snapshot."})
		}
	}
	return out
}

func has(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// SetFreshness applies the Kubernetes source state, independently of data revision.
func (r *Report) SetFreshness(state string) {
	for i := range r.Capabilities {
		r.Capabilities[i].Freshness = state
	}
}

func expansionValue(value *bool) string {
	if value == nil {
		return "not recorded"
	}
	return fmt.Sprint(*value)
}
