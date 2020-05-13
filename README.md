# c3tsyncer

Sync chart packages between chart repositories

##


## Usage

#### Sync a specific chart

~~~bash
$ c3tsyncer syncChart --name nginx --version 1.0.0 --config ./c3tsyncer.yaml
~~~

#### Sync all versions for a specific chart

~~~bash
$ c3tsyncer syncChart --name nginx --all-versions --config ./c3tsyncer.yaml
~~~

#### Sync all charts and versions

~~~bash
$ c3tsyncer sync --config ./c3tsyncer.yaml
~~~

#### Sync all charts and versions from specific date

~~~bash
$ c3tsyncer sync --config --from-date 05/12/20 ./c3tsyncer.yaml
~~~

 > Date should be in format MM/DD/YY

----

## Configuration

Below you can find an example configuration file:

~~~yaml
#
# Example config file
#
source:
  url: "http://localhost:8080" # local test source repo
  kind: "chartmuseum"
  # auth:
  #   username: "USERNAME"
  #   password: "PASSWORD"
target:
  url: "http://localhost:9090" # local test target repo
  containerRegistry: "demo.registry.io"
  containerRepository: "tpizarro/demo"
  # auth:
  #   username: "USERNAME"
  #   password: "PASSWORD"
~~~

Credentials can be provided using config file or the following environment variables:

- `SOURCE_AUTH_USERNAME`
- `SOURCE_AUTH_PASSWORD`
- `TARGET_AUTH_USERNAME`
- `TARGET_AUTH_PASSWORD`

## How to build

You need go and the GO protocol buffers pluging:

~~~bash
make gen # To generate Go code from protobuff definition
make build # To actually build the binary
~~~~