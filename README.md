# Teleskope

EKS and Kubernetes inventory, platform component configuration, and workload
resource topology for human review.

The first phase focuses on collecting and organizing evidence: EKS metadata,
add-ons, CNI, CSI, container runtimes, and workload relationships. Automated
capability assessment and migration gap analysis are future work.

## Status

Project scaffold only. The CLI currently provides help and version output;
collection and reporting are not implemented yet.

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
internal/cli/  Internal command-line implementation
docs/          Architecture and design notes
go.mod         Module definition
Makefile       Local development commands
```

Internal packages are added as their functionality is implemented. See
[architecture](docs/architecture.md) for intended module boundaries.
