#!/usr/bin/env python3
"""Sanitize a Teleskope snapshot for long-lived local UI fixtures."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
from pathlib import Path
from typing import Any


PLACEHOLDERS = {
    "account": "000000000000",
    "cluster": "demo-eks",
    "endpoint_id": "DEMOENDPOINT",
    "user": "demo-user",
    "nodegroup": "demo-nodegroup",
}


EXTRA_REPLACEMENTS: list[tuple[re.Pattern[str], str]] = []


RESOURCE_ID_PATTERN = re.compile(r"\b(vpc|subnet|sg|i|lt|ami|rtb|nat|dopt)-[0-9a-f]+\b")
RESOURCE_ID_MAPS: dict[str, dict[str, str]] = {}


PATTERN_REPLACEMENTS = [
    (re.compile(r"\b\d{12}\b"), PLACEHOLDERS["account"]),
    (
        re.compile(r"arn:aws:eks:([a-z0-9-]+):\d{12}:cluster/[^/\s\"']+"),
        r"arn:aws:eks:\1:000000000000:cluster/demo-eks",
    ),
    (
        re.compile(r"arn:aws:eks:([a-z0-9-]+):\d{12}:addon/[^/\s\"']+/([^/\s\"']+)/[^/\s\"']+"),
        r"arn:aws:eks:\1:000000000000:addon/demo-eks/\2/demo-addon-id",
    ),
    (
        re.compile(r"arn:aws:eks:([a-z0-9-]+):\d{12}:nodegroup/[^/\s\"']+/[^/\s\"']+/[^/\s\"']+"),
        r"arn:aws:eks:\1:000000000000:nodegroup/demo-eks/demo-nodegroup/demo-nodegroup-id",
    ),
    (
        re.compile(r"arn:aws:eks:([a-z0-9-]+):\d{12}:access-entry/[^/\s\"']+/.+?(?=\s|$|\"|')"),
        r"arn:aws:eks:\1:000000000000:access-entry/demo-eks/demo-principal",
    ),
    (
        re.compile(r"arn:aws:iam::\d{12}:user/[^/\s\"']+"),
        "arn:aws:iam::000000000000:user/demo-user",
    ),
    (
        re.compile(r"arn:aws:iam::\d{12}:role/aws-service-role/eks\.amazonaws\.com/AWSServiceRoleForAmazonEKS"),
        "arn:aws:iam::000000000000:role/aws-service-role/eks.amazonaws.com/AWSServiceRoleForAmazonEKS",
    ),
    (
        re.compile(r"arn:aws:iam::\d{12}:role/[^,\n\s\"']+"),
        "arn:aws:iam::000000000000:role/demo-role",
    ),
    (
        re.compile(r"arn:aws:sts::\d{12}:assumed-role/[^/\s\"']+/[^/\s\"']+"),
        "arn:aws:sts::000000000000:assumed-role/demo-role/demo-session",
    ),
    (
        re.compile(r"arn:aws:kms:([a-z0-9-]+):\d{12}:key/[0-9a-f-]+"),
        r"arn:aws:kms:\1:000000000000:key/00000000-0000-0000-0000-000000000000",
    ),
    (
        re.compile(r"https://[a-z0-9]+\.([a-z0-9-]+\.)?([a-z0-9-]+)\.eks\.amazonaws\.com", re.IGNORECASE),
        "https://demo-eks.example.invalid",
    ),
    (
        re.compile(r"https://oidc\.eks\.([a-z0-9-]+)\.amazonaws\.com/id/[A-Z0-9]+", re.IGNORECASE),
        r"https://oidc.eks.\1.amazonaws.com/id/DEMOOIDC",
    ),
    (
        re.compile(r"\barn:aws:eks:([a-z0-9-]+):\d{12}:cluster/[^/\s\"']+\b"),
        r"arn:aws:eks:\1:000000000000:cluster/demo-eks",
    ),
    (re.compile(r"\bexample-[a-z0-9-]+\b"), PLACEHOLDERS["nodegroup"]),
    (re.compile(r"\bip-\d+-\d+-\d+-\d+\.[a-z0-9-]+\.compute\.internal\b"), "demo-node.internal"),
    (re.compile(r"\b\d{1,3}(?:\.\d{1,3}){3}/\d{1,2}\b"), "demo-cidr"),
    (re.compile(r"\b\d{1,3}(?:\.\d{1,3}){3}\b"), "demo-ip"),
    (re.compile(r"\b[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\b"), "00000000-0000-0000-0000-000000000000"),
]


def sanitize_string(value: str) -> str:
    out = value
    out = RESOURCE_ID_PATTERN.sub(pseudonymize_resource_id, out)
    for pattern, replacement in [*PATTERN_REPLACEMENTS, *EXTRA_REPLACEMENTS]:
        out = pattern.sub(replacement, out)
    return out


def pseudonymize_resource_id(match: re.Match[str]) -> str:
    prefix = match.group(1)
    original = match.group(0)
    mapping = RESOURCE_ID_MAPS.setdefault(prefix, {})
    if original not in mapping:
        mapping[original] = f"{prefix}-{len(mapping) + 1:017x}"
    return mapping[original]


def sanitize(value: Any) -> Any:
    if isinstance(value, dict):
        return {key: sanitize(item) for key, item in value.items()}
    if isinstance(value, list):
        return [sanitize(item) for item in value]
    if isinstance(value, str):
        return sanitize_string(value)
    return value


def stable_name(prefix: str, value: str) -> str:
    digest = hashlib.sha256(value.encode("utf-8")).hexdigest()[:6]
    return f"{prefix}-{digest}"


def normalize_nodes(snapshot: dict[str, Any]) -> None:
    kubernetes = snapshot.get("kubernetes") or {}
    node_names: dict[str, str] = {}
    instance_ids: dict[str, str] = {}
    for index, node in enumerate(kubernetes.get("nodes") or [], start=1):
        old = node.get("name")
        new = f"demo-node-{index}"
        if old:
            node_names[old] = new
        node["name"] = new
        provider_id = node.get("providerId")
        if provider_id:
            provider_prefix, _, old_instance_id = provider_id.rpartition("/")
            new_instance_id = f"i-{index:017x}"
            if old_instance_id:
                instance_ids[old_instance_id] = new_instance_id
            node["providerId"] = f"{provider_prefix}/{new_instance_id}"
        labels = node.get("labels") or {}
        labels["kubernetes.io/hostname"] = new
        labels["eks.amazonaws.com/nodegroup"] = PLACEHOLDERS["nodegroup"]
        node["labels"] = labels
    replace_node_refs(kubernetes, node_names)
    replace_exact_strings(snapshot, instance_ids)


def replace_node_refs(value: Any, node_names: dict[str, str]) -> None:
    if not node_names:
        return
    if isinstance(value, dict):
        for key, item in value.items():
            if isinstance(item, str) and item in node_names:
                value[key] = node_names[item]
            else:
                replace_node_refs(item, node_names)
    elif isinstance(value, list):
        for item in value:
            replace_node_refs(item, node_names)


def replace_exact_strings(value: Any, replacements: dict[str, str]) -> None:
    if not replacements:
        return
    if isinstance(value, dict):
        for key, item in value.items():
            if isinstance(item, str) and item in replacements:
                value[key] = replacements[item]
            else:
                replace_exact_strings(item, replacements)
    elif isinstance(value, list):
        for index, item in enumerate(value):
            if isinstance(item, str) and item in replacements:
                value[index] = replacements[item]
            else:
                replace_exact_strings(item, replacements)


def normalize_generated_names(snapshot: dict[str, Any]) -> None:
    kubernetes = snapshot.get("kubernetes") or {}
    for pod in kubernetes.get("pods") or []:
        name = pod.get("name")
        if isinstance(name, str) and re.search(r"-[a-z0-9]{5}$", name):
            pod["name"] = stable_name(name.rsplit("-", 1)[0], name)
    for endpoint_slice in kubernetes.get("endpointSlices") or []:
        name = endpoint_slice.get("name")
        if isinstance(name, str) and re.search(r"-[a-z0-9]{5}$", name):
            endpoint_slice["name"] = stable_name(name.rsplit("-", 1)[0], name)


def normalize_top_level(snapshot: dict[str, Any]) -> None:
    aws = snapshot.setdefault("aws", {})
    aws["accountId"] = PLACEHOLDERS["account"]
    if aws.get("arn"):
        aws["arn"] = "arn:aws:iam::000000000000:user/demo-user"
    if aws.get("userId"):
        aws["userId"] = "demo-user"

    cluster = snapshot.setdefault("eks", {}).setdefault("cluster", {})
    if cluster:
        cluster["name"] = PLACEHOLDERS["cluster"]
        if cluster.get("arn"):
            cluster["arn"] = "arn:aws:eks:eu-west-1:000000000000:cluster/demo-eks"
        if cluster.get("endpoint"):
            cluster["endpoint"] = "https://demo-eks.example.invalid"
        if cluster.get("roleArn"):
            cluster["roleArn"] = "arn:aws:iam::000000000000:role/demo-eks-cluster-role"
        if cluster.get("oidcIssuer"):
            cluster["oidcIssuer"] = "https://oidc.eks.eu-west-1.amazonaws.com/id/DEMOOIDC"

    kubernetes = snapshot.setdefault("kubernetes", {})
    if kubernetes.get("context"):
        kubernetes["context"] = "demo-eks"
    if kubernetes.get("server"):
        kubernetes["server"] = "https://demo-eks.example.invalid"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--cluster-name", help="raw cluster name to replace with demo-eks")
    parser.add_argument("--user-name", help="raw IAM or Kubernetes username to replace with demo-user")
    args = parser.parse_args()

    if args.cluster_name:
        EXTRA_REPLACEMENTS.append((re.compile(rf"\b{re.escape(args.cluster_name)}\b"), PLACEHOLDERS["cluster"]))
    if args.user_name:
        EXTRA_REPLACEMENTS.append((re.compile(rf"\b{re.escape(args.user_name)}\b"), PLACEHOLDERS["user"]))

    with args.input.open() as fh:
        snapshot = json.load(fh)
    normalize_nodes(snapshot)
    normalize_generated_names(snapshot)
    snapshot = sanitize(snapshot)
    normalize_top_level(snapshot)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w") as fh:
        json.dump(snapshot, fh, indent=2, sort_keys=False)
        fh.write("\n")


if __name__ == "__main__":
    main()
