# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause

SHELL = /bin/bash

default: build


# #### GO Binary Management ####
.PHONY: deps-go-binary deps-counterfeiter deps-ginkgo deps-golangci-lint

GO_VERSION := $(shell go version)
GO_VERSION_REQUIRED = go1.17
GO_VERSION_MATCHED := $(shell go version | grep $(GO_VERSION_REQUIRED))

deps-go-binary:
ifndef GO_VERSION
	$(error Go not installed)
endif
ifndef GO_VERSION_MATCHED
	$(error Required Go version is $(GO_VERSION_REQUIRED), but was $(GO_VERSION))
endif
	@:

HAS_COUNTERFEITER := $(shell command -v counterfeiter;)
HAS_GINKGO := $(shell command -v ginkgo;)
HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)

# If go get is run from inside the project directory it will add the dependencies
# to the go.mod file. To avoid that we import from another directory
deps-counterfeiter: deps-go-binary
ifndef HAS_COUNTERFEITER
	cd /; go get -u github.com/maxbrunsfeld/counterfeiter/v6
endif

deps-ginkgo: deps-go-binary
ifndef HAS_GINKGO
	cd /; go get github.com/onsi/ginkgo/ginkgo github.com/onsi/gomega
endif

deps-golangci-lint: deps-go-binary
ifndef HAS_GOLANGCI_LINT
	cd /; go get github.com/golangci/golangci-lint/cmd/golangci-lint
endif

# #### CLEAN ####
.PHONY: clean

clean: deps-go-binary 
	rm -rf build/* vendor/*

# #### DEPS ####
.PHONY: deps

vendor/modules.txt: go.mod
	go mod vendor

deps: vendor/modules.txt deps-counterfeiter deps-ginkgo


# #### BUILD ####
.PHONY: build

SRC = $(shell find . -name "*.go" | grep -v "_test\." )
VERSION := $(or $(VERSION), dev)
LDFLAGS="-X github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/cmd.Version=$(VERSION)"

build/relok8s: $(SRC)
	go build -o build/relok8s -ldflags ${LDFLAGS} ./main.go

build/relok8s-darwin: $(SRC)
	GOARCH=amd64 GOOS=darwin go build -o build/relok8s-darwin -ldflags ${LDFLAGS} ./main.go

build/relok8s-linux: $(SRC)
	GOARCH=amd64 GOOS=linux go build -o build/relok8s-linux -ldflags ${LDFLAGS} ./main.go

build: deps build/relok8s

build-all: build/relok8s-darwin build/relok8s-linux

# #### TESTS ####
.PHONY: lint test test-features test-units

test-units: deps
	ginkgo -r -skipPackage test .

test-fixtures:
	make --directory test/fixtures

test-features: deps test-fixtures
	ginkgo -r test/features

test-external: deps test-fixtures
	ginkgo -r test/external

local-registry-image: deps test
	docker build -f test/registry/Dockerfile -t registry-tester .

test-registry: local-registry-image
	docker run -it --rm -p 5443:443 registry-tester /bin/registry-test -test.v -test.run=TestRegistry

test-performance: local-registry-image
	docker run -it --rm -p 5443:443 registry-tester /bin/registry-test -test.v -test.run=TestMove
	docker run -it --rm -p 5443:443 registry-tester /bin/registry-test -test.v -test.run=TestSaveNLoad

test: deps test-units test-features

test-all: test test-registry test-external

# https://golangci-lint.run/usage/install/#local-installation
lint: deps-golangci-lint
	golangci-lint run

# #### DEVOPS ####
.PHONY: set-pipeline set-example-pipeline
set-pipeline: ci/pipeline.yaml
	fly -t tie set-pipeline --config ci/pipeline.yaml --pipeline relok8s

set-example-pipeline: examples/concourse-pipeline/pipeline.yaml
	fly -t tie set-pipeline --config examples/concourse-pipeline/pipeline.yaml --pipeline relok8s-example
