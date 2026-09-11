package render

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// Human writes a compact terminal summary inspired by small TUI status tools.
func Human(w io.Writer, snapshot *inventory.Snapshot) error {
	cluster := snapshot.EKS.Cluster
	title := cluster.Name
	if title == "" && snapshot.Kubernetes.Context != "" {
		title = snapshot.Kubernetes.Context
	}
	fmt.Fprintf(w, "teleskope  %s\n", value(title))

	if hasEKS(snapshot) {
		fmt.Fprintf(w, "region     %s\n", snapshot.AWS.Region)
		if snapshot.AWS.ARN != "" {
			fmt.Fprintf(w, "identity   %s\n", snapshot.AWS.ARN)
		}
		fmt.Fprintf(w, "version    k8s %s / platform %s / status %s\n", value(cluster.Version), value(cluster.PlatformVersion), value(cluster.Status))
		fmt.Fprintf(w, "endpoint   public=%t private=%t vpc=%s\n", cluster.VPC.EndpointPublicAccess, cluster.VPC.EndpointPrivateAccess, value(cluster.VPC.VPCID))
		fmt.Fprintf(w, "network    ipFamily=%s serviceCIDR=%s\n", value(cluster.Network.IPFamily), serviceCIDR(cluster.Network))
		fmt.Fprintf(w, "auth       %s\n", value(cluster.AccessConfig.AuthenticationMode))
		fmt.Fprintln(w)

		fmt.Fprintf(w, "addons     %d\n", len(snapshot.EKS.Addons))
		for _, addon := range snapshot.EKS.Addons {
			fmt.Fprintf(w, "  %s  %s  %s", marker(addon.Status), addon.Name, value(addon.Version))
			if addon.Namespace != "" {
				fmt.Fprintf(w, "  ns=%s", addon.Namespace)
			}
			if addon.ServiceAccountRoleARN != "" {
				fmt.Fprintf(w, "  irsa")
			}
			if len(addon.PodIdentityAssociations) > 0 {
				fmt.Fprintf(w, "  pod-identity=%d", len(addon.PodIdentityAssociations))
			}
			if addon.ConfigurationValues != "" {
				fmt.Fprintf(w, "  config")
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)

		fmt.Fprintf(w, "nodegroups %d\n", len(snapshot.EKS.Nodegroups))
		for _, nodegroup := range snapshot.EKS.Nodegroups {
			fmt.Fprintf(w, "  %s  %s  k8s=%s ami=%s capacity=%s desired=%s",
				marker(nodegroup.Status),
				nodegroup.Name,
				value(nodegroup.Version),
				value(nodegroup.ReleaseVersion),
				value(nodegroup.CapacityType),
				int32Value(nodegroup.DesiredSize),
			)
			if len(nodegroup.InstanceTypes) > 0 {
				fmt.Fprintf(w, "  type=%s", strings.Join(nodegroup.InstanceTypes, ","))
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)

		fmt.Fprintf(w, "identity   accessEntries=%d podIdentity=%d\n", len(snapshot.EKS.AccessEntries), len(snapshot.EKS.PodIdentityAssociations))
		fmt.Fprintln(w)
	}

	if hasKubernetes(snapshot) {
		fmt.Fprintf(w, "kubernetes %s\n", value(snapshot.Kubernetes.Version.GitVersion))
		if snapshot.Kubernetes.Context != "" {
			fmt.Fprintf(w, "  context  %s\n", snapshot.Kubernetes.Context)
		}
		fmt.Fprintf(w, "  api      resources=%d crds=%d crInstances=%d apiServices=%d\n", len(snapshot.Kubernetes.APIResources), len(snapshot.Kubernetes.CustomResourceDefinitions), len(snapshot.Kubernetes.CustomResourceInstances), len(snapshot.Kubernetes.APIServices))
		fmt.Fprintf(w, "  compute  nodes=%d runtimeClasses=%d\n", len(snapshot.Kubernetes.Nodes), len(snapshot.Kubernetes.RuntimeClasses))
		fmt.Fprintf(w, "  workload workloads=%d pods=%d images=%d runningImages=%d runningContainers=%d\n", len(snapshot.Kubernetes.Workloads), len(snapshot.Kubernetes.Pods), imageCount(snapshot), len(snapshot.Kubernetes.RunningImages), len(snapshot.Kubernetes.RunningContainers))
		fmt.Fprintf(w, "  network  services=%d endpointSlices=%d ingresses=%d classes=%d gatewayClasses=%d gateways=%d gatewayRoutes=%d\n", len(snapshot.Kubernetes.Services), len(snapshot.Kubernetes.EndpointSlices), len(snapshot.Kubernetes.Ingresses), len(snapshot.Kubernetes.IngressClasses), len(snapshot.Kubernetes.GatewayClasses), len(snapshot.Kubernetes.Gateways), len(snapshot.Kubernetes.GatewayRoutes))
		fmt.Fprintf(w, "  storage  classes=%d pv=%d pvc=%d csiDrivers=%d csiNodes=%d attachments=%d\n", len(snapshot.Kubernetes.StorageClasses), len(snapshot.Kubernetes.PersistentVolumes), len(snapshot.Kubernetes.PersistentVolumeClaims), len(snapshot.Kubernetes.CSIDrivers), len(snapshot.Kubernetes.CSINodes), len(snapshot.Kubernetes.VolumeAttachments))
		fmt.Fprintf(w, "  config   configMaps=%d secrets=%d(metadata only)\n", len(snapshot.Kubernetes.ConfigMaps), len(snapshot.Kubernetes.Secrets))
		fmt.Fprintln(w)
		writeKubernetesDetails(w, snapshot)
	}

	fmt.Fprintf(w, "coverage   %d\n", len(snapshot.Coverage))
	coverage := newTable(w)
	fmt.Fprintln(coverage, "STATUS\tAREA\tRESOURCE\tOBJECTS\tREASON")
	for _, item := range snapshot.Coverage {
		fmt.Fprintf(coverage, "%s\t%s\t%s\t%d\t%s\n", marker(item.Status), item.Area, item.Resource, item.ObjectCount, value(item.Reason))
	}
	coverage.Flush()
	return nil
}

func writeKubernetesDetails(w io.Writer, snapshot *inventory.Snapshot) {
	kubernetes := snapshot.Kubernetes
	writeRouting(w, kubernetes)
	writeStorage(w, kubernetes)
	writeRuntime(w, kubernetes)
	writeCustomResources(w, kubernetes)
	writeRunningImages(w, kubernetes)
	writeRunningContainers(w, kubernetes)
	writePlatformComponents(w, kubernetes)
	writeWorkloadRelations(w, kubernetes)
}

func writeCustomResources(w io.Writer, kubernetes inventory.Kubernetes) {
	if len(kubernetes.CustomResourceCounts) == 0 && len(kubernetes.CustomResourceInstances) == 0 {
		return
	}
	fmt.Fprintln(w, "custom-resources")
	if len(kubernetes.CustomResourceCounts) > 0 {
		counts := newTable(w)
		fmt.Fprintln(counts, "CRD TYPE\tINSTANCES\tNAMESPACES\tSCOPE\tVERSION\tPLURAL")
		for _, count := range sortedCustomResourceCounts(kubernetes.CustomResourceCounts) {
			fmt.Fprintf(counts, "%s\t%d\t%d\t%s\t%s\t%s\n",
				crdType(count.Group, count.Kind),
				count.InstanceCount,
				count.NamespaceCount,
				value(count.Scope),
				value(count.Version),
				value(count.Plural),
			)
		}
		counts.Flush()
	}
	if len(kubernetes.CustomResourceInstances) > 0 {
		instances := newTable(w)
		fmt.Fprintln(instances, "INSTANCE\tCRD TYPE\tVERSION\tCRD\tOWNERS")
		for _, instance := range sortedCustomResourceInstances(kubernetes.CustomResourceInstances) {
			fmt.Fprintf(instances, "%s\t%s\t%s\t%s\t%s\n",
				refValue(instance.ObjectRef),
				crdType(instance.CRDGroup, instance.CRDKind),
				value(instance.CRDVersion),
				value(instance.CRDName),
				refsValue(instance.OwnerReferences),
			)
		}
		instances.Flush()
	}
	fmt.Fprintln(w)
}

func writeRunningImages(w io.Writer, kubernetes inventory.Kubernetes) {
	if len(kubernetes.RunningImages) == 0 {
		return
	}
	fmt.Fprintln(w, "running-images")
	table := newTable(w)
	fmt.Fprintln(table, "IMAGE\tCONTAINERS\tRUNTIMES\tNAMESPACES\tWORKLOADS\tIMAGE IDS")
	for _, image := range sortedRunningImages(kubernetes.RunningImages) {
		fmt.Fprintf(table, "%s (pods=%d)\t%d\t%s\t%s\t%s\t%s\n",
			value(image.Image),
			image.PodCount,
			image.ContainerCount,
			stringList(image.Runtimes),
			stringList(image.Namespaces),
			refsValue(image.Workloads),
			shortImageIDs(image.ImageIDs),
		)
	}
	table.Flush()
	fmt.Fprintln(w)
}

func writeRunningContainers(w io.Writer, kubernetes inventory.Kubernetes) {
	if len(kubernetes.RunningContainers) == 0 {
		return
	}
	fmt.Fprintln(w, "running-containers")
	table := newTable(w)
	fmt.Fprintln(table, "NAMESPACE\tPOD\tCONTAINER\tTYPE\tIMAGE\tIMAGE ID\tNODE\tWORKLOAD")
	for _, container := range sortedRunningContainers(kubernetes.RunningContainers) {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			value(container.Namespace),
			value(container.Pod),
			value(container.Container),
			value(container.ContainerType),
			value(container.Image),
			shortImageID(container.ImageID),
			value(container.NodeName),
			refValue(container.Workload),
		)
	}
	table.Flush()
	fmt.Fprintln(w)
}

func writeRouting(w io.Writer, kubernetes inventory.Kubernetes) {
	if len(kubernetes.IngressClasses) == 0 && len(kubernetes.Ingresses) == 0 && len(kubernetes.GatewayClasses) == 0 && len(kubernetes.Gateways) == 0 && len(kubernetes.GatewayRoutes) == 0 && len(kubernetes.ReferenceGrants) == 0 && len(kubernetes.GatewayPolicies) == 0 && len(kubernetes.Services) == 0 {
		return
	}
	fmt.Fprintln(w, "routing")
	table := newTable(w)
	fmt.Fprintln(table, "KIND\tNAME\tCLASS/TYPE\tROUTE/SELECTOR\tTARGET/PORTS\tENDPOINTS")
	for _, class := range sortedIngressClasses(kubernetes.IngressClasses) {
		fmt.Fprintf(table, "class\t%s\t-\t-\tcontroller=%s\t-\n", class.Name, value(class.Controller))
	}
	for _, ingress := range sortedIngresses(kubernetes.Ingresses) {
		if len(ingress.Rules) == 0 {
			fmt.Fprintf(table, "ingress\t%s\t%s\trules=0\t-\t-\n", namespaced(ingress.ObjectRef), value(ingress.ClassName))
			continue
		}
		for _, rule := range ingress.Rules {
			fmt.Fprintf(table, "ingress\t%s\t%s\t%s%s\tservice/%s:%s\t-\n",
				namespaced(ingress.ObjectRef),
				value(ingress.ClassName),
				value(rule.Host),
				pathValue(rule.Path),
				value(rule.ServiceName),
				value(rule.ServicePort),
			)
		}
	}

	endpointsByService := endpointTargetCounts(kubernetes.EndpointSlices)
	for _, class := range sortedGatewayClasses(kubernetes.GatewayClasses) {
		fmt.Fprintf(table, "gateway-class\t%s\t-\t-\tcontroller=%s\t-\n", class.Name, value(class.ControllerName))
	}
	for _, gateway := range sortedGateways(kubernetes.Gateways) {
		fmt.Fprintf(table, "gateway\t%s\t%s\tlisteners=%s\taddresses=%s\t-\n", namespaced(gateway.ObjectRef), value(gateway.ClassName), gatewayListeners(gateway.Listeners), strings.Join(gateway.Addresses, ","))
	}
	for _, route := range sortedGatewayRoutes(kubernetes.GatewayRoutes) {
		fmt.Fprintf(table, "%s\t%s\t%s\thosts=%s parents=%s\t%s\t-\n", strings.ToLower(route.Kind), namespaced(route.ObjectRef), routeRuleSummary(route.Rules), strings.Join(route.Hostnames, ","), parentRefsValue(route.ParentRefs), routeBackends(route.Rules))
	}
	for _, grant := range sortedReferenceGrants(kubernetes.ReferenceGrants) {
		fmt.Fprintf(table, "reference-grant\t%s\t-\tfrom=%s\tto=%s\t-\n", namespaced(grant.ObjectRef), grantRefsValue(grant.From), grantRefsValue(grant.To))
	}
	for _, policy := range sortedGatewayPolicies(kubernetes.GatewayPolicies) {
		fmt.Fprintf(table, "%s\t%s\t-\ttargets=%s\t%s\t-\n", strings.ToLower(policy.Kind), namespaced(policy.ObjectRef), refsValue(policy.TargetRefs), strings.Join(policy.Details, ","))
	}
	for _, service := range sortedServices(kubernetes.Services) {
		fmt.Fprintf(table, "service\t%s\t%s\t%s\t%s\t%d\n",
			namespaced(service.ObjectRef),
			value(service.Type),
			mapValue(service.Selector),
			servicePorts(service.Ports),
			endpointsByService[namespaced(service.ObjectRef)],
		)
	}
	table.Flush()
	fmt.Fprintln(w)
}

func writeStorage(w io.Writer, kubernetes inventory.Kubernetes) {
	if len(kubernetes.StorageClasses) == 0 && len(kubernetes.PersistentVolumeClaims) == 0 && len(kubernetes.PersistentVolumes) == 0 && len(kubernetes.CSIDrivers) == 0 {
		return
	}
	fmt.Fprintln(w, "storage")
	table := newTable(w)
	fmt.Fprintln(table, "KIND\tNAME\tACCESS\tMODE\tSIZE\tREF\tCSI/PROVISIONER\tSTATUS/DETAILS")
	for _, class := range sortedStorageClasses(kubernetes.StorageClasses) {
		fmt.Fprintf(table, "class\t%s\t-\t-\t-\t-\t%s\tbinding=%s reclaim=%s expand=%s params=%s\n",
			class.Name,
			value(class.Provisioner),
			value(class.VolumeBindingMode),
			value(class.ReclaimPolicy),
			boolPtrValue(class.AllowVolumeExpansion),
			mapValue(class.Parameters),
		)
	}

	pvByName := persistentVolumeByName(kubernetes.PersistentVolumes)
	for _, claim := range sortedPVCs(kubernetes.PersistentVolumeClaims) {
		driver := "-"
		if pv, ok := pvByName[claim.VolumeName]; ok && pv.CSI != nil {
			driver = pv.CSI.Driver
		}
		fmt.Fprintf(table, "pvc\t%s\t%s\t%s\t%s\t%s -> %s\t%s\t%s\n",
			namespaced(claim.ObjectRef),
			stringList(claim.AccessModes),
			value(claim.VolumeMode),
			value(claim.RequestedStorage),
			value(claim.StorageClassName),
			value(claim.VolumeName),
			value(driver),
			value(claim.Phase),
		)
	}
	for _, volume := range sortedPVs(kubernetes.PersistentVolumes) {
		csi := "-"
		if volume.CSI != nil {
			csi = fmt.Sprintf("%s handle=%s fs=%s", value(volume.CSI.Driver), value(volume.CSI.VolumeHandle), value(volume.CSI.FSType))
		}
		fmt.Fprintf(table, "pv\t%s\t%s\t%s\t%s\t%s -> %s\t%s\t%s\n",
			volume.Name,
			stringList(volume.AccessModes),
			value(volume.VolumeMode),
			value(volume.Capacity),
			value(volume.StorageClassName),
			refValue(volume.ClaimRef),
			csi,
			value(volume.Phase),
		)
	}
	for _, driver := range sortedCSIDrivers(kubernetes.CSIDrivers) {
		fmt.Fprintf(table, "csi-driver\t%s\t-\t-\t-\t-\t%s\tattachRequired=%s podInfoOnMount=%s lifecycle=%s nodes=%d\n",
			driver.Name,
			driver.Name,
			boolPtrValue(driver.AttachRequired),
			boolPtrValue(driver.PodInfoOnMount),
			stringList(driver.VolumeLifecycleModes),
			csiNodeCount(kubernetes.CSINodes, driver.Name),
		)
	}
	for _, attachment := range sortedVolumeAttachments(kubernetes.VolumeAttachments) {
		fmt.Fprintf(table, "attachment\t%s\t-\t-\t-\t%s -> %s\t%s\tattached=%t error=%s\n",
			attachment.Name,
			value(attachment.PVName),
			value(attachment.NodeName),
			value(attachment.Attacher),
			attachment.Attached,
			value(attachment.AttachError),
		)
	}
	table.Flush()
	fmt.Fprintln(w)
}

func writeRuntime(w io.Writer, kubernetes inventory.Kubernetes) {
	if len(kubernetes.Nodes) == 0 && len(kubernetes.RuntimeClasses) == 0 {
		return
	}
	fmt.Fprintln(w, "runtime")
	table := newTable(w)
	fmt.Fprintln(table, "KIND\tNAME\tREADY/HANDLER\tRUNTIME\tKUBELET\tOS/ARCH\tUSERS")
	for _, runtime := range sortedCounts(nodeRuntimeCounts(kubernetes.Nodes)) {
		fmt.Fprintf(table, "cri\t%s\t-\t%s\t-\t-\tnodes=%d\n", runtime.name, runtime.name, runtime.count)
	}
	for _, class := range sortedRuntimeClasses(kubernetes.RuntimeClasses) {
		fmt.Fprintf(table, "runtimeclass\t%s\t%s\t-\t-\t-\tworkloads=%d pods=%d\n",
			class.Name,
			value(class.Handler),
			workloadRuntimeUse(kubernetes.Workloads, class.Name),
			podRuntimeUse(kubernetes.Pods, class.Name),
		)
	}
	for _, node := range sortedNodes(kubernetes.Nodes) {
		fmt.Fprintf(table, "node\t%s\t%s\t%s\t%s\t%s/%s\t-\n",
			node.Name,
			value(node.Ready),
			value(node.ContainerRuntime),
			value(node.KubeletVersion),
			value(node.OSImage),
			value(node.Architecture),
		)
	}
	table.Flush()
	fmt.Fprintln(w)
}

func writePlatformComponents(w io.Writer, kubernetes inventory.Kubernetes) {
	cni := componentCandidates(kubernetes.Workloads, "cni")
	csi := componentCandidates(kubernetes.Workloads, "csi")
	if len(cni) == 0 && len(csi) == 0 {
		return
	}
	fmt.Fprintln(w, "platform")
	table := newTable(w)
	fmt.Fprintln(table, "TYPE\tWORKLOAD\tIMAGES\tCONFIG\tSECRETS\tVOLUMES/SA")
	for _, workload := range cni {
		fmt.Fprintf(table, "cni\t%s\t%s\t%s\t%s\t-\n", refValue(workload.ObjectRef), images(workload), refsValue(workload.ConfigRefs), refsValue(workload.SecretRefs))
	}
	for _, workload := range csi {
		fmt.Fprintf(table, "csi\t%s\t%s\t-\t-\tvolumes=%s sa=%s\n", refValue(workload.ObjectRef), images(workload), volumesValue(workload.Volumes), value(workload.ServiceAccountName))
	}
	table.Flush()
	fmt.Fprintln(w)
}

func writeWorkloadRelations(w io.Writer, kubernetes inventory.Kubernetes) {
	if len(kubernetes.Workloads) == 0 {
		return
	}
	pvcByKey := pvcByNamespacedName(kubernetes.PersistentVolumeClaims)
	pvByName := persistentVolumeByName(kubernetes.PersistentVolumes)
	fmt.Fprintln(w, "workload-relations")
	workloads := newTable(w)
	fmt.Fprintln(workloads, "WORKLOAD\tREPLICAS\tREADY\tIMAGES\tSA\tRUNTIME\tSELECTOR")
	for _, workload := range sortedWorkloads(kubernetes.Workloads) {
		fmt.Fprintf(workloads, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
			refValue(workload.ObjectRef),
			int32PtrValue(workload.Replicas),
			workload.ReadyReplicas,
			images(workload),
			value(workload.ServiceAccountName),
			value(workload.RuntimeClassName),
			mapValue(workload.Selector),
		)
	}
	workloads.Flush()

	relations := newTable(w)
	fmt.Fprintln(relations, "WORKLOAD\tRELATION\tSOURCE\tTARGET\tDETAILS")
	for _, workload := range sortedWorkloads(kubernetes.Workloads) {
		for _, volume := range workload.Volumes {
			switch {
			case volume.PersistentVolumeClaim != "":
				key := namespacedName(workload.Namespace, volume.PersistentVolumeClaim)
				claim, ok := pvcByKey[key]
				if !ok {
					fmt.Fprintf(relations, "%s\tvolume\t%s\tpvc/%s\tunresolved\n", refValue(workload.ObjectRef), volume.Name, key)
					continue
				}
				driver := "-"
				if pv, ok := pvByName[claim.VolumeName]; ok && pv.CSI != nil {
					driver = pv.CSI.Driver
				}
				fmt.Fprintf(relations, "%s\tvolume\t%s\tpvc/%s\taccess=%s mode=%s sc=%s pv=%s csi=%s\n",
					refValue(workload.ObjectRef),
					volume.Name,
					key,
					stringList(claim.AccessModes),
					value(claim.VolumeMode),
					value(claim.StorageClassName),
					value(claim.VolumeName),
					value(driver),
				)
			case volume.ConfigMap != "":
				fmt.Fprintf(relations, "%s\tconfig\tvolume/%s\tconfigmap/%s\t-\n", refValue(workload.ObjectRef), volume.Name, namespacedName(workload.Namespace, volume.ConfigMap))
			case volume.Secret != "":
				fmt.Fprintf(relations, "%s\tsecret\tvolume/%s\tsecret/%s\tmetadata-only\n", refValue(workload.ObjectRef), volume.Name, namespacedName(workload.Namespace, volume.Secret))
			case volume.CSI != "":
				fmt.Fprintf(relations, "%s\tcsi\tvolume/%s\tdriver/%s\t-\n", refValue(workload.ObjectRef), volume.Name, volume.CSI)
			}
		}
		for _, ref := range workload.ConfigRefs {
			fmt.Fprintf(relations, "%s\tconfig\tenv\t%s\t-\n", refValue(workload.ObjectRef), refValue(ref))
		}
		for _, ref := range workload.SecretRefs {
			fmt.Fprintf(relations, "%s\tsecret\tenv\t%s\tmetadata-only\n", refValue(workload.ObjectRef), refValue(ref))
		}
	}
	relations.Flush()
	fmt.Fprintln(w)
}

func newTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
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
	return kubernetes.Context != "" ||
		kubernetes.Version.GitVersion != "" ||
		len(kubernetes.APIResources) > 0 ||
		len(kubernetes.Nodes) > 0 ||
		len(kubernetes.Workloads) > 0 ||
		len(kubernetes.Pods) > 0
}

func imageCount(snapshot *inventory.Snapshot) int {
	seen := map[string]struct{}{}
	for _, workload := range snapshot.Kubernetes.Workloads {
		for _, container := range workload.Containers {
			if container.Image != "" {
				seen[container.Image] = struct{}{}
			}
		}
		for _, container := range workload.InitContainers {
			if container.Image != "" {
				seen[container.Image] = struct{}{}
			}
		}
	}
	for _, pod := range snapshot.Kubernetes.Pods {
		for _, container := range pod.Containers {
			if container.Image != "" {
				seen[container.Image] = struct{}{}
			}
		}
		for _, container := range pod.InitContainers {
			if container.Image != "" {
				seen[container.Image] = struct{}{}
			}
		}
	}
	return len(seen)
}

func sortedIngressClasses(items []inventory.IngressClass) []inventory.IngressClass {
	out := append([]inventory.IngressClass(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedIngresses(items []inventory.Ingress) []inventory.Ingress {
	out := append([]inventory.Ingress(nil), items...)
	sort.Slice(out, func(i, j int) bool { return namespaced(out[i].ObjectRef) < namespaced(out[j].ObjectRef) })
	return out
}

func sortedGatewayClasses(items []inventory.GatewayClass) []inventory.GatewayClass {
	out := append([]inventory.GatewayClass(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedGateways(items []inventory.Gateway) []inventory.Gateway {
	out := append([]inventory.Gateway(nil), items...)
	sort.Slice(out, func(i, j int) bool { return namespaced(out[i].ObjectRef) < namespaced(out[j].ObjectRef) })
	return out
}

func sortedGatewayRoutes(items []inventory.GatewayRoute) []inventory.GatewayRoute {
	out := append([]inventory.GatewayRoute(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Kind, out[i].Namespace, out[i].Name}, "\x00")
		right := strings.Join([]string{out[j].Kind, out[j].Namespace, out[j].Name}, "\x00")
		return left < right
	})
	return out
}

func sortedReferenceGrants(items []inventory.ReferenceGrant) []inventory.ReferenceGrant {
	out := append([]inventory.ReferenceGrant(nil), items...)
	sort.Slice(out, func(i, j int) bool { return namespaced(out[i].ObjectRef) < namespaced(out[j].ObjectRef) })
	return out
}

func sortedGatewayPolicies(items []inventory.GatewayPolicy) []inventory.GatewayPolicy {
	out := append([]inventory.GatewayPolicy(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].Kind, out[i].Namespace, out[i].Name}, "\x00")
		right := strings.Join([]string{out[j].Kind, out[j].Namespace, out[j].Name}, "\x00")
		return left < right
	})
	return out
}

func sortedServices(items []inventory.Service) []inventory.Service {
	out := append([]inventory.Service(nil), items...)
	sort.Slice(out, func(i, j int) bool { return namespaced(out[i].ObjectRef) < namespaced(out[j].ObjectRef) })
	return out
}

func sortedStorageClasses(items []inventory.StorageClass) []inventory.StorageClass {
	out := append([]inventory.StorageClass(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedPVCs(items []inventory.PersistentVolumeClaim) []inventory.PersistentVolumeClaim {
	out := append([]inventory.PersistentVolumeClaim(nil), items...)
	sort.Slice(out, func(i, j int) bool { return namespaced(out[i].ObjectRef) < namespaced(out[j].ObjectRef) })
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

func sortedVolumeAttachments(items []inventory.VolumeAttachment) []inventory.VolumeAttachment {
	out := append([]inventory.VolumeAttachment(nil), items...)
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
	sort.Slice(out, func(i, j int) bool { return refValue(out[i].ObjectRef) < refValue(out[j].ObjectRef) })
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

func sortedCustomResourceCounts(items []inventory.CustomResourceCount) []inventory.CustomResourceCount {
	out := append([]inventory.CustomResourceCount(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].InstanceCount != out[j].InstanceCount {
			return out[i].InstanceCount > out[j].InstanceCount
		}
		left := strings.Join([]string{out[i].Group, out[i].Kind, out[i].Version}, "\x00")
		right := strings.Join([]string{out[j].Group, out[j].Kind, out[j].Version}, "\x00")
		return left < right
	})
	return out
}

func sortedCustomResourceInstances(items []inventory.CustomResourceInstance) []inventory.CustomResourceInstance {
	out := append([]inventory.CustomResourceInstance(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].CRDGroup, out[i].CRDKind, out[i].Namespace, out[i].Name}, "\x00")
		right := strings.Join([]string{out[j].CRDGroup, out[j].CRDKind, out[j].Namespace, out[j].Name}, "\x00")
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

func endpointTargetCounts(slices []inventory.EndpointSlice) map[string]int {
	counts := map[string]int{}
	for _, slice := range slices {
		if slice.ServiceName == "" {
			continue
		}
		key := namespacedName(slice.Namespace, slice.ServiceName)
		for _, endpoint := range slice.Endpoints {
			if endpoint.Ready != nil && !*endpoint.Ready {
				continue
			}
			counts[key]++
		}
	}
	return counts
}

func persistentVolumeByName(volumes []inventory.PersistentVolume) map[string]inventory.PersistentVolume {
	out := make(map[string]inventory.PersistentVolume, len(volumes))
	for _, volume := range volumes {
		out[volume.Name] = volume
	}
	return out
}

func pvcByNamespacedName(claims []inventory.PersistentVolumeClaim) map[string]inventory.PersistentVolumeClaim {
	out := make(map[string]inventory.PersistentVolumeClaim, len(claims))
	for _, claim := range claims {
		out[namespaced(claim.ObjectRef)] = claim
	}
	return out
}

func csiNodeCount(nodes []inventory.CSINode, driver string) int {
	count := 0
	for _, node := range nodes {
		for _, item := range node.Drivers {
			if item.Name == driver {
				count++
				break
			}
		}
	}
	return count
}

func nodeRuntimeCounts(nodes []inventory.Node) map[string]int {
	counts := map[string]int{}
	for _, node := range nodes {
		counts[value(node.ContainerRuntime)]++
	}
	return counts
}

type countEntry struct {
	name  string
	count int
}

func sortedCounts(counts map[string]int) []countEntry {
	out := make([]countEntry, 0, len(counts))
	for name, count := range counts {
		out = append(out, countEntry{name: name, count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
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

func componentCandidates(workloads []inventory.Workload, kind string) []inventory.Workload {
	var out []inventory.Workload
	for _, workload := range workloads {
		if workload.Kind != "DaemonSet" && workload.Kind != "Deployment" && workload.Kind != "StatefulSet" {
			continue
		}
		text := strings.ToLower(workload.Name + " " + workload.Namespace + " " + images(workload))
		if kind == "cni" && containsAny(text, []string{"aws-node", "vpc-cni", "cilium", "calico", "flannel", "weave", "antrea", "canal"}) {
			out = append(out, workload)
		}
		if kind == "csi" && containsAny(text, []string{"csi", "ebs-csi", "efs-csi", "external-provisioner", "external-attacher", "node-driver-registrar"}) {
			out = append(out, workload)
		}
	}
	return sortedWorkloads(out)
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func images(workload inventory.Workload) string {
	var values []string
	for _, container := range workload.Containers {
		if container.Image != "" {
			values = append(values, container.Image)
		}
	}
	for _, container := range workload.InitContainers {
		if container.Image != "" {
			values = append(values, container.Image)
		}
	}
	return stringList(uniqueStrings(values))
}

func refsValue(refs []inventory.ObjectRef) string {
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, refValue(ref))
	}
	return stringList(values)
}

func volumesValue(volumes []inventory.Volume) string {
	values := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		if volume.CSI != "" {
			values = append(values, volume.Name+"->csi/"+volume.CSI)
			continue
		}
		if volume.PersistentVolumeClaim != "" {
			values = append(values, volume.Name+"->pvc/"+volume.PersistentVolumeClaim)
		}
	}
	return stringList(values)
}

func gatewayListeners(listeners []inventory.GatewayListener) string {
	if len(listeners) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(listeners))
	for _, listener := range listeners {
		parts = append(parts, fmt.Sprintf("%s:%s/%d", value(listener.Name), value(listener.Protocol), listener.Port))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func parentRefsValue(refs []inventory.GatewayParentRef) string {
	if len(refs) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		value := ref.Kind
		if value == "" {
			value = "Gateway"
		}
		if ref.Namespace != "" {
			value += "/" + ref.Namespace
		}
		value += "/" + ref.Name
		if ref.SectionName != "" {
			value += "#" + ref.SectionName
		}
		parts = append(parts, value)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func grantRefsValue(refs []inventory.GatewayGrantRef) string {
	if len(refs) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(refs))
	for _, ref := range refs {
		value := gatewayKind(ref.Group, ref.Kind)
		if ref.Namespace != "" {
			value += " ns=" + ref.Namespace
		}
		if ref.Name != "" {
			value += " name=" + ref.Name
		}
		parts = append(parts, value)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func gatewayKind(group, kind string) string {
	if group == "" {
		return value(kind)
	}
	if kind == "" {
		return group
	}
	return group + "/" + kind
}

func routeRuleSummary(rules []inventory.GatewayRouteRule) string {
	if len(rules) == 0 {
		return "rules=0"
	}
	matches := make([]string, 0)
	for _, rule := range rules {
		matches = append(matches, rule.Matches...)
	}
	if len(matches) == 0 {
		return fmt.Sprintf("rules=%d", len(rules))
	}
	sort.Strings(matches)
	return strings.Join(matches, ",")
}

func routeBackends(rules []inventory.GatewayRouteRule) string {
	refs := make([]inventory.ObjectRef, 0)
	for _, rule := range rules {
		refs = append(refs, rule.BackendRefs...)
	}
	return refsValue(dedupeObjectRefs(refs))
}

func dedupeObjectRefs(refs []inventory.ObjectRef) []inventory.ObjectRef {
	seen := map[string]struct{}{}
	out := make([]inventory.ObjectRef, 0, len(refs))
	for _, ref := range refs {
		key := strings.Join([]string{ref.APIVersion, ref.Kind, ref.Namespace, ref.Name}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func servicePorts(ports []inventory.ServicePort) string {
	values := make([]string, 0, len(ports))
	for _, port := range ports {
		target := port.TargetPort
		if target == "" {
			target = fmt.Sprintf("%d", port.Port)
		}
		values = append(values, fmt.Sprintf("%s/%d->%s", value(port.Protocol), port.Port, target))
	}
	return stringList(values)
}

func mapValue(values map[string]string) string {
	if len(values) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	return strings.Join(parts, ",")
}

func crdType(group, kind string) string {
	if group == "" {
		return value(kind)
	}
	if kind == "" {
		return group
	}
	return group + "/" + kind
}

func stringList(values []string) string {
	values = uniqueStrings(values)
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
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

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func namespaced(ref inventory.ObjectRef) string {
	return namespacedName(ref.Namespace, ref.Name)
}

func namespacedName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func refValue(ref inventory.ObjectRef) string {
	if ref.Name == "" {
		return "-"
	}
	prefix := strings.ToLower(ref.Kind)
	if prefix == "" {
		prefix = "object"
	}
	return prefix + "/" + namespaced(ref)
}

func pathValue(path string) string {
	if path == "" {
		return ""
	}
	return path
}

func boolPtrValue(value *bool) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%t", *value)
}

func int32PtrValue(value *int32) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}

func serviceCIDR(network inventory.NetworkConfig) string {
	if network.ServiceIPv4CIDR != "" {
		return network.ServiceIPv4CIDR
	}
	if network.ServiceIPv6CIDR != "" {
		return network.ServiceIPv6CIDR
	}
	return "-"
}

func value(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func int32Value(v *int32) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}

func marker(status string) string {
	switch strings.ToLower(status) {
	case "active", "complete":
		return "ok"
	case "creating", "updating", "pending":
		return ".."
	case "denied", "degraded", "failed":
		return "!!"
	case "partial":
		return "~~"
	case "unavailable":
		return "xx"
	default:
		return "--"
	}
}
