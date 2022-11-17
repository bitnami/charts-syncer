// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamlops_test

import (
	"reflect"
	"testing"

	yamlops2 "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal/yamlops"
)

type node struct {
	path  string
	tag   string
	value string
}

func TestScannerSimple(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		visited []node
	}{
		{
			"simple scalar",
			"hello world",
			[]node{
				{".", "!!str", "hello world"},
			},
		},
		{
			"map",
			"a: 1\nb: 2",
			[]node{
				{".", "!!map", ""},
				{".a", "!!int", "1"},
				{".b", "!!int", "2"},
			},
		},
		{
			"list",
			"- a\n- b",
			[]node{
				{".", "!!seq", ""},
				{".[0]", "!!str", "a"},
				{".[1]", "!!str", "b"},
			},
		},
		{
			"list of maps",
			"- a: 1\n  b: 2\n- a: 3\n  b: 4",
			[]node{
				{".", "!!seq", ""},
				{".[0]", "!!map", ""},
				{".[0].a", "!!int", "1"},
				{".[0].b", "!!int", "2"},
				{".[1]", "!!map", ""},
				{".[1].a", "!!int", "3"},
				{".[1].b", "!!int", "4"},
			},
		},
		{
			"map with list",
			"foo:\n- a\n- b",
			[]node{
				{".", "!!map", ""},
				{".foo", "!!seq", ""},
				{".foo[0]", "!!str", "a"},
				{".foo[1]", "!!str", "b"},
			},
		},
		{
			"list of lists",
			"- [a, b]\n- [c, d]",
			[]node{
				{".", "!!seq", ""},
				{".[0]", "!!seq", ""},
				{".[0][0]", "!!str", "a"},
				{".[0][1]", "!!str", "b"},
				{".[1]", "!!seq", ""},
				{".[1][0]", "!!str", "c"},
				{".[1][1]", "!!str", "d"},
			},
		},
		{
			"map with map",
			"outer:\n  inner:\n    foo: bar",
			[]node{
				{".", "!!map", ""},
				{".outer", "!!map", ""},
				{".outer.inner", "!!map", ""},
				{".outer.inner.foo", "!!str", "bar"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := yamlRootNode(t, test.doc)
			visited := []node{}
			s := yamlops2.NewNodeScanner(root, ".")
			for s.Next() {
				n, path := s.Current()
				visited = append(visited, node{path, n.Tag, n.Value})
			}
			if got, want := visited, test.visited; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
