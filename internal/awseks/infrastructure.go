package awseks

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	autoscalingtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

type autoScalingAPI interface {
	DescribeAutoScalingGroups(context.Context, *autoscaling.DescribeAutoScalingGroupsInput, ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
}

type ec2API interface {
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeVpcs(context.Context, *ec2.DescribeVpcsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeVpcAttribute(context.Context, *ec2.DescribeVpcAttributeInput, ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error)
	DescribeSubnets(context.Context, *ec2.DescribeSubnetsInput, ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeRouteTables(context.Context, *ec2.DescribeRouteTablesInput, ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DescribeSecurityGroups(context.Context, *ec2.DescribeSecurityGroupsInput, ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
}

func collectInfrastructure(ctx context.Context, asgClient autoScalingAPI, ec2Client ec2API, snapshot *inventory.Snapshot, now time.Time) {
	infra := &snapshot.EKS.Infrastructure
	nodegroupByASG := nodegroupASGIndex(snapshot.EKS.Nodegroups)

	groups, err := collectAutoScalingGroups(ctx, asgClient, sortedMapKeys(nodegroupByASG), nodegroupByASG)
	infra.AutoScalingGroups = groups
	recordInfrastructureCoverage(&snapshot.Coverage, "autoscaling", "DescribeAutoScalingGroups", len(groups), err, now)

	asgByInstance := map[string]string{}
	groupByName := map[string]inventory.AutoScalingGroup{}
	for _, group := range groups {
		groupByName[group.Name] = group
		for _, instanceID := range group.InstanceIDs {
			asgByInstance[instanceID] = group.Name
		}
	}
	instances, err := collectEC2Instances(ctx, ec2Client, sortedMapKeys(asgByInstance), asgByInstance, nodegroupByASG, groupByName)
	infra.Instances = instances
	recordInfrastructureCoverage(&snapshot.Coverage, "ec2", "DescribeInstances", len(instances), err, now)

	vpcIDs := stringSet(snapshot.EKS.Cluster.VPC.VPCID)
	subnetIDs := stringSet(snapshot.EKS.Cluster.VPC.SubnetIDs...)
	securityGroupIDs := stringSet(snapshot.EKS.Cluster.VPC.SecurityGroupIDs...)
	addString(securityGroupIDs, snapshot.EKS.Cluster.VPC.ClusterSecurityGroupID)
	for _, nodegroup := range snapshot.EKS.Nodegroups {
		addStrings(subnetIDs, nodegroup.Subnets...)
		addString(securityGroupIDs, nodegroup.RemoteAccessSecurityGroupID)
	}
	for _, group := range groups {
		addStrings(subnetIDs, group.SubnetIDs...)
	}
	for _, instance := range instances {
		addString(vpcIDs, instance.VPCID)
		addString(subnetIDs, instance.SubnetID)
		addStrings(securityGroupIDs, instance.SecurityGroupIDs...)
	}

	vpcs, err := collectVPCs(ctx, ec2Client, sortedMapKeys(vpcIDs))
	infra.VPCs = vpcs
	recordInfrastructureCoverage(&snapshot.Coverage, "ec2", "DescribeVpcs", len(vpcs), err, now)

	subnets, err := collectSubnets(ctx, ec2Client, sortedMapKeys(subnetIDs))
	recordInfrastructureCoverage(&snapshot.Coverage, "ec2", "DescribeSubnets", len(subnets), err, now)
	routeTables, routeErr := collectRouteTables(ctx, ec2Client, sortedMapKeys(vpcIDs))
	recordInfrastructureCoverage(&snapshot.Coverage, "ec2", "DescribeRouteTables", len(routeTables), routeErr, now)
	infra.Subnets = applyRouteTables(subnets, routeTables)

	securityGroups, err := collectSecurityGroups(ctx, ec2Client, sortedMapKeys(securityGroupIDs))
	infra.SecurityGroups = securityGroups
	recordInfrastructureCoverage(&snapshot.Coverage, "ec2", "DescribeSecurityGroups", len(securityGroups), err, now)
}

func recordInfrastructureCoverage(coverage *[]inventory.CoverageItem, area, resource string, count int, err error, now time.Time) {
	if err != nil {
		*coverage = append(*coverage, partial(area, resource, count, err, now))
		return
	}
	*coverage = append(*coverage, complete(area, resource, count, now))
}

func nodegroupASGIndex(nodegroups []inventory.Nodegroup) map[string]string {
	out := map[string]string{}
	for _, nodegroup := range nodegroups {
		for _, name := range nodegroup.AutoScalingGroups {
			if name != "" {
				out[name] = nodegroup.Name
			}
		}
	}
	return out
}

func collectAutoScalingGroups(ctx context.Context, client autoScalingAPI, names []string, nodegroupByASG map[string]string) ([]inventory.AutoScalingGroup, error) {
	if len(names) == 0 {
		return nil, nil
	}
	groups := []inventory.AutoScalingGroup{}
	seen := map[string]struct{}{}
	var collectedErr error
	for _, batch := range batches(names, 50) {
		input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: batch}
		for {
			output, err := client.DescribeAutoScalingGroups(ctx, input)
			if err != nil {
				collectedErr = errors.Join(collectedErr, err)
				break
			}
			for _, group := range output.AutoScalingGroups {
				name := awssdk.ToString(group.AutoScalingGroupName)
				seen[name] = struct{}{}
				groups = append(groups, mapAutoScalingGroup(group, nodegroupByASG[name]))
			}
			if output.NextToken == nil || awssdk.ToString(output.NextToken) == "" {
				break
			}
			input.NextToken = output.NextToken
		}
	}
	collectedErr = errors.Join(collectedErr, missingRequested("Auto Scaling groups", names, seen))
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups, collectedErr
}

func mapAutoScalingGroup(group autoscalingtypes.AutoScalingGroup, nodegroupName string) inventory.AutoScalingGroup {
	out := inventory.AutoScalingGroup{
		Name:              awssdk.ToString(group.AutoScalingGroupName),
		ARN:               awssdk.ToString(group.AutoScalingGroupARN),
		NodegroupName:     nodegroupName,
		MinSize:           awssdk.ToInt32(group.MinSize),
		MaxSize:           awssdk.ToInt32(group.MaxSize),
		DesiredCapacity:   awssdk.ToInt32(group.DesiredCapacity),
		AvailabilityZones: append([]string(nil), group.AvailabilityZones...),
		SubnetIDs:         splitCommaSeparated(awssdk.ToString(group.VPCZoneIdentifier)),
		HealthCheckType:   awssdk.ToString(group.HealthCheckType),
	}
	for _, instance := range group.Instances {
		addStringSlice(&out.InstanceIDs, awssdk.ToString(instance.InstanceId))
	}
	if group.LaunchTemplate != nil {
		out.LaunchTemplateID = awssdk.ToString(group.LaunchTemplate.LaunchTemplateId)
		out.LaunchTemplateVersion = awssdk.ToString(group.LaunchTemplate.Version)
	} else if group.MixedInstancesPolicy != nil && group.MixedInstancesPolicy.LaunchTemplate != nil && group.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification != nil {
		template := group.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification
		out.LaunchTemplateID = awssdk.ToString(template.LaunchTemplateId)
		out.LaunchTemplateVersion = awssdk.ToString(template.Version)
	}
	sort.Strings(out.InstanceIDs)
	return out
}

func collectEC2Instances(ctx context.Context, client ec2API, ids []string, asgByInstance, nodegroupByASG map[string]string, groupByName map[string]inventory.AutoScalingGroup) ([]inventory.EC2Instance, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	instances := []inventory.EC2Instance{}
	seen := map[string]struct{}{}
	var collectedErr error
	for _, batch := range batches(ids, 1000) {
		input := &ec2.DescribeInstancesInput{InstanceIds: batch}
		for {
			output, err := client.DescribeInstances(ctx, input)
			if err != nil {
				collectedErr = errors.Join(collectedErr, err)
				break
			}
			for _, reservation := range output.Reservations {
				for _, instance := range reservation.Instances {
					instanceID := awssdk.ToString(instance.InstanceId)
					seen[instanceID] = struct{}{}
					asgName := asgByInstance[instanceID]
					item := mapEC2Instance(instance, asgName, nodegroupByASG[asgName])
					item.LaunchTemplateID = groupByName[asgName].LaunchTemplateID
					item.LaunchTemplateVersion = groupByName[asgName].LaunchTemplateVersion
					instances = append(instances, item)
				}
			}
			if output.NextToken == nil || awssdk.ToString(output.NextToken) == "" {
				break
			}
			input.NextToken = output.NextToken
		}
	}
	collectedErr = errors.Join(collectedErr, missingRequested("EC2 instances", ids, seen))
	sort.Slice(instances, func(i, j int) bool { return instances[i].InstanceID < instances[j].InstanceID })
	return instances, collectedErr
}

func mapEC2Instance(instance ec2types.Instance, asgName, nodegroupName string) inventory.EC2Instance {
	tags := mapEC2Tags(instance.Tags)
	out := inventory.EC2Instance{
		InstanceID:           awssdk.ToString(instance.InstanceId),
		Name:                 tags["Name"],
		InstanceType:         string(instance.InstanceType),
		Architecture:         string(instance.Architecture),
		PrivateIP:            awssdk.ToString(instance.PrivateIpAddress),
		SubnetID:             awssdk.ToString(instance.SubnetId),
		VPCID:                awssdk.ToString(instance.VpcId),
		NodegroupName:        nodegroupName,
		AutoScalingGroupName: asgName,
		KubernetesNodeName:   awssdk.ToString(instance.PrivateDnsName),
		ImageID:              awssdk.ToString(instance.ImageId),
		Tags:                 tags,
	}
	if instance.State != nil {
		out.State = string(instance.State.Name)
	}
	if instance.Placement != nil {
		out.AvailabilityZone = awssdk.ToString(instance.Placement.AvailabilityZone)
	}
	for _, group := range instance.SecurityGroups {
		addStringSlice(&out.SecurityGroupIDs, awssdk.ToString(group.GroupId))
	}
	sort.Strings(out.SecurityGroupIDs)
	return out
}

func collectVPCs(ctx context.Context, client ec2API, ids []string) ([]inventory.VPCDetail, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	output, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{VpcIds: ids})
	if err != nil {
		return nil, err
	}
	vpcs := make([]inventory.VPCDetail, 0, len(output.Vpcs))
	seen := map[string]struct{}{}
	var collectedErr error
	for _, vpc := range output.Vpcs {
		seen[awssdk.ToString(vpc.VpcId)] = struct{}{}
		item := mapVPC(vpc)
		dnsSupport, attrErr := client.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{VpcId: vpc.VpcId, Attribute: ec2types.VpcAttributeNameEnableDnsSupport})
		if attrErr != nil {
			collectedErr = errors.Join(collectedErr, fmt.Errorf("describe VPC %s DNS support: %w", item.VPCID, attrErr))
		} else if dnsSupport.EnableDnsSupport != nil {
			item.DNSSupport = dnsSupport.EnableDnsSupport.Value
		}
		dnsHostnames, attrErr := client.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{VpcId: vpc.VpcId, Attribute: ec2types.VpcAttributeNameEnableDnsHostnames})
		if attrErr != nil {
			collectedErr = errors.Join(collectedErr, fmt.Errorf("describe VPC %s DNS hostnames: %w", item.VPCID, attrErr))
		} else if dnsHostnames.EnableDnsHostnames != nil {
			item.DNSHostnames = dnsHostnames.EnableDnsHostnames.Value
		}
		vpcs = append(vpcs, item)
	}
	collectedErr = errors.Join(collectedErr, missingRequested("VPCs", ids, seen))
	sort.Slice(vpcs, func(i, j int) bool { return vpcs[i].VPCID < vpcs[j].VPCID })
	return vpcs, collectedErr
}

func mapVPC(vpc ec2types.Vpc) inventory.VPCDetail {
	return inventory.VPCDetail{
		VPCID:         awssdk.ToString(vpc.VpcId),
		CIDR:          awssdk.ToString(vpc.CidrBlock),
		State:         string(vpc.State),
		Tenancy:       string(vpc.InstanceTenancy),
		DHCPOptionsID: awssdk.ToString(vpc.DhcpOptionsId),
		Tags:          mapEC2Tags(vpc.Tags),
	}
}

func collectSubnets(ctx context.Context, client ec2API, ids []string) ([]inventory.SubnetDetail, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	subnets := []inventory.SubnetDetail{}
	seen := map[string]struct{}{}
	var collectedErr error
	for _, batch := range batches(ids, 200) {
		input := &ec2.DescribeSubnetsInput{SubnetIds: batch}
		for {
			output, err := client.DescribeSubnets(ctx, input)
			if err != nil {
				collectedErr = errors.Join(collectedErr, err)
				break
			}
			for _, subnet := range output.Subnets {
				seen[awssdk.ToString(subnet.SubnetId)] = struct{}{}
				subnets = append(subnets, mapSubnet(subnet))
			}
			if output.NextToken == nil || awssdk.ToString(output.NextToken) == "" {
				break
			}
			input.NextToken = output.NextToken
		}
	}
	collectedErr = errors.Join(collectedErr, missingRequested("subnets", ids, seen))
	sort.Slice(subnets, func(i, j int) bool { return subnets[i].SubnetID < subnets[j].SubnetID })
	return subnets, collectedErr
}

func mapSubnet(subnet ec2types.Subnet) inventory.SubnetDetail {
	return inventory.SubnetDetail{
		SubnetID:                awssdk.ToString(subnet.SubnetId),
		VPCID:                   awssdk.ToString(subnet.VpcId),
		CIDR:                    awssdk.ToString(subnet.CidrBlock),
		AvailabilityZone:        awssdk.ToString(subnet.AvailabilityZone),
		AvailabilityZoneID:      awssdk.ToString(subnet.AvailabilityZoneId),
		State:                   string(subnet.State),
		AvailableIPAddressCount: awssdk.ToInt32(subnet.AvailableIpAddressCount),
		MapPublicIPOnLaunch:     awssdk.ToBool(subnet.MapPublicIpOnLaunch),
		Tags:                    mapEC2Tags(subnet.Tags),
	}
}

func collectRouteTables(ctx context.Context, client ec2API, vpcIDs []string) ([]ec2types.RouteTable, error) {
	if len(vpcIDs) == 0 {
		return nil, nil
	}
	input := &ec2.DescribeRouteTablesInput{Filters: []ec2types.Filter{{Name: awssdk.String("vpc-id"), Values: vpcIDs}}}
	routeTables := []ec2types.RouteTable{}
	for {
		output, err := client.DescribeRouteTables(ctx, input)
		if err != nil {
			return routeTables, err
		}
		routeTables = append(routeTables, output.RouteTables...)
		if output.NextToken == nil || awssdk.ToString(output.NextToken) == "" {
			break
		}
		input.NextToken = output.NextToken
	}
	sort.Slice(routeTables, func(i, j int) bool {
		return awssdk.ToString(routeTables[i].RouteTableId) < awssdk.ToString(routeTables[j].RouteTableId)
	})
	return routeTables, nil
}

func applyRouteTables(subnets []inventory.SubnetDetail, routeTables []ec2types.RouteTable) []inventory.SubnetDetail {
	explicit := map[string]ec2types.RouteTable{}
	main := map[string]ec2types.RouteTable{}
	for _, routeTable := range routeTables {
		for _, association := range routeTable.Associations {
			if subnetID := awssdk.ToString(association.SubnetId); subnetID != "" {
				explicit[subnetID] = routeTable
			}
			if awssdk.ToBool(association.Main) {
				main[awssdk.ToString(routeTable.VpcId)] = routeTable
			}
		}
	}
	for i := range subnets {
		routeTable, ok := explicit[subnets[i].SubnetID]
		if !ok {
			routeTable, ok = main[subnets[i].VPCID]
		}
		if !ok {
			continue
		}
		subnets[i].RouteTableID = awssdk.ToString(routeTable.RouteTableId)
		for _, route := range routeTable.Routes {
			addStringSlice(&subnets[i].NATGatewayIDs, awssdk.ToString(route.NatGatewayId))
		}
		sort.Strings(subnets[i].NATGatewayIDs)
	}
	return subnets
}

func collectSecurityGroups(ctx context.Context, client ec2API, ids []string) ([]inventory.SecurityGroupDetail, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	groups := []inventory.SecurityGroupDetail{}
	seen := map[string]struct{}{}
	var collectedErr error
	for _, batch := range batches(ids, 200) {
		input := &ec2.DescribeSecurityGroupsInput{GroupIds: batch}
		for {
			output, err := client.DescribeSecurityGroups(ctx, input)
			if err != nil {
				collectedErr = errors.Join(collectedErr, err)
				break
			}
			for _, group := range output.SecurityGroups {
				seen[awssdk.ToString(group.GroupId)] = struct{}{}
				groups = append(groups, mapSecurityGroup(group))
			}
			if output.NextToken == nil || awssdk.ToString(output.NextToken) == "" {
				break
			}
			input.NextToken = output.NextToken
		}
	}
	collectedErr = errors.Join(collectedErr, missingRequested("security groups", ids, seen))
	sort.Slice(groups, func(i, j int) bool { return groups[i].GroupID < groups[j].GroupID })
	return groups, collectedErr
}

func mapSecurityGroup(group ec2types.SecurityGroup) inventory.SecurityGroupDetail {
	return inventory.SecurityGroupDetail{
		GroupID:          awssdk.ToString(group.GroupId),
		GroupName:        awssdk.ToString(group.GroupName),
		Description:      awssdk.ToString(group.Description),
		VPCID:            awssdk.ToString(group.VpcId),
		IngressRuleCount: len(group.IpPermissions),
		EgressRuleCount:  len(group.IpPermissionsEgress),
		Tags:             mapEC2Tags(group.Tags),
	}
}

func mapEC2Tags(tags []ec2types.Tag) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for _, tag := range tags {
		if key := awssdk.ToString(tag.Key); key != "" {
			out[key] = awssdk.ToString(tag.Value)
		}
	}
	return out
}

func splitCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}
	values := strings.Split(value, ",")
	out := make([]string, 0, len(values))
	for _, item := range values {
		addStringSlice(&out, strings.TrimSpace(item))
	}
	sort.Strings(out)
	return out
}

func batches(values []string, size int) [][]string {
	var out [][]string
	for start := 0; start < len(values); start += size {
		end := min(start+size, len(values))
		out = append(out, values[start:end])
	}
	return out
}

func stringSet(values ...string) map[string]struct{} {
	out := map[string]struct{}{}
	addStrings(out, values...)
	return out
}

func addStrings(set map[string]struct{}, values ...string) {
	for _, value := range values {
		addString(set, value)
	}
}

func addString(set map[string]struct{}, value string) {
	if value != "" {
		set[value] = struct{}{}
	}
}

func addStringSlice(values *[]string, value string) {
	if value == "" {
		return
	}
	for _, existing := range *values {
		if existing == value {
			return
		}
	}
	*values = append(*values, value)
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func missingRequested(resource string, requested []string, seen map[string]struct{}) error {
	missing := make([]string, 0)
	for _, id := range requested {
		if _, ok := seen[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s missing from AWS response: %s", resource, strings.Join(missing, ","))
}
