package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// Human writes a compact terminal summary inspired by small TUI status tools.
func Human(w io.Writer, snapshot *inventory.Snapshot) error {
	cluster := snapshot.EKS.Cluster
	fmt.Fprintf(w, "teleskope  %s\n", cluster.Name)
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

	fmt.Fprintf(w, "coverage   %d\n", len(snapshot.Coverage))
	for _, item := range snapshot.Coverage {
		fmt.Fprintf(w, "  %s  %s/%s  objects=%d", marker(item.Status), item.Area, item.Resource, item.ObjectCount)
		if item.Reason != "" {
			fmt.Fprintf(w, "  %s", item.Reason)
		}
		fmt.Fprintln(w)
	}
	return nil
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
	default:
		return "--"
	}
}
