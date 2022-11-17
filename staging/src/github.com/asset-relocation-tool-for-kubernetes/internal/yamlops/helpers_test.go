// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamlops_test

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// Parse yaml document and return the root node.
func yamlRootNode(t *testing.T, doc string) *yaml.Node {
	node := &yaml.Node{}
	if err := yaml.Unmarshal([]byte(doc), node); err != nil {
		t.Fatal(err)
	}
	return node.Content[0]
}
