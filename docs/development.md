# Development docs

## How to build

You need go and the Go protocol buffers pluging:

~~~bash
make gen # To generate Go code from buf definition
make build # To actually build the binary
~~~

## Developing with Distribution Tooling for Helm

For changes spanning both repositories, check out `distribution-tooling-for-helm`
beside `charts-syncer` and create a local Go workspace from this directory:

```bash
go work init . ../distribution-tooling-for-helm
go test ./...
make build
```

If a workspace already exists, use `go work use . ../distribution-tooling-for-helm`.
`go.work` and `go.work.sum` are ignored. Before submitting changes, also validate
against the dependency recorded in `go.mod`, without the sibling checkout:

```bash
GOWORK=off go test ./...
GOWORK=off make build
GOWORK=off make test-style
```

Release builds must use a published DT version in `go.mod`.

Run DT's tests with its own dependency versions:

```bash
cd ../distribution-tooling-for-helm
GOWORK=off go test ./pkg/imagelock ./pkg/artifacts ./pkg/chartutils ./pkg/relocator ./pkg/wrapping ./cmd/dt/unwrap
```

The alias integration tests use temporary in-process OCI registries; Docker is not
required. They cover direct syncing, offline bundles, tag exclusions, overwrites,
platform filtering, digest preservation, retries, and existing-content skips.
