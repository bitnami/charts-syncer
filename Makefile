OUTPUT = ./dist/charts-syncer
GO_SOURCES = $(shell find . -type f -name '*.go')
GOARCH ?= $(shell go env GOARCH)
VERSION := $(or $(VERSION), dev)
LDFLAGS="-X github.com/bitnami-labs/charts-syncer/cmd.version=$(VERSION)"

# build args
IMAGE_VERSION := $(shell echo $(VERSION) | sed 's/-/+/1')
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
	cd api && protoc --go_out=plugins=grpc:. config.proto

build: $(GO_SOURCES)
	GO111MODULE=on CGO_ENABLED=0 go build -o $(OUTPUT) -ldflags ${LDFLAGS} ./

build-image: $(SOURCES)
	echo "Building charts-syncer for arch = $(BUILD_ARCH)"
	export DOCKER_CLI_EXPERIMENTAL=enabled ;\
	! ( docker buildx ls | grep charts-syncer-multi-platform-builder ) && docker buildx create --use --platform=$(BUILD_ARCH) --name charts-syncer-multi-platform-builder ;\
	docker buildx build \
			--build-arg version=$(IMAGE_VERSION) \
			--builder charts-syncer-multi-platform-builder \
			--platform $(BUILD_ARCH) \
			--tag $(REGISTRY_REPO)/charts-syncer:$(IMAGE_VERSION)  \
			--tag $(REGISTRY_REPO)/charts-syncer:latest  \
			-f ./Dockerfile \
			--load \
			.