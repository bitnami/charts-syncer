// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamlops

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// NodeMatchFunc is a func that takes a node and path and returns a bool to
// indicate if the node matches some test.
type NodeMatchFunc func(node *yaml.Node, path string) bool

// SearchNodes walks the node tree adding the node to the result if matchers all
// return true.
func SearchNodes(root *yaml.Node, rootPath string, matchers ...NodeMatchFunc) map[string]*yaml.Node {
	// Don't match everything by accident.
	if len(matchers) == 0 {
		return nil
	}

	nodes := map[string]*yaml.Node{}
	scanner := NewNodeScanner(root, rootPath)

NextNode:
	for scanner.Next() {
		node, path := scanner.Current()
		for _, fn := range matchers {
			if !fn(node, path) {
				continue NextNode
			}
		}

		nodes[path] = node
	}

	return nodes
}

// NodeHasPath returns a NodeMatchFunc that tests if the node's path matches the
// query spec.
func NodeHasPath(spec string) NodeMatchFunc {
	// TODO(mgoodall): parse the spec for complex key names, any list item,
	// etc.  But for now, assume it's exact all the way down, e.g. a relatively
	// simple ".some.path.in.a.list[0].to.a.field".
	return func(node *yaml.Node, path string) bool {
		return path == spec
	}
}

// map key tags (types) that make sense to match as if a string.
var validMapKeyTags = map[string]struct{}{
	"!!bool": {},
	"!!int":  {},
	"!!null": {},
	"!!str":  {},
}

// MapNodeContains returns a NodeMatchFunc that tests if a map node contains all
// the given field values.
func MapNodeContains(values map[string]string) NodeMatchFunc {
	return func(node *yaml.Node, path string) bool {
		return SearchInMap(node, values)
	}
}

// SelectorMatchFilter returns a NodeMatchFunc that tests if a map node inside a prefix
// contains all the given field values.
func SelectorMatchFilter(filter string, values map[string]string) NodeMatchFunc {
	return func(node *yaml.Node, path string) bool {
		if strings.HasPrefix(path, filter) {
			return SearchInMap(node, values)
		}
		return false
	}
}

// SearchInMap look for provided key=values inside a yaml map
func SearchInMap(node *yaml.Node, values map[string]string) bool {
	if node.Tag != "!!map" || len(values) == 0 {
		return false
	}
	fields := map[string]string{}
	for i := 0; i < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if _, ok := validMapKeyTags[k.Tag]; !ok {
			continue
		}
		// TODO(mgoodall): match other types ... one day
		if v.Tag != "!!str" {
			continue
		}
		fields[k.Value] = v.Value
	}

	for k, want := range values {
		if got, ok := fields[k]; !ok || got != want {
			return false
		}
	}
	return true
}
