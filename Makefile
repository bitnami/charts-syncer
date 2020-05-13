OUTPUT = ./dist/c3tsyncer
GO_SOURCES = $(shell find . -type f -name '*.go')

test:
	GO111MODULE=on go test ./...

gen:
	protoc -I=./api --go_out=./api ./api/config.proto

build: $(GO_SOURCES)
	GO111MODULE=on go build -o $(OUTPUT) main.go

snapshot:
	rm -rf dist && \
	if [ ! -f ./bin/goreleaser ]; then curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh; fi	&& \
	./bin/goreleaser release --skip-publish --snapshot
