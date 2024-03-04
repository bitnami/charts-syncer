SHELL      = /usr/bin/env bash

OUTPUT = ./dist/charts-syncer
GO_SOURCES = $(shell find . -type f -name '*.go')
VERSION := $(or $(VERSION), dev)
LDFLAGS="-X main.version=$(VERSION)"
GOPATH ?= $(shell go env GOPATH)
export GOBIN := $(abspath $(GOPATH)/bin)
export GO111MODULE := on

GOLANGCILINT  = $(GOBIN)/golangci-lint

export PATH := $(GOBIN):$(PATH)


$(GOLANGCILINT):
	(cd /; GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2)

.PHONY: test
test:
	GO111MODULE=on go test $(GO_TEST_FLAGS) ./...

.PHONY: test-style
test-style: $(GOLANGCILINT)
	GO111MODULE=on $(GOLANGCILINT) run

cover:
	GO111MODULE=on go test -cover ./...

fullcover:
	GO111MODULE=on go test -coverprofile=coverage.out ./...
	GO111MODULE=on go tool cover -func=coverage.out

gen:
	go generate github.com/bitnami/charts-syncer/...

build: $(GO_SOURCES)
	GO111MODULE=on CGO_ENABLED=0 go build -o $(OUTPUT) -ldflags ${LDFLAGS} ./cmd
