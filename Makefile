OUTPUT = ./dist/charts-syncer
GO_SOURCES = $(shell find . -type f -name '*.go')

test:
	GO111MODULE=on go test ./...

cover:
	GO111MODULE=on go test -cover ./...

fullcover:
	GO111MODULE=on go test -coverprofile=coverage.out ./...
	GO111MODULE=on go tool cover -func=coverage.out

gen:
	go generate github.com/bitnami-labs/charts-syncer/...

build: $(GO_SOURCES)
	GO111MODULE=on CGO_ENABLED=0 go build -o $(OUTPUT) ./
