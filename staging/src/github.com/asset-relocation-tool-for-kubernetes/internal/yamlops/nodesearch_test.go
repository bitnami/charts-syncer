// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamlops_test

import (
	"reflect"
	"strconv"
	"testing"

	yamlops2 "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal/yamlops"
	"gopkg.in/yaml.v3"
)

func TestSearchNodes(t *testing.T) {
	root := yamlRootNode(t, "{foo: {bar: fum}}")

	tests := []struct {
		name        string
		matchers    []yamlops2.NodeMatchFunc
		matchValues map[string]string
	}{
		{
			"visits all nodes",
			[]yamlops2.NodeMatchFunc{func(*yaml.Node, string) bool { return true }},
			map[string]string{
				".":        "",
				".foo":     "",
				".foo.bar": "fum",
			},
		},
		{
			"matches no nodes",
			[]yamlops2.NodeMatchFunc{func(*yaml.Node, string) bool { return false }},
			map[string]string{},
		},
		{
			"matches some nodes",
			[]yamlops2.NodeMatchFunc{func(_ *yaml.Node, path string) bool { return path == ".foo.bar" }},
			map[string]string{
				".foo.bar": "fum",
			},
		},
		{
			"no matchers",
			[]yamlops2.NodeMatchFunc{},
			map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matches := yamlops2.SearchNodes(root, ".", test.matchers...)

			matchValues := map[string]string{}
			for path, node := range matches {
				matchValues[path] = node.Value
			}

			if got, want := matchValues, test.matchValues; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestNodeHasPath(t *testing.T) {
	tests := []struct {
		spec  string
		path  string
		match bool
	}{
		{".foo", ".foo", true},
		{".foo", ".bar", false},
		{".[0]", ".[0]", true},
		{".[0]", ".[1]", false},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			fn := yamlops2.NodeHasPath(test.spec)
			match := fn(&yaml.Node{}, test.path)
			if got, want := match, test.match; got != want {
				t.Errorf("got: %t, want: %t", got, want)
			}
		})
	}
}

func TestMapNodeContains(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		selectors map[string]string
		match     bool
	}{
		{
			"full match",
			"{a: foo, b: bar}",
			map[string]string{
				"a": "foo",
				"b": "bar",
			},
			true,
		},
		{
			"partial match",
			"{a: foo, b: bar}",
			map[string]string{
				"a": "foo",
				"b": "bar",
			},
			true,
		},
		{
			"full mismatch",
			"{a: foo, b: bar}",
			map[string]string{
				"a": "xfoo",
				"b": "xbar",
			},
			false,
		},
		{
			"partial mismatch",
			"{a: foo, b: bar}",
			map[string]string{
				"a": "xfoo",
			},
			false,
		},
		{
			"over match",
			"{a: foo, b: bar}",
			map[string]string{
				"a": "foo",
				"b": "bar",
				"c": "fum",
			},
			false,
		},
		{
			"no match values",
			"{a: foo, b: bar}",
			map[string]string{},
			false,
		},
		{
			"not a map",
			"- foo\n- bar",
			map[string]string{},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fn := yamlops2.MapNodeContains(test.selectors)
			node := yamlRootNode(t, test.yaml)
			match := fn(node, ".")
			if got, want := match, test.match; got != want {
				t.Errorf("got: %t, want: %t", got, want)
			}
		})
	}
}

func TestMapNodeContainsKeyTypes(t *testing.T) {
	tests := []struct {
		yaml      string
		fieldName string
	}{
		{
			"{a: foo}}",
			"a",
		},
		{
			"{1: foo}}",
			"1",
		},
		{
			"{true: foo}}",
			"true",
		},

		{
			"{null: foo}}",
			"null",
		},
	}

	for _, test := range tests {
		t.Run(test.yaml, func(t *testing.T) {
			fn := yamlops2.MapNodeContains(map[string]string{
				test.fieldName: "foo",
			})
			node := yamlRootNode(t, test.yaml)
			if got, want := fn(node, "."), true; got != want {
				t.Errorf("got: %t, want: %t", got, want)
			}
		})
	}
}

func TestSelectorMatchFilter(t *testing.T) {
	yamlTest := `{
        "a": "foo",
        "b": "bar",
        "components": [
            {
                "name": "asset1",
                "version": "v1"
            },
            {
                "name": "asset1",
                "version": "v1.1"
            },
            {
                "name": "asset2",
                "version": "v2"
            }
        ],
        "dependencies": [
            {
                "name": "asset1",
                "version": "v1"
            },
            {
                "name": "asset2",
                "version": "v2"
            }
        ]
    }`
	tests := []struct {
		name         string
		filter       string
		selectors    map[string]string
		match        bool
		expectedKeys map[string]string
	}{
		{
			"single match",
			".dependencies",
			map[string]string{
				"name": "asset1",
			},
			true,
			map[string]string{
				"1": ".dependencies[0]",
			},
		},
		{
			"no match",
			".dependencies",
			map[string]string{
				"name": "asset4",
			},
			false,
			map[string]string{
				"1": ".dependencies[0]",
			},
		},
		{
			"multi match",
			".components",
			map[string]string{
				"name": "asset1",
			},
			true,
			map[string]string{
				"1": ".components[0]",
				"2": ".components[1]",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fn := yamlops2.SelectorMatchFilter(test.filter, test.selectors)
			node := yamlRootNode(t, yamlTest)
			maps := yamlops2.SearchNodes(node, ".", fn)
			if test.match {
				for k, v := range test.expectedKeys {
					if _, ok := maps[v]; !ok {
						t.Errorf("expected key %q not found in result map", test.expectedKeys[k])
					}
				}
				if len(test.expectedKeys) != len(maps) {
					t.Errorf("wrong map legth: want: %d , got: %d", len(test.expectedKeys), len(maps))
				}
			}
		})
	}
}
