package awseks

import (
	"context"
	"errors"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	autoscalingtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestScalingActivitiesBoundedAndScoped(t *testing.T) {
	started := time.Date(2026, 10, 25, 6, 30, 0, 0, time.UTC)
	client := &fakeAutoScaling{activities: &autoscaling.DescribeScalingActivitiesOutput{
		Activities: []autoscalingtypes.Activity{{ActivityId: awssdk.String("activity"), Description: awssdk.String("Launching instance"), StatusCode: autoscalingtypes.ScalingActivityStatusCodeInProgress, StartTime: &started, Progress: awssdk.Int32(20)}}, NextToken: awssdk.String("older"),
	}}
	got, err := collectScalingActivities(context.Background(), client, map[string]string{"asg-app": "app"})
	if err == nil {
		t.Fatal("truncated history must be reported as partial")
	}
	if len(client.activityInputs) != 1 || awssdk.ToString(client.activityInputs[0].AutoScalingGroupName) != "asg-app" || awssdk.ToInt32(client.activityInputs[0].MaxRecords) != 100 {
		t.Fatalf("unscoped activity request: %#v", client.activityInputs)
	}
	if len(got) != 1 || got[0].NodegroupName != "app" || got[0].StatusCode != "InProgress" || !got[0].StartTime.Equal(started) {
		t.Fatalf("activity facts: %#v", got)
	}
}

func TestActivitiesPermissionFailureDoesNotDiscardInfrastructure(t *testing.T) {
	snapshot := infrastructureTestSnapshot()
	client := &fakeAutoScaling{output: &autoscaling.DescribeAutoScalingGroupsOutput{}, activityErr: errors.New("AccessDenied")}
	collectInfrastructure(context.Background(), client, infrastructureFakeEC2(nil), snapshot, time.Now())
	assertCoverage(t, snapshot.Coverage, "DescribeScalingActivities", "partial")
	if len(snapshot.EKS.Infrastructure.Subnets) != 1 {
		t.Fatal("activity denial lost unrelated infrastructure")
	}
	client.activityInputs = nil
	_, err := collectScalingActivities(context.Background(), client, map[string]string{})
	if err != nil || len(client.activityInputs) != 0 {
		t.Fatal("empty scope must not make account-wide requests")
	}
}

func TestInstanceDoesNotInheritGroupTargetTemplate(t *testing.T) {
	client := infrastructureFakeEC2(nil)
	launched := time.Date(2026, 10, 25, 6, 0, 0, 0, time.UTC)
	client.instances.Reservations[0].Instances[0].LaunchTime = &launched
	groups := map[string]inventory.AutoScalingGroup{"asg": {LaunchTemplateID: "lt-target", LaunchTemplateVersion: "18"}}
	instances, err := collectEC2Instances(context.Background(), client, []string{"i-example"}, map[string]string{"i-example": "asg"}, map[string]string{"asg": "app"}, groups)
	if err != nil {
		t.Fatal(err)
	}
	if instances[0].LaunchTemplateID != "" || instances[0].LaunchTemplateVersion != "" || instances[0].LaunchTemplateSource != "" {
		t.Fatal("missing actual template incorrectly inherited group target")
	}
	if instances[0].LaunchTime == nil || !instances[0].LaunchTime.Equal(launched) {
		t.Fatal("EC2 launch time missing")
	}
	groups["asg"] = inventory.AutoScalingGroup{Instances: []inventory.AutoScalingInstance{{InstanceID: "i-example", LaunchTemplateID: "lt-old", LaunchTemplateVersion: "17", LifecycleState: "Terminating:Wait"}}}
	instances, err = collectEC2Instances(context.Background(), client, []string{"i-example"}, map[string]string{"i-example": "asg"}, nil, groups)
	if err != nil {
		t.Fatal(err)
	}
	if instances[0].LaunchTemplateVersion != "17" || instances[0].LifecycleState != "Terminating:Wait" {
		t.Fatalf("actual instance facts lost: %#v", instances[0])
	}
}
