// Package report writes scan artifacts for offline review.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// Options configures report artifact creation.
type Options struct {
	BaseDir string
	Target  string
	Now     time.Time
}

// Artifact describes the files produced by a report run.
type Artifact struct {
	Dir   string
	Files []string
}

// WriteDirectory creates a timestamped report directory with raw JSON files and
// a Markdown summary.
func WriteDirectory(snapshot *inventory.Snapshot, opts Options) (*Artifact, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot is nil")
	}
	baseDir := opts.BaseDir
	if baseDir == "" {
		baseDir = "."
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	target := opts.Target
	if target == "" {
		target = targetName(snapshot)
	}

	dir, err := createReportDir(baseDir, target, now)
	if err != nil {
		return nil, err
	}

	artifact := &Artifact{Dir: dir}
	if err := artifact.writeJSON("snapshot.json", snapshot); err != nil {
		return nil, err
	}
	if err := artifact.writeJSON("source.json", snapshot.Source); err != nil {
		return nil, err
	}
	if hasAWS(snapshot) {
		if err := artifact.writeJSON("aws.json", snapshot.AWS); err != nil {
			return nil, err
		}
	}
	if hasEKS(snapshot) {
		if err := artifact.writeJSON("eks.json", snapshot.EKS); err != nil {
			return nil, err
		}
	}
	if hasKubernetes(snapshot) {
		if err := artifact.writeJSON("kubernetes.json", snapshot.Kubernetes); err != nil {
			return nil, err
		}
	}
	if err := artifact.writeJSON("coverage.json", snapshot.Coverage); err != nil {
		return nil, err
	}
	if err := artifact.writeText("summary.md", markdown(snapshot, target)); err != nil {
		return nil, err
	}

	return artifact, nil
}

func (artifact *Artifact) writeJSON(name string, value any) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	return artifact.writeText(name, buf.String())
}

func (artifact *Artifact) writeText(name string, value string) error {
	path := filepath.Join(artifact.Dir, name)
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	artifact.Files = append(artifact.Files, path)
	return nil
}

func createReportDir(baseDir, target string, now time.Time) (string, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("create report base directory %s: %w", baseDir, err)
	}
	name := fmt.Sprintf("%s-%s", sanitize(target), now.Format("20060102-150405"))
	for i := 0; i < 100; i++ {
		candidate := filepath.Join(baseDir, name)
		if i > 0 {
			candidate = filepath.Join(baseDir, fmt.Sprintf("%s-%02d", name, i+1))
		}
		if err := os.Mkdir(candidate, 0o755); err == nil {
			return candidate, nil
		} else if os.IsExist(err) {
			continue
		} else {
			return "", fmt.Errorf("create report directory %s: %w", candidate, err)
		}
	}
	return "", fmt.Errorf("create report directory: exhausted unique names for %s", name)
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = unsafeName.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-_.")
	if value == "" {
		return "teleskope-scan"
	}
	return value
}

// Markdown writes the scan summary used by report directories.
func Markdown(snapshot *inventory.Snapshot) string {
	return markdown(snapshot, targetName(snapshot))
}

func markdown(snapshot *inventory.Snapshot, title string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Teleskope scan summary\n\n")
	fmt.Fprintf(&b, "- Target: `%s`\n", mdInline(title))
	fmt.Fprintf(&b, "- Collected at: `%s`\n", snapshot.CollectedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "- Schema: `%s`\n", mdInline(snapshot.SchemaVersion))
	fmt.Fprintf(&b, "- Mode: `%s`\n", mdInline(snapshot.Source.Mode))
	fmt.Fprintf(&b, "- Coverage items: `%d`\n\n", len(snapshot.Coverage))

	if hasEKS(snapshot) {
		writeEKS(&b, snapshot)
	}
	if hasKubernetes(snapshot) {
		writeKubernetes(&b, snapshot.Kubernetes)
	}
	writeCoverage(&b, snapshot.Coverage)
	return b.String()
}

func writeEKS(b *strings.Builder, snapshot *inventory.Snapshot) {
	cluster := snapshot.EKS.Cluster
	fmt.Fprintf(b, "## EKS\n\n")
	fmt.Fprintf(b, "| Field | Value |\n| --- | --- |\n")
	fmt.Fprintf(b, "| Cluster | %s |\n", mdCell(cluster.Name))
	fmt.Fprintf(b, "| Region | %s |\n", mdCell(snapshot.AWS.Region))
	fmt.Fprintf(b, "| Version | %s |\n", mdCell(cluster.Version))
	fmt.Fprintf(b, "| Platform | %s |\n", mdCell(cluster.PlatformVersion))
	fmt.Fprintf(b, "| Status | %s |\n", mdCell(cluster.Status))
	fmt.Fprintf(b, "| VPC | %s |\n", mdCell(cluster.VPC.VPCID))
	fmt.Fprintf(b, "| Endpoint | public=%t private=%t |\n", cluster.VPC.EndpointPublicAccess, cluster.VPC.EndpointPrivateAccess)
	fmt.Fprintf(b, "| Auth | %s |\n\n", mdCell(cluster.AccessConfig.AuthenticationMode))

	if len(snapshot.EKS.Addons) > 0 {
		fmt.Fprintf(b, "### Managed add-ons\n\n")
		fmt.Fprintf(b, "| Name | Version | Status | Namespace | IAM |\n| --- | --- | --- | --- | --- |\n")
		for _, addon := range snapshot.EKS.Addons {
			iam := "-"
			if addon.ServiceAccountRoleARN != "" {
				iam = "IRSA"
			}
			if len(addon.PodIdentityAssociations) > 0 {
				iam = fmt.Sprintf("%s PodIdentity=%d", iam, len(addon.PodIdentityAssociations))
			}
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", mdCell(addon.Name), mdCell(addon.Version), mdCell(addon.Status), mdCell(addon.Namespace), mdCell(iam))
		}
		fmt.Fprintln(b)
	}
}

func writeKubernetes(b *strings.Builder, kubernetes inventory.Kubernetes) {
	fmt.Fprintf(b, "## Kubernetes\n\n")
	fmt.Fprintf(b, "| Area | Count |\n| --- | ---: |\n")
	fmt.Fprintf(b, "| API resources | %d |\n", len(kubernetes.APIResources))
	fmt.Fprintf(b, "| CRDs | %d |\n", len(kubernetes.CustomResourceDefinitions))
	fmt.Fprintf(b, "| API services | %d |\n", len(kubernetes.APIServices))
	fmt.Fprintf(b, "| Namespaces | %d |\n", len(kubernetes.Namespaces))
	fmt.Fprintf(b, "| Nodes | %d |\n", len(kubernetes.Nodes))
	fmt.Fprintf(b, "| Workloads | %d |\n", len(kubernetes.Workloads))
	fmt.Fprintf(b, "| Pods | %d |\n", len(kubernetes.Pods))
	fmt.Fprintf(b, "| Running images | %d |\n", len(kubernetes.RunningImages))
	fmt.Fprintf(b, "| Running containers | %d |\n", len(kubernetes.RunningContainers))
	fmt.Fprintf(b, "| Services | %d |\n", len(kubernetes.Services))
	fmt.Fprintf(b, "| Ingresses | %d |\n", len(kubernetes.Ingresses))
	fmt.Fprintf(b, "| StorageClasses | %d |\n", len(kubernetes.StorageClasses))
	fmt.Fprintf(b, "| PVCs | %d |\n", len(kubernetes.PersistentVolumeClaims))
	fmt.Fprintf(b, "| PVs | %d |\n", len(kubernetes.PersistentVolumes))
	fmt.Fprintf(b, "| CSIDrivers | %d |\n", len(kubernetes.CSIDrivers))
	fmt.Fprintf(b, "| RuntimeClasses | %d |\n\n", len(kubernetes.RuntimeClasses))

	writeRouting(b, kubernetes)
	writeStorage(b, kubernetes)
	writeRuntime(b, kubernetes)
	writeRunningImages(b, kubernetes)
	writeRunningContainers(b, kubernetes)
	writeWorkloads(b, kubernetes)
}

func writeRunningImages(b *strings.Builder, kubernetes inventory.Kubernetes) {
	if len(kubernetes.RunningImages) == 0 {
		return
	}
	fmt.Fprintf(b, "### Running images\n\n")
	fmt.Fprintf(b, "| Image | Containers | Runtimes | Namespaces | Workloads | Image IDs |\n| --- | ---: | --- | --- | --- | --- |\n")
	for _, image := range sortedRunningImages(kubernetes.RunningImages) {
		fmt.Fprintf(b, "| %s (pods=%d) | %d | %s | %s | %s | %s |\n",
			mdCell(image.Image),
			image.PodCount,
			image.ContainerCount,
			mdCell(strings.Join(image.Runtimes, ",")),
			mdCell(strings.Join(image.Namespaces, ",")),
			mdCell(refsValue(image.Workloads)),
			mdCell(shortImageIDs(image.ImageIDs)),
		)
	}
	fmt.Fprintln(b)
}

func writeRunningContainers(b *strings.Builder, kubernetes inventory.Kubernetes) {
	if len(kubernetes.RunningContainers) == 0 {
		return
	}
	fmt.Fprintf(b, "### Running containers\n\n")
	fmt.Fprintf(b, "| Namespace | Pod | Container | Type | Image | Image ID | Node | Workload |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, container := range sortedRunningContainers(kubernetes.RunningContainers) {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			mdCell(container.Namespace),
			mdCell(container.Pod),
			mdCell(container.Container),
			mdCell(container.ContainerType),
			mdCell(container.Image),
			mdCell(shortImageID(container.ImageID)),
			mdCell(container.NodeName),
			mdCell(ref(container.Workload)),
		)
	}
	fmt.Fprintln(b)
}

func writeRouting(b *strings.Builder, kubernetes inventory.Kubernetes) {
	if len(kubernetes.IngressClasses) == 0 && len(kubernetes.Ingresses) == 0 && len(kubernetes.Services) == 0 {
		return
	}
	fmt.Fprintf(b, "### Routing\n\n")
	fmt.Fprintf(b, "| Kind | Name | Class/Type | Route/Selector | Target/Ports |\n| --- | --- | --- | --- | --- |\n")
	for _, class := range sortedIngressClasses(kubernetes.IngressClasses) {
		fmt.Fprintf(b, "| IngressClass | %s | - | - | controller=%s |\n", mdCell(class.Name), mdCell(class.Controller))
	}
	for _, ingress := range sortedIngresses(kubernetes.Ingresses) {
		if len(ingress.Rules) == 0 {
			fmt.Fprintf(b, "| Ingress | %s | %s | rules=0 | - |\n", mdCell(ref(ingress.ObjectRef)), mdCell(ingress.ClassName))
			continue
		}
		for _, rule := range ingress.Rules {
			fmt.Fprintf(b, "| Ingress | %s | %s | %s%s | service/%s:%s |\n",
				mdCell(ref(ingress.ObjectRef)),
				mdCell(ingress.ClassName),
				mdCell(rule.Host),
				mdCell(rule.Path),
				mdCell(rule.ServiceName),
				mdCell(rule.ServicePort),
			)
		}
	}
	for _, service := range sortedServices(kubernetes.Services) {
		fmt.Fprintf(b, "| Service | %s | %s | %s | %s |\n", mdCell(ref(service.ObjectRef)), mdCell(service.Type), mdCell(mapValue(service.Selector)), mdCell(servicePorts(service.Ports)))
	}
	fmt.Fprintln(b)
}

func writeStorage(b *strings.Builder, kubernetes inventory.Kubernetes) {
	if len(kubernetes.StorageClasses) == 0 && len(kubernetes.PersistentVolumeClaims) == 0 && len(kubernetes.PersistentVolumes) == 0 && len(kubernetes.CSIDrivers) == 0 {
		return
	}
	fmt.Fprintf(b, "### Storage\n\n")
	fmt.Fprintf(b, "| Kind | Name | Access | Mode | Size | Ref | CSI/Provisioner | Status |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, class := range sortedStorageClasses(kubernetes.StorageClasses) {
		fmt.Fprintf(b, "| StorageClass | %s | - | - | - | - | %s | binding=%s reclaim=%s expand=%s |\n", mdCell(class.Name), mdCell(class.Provisioner), mdCell(class.VolumeBindingMode), mdCell(class.ReclaimPolicy), mdCell(boolPtr(class.AllowVolumeExpansion)))
	}
	pvByName := pvByName(kubernetes.PersistentVolumes)
	for _, claim := range sortedPVCs(kubernetes.PersistentVolumeClaims) {
		driver := "-"
		if pv, ok := pvByName[claim.VolumeName]; ok && pv.CSI != nil {
			driver = pv.CSI.Driver
		}
		fmt.Fprintf(b, "| PVC | %s | %s | %s | %s | %s -> %s | %s | %s |\n", mdCell(ref(claim.ObjectRef)), mdCell(strings.Join(claim.AccessModes, ",")), mdCell(claim.VolumeMode), mdCell(claim.RequestedStorage), mdCell(claim.StorageClassName), mdCell(claim.VolumeName), mdCell(driver), mdCell(claim.Phase))
	}
	for _, volume := range sortedPVs(kubernetes.PersistentVolumes) {
		driver := "-"
		if volume.CSI != nil {
			driver = volume.CSI.Driver
		}
		fmt.Fprintf(b, "| PV | %s | %s | %s | %s | %s -> %s | %s | %s |\n", mdCell(volume.Name), mdCell(strings.Join(volume.AccessModes, ",")), mdCell(volume.VolumeMode), mdCell(volume.Capacity), mdCell(volume.StorageClassName), mdCell(ref(volume.ClaimRef)), mdCell(driver), mdCell(volume.Phase))
	}
	for _, driver := range sortedCSIDrivers(kubernetes.CSIDrivers) {
		fmt.Fprintf(b, "| CSIDriver | %s | - | - | - | - | %s | attachRequired=%s podInfoOnMount=%s lifecycle=%s |\n", mdCell(driver.Name), mdCell(driver.Name), mdCell(boolPtr(driver.AttachRequired)), mdCell(boolPtr(driver.PodInfoOnMount)), mdCell(strings.Join(driver.VolumeLifecycleModes, ",")))
	}
	fmt.Fprintln(b)
}

func writeRuntime(b *strings.Builder, kubernetes inventory.Kubernetes) {
	if len(kubernetes.Nodes) == 0 && len(kubernetes.RuntimeClasses) == 0 {
		return
	}
	fmt.Fprintf(b, "### Runtime\n\n")
	fmt.Fprintf(b, "| Kind | Name | Runtime/Handler | Kubelet | OS/Arch | Users |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, counted := range sortedCounts(runtimeCounts(kubernetes.Nodes)) {
		fmt.Fprintf(b, "| CRI | %s | %s | - | - | nodes=%d |\n", mdCell(counted.name), mdCell(counted.name), counted.count)
	}
	for _, class := range sortedRuntimeClasses(kubernetes.RuntimeClasses) {
		fmt.Fprintf(b, "| RuntimeClass | %s | %s | - | - | workloads=%d pods=%d |\n", mdCell(class.Name), mdCell(class.Handler), workloadRuntimeUse(kubernetes.Workloads, class.Name), podRuntimeUse(kubernetes.Pods, class.Name))
	}
	for _, node := range sortedNodes(kubernetes.Nodes) {
		fmt.Fprintf(b, "| Node | %s | %s | %s | %s/%s | - |\n", mdCell(node.Name), mdCell(node.ContainerRuntime), mdCell(node.KubeletVersion), mdCell(node.OSImage), mdCell(node.Architecture))
	}
	fmt.Fprintln(b)
}

func writeWorkloads(b *strings.Builder, kubernetes inventory.Kubernetes) {
	if len(kubernetes.Workloads) == 0 {
		return
	}
	fmt.Fprintf(b, "### Workload topology\n\n")
	fmt.Fprintf(b, "| Workload | Images | ServiceAccount | RuntimeClass | Relations |\n| --- | --- | --- | --- | --- |\n")
	for _, workload := range sortedWorkloads(kubernetes.Workloads) {
		relations := workloadRelations(workload)
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", mdCell(ref(workload.ObjectRef)), mdCell(images(workload)), mdCell(workload.ServiceAccountName), mdCell(workload.RuntimeClassName), mdCell(strings.Join(relations, "<br>")))
	}
	fmt.Fprintln(b)
}

func writeCoverage(b *strings.Builder, coverage []inventory.CoverageItem) {
	if len(coverage) == 0 {
		return
	}
	fmt.Fprintf(b, "## Coverage\n\n")
	fmt.Fprintf(b, "| Status | Area | Resource | Objects | Reason |\n| --- | --- | --- | ---: | --- |\n")
	for _, item := range coverage {
		fmt.Fprintf(b, "| %s | %s | %s | %d | %s |\n", mdCell(item.Status), mdCell(item.Area), mdCell(item.Resource), item.ObjectCount, mdCell(item.Reason))
	}
	fmt.Fprintln(b)
}

func targetName(snapshot *inventory.Snapshot) string {
	if snapshot.EKS.Cluster.Name != "" {
		return snapshot.EKS.Cluster.Name
	}
	if snapshot.Kubernetes.Context != "" {
		return snapshot.Kubernetes.Context
	}
	return "teleskope-scan"
}

func hasAWS(snapshot *inventory.Snapshot) bool {
	return snapshot.AWS.Region != "" || snapshot.AWS.ARN != "" || snapshot.AWS.AccountID != "" || snapshot.AWS.UserID != ""
}

func hasEKS(snapshot *inventory.Snapshot) bool {
	return snapshot.EKS.Cluster.Name != "" ||
		len(snapshot.EKS.Addons) > 0 ||
		len(snapshot.EKS.Nodegroups) > 0 ||
		len(snapshot.EKS.AccessEntries) > 0 ||
		len(snapshot.EKS.PodIdentityAssociations) > 0
}

func hasKubernetes(snapshot *inventory.Snapshot) bool {
	kubernetes := snapshot.Kubernetes
	if kubernetes.Context != "" ||
		kubernetes.Version.GitVersion != "" ||
		len(kubernetes.APIResources) > 0 ||
		len(kubernetes.Nodes) > 0 ||
		len(kubernetes.Workloads) > 0 ||
		len(kubernetes.Pods) > 0 {
		return true
	}
	for _, item := range snapshot.Coverage {
		if item.Area == "kubernetes" {
			return true
		}
	}
	return false
}

func sortedIngressClasses(items []inventory.IngressClass) []inventory.IngressClass {
	out := append([]inventory.IngressClass(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedIngresses(items []inventory.Ingress) []inventory.Ingress {
	out := append([]inventory.Ingress(nil), items...)
	sort.Slice(out, func(i, j int) bool { return ref(out[i].ObjectRef) < ref(out[j].ObjectRef) })
	return out
}

func sortedServices(items []inventory.Service) []inventory.Service {
	out := append([]inventory.Service(nil), items...)
	sort.Slice(out, func(i, j int) bool { return ref(out[i].ObjectRef) < ref(out[j].ObjectRef) })
	return out
}

func sortedStorageClasses(items []inventory.StorageClass) []inventory.StorageClass {
	out := append([]inventory.StorageClass(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedPVCs(items []inventory.PersistentVolumeClaim) []inventory.PersistentVolumeClaim {
	out := append([]inventory.PersistentVolumeClaim(nil), items...)
	sort.Slice(out, func(i, j int) bool { return ref(out[i].ObjectRef) < ref(out[j].ObjectRef) })
	return out
}

func sortedPVs(items []inventory.PersistentVolume) []inventory.PersistentVolume {
	out := append([]inventory.PersistentVolume(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedCSIDrivers(items []inventory.CSIDriver) []inventory.CSIDriver {
	out := append([]inventory.CSIDriver(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedRuntimeClasses(items []inventory.RuntimeClass) []inventory.RuntimeClass {
	out := append([]inventory.RuntimeClass(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedNodes(items []inventory.Node) []inventory.Node {
	out := append([]inventory.Node(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedWorkloads(items []inventory.Workload) []inventory.Workload {
	out := append([]inventory.Workload(nil), items...)
	sort.Slice(out, func(i, j int) bool { return ref(out[i].ObjectRef) < ref(out[j].ObjectRef) })
	return out
}

func sortedRunningContainers(items []inventory.RunningContainer) []inventory.RunningContainer {
	out := append([]inventory.RunningContainer(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Namespace, out[i].Pod, out[i].ContainerType, out[i].Container}, "\x00")
		right := strings.Join([]string{out[j].Namespace, out[j].Pod, out[j].ContainerType, out[j].Container}, "\x00")
		return left < right
	})
	return out
}

func sortedRunningImages(items []inventory.RunningImage) []inventory.RunningImage {
	out := append([]inventory.RunningImage(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].PodCount != out[j].PodCount {
			return out[i].PodCount > out[j].PodCount
		}
		return out[i].Image < out[j].Image
	})
	return out
}

type counted struct {
	name  string
	count int
}

func sortedCounts(counts map[string]int) []counted {
	out := make([]counted, 0, len(counts))
	for name, count := range counts {
		out = append(out, counted{name: name, count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func runtimeCounts(nodes []inventory.Node) map[string]int {
	counts := map[string]int{}
	for _, node := range nodes {
		if node.ContainerRuntime != "" {
			counts[node.ContainerRuntime]++
		}
	}
	return counts
}

func pvByName(items []inventory.PersistentVolume) map[string]inventory.PersistentVolume {
	out := map[string]inventory.PersistentVolume{}
	for _, item := range items {
		out[item.Name] = item
	}
	return out
}

func workloadRuntimeUse(workloads []inventory.Workload, runtimeClass string) int {
	count := 0
	for _, workload := range workloads {
		if workload.RuntimeClassName == runtimeClass {
			count++
		}
	}
	return count
}

func podRuntimeUse(pods []inventory.Pod, runtimeClass string) int {
	count := 0
	for _, pod := range pods {
		if pod.RuntimeClassName == runtimeClass {
			count++
		}
	}
	return count
}

func workloadRelations(workload inventory.Workload) []string {
	var out []string
	for _, volume := range workload.Volumes {
		switch {
		case volume.PersistentVolumeClaim != "":
			out = append(out, fmt.Sprintf("volume/%s -> pvc/%s", volume.Name, volume.PersistentVolumeClaim))
		case volume.ConfigMap != "":
			out = append(out, fmt.Sprintf("volume/%s -> configmap/%s", volume.Name, volume.ConfigMap))
		case volume.Secret != "":
			out = append(out, fmt.Sprintf("volume/%s -> secret/%s(metadata-only)", volume.Name, volume.Secret))
		case volume.CSI != "":
			out = append(out, fmt.Sprintf("volume/%s -> csi/%s", volume.Name, volume.CSI))
		}
	}
	for _, ref := range workload.ConfigRefs {
		out = append(out, "config -> "+refValue(ref))
	}
	for _, ref := range workload.SecretRefs {
		out = append(out, "secret -> "+refValue(ref)+"(metadata-only)")
	}
	if len(out) == 0 {
		out = append(out, "-")
	}
	return out
}

func images(workload inventory.Workload) string {
	seen := map[string]struct{}{}
	var out []string
	for _, container := range append(workload.Containers, workload.InitContainers...) {
		if container.Image == "" {
			continue
		}
		if _, ok := seen[container.Image]; ok {
			continue
		}
		seen[container.Image] = struct{}{}
		out = append(out, container.Image)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return "-"
	}
	return strings.Join(out, ",")
}

func servicePorts(ports []inventory.ServicePort) string {
	if len(ports) == 0 {
		return "-"
	}
	out := make([]string, 0, len(ports))
	for _, port := range ports {
		value := fmt.Sprintf("%s/%d", port.Protocol, port.Port)
		if port.TargetPort != "" {
			value += "->" + port.TargetPort
		}
		if port.NodePort > 0 {
			value += fmt.Sprintf(" node=%d", port.NodePort)
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func ref(value inventory.ObjectRef) string {
	if value.Kind == "" && value.Namespace == "" && value.Name == "" {
		return "-"
	}
	kind := strings.ToLower(value.Kind)
	if value.Namespace == "" {
		if kind == "" {
			return value.Name
		}
		return kind + "/" + value.Name
	}
	return kind + "/" + value.Namespace + "/" + value.Name
}

func refValue(value inventory.ObjectRef) string {
	out := ref(value)
	if out == "" {
		return "-"
	}
	return out
}

func refsValue(refs []inventory.ObjectRef) string {
	if len(refs) == 0 {
		return "-"
	}
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, refValue(ref))
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func mapValue(value map[string]string) string {
	if len(value) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+value[key])
	}
	return strings.Join(parts, ",")
}

func boolPtr(value *bool) string {
	if value == nil {
		return "-"
	}
	if *value {
		return "true"
	}
	return "false"
}

func shortImageID(value string) string {
	value = strings.TrimPrefix(value, "docker-pullable://")
	value = strings.TrimPrefix(value, "docker://")
	value = strings.TrimPrefix(value, "containerd://")
	const digestPrefix = "sha256:"
	if idx := strings.LastIndex(value, digestPrefix); idx >= 0 {
		digest := value[idx+len(digestPrefix):]
		if len(digest) > 12 {
			return digestPrefix + digest[:12]
		}
		return digestPrefix + digest
	}
	if len(value) > 32 {
		return value[:32]
	}
	return value
}

func shortImageIDs(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, shortImageID(value))
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func mdCell(value string) string {
	if value == "" {
		value = "-"
	}
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", `\|`)
	return value
}

func mdInline(value string) string {
	return strings.ReplaceAll(value, "`", "'")
}
