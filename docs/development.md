# Development docs

## How to build

You need go and the Go protocol buffers pluging:

~~~bash
make gen # To generate Go code from protobuff definition
make build # To actually build the binary
~~~

## How to obtain diffs for a chart

As stated in the [README](../README.md) file, the tool performs some change in the chart source code. Specifically these files:

- values.yaml
- values-production.yaml (if exists)
- requirements.yaml (if exists)
- requirements.lock (if exists)
- README.md (if exists)

In order to get a `diff` view with the performed changes you can do the following:

1. Fetch the chart package from the source chart repository
1. Extract it
1. Initialize a git repository and add the current status
1. Execute `charts-syncer` tool
1. Fetch the chart package from target chart repository
1. Extract it in the same folder where the source chart was extracted
1. Run `git diff`