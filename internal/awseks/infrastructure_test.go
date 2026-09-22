package awseks

import (
	"context"
	"errors"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	autoscalingtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

type fakeAutoScaling struct {
	output *autoscaling.DescribeAutoScalingGroupsOutput
	err    error
	inputs []*autoscaling.DescribeAutoScalingGroupsInput
}

func (f *fakeAutoScaling) DescribeAutoScalingGroups(_ context.Context, input *autoscaling.DescribeAutoScalingGroupsInput, _ ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	f.inputs = append(f.inputs, input)
	return f.output, f.err
}

type fakeEC2 struct {
	instances      *ec2.DescribeInstancesOutput
	vpcs           *ec2.DescribeVpcsOutput
	subnets        *ec2.DescribeSubnetsOutput
	routeTables    *ec2.DescribeRouteTablesOutput
	routeTablesErr error
	securityGroups *ec2.DescribeSecurityGroupsOutput
	instanceInputs []*ec2.DescribeInstancesInput
}

func (f *fakeEC2) DescribeInstances(_ context.Context, input *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	f.instanceInputs = append(f.instanceInputs, input)
	return f.instances, nil
}

func (f *fakeEC2) DescribeVpcs(context.Context, *ec2.DescribeVpcsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return f.vpcs, nil
}

func (f *fakeEC2) DescribeVpcAttribute(_ context.Context, input *ec2.DescribeVpcAttributeInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error) {
	value := &ec2types.AttributeBooleanValue{Value: awssdk.Bool(true)}
	if input.Attribute == ec2types.VpcAttributeNameEnableDnsSupport {
		return &ec2.DescribeVpcAttributeOutput{EnableDnsSupport: value}, nil
	}
	return &ec2.DescribeVpcAttributeOutput{EnableDnsHostnames: value}, nil
}

func (f *fakeEC2) DescribeSubnets(context.Context, *ec2.DescribeSubnetsInput, ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return f.subnets, nil
}

func (f *fakeEC2) DescribeRouteTables(context.Context, *ec2.DescribeRouteTablesInput, ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	return f.routeTables, f.routeTablesErr
}

func (f *fakeEC2) DescribeSecurityGroups(context.Context, *ec2.DescribeSecurityGroupsInput, ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return f.securityGroups, nil
}

func TestCollectInfrastructureBuildsRelationships(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	snapshot := infrastructureTestSnapshot()
	asgClient := &fakeAutoScaling{output: &autoscaling.DescribeAutoScalingGroupsOutput{AutoScalingGroups: []autoscalingtypes.AutoScalingGroup{{
		AutoScalingGroupName: awssdk.String("demo-workers"),
		AutoScalingGroupARN:  awssdk.String("arn:aws:autoscaling:example"),
		MinSize:              awssdk.Int32(1),
		MaxSize:              awssdk.Int32(3),
		DesiredCapacity:      awssdk.Int32(2),
		AvailabilityZones:    []string{"example-1a"},
		VPCZoneIdentifier:    awssdk.String("subnet-example"),
		HealthCheckType:      awssdk.String("EC2"),
		Instances:            []autoscalingtypes.Instance{{InstanceId: awssdk.String("i-example")}},
		LaunchTemplate:       &autoscalingtypes.LaunchTemplateSpecification{LaunchTemplateId: awssdk.String("lt-example"), Version: awssdk.String("7")},
	}}}}
	ec2Client := infrastructureFakeEC2(nil)

	collectInfrastructure(context.Background(), asgClient, ec2Client, snapshot, now)
	if len(asgClient.inputs) != 1 || len(asgClient.inputs[0].AutoScalingGroupNames) != 1 || asgClient.inputs[0].AutoScalingGroupNames[0] != "demo-workers" {
		t.Fatalf("expected reference-scoped ASG request, got %#v", asgClient.inputs)
	}
	if len(ec2Client.instanceInputs) != 1 || len(ec2Client.instanceInputs[0].InstanceIds) != 1 || ec2Client.instanceInputs[0].InstanceIds[0] != "i-example" {
		t.Fatalf("expected reference-scoped EC2 request, got %#v", ec2Client.instanceInputs)
	}

	if got := snapshot.EKS.Infrastructure.AutoScalingGroups; len(got) != 1 || got[0].NodegroupName != "workers" || got[0].InstanceIDs[0] != "i-example" {
		t.Fatalf("unexpected Auto Scaling groups: %#v", got)
	}
	if got := snapshot.EKS.Infrastructure.Instances; len(got) != 1 || got[0].NodegroupName != "workers" || got[0].AutoScalingGroupName != "demo-workers" || got[0].KubernetesNodeName != "ip-192-0-2-10.example.internal" || got[0].LaunchTemplateID != "lt-example" {
		t.Fatalf("unexpected instances: %#v", got)
	}
	if got := snapshot.EKS.Infrastructure.VPCs; len(got) != 1 || got[0].DNSSupport == nil || !*got[0].DNSSupport || got[0].DNSHostnames == nil || !*got[0].DNSHostnames {
		t.Fatalf("unexpected VPCs: %#v", got)
	}
	if got := snapshot.EKS.Infrastructure.Subnets; len(got) != 1 || got[0].RouteTableID != "rtb-example" || len(got[0].NATGatewayIDs) != 1 || got[0].NATGatewayIDs[0] != "nat-example" {
		t.Fatalf("unexpected subnets: %#v", got)
	}
	if got := snapshot.EKS.Infrastructure.SecurityGroups; len(got) != 2 || got[0].GroupID != "sg-cluster" || got[1].GroupID != "sg-node" {
		t.Fatalf("unexpected security groups: %#v", got)
	}
	for _, resource := range []string{"DescribeAutoScalingGroups", "DescribeInstances", "DescribeVpcs", "DescribeSubnets", "DescribeRouteTables", "DescribeSecurityGroups"} {
		assertCoverage(t, snapshot.Coverage, resource, "complete")
	}
}

func TestCollectInfrastructureRetainsSubnetsWhenRouteTablesFail(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	snapshot := infrastructureTestSnapshot()
	ec2Client := infrastructureFakeEC2(errors.New("route tables denied"))

	collectInfrastructure(context.Background(), &fakeAutoScaling{output: &autoscaling.DescribeAutoScalingGroupsOutput{}}, ec2Client, snapshot, now)

	if got := snapshot.EKS.Infrastructure.Subnets; len(got) != 1 || got[0].SubnetID != "subnet-example" || got[0].RouteTableID != "" {
		t.Fatalf("expected subnet facts without route data, got %#v", got)
	}
	assertCoverage(t, snapshot.Coverage, "DescribeRouteTables", "partial")
}

func infrastructureTestSnapshot() *inventory.Snapshot {
	return &inventory.Snapshot{EKS: inventory.EKSInventory{
		Cluster: inventory.Cluster{VPC: inventory.VPCConfig{
			VPCID:                  "vpc-example",
			SubnetIDs:              []string{"subnet-example"},
			SecurityGroupIDs:       []string{"sg-node"},
			ClusterSecurityGroupID: "sg-cluster",
		}},
		Nodegroups: []inventory.Nodegroup{{Name: "workers", AutoScalingGroups: []string{"demo-workers"}, Subnets: []string{"subnet-example"}}},
	}}
}

func infrastructureFakeEC2(routeErr error) *fakeEC2 {
	return &fakeEC2{
		instances: &ec2.DescribeInstancesOutput{Reservations: []ec2types.Reservation{{Instances: []ec2types.Instance{{
			InstanceId:       awssdk.String("i-example"),
			InstanceType:     ec2types.InstanceTypeM7iLarge,
			Architecture:     ec2types.ArchitectureValuesX8664,
			ImageId:          awssdk.String("ami-example"),
			PrivateDnsName:   awssdk.String("ip-192-0-2-10.example.internal"),
			PrivateIpAddress: awssdk.String("192.0.2.10"),
			SubnetId:         awssdk.String("subnet-example"),
			VpcId:            awssdk.String("vpc-example"),
			State:            &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning},
			Placement:        &ec2types.Placement{AvailabilityZone: awssdk.String("example-1a")},
			SecurityGroups:   []ec2types.GroupIdentifier{{GroupId: awssdk.String("sg-node")}},
			Tags:             []ec2types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String("demo-worker")}},
		}}}}},
		vpcs: &ec2.DescribeVpcsOutput{Vpcs: []ec2types.Vpc{{
			VpcId: awssdk.String("vpc-example"), CidrBlock: awssdk.String("192.0.2.0/24"), State: ec2types.VpcStateAvailable,
		}}},
		subnets: &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{{
			SubnetId: awssdk.String("subnet-example"), VpcId: awssdk.String("vpc-example"), CidrBlock: awssdk.String("192.0.2.0/28"), AvailabilityZone: awssdk.String("example-1a"), State: ec2types.SubnetStateAvailable,
		}}},
		routeTables: &ec2.DescribeRouteTablesOutput{RouteTables: []ec2types.RouteTable{{
			RouteTableId: awssdk.String("rtb-example"), VpcId: awssdk.String("vpc-example"),
			Associations: []ec2types.RouteTableAssociation{{SubnetId: awssdk.String("subnet-example")}},
			Routes:       []ec2types.Route{{NatGatewayId: awssdk.String("nat-example")}},
		}}},
		routeTablesErr: routeErr,
		securityGroups: &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{
			{GroupId: awssdk.String("sg-node"), GroupName: awssdk.String("nodes"), VpcId: awssdk.String("vpc-example")},
			{GroupId: awssdk.String("sg-cluster"), GroupName: awssdk.String("cluster"), VpcId: awssdk.String("vpc-example")},
		}},
	}
}

func assertCoverage(t *testing.T, coverage []inventory.CoverageItem, resource, status string) {
	t.Helper()
	for _, item := range coverage {
		if item.Resource == resource {
			if item.Status != status {
				t.Fatalf("coverage %s status=%s, want %s", resource, item.Status, status)
			}
			return
		}
	}
	t.Fatalf("coverage %s not found in %#v", resource, coverage)
}
