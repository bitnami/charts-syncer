OUTPUT = ./dist/charts-syncer
GO_SOURCES = $(shell find . -type f -name '*.go')
GOARCH ?= $(shell go env GOARCH)
VERSION := $(or $(VERSION), dev)
LDFLAGS="-X github.com/bitnami-labs/charts-syncer/cmd.version=$(VERSION)"

# build args
IMAGE_VERSION := $(shell echo $(VERSION) | sed 's/+/-/1')
REGISTRY_SERVER_ADDRESS?="release-ci.daocloud.io"
REGISTRY_REPO?="$(REGISTRY_SERVER_ADDRESS)/kpanda"
BUILD_ARCH ?= linux/$(GOARCH)

test:
	GO111MODULE=on go test ./...

cover:
	GO111MODULE=on go test -cover ./...

fullcover:
	GO111MODULE=on go test -coverprofile=coverage.out ./...
	GO111MODULE=on go tool cover -func=coverage.out

gen:
	protoc --proto_path=:. --go_out=plugins=grpc:./api api/config.proto api/manifest.proto

build: $(GO_SOURCES)
	GO111MODULE=on CGO_ENABLED=0 go build -o $(OUTPUT) -ldflags ${LDFLAGS} ./