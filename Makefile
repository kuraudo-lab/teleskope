GO ?= go
PYTHON ?= python3
VERSION ?= dev
SCALE_PROFILE ?= small
SCALE_ADDR ?= 127.0.0.1:8092
LIVE_DEMO_ADDR ?= 127.0.0.1:8093

.PHONY: build run test vet fixture-test topology-bench topology-scale-demo topology-live-demo scope-release-check fmt check clean

build:
	$(GO) build -trimpath -ldflags "-X github.com/kuraudo-lab/teleskope/internal/buildinfo.Version=$(VERSION)" -o bin/teleskope ./cmd/teleskope

run:
	$(GO) run ./cmd/teleskope

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fixture-test:
	PYTHONDONTWRITEBYTECODE=1 $(PYTHON) -m unittest scripts/test_sanitize_snapshot.py

topology-bench:
	$(GO) test -run '^$$' -bench 'TopologyScale' -benchmem ./internal/topology

topology-scale-demo:
	$(GO) run ./scripts/topology-scale -profile $(SCALE_PROFILE) -serve $(SCALE_ADDR)

topology-live-demo:
	$(GO) run ./scripts/topology-live-demo -serve $(LIVE_DEMO_ADDR)

scope-release-check:
	$(GO) run ./scripts/scope-release-check

fmt:
	$(GO) fmt ./...

check: test vet fixture-test

clean:
	$(GO) clean
	rm -f bin/teleskope
