OUTPUT = ./dist/c3tsyncer
GO_SOURCES = $(shell find . -type f -name '*.go')

test:
	GO111MODULE=on go test ./...

gen:
	go generate github.com/bitnami-labs/chart-repository-syncer/...

build: $(GO_SOURCES)
	GO111MODULE=on go build -o $(OUTPUT) ./cmd/
