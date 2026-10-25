import type { Snapshot } from "../runtime/types";
export function watchFixture(): Snapshot {
  return {
    coverage: [
      "Nodes",
      "Pods",
      "DescribeInstances",
      "DescribeAutoScalingGroups",
      "DescribeScalingActivities",
    ].map((resource) => ({
      resource,
      status: "complete",
      collectedAt: "2026-10-25T06:30:00Z",
    })),
    kubernetes: {
      nodes: [
        {
          uid: "node-a",
          name: "worker-a",
          providerId: "aws:///zone/i-a",
          ready: "True",
          unschedulable: true,
          labels: { "eks.amazonaws.com/nodegroup": "app" },
        },
        {
          uid: "node-b",
          name: "worker-b",
          providerId: "aws:///zone/i-b",
          ready: "False",
        },
      ],
      pods: [
        {
          name: "app-a",
          nodeName: "worker-a",
          phase: "Running",
          ownerReferences: [{ kind: "ReplicaSet" }],
        },
        {
          name: "ds",
          nodeName: "worker-a",
          phase: "Running",
          ownerReferences: [{ kind: "DaemonSet" }],
        },
        {
          name: "deleting",
          nodeName: "worker-a",
          phase: "Running",
          deletionTimestamp: "2026-10-25T06:29:00Z",
        },
        { name: "done", nodeName: "worker-a", phase: "Succeeded" },
        { name: "failed", nodeName: "worker-a", phase: "Failed" },
        { name: "scheduled-pending", nodeName: "worker-b", phase: "Pending" },
        { name: "pending", phase: "Pending" },
      ],
      workloads: [
        {
          kind: "Deployment",
          name: "payments",
          namespace: "app",
          replicas: 6,
          readyReplicas: 5,
        },
        { kind: "Deployment", name: "scaled-down", replicas: 0 },
        {
          kind: "DaemonSet",
          name: "agent",
          desiredScheduled: 2,
          readyReplicas: 1,
        },
      ],
      policies: {
        podDisruptionBudgetDetails: [
          { name: "payments", expectedPods: 6 },
          { name: "empty", expectedPods: 0 },
        ],
      },
    },
    eks: {
      infrastructure: {
        instances: [
          {
            instanceId: "i-a",
            nodegroupName: "app",
            autoScalingGroupName: "asg",
            launchTemplateId: "lt",
            launchTemplateVersion: "17",
            launchTemplateSource: "autoscaling-instance",
            imageId: "ami-old",
          },
          {
            instanceId: "i-b",
            nodegroupName: "app",
            autoScalingGroupName: "asg",
            launchTemplateId: "lt",
            launchTemplateVersion: "18",
            launchTemplateSource: "autoscaling-instance",
          },
          { instanceId: "i-new", state: "running" },
        ],
        autoScalingGroups: [
          {
            name: "asg",
            nodegroupName: "app",
            launchTemplateId: "lt",
            launchTemplateVersion: "18",
            desiredCapacity: 2,
          },
        ],
        scalingActivities: [
          {
            activityId: "a",
            autoScalingGroupName: "asg",
            nodegroupName: "app",
            statusCode: "InProgress",
            description: "Launching i-new",
          },
        ],
      },
    },
  };
}
