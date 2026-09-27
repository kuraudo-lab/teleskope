package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/topology"
)

func TestRecordedEKSFixtureInfrastructureRelationships(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "recorded", "eks-demo-snapshot.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot inventory.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode %s: %v", fixture, err)
	}

	infrastructure := snapshot.EKS.Infrastructure
	if len(infrastructure.Instances) != 2 || len(infrastructure.AutoScalingGroups) != 1 || len(infrastructure.VPCs) != 1 || len(infrastructure.Subnets) != 3 || len(infrastructure.SecurityGroups) != 3 {
		t.Fatalf("unexpected infrastructure fixture shape: %+v", infrastructure)
	}

	nodegroups := map[string]inventory.Nodegroup{}
	for _, nodegroup := range snapshot.EKS.Nodegroups {
		nodegroups[nodegroup.Name] = nodegroup
	}
	asgs := map[string]inventory.AutoScalingGroup{}
	for _, group := range infrastructure.AutoScalingGroups {
		asgs[group.Name] = group
		if _, ok := nodegroups[group.NodegroupName]; !ok {
			t.Errorf("ASG %s references missing nodegroup %s", group.Name, group.NodegroupName)
		}
	}
	vpcs := map[string]bool{}
	for _, vpc := range infrastructure.VPCs {
		vpcs[vpc.VPCID] = true
	}
	subnets := map[string]inventory.SubnetDetail{}
	for _, subnet := range infrastructure.Subnets {
		subnets[subnet.SubnetID] = subnet
		if !vpcs[subnet.VPCID] {
			t.Errorf("subnet %s references missing VPC %s", subnet.SubnetID, subnet.VPCID)
		}
	}
	securityGroups := map[string]bool{}
	for _, group := range infrastructure.SecurityGroups {
		securityGroups[group.GroupID] = true
		if !vpcs[group.VPCID] {
			t.Errorf("security group %s references missing VPC %s", group.GroupID, group.VPCID)
		}
	}
	nodes := map[string]inventory.Node{}
	for _, node := range snapshot.Kubernetes.Nodes {
		nodes[node.Name] = node
	}
	instances := map[string]bool{}
	for _, instance := range infrastructure.Instances {
		instances[instance.InstanceID] = true
		if _, ok := nodegroups[instance.NodegroupName]; !ok {
			t.Errorf("instance %s references missing nodegroup %s", instance.InstanceID, instance.NodegroupName)
		}
		if _, ok := asgs[instance.AutoScalingGroupName]; !ok {
			t.Errorf("instance %s references missing ASG %s", instance.InstanceID, instance.AutoScalingGroupName)
		}
		if _, ok := subnets[instance.SubnetID]; !ok {
			t.Errorf("instance %s references missing subnet %s", instance.InstanceID, instance.SubnetID)
		}
		for _, groupID := range instance.SecurityGroupIDs {
			if !securityGroups[groupID] {
				t.Errorf("instance %s references missing security group %s", instance.InstanceID, groupID)
			}
		}
		node, ok := nodes[instance.KubernetesNodeName]
		if !ok {
			t.Errorf("instance %s references missing Kubernetes node %s", instance.InstanceID, instance.KubernetesNodeName)
		} else if !strings.HasSuffix(node.ProviderID, "/"+instance.InstanceID) {
			t.Errorf("node %s provider ID %s does not reference %s", node.Name, node.ProviderID, instance.InstanceID)
		}
	}
	for _, group := range infrastructure.AutoScalingGroups {
		for _, instanceID := range group.InstanceIDs {
			if !instances[instanceID] {
				t.Errorf("ASG %s references missing instance %s", group.Name, instanceID)
			}
		}
	}
	for _, subnetID := range snapshot.EKS.Cluster.VPC.SubnetIDs {
		if _, ok := subnets[subnetID]; !ok {
			t.Errorf("cluster references missing subnet detail %s", subnetID)
		}
	}
}

func TestRecordedEKSFixtureProjectsValidTopologyMetrics(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "recorded", "eks-demo-snapshot.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot inventory.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	graph := topology.Project(&snapshot, 1)
	if err := graph.Validate(); err != nil {
		t.Fatalf("recorded topology graph: %v", err)
	}
	var series int
	nodeNames := map[string]bool{}
	for _, node := range graph.Nodes {
		series += len(node.Metrics)
		nodeNames[node.Name] = true
	}
	if series == 0 {
		t.Fatal("recorded topology did not project any metric series")
	}
	for _, name := range []string{"demo-node-1", "demo-node-2"} {
		if !nodeNames[name] {
			t.Fatalf("recorded topology collapsed %s", name)
		}
	}
	uids := map[string]string{}
	objects := map[string]string{}
	addObject := func(ref inventory.ObjectRef) {
		identity := strings.ToLower(ref.Kind) + "/" + ref.Namespace + "/" + ref.Name
		if ref.UID == "" {
			t.Errorf("recorded object %s has no sanitized UID", identity)
			return
		}
		if previous := uids[ref.UID]; previous != "" {
			t.Errorf("recorded objects %s and %s share sanitized UID %s", previous, identity, ref.UID)
		}
		uids[ref.UID] = identity
		objects[identity] = ref.UID
	}
	for _, node := range snapshot.Kubernetes.Nodes {
		addObject(node.ObjectRef)
	}
	for _, workload := range snapshot.Kubernetes.Workloads {
		addObject(workload.ObjectRef)
	}
	for _, pod := range snapshot.Kubernetes.Pods {
		addObject(pod.ObjectRef)
	}
	for _, service := range snapshot.Kubernetes.Services {
		addObject(service.ObjectRef)
	}
	for _, endpointSlice := range snapshot.Kubernetes.EndpointSlices {
		addObject(endpointSlice.ObjectRef)
	}
	assertReference := func(ref inventory.ObjectRef, namespace string) {
		if ref.Name == "" {
			return
		}
		if ref.Namespace == "" {
			ref.Namespace = namespace
		}
		identity := strings.ToLower(ref.Kind) + "/" + ref.Namespace + "/" + ref.Name
		if want := objects[identity]; want == "" || ref.UID != want {
			t.Errorf("recorded reference %s has UID %s, want %s", identity, ref.UID, want)
		}
	}
	for _, workload := range snapshot.Kubernetes.Workloads {
		for _, owner := range workload.OwnerReferences {
			assertReference(owner, workload.Namespace)
		}
	}
	for _, pod := range snapshot.Kubernetes.Pods {
		for _, owner := range pod.OwnerReferences {
			assertReference(owner, pod.Namespace)
		}
	}
	for _, endpointSlice := range snapshot.Kubernetes.EndpointSlices {
		for _, endpoint := range endpointSlice.Endpoints {
			assertReference(endpoint.TargetRef, endpointSlice.Namespace)
		}
	}
}

func TestRecordedEKSFixtureSanitizesEveryKubernetesObjectIdentity(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "recorded", "eks-demo-snapshot.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	kubernetes, ok := document["kubernetes"]
	if !ok {
		t.Fatal("recorded fixture has no kubernetes inventory")
	}
	identityUIDs, uidIdentities := map[string]string{}, map[string]string{}
	var count int
	var walk func(any, string)
	walk = func(value any, namespace string) {
		switch item := value.(type) {
		case map[string]any:
			if candidate, ok := item["namespace"].(string); ok && candidate != "" {
				namespace = candidate
			}
			kind, kindOK := item["kind"].(string)
			name, nameOK := item["name"].(string)
			uid, uidOK := item["uid"].(string)
			if kindOK && nameOK && uidOK && kind != "" && name != "" {
				count++
				identity := strings.ToLower(kind) + "/" + namespace + "/" + name
				if !strings.HasPrefix(uid, "demo-uid-") {
					t.Errorf("recorded identity %s has non-synthetic UID %s", identity, uid)
				}
				if previous := identityUIDs[identity]; previous != "" && previous != uid {
					t.Errorf("recorded identity %s has inconsistent UIDs %s and %s", identity, previous, uid)
				}
				if previous := uidIdentities[uid]; previous != "" && previous != identity {
					t.Errorf("recorded identities %s and %s share UID %s", previous, identity, uid)
				}
				identityUIDs[identity], uidIdentities[uid] = uid, identity
			}
			for _, child := range item {
				walk(child, namespace)
			}
		case []any:
			for _, child := range item {
				walk(child, namespace)
			}
		}
	}
	walk(kubernetes, "")
	if count < 500 {
		t.Fatalf("validated only %d Kubernetes identities, want complete fixture coverage", count)
	}
}
