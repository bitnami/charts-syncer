OUTPUT = ./dist/c3tsyncer
GO_SOURCES = $(shell find . -type f -name '*.go')

test:
	GO111MODULE=on go test ./...

cover:
	GO111MODULE=on go test -cover ./...

fullcover:
	GO111MODULE=on go test -coverprofile=coverage.out ./...
	GO111MODULE=on go tool cover -func=coverage.out

gen:
	go generate github.com/bitnami-labs/chart-repository-syncer/...

build: $(GO_SOURCES)
	GO111MODULE=on go build -o $(OUTPUT) ./cmd/
