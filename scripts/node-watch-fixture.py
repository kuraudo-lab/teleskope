#!/usr/bin/env python3
"""Build the deterministic synthetic snapshot used by local Node watch demos/tests."""
import json
from pathlib import Path

STAMP = "2026-10-25T06:32:08Z"

def snapshot():
    nodes, pods, instances = [], [], []
    layout = [("a01",18,6,"True",False,18),("a02",16,6,"True",False,18),
              ("a03",3,6,"True",True,17),("a04",0,6,"True",True,17),
              ("a05",21,6,"True",False,17),("a06",19,6,"True",False,17),
              ("b01",14,6,"True",False,18),("b02",0,4,"False",False,18),
              ("s01",8,5,"True",False,12),("s02",9,5,"True",False,12),
              ("s03",7,5,"Unknown",False,12)]
    for index, (name, non_ds, ds, ready, cordon, version) in enumerate(layout):
        group = "system-workers" if name.startswith("s") else "app-workers"
        zone = "ap-northeast-1" + "acd"[index % 3]
        instance_id = f"i-demo-{name}"
        created = "2026-10-25T06:25:02Z" if version == 18 else "2026-10-03T01:13:00Z"
        nodes.append(dict(apiVersion="v1", kind="Node", uid=f"mock-node-{name}", name=name,
                          providerId=f"aws:///{zone}/{instance_id}", ready=ready, unschedulable=cordon,
                          createdAt=created, kubeletVersion="v1.34.2", labels={
                              "eks.amazonaws.com/nodegroup":group,"topology.kubernetes.io/zone":zone,
                              "node.kubernetes.io/instance-type":"m6i.xlarge"}))
        instances.append(dict(instanceId=instance_id, kubernetesNodeName=name, nodegroupName=group,
                              autoScalingGroupName=group+"-asg", availabilityZone=zone,
                              state="running", instanceType="m6i.xlarge", imageId=f"ami-demo-v{version}",
                              launchTemplateId="lt-demo-"+group, launchTemplateVersion=str(version),
                              launchTemplateSource="autoscaling-instance", lifecycleState="InService",
                              launchTime="2026-10-25T06:24:10Z" if version == 18 else "2026-10-03T01:12:00Z"))
        for kind, count in [("ReplicaSet",non_ds),("DaemonSet",ds)]:
            for number in range(count):
                pod = dict(apiVersion="v1",kind="Pod",uid=f"mock-{name}-{kind}-{number}",
                           name=f"{name}-{'agent' if kind == 'DaemonSet' else 'app'}-{number}",
                           namespace="kube-system" if kind == "DaemonSet" else "shop",nodeName=name,
                           phase="Running",ownerReferences=[dict(kind=kind,name=f"agent-{number}" if kind=="DaemonSet" else "payments-api-rs")])
                if name == "a03" and kind == "ReplicaSet" and number < 2:
                    pod["deletionTimestamp"]="2026-10-25T06:30:00Z"
                pods.append(pod)
    for number in range(4):
        pods.append(dict(apiVersion="v1",kind="Pod",uid=f"mock-pending-{number}",name=f"checkout-pending-{number}",namespace="shop",phase="Pending"))
    asgs=[]
    for group, version, desired in [("app-workers",18,8),("system-workers",12,3)]:
        members=[i for i in instances if i["nodegroupName"]==group]
        asgs.append(dict(name=group+"-asg",nodegroupName=group,desiredCapacity=desired,minSize=desired,maxSize=desired+2,
                         launchTemplateId="lt-demo-"+group,launchTemplateVersion=str(version),
                         instanceIds=[i["instanceId"] for i in members],instances=[{k:i[k] for k in ("instanceId","launchTemplateId","launchTemplateVersion","lifecycleState")} for i in members]))
    activities=[dict(activityId="mock-launch-b02",autoScalingGroupName="app-workers-asg",nodegroupName="app-workers",description="Launching replacement instance i-demo-b02",statusCode="InProgress",progress=70,startTime="2026-10-25T06:31:00Z",cause="Synthetic rolling replacement"),
                dict(activityId="mock-terminate-a04",autoScalingGroupName="app-workers-asg",nodegroupName="app-workers",description="Terminating instance i-demo-a04",statusCode="MidLifecycleAction",progress=40,startTime="2026-10-25T06:29:00Z",statusMessage="Synthetic lifecycle hook wait"),
                dict(activityId="mock-launch-b01",autoScalingGroupName="app-workers-asg",nodegroupName="app-workers",description="Launched replacement instance i-demo-b01",statusCode="Successful",progress=100,startTime="2026-10-25T06:24:00Z",endTime="2026-10-25T06:26:00Z")]
    instances[3]["lifecycleState"]="Terminating:Wait"
    asgs[0]["instances"][3]["lifecycleState"]="Terminating:Wait"
    k=dict(context="mock-upgrade-demo",version={"gitVersion":"v1.34.2"},nodes=nodes,pods=pods,
           namespaces=[dict(name="shop"),dict(name="kube-system")],
           workloads=[dict(apiVersion="apps/v1",kind="Deployment",name=name,namespace="shop",replicas=desired,readyReplicas=ready) for name,desired,ready in [("payments-api",6,5),("checkout",12,10),("worker-queue",8,6)]],
           policies={"podDisruptionBudgetDetails":[dict(apiVersion="policy/v1",kind="PodDisruptionBudget",name="payments-api",namespace="shop",expectedPods=6,currentHealthy=5,desiredHealthy=5,disruptionsAllowed=0)]})
    k["workloads"].extend(dict(apiVersion="apps/v1",kind="DaemonSet",name=f"agent-{i}",namespace="kube-system",desiredScheduled=11 if i<5 else 8,readyReplicas=10 if i<4 else 9 if i==4 else 7) for i in range(6))
    counts={"Nodes":len(nodes),"Pods":len(pods),"Nodegroups":2,"DescribeInstances":len(instances),"DescribeAutoScalingGroups":2,"DescribeScalingActivities":3,"Deployments":3,"StatefulSets":0,"DaemonSets":6,"PodDisruptionBudgets":1}
    return dict(schemaVersion="teleskope.io/snapshot/v1alpha1",collectedAt=STAMP,source={"tool":"teleskope-node-watch-fixture","version":"1","mode":"synthetic"},aws={"region":"ap-northeast-1"},
                eks={"cluster":{"name":"MOCK · EKS upgrade demo","version":"1.34"},"nodegroups":[dict(name=g,autoScalingGroups=[g+"-asg"]) for g in ("app-workers","system-workers")],"infrastructure":{"instances":instances,"autoScalingGroups":asgs,"scalingActivities":activities}},
                kubernetes=k,coverage=[dict(area="kubernetes" if resource in ("Nodes","Pods","Deployments","StatefulSets","DaemonSets","PodDisruptionBudgets") else "aws",resource=resource,status="complete",objectCount=count,collectedAt=STAMP) for resource,count in counts.items()])

if __name__ == "__main__":
    output=Path(__file__).resolve().parents[1]/"testdata/synthetic/node-watch.json"
    output.parent.mkdir(parents=True,exist_ok=True)
    output.write_text(json.dumps(snapshot(),ensure_ascii=False,indent=2)+"\n")
    print(output)
