# Teleskope

EKS and Kubernetes inventory, platform component configuration, and workload
resource topology for human review.

The first phase focuses on collecting and organizing evidence: EKS metadata,
add-ons, CNI, CSI, container runtimes, and workload relationships. Automated
capability assessment and migration gap analysis are future work.

## Status

Early EKS collection is implemented for AWS-side inventory. The CLI reads AWS
shared configuration and credentials through the AWS SDK default config chain,
then collects cluster metadata, managed add-ons, managed node groups, EKS access
entries, and Pod Identity associations. It also attempts Kubernetes API
collection from kubeconfig unless `--skip-kubernetes` is set.

```sh
teleskope scan eks --cluster my-cluster
teleskope scan eks --cluster my-cluster --profile prod --region ap-northeast-1
teleskope scan eks --cluster my-cluster --output json
teleskope scan k8s --kube-context prod
```

The default terminal output is a compact human-readable summary. JSON preserves
the full snapshot structure for later renderers and offline browsing. Kubernetes
Secret values are not collected; Secrets are currently recorded as metadata-only
objects.

## Development

Requires Go 1.26.0 or later. Make is optional.

```sh
make build
./bin/teleskope --help
./bin/teleskope --version
make check
```

Without Make:

```sh
go build -o bin/teleskope ./cmd/teleskope
go test ./...
go vet ./...
```

Set a build version with `make build VERSION=0.1.0`.

## Layout

```text
cmd/teleskope/  Executable entry point
internal/awseks/     AWS-side EKS collector
internal/buildinfo/  Build metadata
internal/cli/        Internal command-line implementation
internal/inventory/  Snapshot data model
internal/k8s/        Kubernetes API collector
internal/render/     Output renderers
docs/                Architecture and design notes
go.mod               Module definition
Makefile             Local development commands
```

Internal packages are added as their functionality is implemented. See
[architecture](docs/architecture.md) for intended module boundaries.
