#!/bin/bash

set -o nounset
set -o pipefail

reg_port=8080
reg_name=${HARBOR_IP:-${1:?Missing harbor ip}}

cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:8080"]
  [plugins."io.containerd.grpc.v1.cri".registry.configs]
    [plugins."io.containerd.grpc.v1.cri".registry.configs."localhost:${reg_port}".tls]
      insecure_skip_verify = true
    [plugins."io.containerd.grpc.v1.cri".registry.configs."${reg_name}:8080".tls]
      insecure_skip_verify = true
nodes:
- role: control-plane
- role: worker
EOF
