package awseks

import (
	"context"
	"errors"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// Keep collection bounded: AWS returns in-progress activities first, then recent
// activities. A continuation token explicitly records omitted older history.
func collectScalingActivities(ctx context.Context, client autoScalingAPI, groups map[string]string) ([]inventory.ScalingActivity, error) {
	var result []inventory.ScalingActivity
	var collectedErr error
	for _, name := range sortedMapKeys(groups) {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(collectedErr, err)
		}
		output, err := client.DescribeScalingActivities(ctx, &autoscaling.DescribeScalingActivitiesInput{AutoScalingGroupName: awssdk.String(name), MaxRecords: awssdk.Int32(100)})
		if err != nil {
			collectedErr = errors.Join(collectedErr, fmt.Errorf("activities %s: %w", name, err))
			continue
		}
		for _, a := range output.Activities {
			result = append(result, inventory.ScalingActivity{ActivityID: awssdk.ToString(a.ActivityId), AutoScalingGroupName: name, NodegroupName: groups[name], Description: awssdk.ToString(a.Description), Cause: awssdk.ToString(a.Cause), StatusCode: string(a.StatusCode), StatusMessage: awssdk.ToString(a.StatusMessage), Progress: a.Progress, StartTime: a.StartTime, EndTime: a.EndTime})
		}
		if awssdk.ToString(output.NextToken) != "" {
			collectedErr = errors.Join(collectedErr, fmt.Errorf("activities %s: limited to 100 recent/in-progress activities; older history omitted", name))
		}
	}
	return result, collectedErr
}
