GO ?= go
PYTHON ?= python3
VERSION ?= dev

.PHONY: build run test vet fixture-test fmt check clean

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

fmt:
	$(GO) fmt ./...

check: test vet fixture-test

clean:
	$(GO) clean
	rm -f bin/teleskope
