package topology

import (
	"sort"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Project builds a full deterministic graph revision from one inventory snapshot.
func Project(snapshot *inventory.Snapshot, revision uint64) Graph {
	return ProjectForCluster(snapshot, revision, clusterIdentity(snapshot))
}

// ProjectForCluster builds a graph using an explicit transport cluster identity.
func ProjectForCluster(snapshot *inventory.Snapshot, revision uint64, cluster string) Graph {
	cluster = firstNonEmpty(cluster, clusterIdentity(snapshot))
	provider := "kubernetes"
	if snapshot != nil && snapshot.EKS.Cluster.Name != "" {
		provider = "eks"
	}
	generatedAt := time.Time{}
	if snapshot != nil {
		generatedAt = snapshot.CollectedAt
	}
	if generatedAt.IsZero() {
		generatedAt = time.Unix(0, 0).UTC()
	}
	g := Graph{SchemaVersion: SchemaVersion, ClusterID: cluster, Revision: revision, GeneratedAt: generatedAt}
	scope := Scope{Provider: provider, Cluster: cluster}
	if snapshot == nil {
		return g
	}
	scope.Account, scope.Region = snapshot.AWS.AccountID, snapshot.AWS.Region
	clusterID := NodeID(provider, cluster, "cluster", "", cluster)
	g.Nodes = append(g.Nodes, Node{ID: clusterID, Kind: "cluster", Name: cluster, Identity: cluster, Scope: scope})

	findings := advisor.Analyze(snapshot)
	uidIdentities := collectedUIDIdentities(snapshot, findings)
	nodeIDs := map[string]string{"cluster": clusterID}
	addNamespace := func(namespace string) string {
		if namespace == "" {
			return clusterID
		}
		key := "namespace/" + namespace
		if id := nodeIDs[key]; id != "" {
			return id
		}
		id := NodeID("kubernetes", cluster, "namespace", "", namespace)
		g.Nodes = append(g.Nodes, Node{ID: id, Kind: "namespace", Name: namespace, Identity: namespace, Scope: Scope{Provider: "kubernetes", Cluster: cluster, Account: scope.Account, Region: scope.Region}, ParentID: clusterID})
		nodeIDs[key] = id
		g.addEdge(EdgeOwnership, clusterID, id, "kubernetes", "contains", declaredEvidence("kubernetes", "namespace belongs to cluster"))
		return id
	}
	for _, namespace := range snapshot.Kubernetes.Namespaces {
		addNamespace(namespace.Name)
	}
	addObject := func(ref inventory.ObjectRef, kind string, metadata map[string]string) string {
		if ref.Name == "" {
			return ""
		}
		if kind == "" {
			kind = ref.Kind
		}
		nameKey := ref.Name
		if ref.UID != "" && len(uidIdentities[ref.UID]) == 1 {
			nameKey = ref.UID
		}
		if kind == "workload" && ref.Kind != "" {
			nameKey = strings.ToLower(ref.Kind) + ":" + nameKey
		}
		id := NodeID("kubernetes", cluster, kind, ref.Namespace, nameKey)
		if nodeIDs["id/"+id] != "" {
			return id
		}
		parent := clusterID
		if ref.Namespace != "" {
			parent = addNamespace(ref.Namespace)
		}
		g.Nodes = append(g.Nodes, Node{ID: id, Kind: strings.ToLower(kind), Name: ref.Name, Identity: nameKey, APIVersion: ref.APIVersion, UID: ref.UID, Scope: Scope{Provider: "kubernetes", Cluster: cluster, Namespace: ref.Namespace, Account: scope.Account, Region: scope.Region}, ParentID: parent, Metadata: metadata})
		nodeIDs["id/"+id] = id
		if ref.UID != "" && len(uidIdentities[ref.UID]) == 1 {
			nodeIDs["uid/"+ref.UID] = id
		}
		nodeIDs[objectKey(ref, firstNonEmpty(ref.Kind, kind))] = id
		if kind != "workload" {
			nodeIDs[objectKey(ref, kind)] = id
		}
		return id
	}
	addPseudo := func(pseudoProvider, kind, identity, namespace, name string) string {
		id := PseudoNodeID(pseudoProvider, cluster, kind, namespace, identity)
		if nodeIDs["id/"+id] != "" {
			return id
		}
		parent := clusterID
		if namespace != "" {
			parent = addNamespace(namespace)
		}
		g.Nodes = append(g.Nodes, Node{ID: id, Kind: "pseudo-" + kind, Name: name, Identity: identity, Scope: Scope{Provider: pseudoProvider, Cluster: cluster, Namespace: namespace, Account: scope.Account, Region: scope.Region}, ParentID: parent, PseudoKind: kind})
		nodeIDs["id/"+id] = id
		return id
	}
	if snapshot.EKS.Cluster.VPC.EndpointPublicAccess {
		internetID := addPseudo("external", "internet", "internet", "", "Internet")
		evidence := declaredEvidence("eks", "EKS cluster endpointPublicAccess is enabled subject to declared public access CIDRs")
		evidence.Fields = []FieldReference{{Resource: clusterID, Field: "eks.cluster.resourcesVpcConfig.publicAccessCidrs", Value: strings.Join(snapshot.EKS.Cluster.VPC.PublicAccessCIDRs, ",")}}
		g.addEdgeWithMetadata(EdgeDeclared, clusterID, internetID, "eks", "public-api-access", compactMetadata(map[string]string{"publicAccessCidrs": strings.Join(snapshot.EKS.Cluster.VPC.PublicAccessCIDRs, ",")}), evidence)
	}

	for _, node := range snapshot.Kubernetes.Nodes {
		id := addObject(node.ObjectRef, "node", compactMetadata(map[string]string{"providerId": node.ProviderID, "ready": node.Ready, "nodegroup": node.Labels["eks.amazonaws.com/nodegroup"]}))
		g.setNodeMetrics(id, resourceMetrics(node.Capacity, node.Allocatable, generatedAt))
	}
	for _, workload := range snapshot.Kubernetes.Workloads {
		id := addObject(workload.ObjectRef, "workload", compactMetadata(map[string]string{"resourceKind": workload.Kind}))
		g.setNodeMetrics(id, declaredContainerMetrics(workload.Containers, generatedAt))
	}
	for _, pod := range snapshot.Kubernetes.Pods {
		id := addObject(pod.ObjectRef, "pod", compactMetadata(map[string]string{"phase": pod.Phase, "podIp": pod.PodIP}))
		g.setNodeMetrics(id, declaredContainerMetrics(pod.Containers, generatedAt))
	}
	for _, service := range snapshot.Kubernetes.Services {
		addObject(service.ObjectRef, "service", compactMetadata(map[string]string{"type": service.Type, "clusterIp": service.ClusterIP}))
	}
	for _, endpointSlice := range snapshot.Kubernetes.EndpointSlices {
		addObject(endpointSlice.ObjectRef, "endpoint-slice", compactMetadata(map[string]string{"service": endpointSlice.ServiceName, "addressType": endpointSlice.AddressType}))
	}

	for _, workload := range snapshot.Kubernetes.Workloads {
		child := findObjectID(nodeIDs, workload.ObjectRef)
		for _, owner := range workload.OwnerReferences {
			if parent := findObjectID(nodeIDs, owner); parent != "" {
				g.addEdge(EdgeOwnership, parent, child, "kubernetes", "owns", declaredEvidence("kubernetes", "metadata.ownerReferences"))
			} else {
				identity := objectKey(owner, firstNonEmpty(owner.Kind, "object"))
				parent := addPseudo("kubernetes", "unresolved", identity, owner.Namespace, firstNonEmpty(owner.Kind+"/"+owner.Name, owner.Name))
				g.addEdge(EdgeDeclared, parent, child, "kubernetes", "unresolved-owner", declaredEvidence("kubernetes", "metadata.ownerReferences target was not collected"))
			}
		}
	}
	for _, pod := range snapshot.Kubernetes.Pods {
		podID := nodeIDs[objectKey(pod.ObjectRef, "pod")]
		for _, owner := range pod.OwnerReferences {
			if parent := findObjectID(nodeIDs, owner); parent != "" {
				g.addEdge(EdgeOwnership, parent, podID, "kubernetes", "owns", declaredEvidence("kubernetes", "metadata.ownerReferences"))
			} else {
				identity := objectKey(owner, firstNonEmpty(owner.Kind, "object"))
				parent := addPseudo("kubernetes", "unresolved", identity, owner.Namespace, firstNonEmpty(owner.Kind+"/"+owner.Name, owner.Name))
				g.addEdge(EdgeDeclared, parent, podID, "kubernetes", "unresolved-owner", declaredEvidence("kubernetes", "metadata.ownerReferences target was not collected"))
			}
		}
		if nodeID := nodeIDs[objectKey(inventory.ObjectRef{Name: pod.NodeName}, "node")]; nodeID != "" {
			g.addEdge(EdgeResolved, podID, nodeID, "kubernetes", "scheduled-to", resolvedEvidence("kubernetes", "pod.spec.nodeName resolved to collected Node"))
		}
	}
	for _, service := range snapshot.Kubernetes.Services {
		serviceID := nodeIDs[objectKey(service.ObjectRef, "service")]
		for _, workload := range snapshot.Kubernetes.Workloads {
			if workload.Namespace == service.Namespace && equalSelector(service.Selector, workload.Selector) {
				g.addEdge(EdgeDeclared, serviceID, findObjectID(nodeIDs, workload.ObjectRef), "kubernetes", "selects", declaredEvidence("kubernetes", "service selector and workload selector are equal"))
			}
		}
		for _, externalIP := range service.ExternalIPs {
			target := addPseudo("external", "external-endpoint", externalIP, service.Namespace, externalIP)
			g.addEdge(EdgeDeclared, serviceID, target, "kubernetes", "exposes", declaredEvidence("kubernetes", "service.spec.externalIPs"))
		}
	}
	for _, endpointSlice := range snapshot.Kubernetes.EndpointSlices {
		serviceRef := inventory.ObjectRef{Namespace: endpointSlice.Namespace, Name: endpointSlice.ServiceName}
		serviceID := nodeIDs[objectKey(serviceRef, "service")]
		endpointID := nodeIDs[objectKey(endpointSlice.ObjectRef, "endpoint-slice")]
		if serviceID != "" {
			g.addEdge(EdgeResolved, serviceID, endpointID, "kubernetes", "has-endpoints", resolvedEvidence("kubernetes", "EndpointSlice service label resolved to collected Service"))
		}
		for _, endpoint := range endpointSlice.Endpoints {
			if target := findObjectID(nodeIDs, endpoint.TargetRef); target != "" {
				g.addEdge(EdgeResolved, endpointID, target, "kubernetes", "targets", resolvedEvidence("kubernetes", "EndpointSlice targetRef resolved to collected object"))
				continue
			}
			if endpoint.TargetRef.Name != "" {
				identity := objectKey(endpoint.TargetRef, firstNonEmpty(endpoint.TargetRef.Kind, "object"))
				target := addPseudo("kubernetes", "unresolved", identity, endpointSlice.Namespace, firstNonEmpty(endpoint.TargetRef.Kind+"/"+endpoint.TargetRef.Name, endpoint.TargetRef.Name))
				g.addEdge(EdgeDeclared, endpointID, target, "kubernetes", "unresolved-target", declaredEvidence("kubernetes", "EndpointSlice targetRef was not collected"))
				continue
			}
			for _, address := range endpoint.Addresses {
				target := addPseudo("external", "external-endpoint", address, endpointSlice.Namespace, address)
				g.addEdge(EdgeResolved, endpointID, target, "kubernetes", "addresses", resolvedEvidence("kubernetes", "EndpointSlice contains an endpoint address without targetRef"))
			}
		}
	}

	for _, group := range snapshot.EKS.Nodegroups {
		id := NodeID("eks", cluster, "nodegroup", "", firstNonEmpty(group.ARN, group.Name))
		g.Nodes = append(g.Nodes, Node{ID: id, Kind: "nodegroup", Name: group.Name, Identity: firstNonEmpty(group.ARN, group.Name), Scope: Scope{Provider: "eks", Cluster: cluster, Account: scope.Account, Region: scope.Region}, ParentID: clusterID, Metadata: compactMetadata(map[string]string{"arn": group.ARN, "status": group.Status})})
		nodeIDs["nodegroup/"+group.Name] = id
		nodeIDs[objectKey(inventory.ObjectRef{Kind: "Nodegroup", Name: group.Name}, "Nodegroup")] = id
		g.addEdge(EdgeOwnership, clusterID, id, "eks", "contains", declaredEvidence("eks", "EKS managed node group belongs to cluster"))
	}
	for _, addon := range snapshot.EKS.Addons {
		identity := firstNonEmpty(addon.ARN, addon.Name)
		id := NodeID("eks", cluster, "addon", addon.Namespace, identity)
		g.Nodes = append(g.Nodes, Node{ID: id, Kind: "addon", Name: addon.Name, Identity: identity, Scope: Scope{Provider: "eks", Cluster: cluster, Namespace: addon.Namespace, Account: scope.Account, Region: scope.Region}, ParentID: clusterID, Metadata: compactMetadata(map[string]string{"arn": addon.ARN, "status": addon.Status, "version": addon.Version})})
		nodeIDs[objectKey(inventory.ObjectRef{Kind: "Addon", Name: addon.Name}, "Addon")] = id
		g.addEdge(EdgeOwnership, clusterID, id, "eks", "contains", declaredEvidence("eks", "EKS managed add-on belongs to cluster"))
	}
	for _, insight := range snapshot.EKS.Insights {
		name := topologyInsightScope(insight)
		identity := firstNonEmpty(insight.ID, name)
		id := NodeID("eks", cluster, "eksinsight", "", identity)
		g.Nodes = append(g.Nodes, Node{ID: id, Kind: "eksinsight", Name: name, Identity: identity, Scope: Scope{Provider: "eks", Cluster: cluster, Account: scope.Account, Region: scope.Region}, ParentID: clusterID, Metadata: compactMetadata(map[string]string{"status": insight.Status, "category": insight.Category, "targetVersion": insight.KubernetesVersion})})
		nodeIDs[objectKey(inventory.ObjectRef{Kind: "EKSInsight", Name: name}, "EKSInsight")] = id
		g.addEdge(EdgeOwnership, clusterID, id, "eks", "contains", declaredEvidence("eks", "EKS insight belongs to cluster"))
	}
	for _, node := range snapshot.Kubernetes.Nodes {
		groupID := nodeIDs["nodegroup/"+node.Labels["eks.amazonaws.com/nodegroup"]]
		nodeID := nodeIDs[objectKey(node.ObjectRef, "node")]
		if groupID != "" && nodeID != "" {
			g.addEdge(EdgeResolved, groupID, nodeID, "kubernetes", "manages", resolvedEvidence("kubernetes", "node label resolved to collected EKS node group"))
		}
	}
	for _, instance := range snapshot.EKS.Infrastructure.Instances {
		id := NodeID("aws", cluster, "ec2-instance", "", instance.InstanceID)
		g.Nodes = append(g.Nodes, Node{ID: id, Kind: "ec2-instance", Name: instance.InstanceID, Identity: instance.InstanceID, Scope: Scope{Provider: "aws", Cluster: cluster, Account: scope.Account, Region: scope.Region}, ParentID: clusterID, Metadata: compactMetadata(map[string]string{"state": instance.State, "privateIp": instance.PrivateIP, "nodegroup": instance.NodegroupName})})
		if nodeID := nodeIDs[objectKey(inventory.ObjectRef{Name: instance.KubernetesNodeName}, "node")]; nodeID != "" {
			g.addEdge(EdgeResolved, nodeID, id, "aws", "backed-by", resolvedEvidence("aws", "EC2 instance association resolved to collected Kubernetes Node"))
		}
	}

	for _, capability := range findings.Capabilities {
		for _, evidence := range capability.Evidence {
			if evidence.Resource.Name != "" && findObjectID(nodeIDs, evidence.Resource) == "" {
				addObject(evidence.Resource, strings.ToLower(firstNonEmpty(evidence.Resource.Kind, "resource")), nil)
			}
		}
	}
	coverageByIdentity := map[string]Coverage{}
	for _, item := range snapshot.Coverage {
		at := item.CollectedAt
		status := item.Status
		if status != "complete" && status != "partial" && status != "unavailable" {
			status = "unknown"
		}
		candidate := Coverage{Provider: firstNonEmpty(item.Area, provider), Capability: item.Resource, Status: status, Reason: item.Reason, ObservedAt: timePtr(at)}
		key := candidate.Provider + "\x00" + candidate.Capability + "\x00" + candidate.Namespace
		if current, exists := coverageByIdentity[key]; exists {
			candidate = mergeCoverage(current, candidate)
		}
		coverageByIdentity[key] = candidate
	}
	for _, coverage := range coverageByIdentity {
		g.Coverage = append(g.Coverage, coverage)
	}
	g.Coverage = append(g.Coverage, Coverage{Provider: "external", Capability: "runtime-connections", Status: "unavailable", Reason: "no external connection evidence provider configured"})
	attachFindings(&g, nodeIDs, findings)
	g.Normalize()
	return g
}

func collectedUIDIdentities(snapshot *inventory.Snapshot, findings advisor.Report) map[string]map[string]struct{} {
	identities := map[string]map[string]struct{}{}
	add := func(ref inventory.ObjectRef, fallbackKind string) {
		if ref.UID == "" || ref.Name == "" {
			return
		}
		identity := objectKey(ref, firstNonEmpty(ref.Kind, fallbackKind))
		if identities[ref.UID] == nil {
			identities[ref.UID] = map[string]struct{}{}
		}
		identities[ref.UID][identity] = struct{}{}
	}
	for _, node := range snapshot.Kubernetes.Nodes {
		add(node.ObjectRef, "node")
	}
	for _, workload := range snapshot.Kubernetes.Workloads {
		add(workload.ObjectRef, "workload")
	}
	for _, pod := range snapshot.Kubernetes.Pods {
		add(pod.ObjectRef, "pod")
	}
	for _, service := range snapshot.Kubernetes.Services {
		add(service.ObjectRef, "service")
	}
	for _, endpointSlice := range snapshot.Kubernetes.EndpointSlices {
		add(endpointSlice.ObjectRef, "endpoint-slice")
	}
	for _, capability := range findings.Capabilities {
		for _, evidence := range capability.Evidence {
			add(evidence.Resource, "resource")
		}
	}
	return identities
}

func (g *Graph) setNodeMetrics(id string, metrics []MetricSeries) {
	if id == "" || len(metrics) == 0 {
		return
	}
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			for _, metric := range metrics {
				replaced := false
				for j := range g.Nodes[i].Metrics {
					if g.Nodes[i].Metrics[j].Key != metric.Key {
						continue
					}
					if metricSeriesKey(metric) < metricSeriesKey(g.Nodes[i].Metrics[j]) {
						g.Nodes[i].Metrics[j] = metric
					}
					replaced = true
					break
				}
				if !replaced {
					g.Nodes[i].Metrics = append(g.Nodes[i].Metrics, metric)
				}
			}
			return
		}
	}
}

func resourceMetrics(capacity, allocatable map[string]string, at time.Time) []MetricSeries {
	var out []MetricSeries
	for _, item := range []struct {
		key, label, raw, unit string
		semantic              MetricSemantic
	}{
		{"cpu.capacity", "CPU capacity", capacity["cpu"], "mCPU", MetricCapacity},
		{"cpu.allocatable", "CPU allocatable", allocatable["cpu"], "mCPU", MetricAllocatable},
		{"memory.capacity", "Memory capacity", capacity["memory"], "bytes", MetricCapacity},
		{"memory.allocatable", "Memory allocatable", allocatable["memory"], "bytes", MetricAllocatable},
	} {
		if metric, ok := quantityMetric(item.key, item.label, item.raw, item.unit, item.semantic, "kubernetes", "snapshot", at); ok {
			out = append(out, metric)
		}
	}
	return out
}

func declaredContainerMetrics(containers []inventory.Container, at time.Time) []MetricSeries {
	values := map[string]float64{}
	for _, container := range containers {
		for _, item := range []struct{ source, key, unit string }{
			{"requests.cpu", "cpu.request", "mCPU"},
			{"limits.cpu", "cpu.limit", "mCPU"},
			{"requests.memory", "memory.request", "bytes"},
			{"limits.memory", "memory.limit", "bytes"},
		} {
			if value, ok := quantityValue(container.Resources[item.source], item.unit); ok {
				values[item.key] += value
			}
		}
	}
	labels := map[string]string{"cpu.request": "CPU request", "cpu.limit": "CPU limit", "memory.request": "Memory request", "memory.limit": "Memory limit"}
	units := map[string]string{"cpu.request": "mCPU", "cpu.limit": "mCPU", "memory.request": "bytes", "memory.limit": "bytes"}
	keys := []string{"cpu.request", "cpu.limit", "memory.request", "memory.limit"}
	var out []MetricSeries
	for _, key := range keys {
		if value, ok := values[key]; ok {
			out = append(out, pointMetric(key, labels[key], value, units[key], MetricDeclared, "kubernetes", "snapshot regular-container sum", at))
		}
	}
	return out
}

func quantityMetric(key, label, raw, unit string, semantic MetricSemantic, provider, sampling string, at time.Time) (MetricSeries, bool) {
	value, ok := quantityValue(raw, unit)
	if !ok {
		return MetricSeries{}, false
	}
	return pointMetric(key, label, value, unit, semantic, provider, sampling, at), true
}

func quantityValue(raw, unit string) (float64, bool) {
	if strings.TrimSpace(raw) == "" {
		return 0, false
	}
	quantity, err := resource.ParseQuantity(raw)
	if err != nil {
		return 0, false
	}
	if unit == "mCPU" {
		return float64(quantity.MilliValue()), true
	}
	return float64(quantity.Value()), true
}

func pointMetric(key, label string, value float64, unit string, semantic MetricSemantic, provider, sampling string, at time.Time) MetricSeries {
	start, end := at, at
	return MetricSeries{Key: key, Label: label, Provider: provider, Semantic: semantic, Unit: unit, Sampling: sampling, Freshness: "current", WindowStart: &start, WindowEnd: &end, Samples: []MetricSample{{At: at, Value: value}}}
}

func topologyInsightScope(insight inventory.EKSInsight) string {
	var parts []string
	for _, value := range []string{insight.Category, insight.Name, insight.KubernetesVersion} {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, strings.TrimSpace(value))
		}
	}
	if len(parts) == 0 {
		return "cluster"
	}
	return strings.Join(parts, "/")
}

func (g *Graph) addEdge(kind EdgeKind, source, target, provider, relation string, evidence Evidence) {
	g.addEdgeWithMetadata(kind, source, target, provider, relation, nil, evidence)
}

func (g *Graph) addEdgeWithMetadata(kind EdgeKind, source, target, provider, relation string, metadata map[string]string, evidence Evidence) {
	if source == "" || target == "" {
		return
	}
	id := EdgeID(kind, source, target, provider)
	for _, edge := range g.Edges {
		if edge.ID == id {
			return
		}
	}
	g.Edges = append(g.Edges, Edge{ID: id, Source: source, Target: target, Kind: kind, Provider: provider, Relation: relation, Directed: true, Metadata: metadata, Evidence: []Evidence{evidence}})
}

func mergeCoverage(a, b Coverage) Coverage {
	if coverageRank(b.Status) > coverageRank(a.Status) {
		a.Status = b.Status
	}
	var reasons []string
	for _, reason := range []string{a.Reason, b.Reason} {
		if reason != "" && !contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	sort.Strings(reasons)
	a.Reason = strings.Join(reasons, "; ")
	if a.ObservedAt == nil || (b.ObservedAt != nil && b.ObservedAt.Before(*a.ObservedAt)) {
		a.ObservedAt = b.ObservedAt
	}
	return a
}

func coverageRank(status string) int {
	switch status {
	case "unavailable":
		return 3
	case "partial":
		return 2
	case "unknown":
		return 1
	default:
		return 0
	}
}

func clusterIdentity(snapshot *inventory.Snapshot) string {
	if snapshot == nil {
		return "unknown"
	}
	return firstNonEmpty(snapshot.EKS.Cluster.ARN, snapshot.EKS.Cluster.Name, snapshot.Kubernetes.Context, "unknown")
}

func objectKey(ref inventory.ObjectRef, kind string) string {
	return strings.ToLower(kind) + "/" + ref.Namespace + "/" + ref.Name
}

func findObjectID(ids map[string]string, ref inventory.ObjectRef) string {
	if ref.Name == "" {
		return ""
	}
	if ref.UID != "" {
		if id := ids["uid/"+ref.UID]; id != "" {
			return id
		}
	}
	candidates := []string{strings.ToLower(ref.Kind), "pod", "service", "node", "endpoint-slice"}
	for _, kind := range candidates {
		if id := ids[objectKey(ref, kind)]; id != "" {
			return id
		}
	}
	return ""
}

func attachFindings(graph *Graph, ids map[string]string, report advisor.Report) {
	for _, capability := range report.Capabilities {
		findingRef := findingReference(capability.RuleID, capability.Scope)
		var referenced []string
		for _, evidence := range capability.Evidence {
			if id := findObjectID(ids, evidence.Resource); id != "" {
				referenced = appendUnique(referenced, id)
			}
		}
		for i := range graph.Nodes {
			if contains(referenced, graph.Nodes[i].ID) {
				graph.Nodes[i].FindingRefs = appendUnique(graph.Nodes[i].FindingRefs, findingRef)
			}
		}
		if len(referenced) < 2 {
			continue
		}
		for i := range graph.Edges {
			if contains(referenced, graph.Edges[i].Source) && contains(referenced, graph.Edges[i].Target) {
				graph.Edges[i].FindingRefs = appendUnique(graph.Edges[i].FindingRefs, findingRef)
			}
		}
	}
}

// findingReference identifies one deterministic advisor result, rather than
// only the rule that produced it. A rule can emit one result per object scope.
func findingReference(ruleID, scope string) string {
	return ruleID + "::" + scope
}

func appendUnique(values []string, value string) []string {
	if value == "" || contains(values, value) {
		return values
	}
	return append(values, value)
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func equalSelector(a, b map[string]string) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func declaredEvidence(provider, summary string) Evidence {
	return Evidence{Provider: provider, Basis: BasisDeclared, Freshness: "current", Confidence: "high", Summary: summary}
}

func resolvedEvidence(provider, summary string) Evidence {
	return Evidence{Provider: provider, Basis: BasisResolved, Freshness: "current", Confidence: "high", Summary: summary}
}

func compactMetadata(values map[string]string) map[string]string {
	for key, value := range values {
		if value == "" {
			delete(values, key)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copy := value
	return &copy
}
