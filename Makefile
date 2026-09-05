GO ?= go
VERSION ?= dev

.PHONY: build run test vet fmt check clean

build:
	$(GO) build -trimpath -ldflags "-X github.com/kuraudo-lab/teleskope/internal/buildinfo.Version=$(VERSION)" -o bin/teleskope ./cmd/teleskope

run:
	$(GO) run ./cmd/teleskope

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test vet

clean:
	$(GO) clean
	rm -f bin/teleskope
