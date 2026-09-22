package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
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
