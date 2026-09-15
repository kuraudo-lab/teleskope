package hub

import (
	"sort"
	"strings"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

const maxSearchResults = 100

// SearchResponse is the hub-wide resource search payload.
type SearchResponse struct {
	Query     string         `json:"query,omitempty"`
	Kind      string         `json:"kind,omitempty"`
	Namespace string         `json:"namespace,omitempty"`
	Image     string         `json:"image,omitempty"`
	Total     int            `json:"total"`
	Truncated bool           `json:"truncated,omitempty"`
	Results   []SearchResult `json:"results"`
}

// SearchResult is one resource-level match from a stored cluster envelope.
type SearchResult struct {
	Cluster    Cluster `json:"cluster"`
	ClusterURL string  `json:"clusterUrl"`
	Kind       string  `json:"kind"`
	Namespace  string  `json:"namespace,omitempty"`
	Name       string  `json:"name"`
	Image      string  `json:"image,omitempty"`
	Detail     string  `json:"detail,omitempty"`
}

type searchFilter struct {
	Query     string
	Kind      string
	Namespace string
	Image     string
}

// Search scans the latest stored envelopes without contacting any cluster APIs.
func (s *Store) Search(query, kind, namespace, image string) SearchResponse {
	filter := searchFilter{
		Query:     strings.ToLower(strings.TrimSpace(query)),
		Kind:      strings.ToLower(strings.TrimSpace(kind)),
		Namespace: strings.ToLower(strings.TrimSpace(namespace)),
		Image:     strings.ToLower(strings.TrimSpace(image)),
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []SearchResult
	for _, env := range s.latest {
		results = append(results, searchEnvelope(env, filter)...)
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		for _, less := range []struct{ left, right string }{
			{a.Cluster.Name, b.Cluster.Name},
			{a.Kind, b.Kind},
			{a.Namespace, b.Namespace},
			{a.Name, b.Name},
			{a.Image, b.Image},
		} {
			if less.left != less.right {
				return less.left < less.right
			}
		}
		return false
	})
	total := len(results)
	truncated := total > maxSearchResults
	if truncated {
		results = results[:maxSearchResults]
	}
	return SearchResponse{
		Query:     strings.TrimSpace(query),
		Kind:      strings.TrimSpace(kind),
		Namespace: strings.TrimSpace(namespace),
		Image:     strings.TrimSpace(image),
		Total:     total,
		Truncated: truncated,
		Results:   results,
	}
}

func searchEnvelope(env Envelope, filter searchFilter) []SearchResult {
	if env.Snapshot == nil {
		return nil
	}
	var results []SearchResult
	add := func(kind, namespace, name, image, detail string) {
		result := SearchResult{
			Cluster:    env.Cluster,
			ClusterURL: "/cluster?id=" + urlQueryEscape(env.Cluster.ID),
			Kind:       kind,
			Namespace:  namespace,
			Name:       name,
			Image:      image,
			Detail:     detail,
		}
		if resultMatches(result, filter) {
			results = append(results, result)
		}
	}
	ref := func(ref inventory.ObjectRef, fallbackKind string) (string, string, string) {
		kind := firstNonEmpty(ref.Kind, fallbackKind)
		return kind, ref.Namespace, ref.Name
	}
	k := env.Snapshot.Kubernetes
	for _, ns := range k.Namespaces {
		kind, namespace, name := ref(ns.ObjectRef, "Namespace")
		add(kind, namespace, name, "", ns.Phase)
	}
	for _, workload := range k.Workloads {
		kind, namespace, name := ref(workload.ObjectRef, "Workload")
		add(kind, namespace, name, workloadImages(workload), workload.ServiceAccountName)
	}
	for _, image := range k.RunningImages {
		add("Image", strings.Join(image.Namespaces, ","), image.Image, image.Image, imageDetail(image))
	}
	for _, service := range k.Services {
		kind, namespace, name := ref(service.ObjectRef, "Service")
		add(kind, namespace, name, "", service.Type)
	}
	for _, class := range k.GatewayClasses {
		kind, namespace, name := ref(class.ObjectRef, "GatewayClass")
		add(kind, namespace, name, "", class.ControllerName)
	}
	for _, gateway := range k.Gateways {
		kind, namespace, name := ref(gateway.ObjectRef, "Gateway")
		add(kind, namespace, name, "", gateway.ClassName)
	}
	for _, route := range k.GatewayRoutes {
		kind, namespace, name := ref(route.ObjectRef, "GatewayRoute")
		add(kind, namespace, name, "", strings.Join(route.Hostnames, ","))
	}
	for _, claim := range k.PersistentVolumeClaims {
		kind, namespace, name := ref(claim.ObjectRef, "PersistentVolumeClaim")
		add(kind, namespace, name, "", firstNonEmpty(claim.StorageClassName, claim.VolumeName, claim.Phase))
	}
	for _, role := range iamRoles(env.Snapshot) {
		add("IAMRole", role.Namespace, role.Name, "", role.Detail)
	}
	return results
}

func resultMatches(result SearchResult, filter searchFilter) bool {
	if filter.Kind != "" && !strings.Contains(strings.ToLower(result.Kind), filter.Kind) {
		return false
	}
	if filter.Namespace != "" && !strings.Contains(strings.ToLower(result.Namespace), filter.Namespace) {
		return false
	}
	if filter.Image != "" && !strings.Contains(strings.ToLower(result.Image), filter.Image) {
		return false
	}
	if filter.Query == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		result.Cluster.ID,
		result.Cluster.Name,
		result.Cluster.Provider,
		result.Cluster.Region,
		result.Kind,
		result.Namespace,
		result.Name,
		result.Image,
		result.Detail,
	}, " "))
	return strings.Contains(haystack, filter.Query)
}

func workloadImages(workload inventory.Workload) string {
	var images []string
	for _, container := range append(append([]inventory.Container{}, workload.Containers...), workload.InitContainers...) {
		if container.Image != "" {
			images = append(images, container.Image)
		}
	}
	return strings.Join(images, ",")
}

func imageDetail(image inventory.RunningImage) string {
	parts := []string{}
	if image.PodCount > 0 {
		parts = append(parts, "pods="+itoa(image.PodCount))
	}
	if image.ContainerCount > 0 {
		parts = append(parts, "containers="+itoa(image.ContainerCount))
	}
	return strings.Join(parts, " ")
}

type iamRole struct {
	Namespace string
	Name      string
	Detail    string
}

func iamRoles(snapshot *inventory.Snapshot) []iamRole {
	var roles []iamRole
	for _, addon := range snapshot.EKS.Addons {
		if addon.ServiceAccountRoleARN != "" {
			roles = append(roles, iamRole{Name: addon.ServiceAccountRoleARN, Detail: "EKS add-on " + addon.Name})
		}
	}
	for _, nodegroup := range snapshot.EKS.Nodegroups {
		if nodegroup.NodeRoleARN != "" {
			roles = append(roles, iamRole{Name: nodegroup.NodeRoleARN, Detail: "EKS nodegroup " + nodegroup.Name})
		}
	}
	for _, assoc := range snapshot.EKS.PodIdentityAssociations {
		if assoc.RoleARN != "" {
			roles = append(roles, iamRole{Namespace: assoc.Namespace, Name: assoc.RoleARN, Detail: "Pod Identity " + assoc.ServiceAccount})
		}
		if assoc.TargetRoleARN != "" {
			roles = append(roles, iamRole{Namespace: assoc.Namespace, Name: assoc.TargetRoleARN, Detail: "Pod Identity target " + assoc.ServiceAccount})
		}
	}
	for _, account := range snapshot.Kubernetes.ServiceAccounts {
		for key, value := range account.Annotations {
			if strings.Contains(strings.ToLower(key), "role") && strings.Contains(value, ":role/") {
				roles = append(roles, iamRole{Namespace: account.Namespace, Name: value, Detail: "ServiceAccount " + account.Name})
			}
		}
	}
	return roles
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
