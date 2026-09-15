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
	r.Summary = fmt.Sprintf("%d Ingress classes, %d Gateway classes, %d StorageClasses, %d custom API definitions and %d EKS managed add-ons observed. Conclusions describe configuration and collected evidence; runtime behavior is not verified.", len(k.IngressClasses), len(k.GatewayClasses), len(k.StorageClasses), len(k.CustomResourceDefinitions), len(s.EKS.Addons))
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
