// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamlops_test

import (
	"strings"
	"testing"

	yamlops2 "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal/yamlops"
)

func TestUpdateMap(t *testing.T) {
	tests := []struct {
		name      string
		doc       string
		path      string
		selectors map[string]string
		values    map[string]string
		result    string
	}{
		{
			"updates map with path",
			"a:\n  foo: a foo\nb:\n  foo: b foo",
			".a",
			nil,
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx\nb:\n  foo: b foo",
		},
		{
			"updates map in list with path",
			"- name: a\n  foo: a foo\n- name: b\n  foo: b foo",
			".[0]",
			nil,
			map[string]string{"foo": "xxx"},
			"- name: a\n  foo: xxx\n- name: b\n  foo: b foo",
		},
		{
			"updates nested map with path",
			"a:\n  b:\n    foo: a b foo",
			".a.b",
			nil,
			map[string]string{"foo": "xxx"},
			"a:\n  b:\n    foo: xxx",
		},
		{
			"updates inline map",
			"a: {foo: a foo}",
			".a",
			nil,
			map[string]string{"foo": "xxx"},
			"a: {foo: xxx}",
		},
		{
			"updates map with field",
			"a:\n  foo: a foo\nb:\n  foo: b foo",
			"",
			map[string]string{"foo": "a foo"},
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx\nb:\n  foo: b foo",
		},
		{
			"updates map with multiple field",
			"a:\n  foo: a foo\n  other: a other\nb:\n  foo: b foo",
			"",
			map[string]string{"foo": "a foo", "other": "a other"},
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx\n  other: a other\nb:\n  foo: b foo",
		},
		{
			"updates multiple maps with field",
			"a:\n  foo: old\nb:\n  foo: old",
			"",
			map[string]string{"foo": "old"},
			map[string]string{"foo": "new"},
			"a:\n  foo: new\nb:\n  foo: new",
		},
		{
			"updates map with path when field matches multiple maps",
			"a:\n  foo: old\nb:\n  foo: old",
			".a",
			map[string]string{"foo": "old"},
			map[string]string{"foo": "new"},
			"a:\n  foo: new\nb:\n  foo: old",
		},
		{
			"doesn't update map with non-existent path",
			"a:\n  foo: a foo",
			".b",
			nil,
			map[string]string{"foo": "new"},
			"a:\n  foo: a foo",
		},
		{
			"doesn't update map with different field",
			"a:\n  foo: a foo",
			"",
			map[string]string{"foo": "not a foo"},
			map[string]string{"foo": "xxx"},
			"a:\n  foo: a foo",
		},
		{
			"doesn't update map with multiple fields that don't all match",
			"a:\n  foo: a foo\n  other: a other\nb:\n  foo: b foo",
			"",
			map[string]string{"foo": "a foo", "other": "not other"},
			map[string]string{"foo": "xxx"},
			"a:\n  foo: a foo\n  other: a other\nb:\n  foo: b foo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := yamlops2.UpdateMap(
				[]byte(test.doc), test.path, "", test.selectors, test.values,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(result), test.result; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestUpdateMapUnicode(t *testing.T) {
	tests := []struct {
		name   string
		doc    string
		path   string
		values map[string]string
		result string
	}{
		{
			"earlier unicode comment",
			"## unicöde comment\nfoo: old",
			".",
			map[string]string{"foo": "new"},
			"## unicöde comment\nfoo: new",
		},
		{
			"later unicode comment",
			"foo: old\n## unicöde comment",
			".",
			map[string]string{"foo": "new"},
			"foo: new\n## unicöde comment",
		},
		{
			"earlier unicode value",
			"x: unicöde value\nfoo: old",
			".",
			map[string]string{"foo": "new"},
			"x: unicöde value\nfoo: new",
		},
		{
			"later unicode value",
			"foo: old\nx: unicöde value",
			".",
			map[string]string{"foo": "new"},
			"foo: new\nx: unicöde value",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := yamlops2.UpdateMap(
				[]byte(test.doc), test.path, "", nil, test.values,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(result), test.result; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestUpdateMapEncodesValues(t *testing.T) {
	doc := "a:\n  foo: a foo"
	path := ".a"
	values := map[string]string{"foo": "1"}
	expected := "a:\n  foo: \"1\""

	result, err := yamlops2.UpdateMap([]byte(doc), path, "", nil, values)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(result), expected; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}

func TestUpdateMapValueNodeStyle(t *testing.T) {
	tests := []struct {
		name   string
		doc    string
		path   string
		values map[string]string
		result string
	}{
		{
			"updates unquoted value",
			"a:\n  foo: a foo",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
		{
			"updates single quoted value",
			"a:\n  foo: 'a foo'",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
		{
			"updates double quoted value",
			"a:\n  foo: \"a foo\"",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
		{
			"updates unquoted value in inline map",
			"a: {foo: a foo}",
			".a",
			map[string]string{"foo": "xxx"},
			"a: {foo: xxx}",
		},
		{
			"updates single quoted value in inline map",
			"a: {foo: 'a foo'}",
			".a",
			map[string]string{"foo": "xxx"},
			"a: {foo: xxx}",
		},
		{
			"updates double quoted value in inline map",
			"a: {foo: \"a foo\"}",
			".a",
			map[string]string{"foo": "xxx"},
			"a: {foo: xxx}",
		},
		{
			"updates flow value",
			"a:\n  foo: a\n    foo",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
		{
			"updates single quoted flow value",
			"a:\n  foo: 'a\n  foo'",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
		{
			"updates double quoted flow value",
			"a:\n  foo: \"a\n  foo\"",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
		{
			"updates block folded value",
			"a:\n  foo: >\n    a foo",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
		{
			"updates block literal value",
			"a:\n  foo: |\n    a foo",
			".a",
			map[string]string{"foo": "xxx"},
			"a:\n  foo: xxx",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := yamlops2.UpdateMap(
				[]byte(test.doc), test.path, "", nil, test.values,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(result), test.result; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestUpdateMapComments(t *testing.T) {
	path := ".a"
	values := map[string]string{"foo": "xxx"}

	tests := []struct {
		name   string
		doc    string
		result string
	}{
		{
			"retains values's head comment",
			"a:\n  # a foo head comment\n  foo: a foo",
			"a:\n  # a foo head comment\n  foo: xxx",
		},
		{
			"retains value's foot comment",
			"a:\n  foo: a foo\n  # a foo foot comment",
			"a:\n  foo: xxx\n  # a foo foot comment",
		},
		{
			"retains value's line comment",
			"a:\n  foo: a foo # a foo line comment",
			"a:\n  foo: xxx # a foo line comment",
		},
		{
			"retains maps's head comment",
			"# a head comment\na:\n  foo: a foo",
			"# a head comment\na:\n  foo: xxx",
		},
		{
			"retains map's foot comment",
			"a:\n  foo: a foo\n# a foot comment",
			"a:\n  foo: xxx\n# a foot comment",
		},
		{
			"retains map's line comment",
			"a: # a line comment\n  foo: a foo",
			"a: # a line comment\n  foo: xxx",
		},
		{
			"retains unrelated comments",
			"a: \n  foo: a foo\n\n# b head comment\nb: b\n# b foot comment",
			"a: \n  foo: xxx\n\n# b head comment\nb: b\n# b foot comment",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := yamlops2.UpdateMap([]byte(test.doc), path, "", nil, values)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(result), test.result; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestUpdateMapErrors(t *testing.T) {
	tests := []struct {
		name      string
		doc       string
		path      string
		selectors map[string]string
		values    map[string]string
		errText   string
	}{
		{
			"no matchers",
			"a:\n  foo: a foo",
			"",
			nil,
			map[string]string{"foo": "xxx"},
			"need path or selectors or both",
		},
		{
			"field to update does not exist",
			"a:\n  foo: a foo",
			".a",
			nil,
			map[string]string{"bar": "xxx"},
			"map \".a\" has no field \"bar\"",
		},
		{
			"field to update has wrong type",
			"a:\n  foo: 1",
			".a",
			nil,
			map[string]string{"foo": "xxx"},
			"map \".a\" has field \"foo\" with tag !!int but expected !!str",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := yamlops2.UpdateMap(
				[]byte(test.doc), test.path, "", test.selectors, test.values,
			)
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			if got, want := errText, test.errText; !strings.Contains(got, want) {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
