// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamlops

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// UpdateMap finds maps in a YAML doc that match a path and/or contain a
// specific set of fields. It then updates the maps' fields with new values.
//
// Values are replaced without disturbing any other content (as far as is
// possible) although the original style (inline, flow, folded, etc.) of value
// nodes being replaced may not be retained. To achieve this, the func works
// close to the YAML AST imposing some limitations:
//
// * New values must be strings.
// * Matching maps must already define the values being replaced.
// * Existing map values must be strings.
//
// It should be possible to remove some of these limitations as needed.
//
// Examples:
//
// Update a Chart's values.yaml to set a specific image's reference:
//
//     UpdateMap(doc, ".images.postgresql", "", nil, map[string]string{
//         "registry": "docker.io",
//         "repository": "bitnami/postgresql",
//         "tag": "11.6.0-debian-10-r5",
//     })
//
// Update a Chart's values.yaml to replace any uses of a general purpose image:
//
//     UpdateMap(doc, "", "",map[string]string{
//         "repository": "bitnami/minideb",
//     }, map[string]string{
//         "registry": "custom.images.org",
//         "repository": "custom-general-purpose",
//         "tag": "1.2.3",
//     })
//
// Update a Chart's dependencies to rewrite a chart registry:
//
//     UpdateMap(doc, "", ".dependencies", map[string]string{
//         "repository": "https://charts.bitnami.com/bitnami",
//     }, map[string]string{
//         "repository": "custom.charts.org",
//     })
//
func UpdateMap(doc []byte, pathSpec, selectorFilter string, selectors, values map[string]string) ([]byte, error) {
	root, err := parse(doc)
	if err != nil {
		return nil, err
	}

	// Encode new values as yaml ready for direct insertion in doc.
	// TODO(mgoodall): consider rendering later, to match the node's style.
	yamlValues := make(map[string][]byte, len(values))
	for k, v := range values {
		y, err := yaml.Marshal(v)
		if err != nil {
			return nil, err
		}
		yamlValues[k] = bytes.TrimSpace(y)
	}

	// Scan for matching maps.
	var matchers []NodeMatchFunc

	// if both selectorFilter and selectors inputs are specified, yamlops will update those yaml paths
	// matching the condition inside the provided selectorFilter with the provided value
	if selectorFilter != "" && len(selectors) != 0 {
		matchers = append(matchers, SelectorMatchFilter(selectorFilter, selectors))
	} else {
		// if only pathSpec is provided, yamlops will just update that path with the provided value
		if pathSpec != "" {
			matchers = append(matchers, NodeHasPath(pathSpec))
		}
		// if only selectors is provided, yamlops will iterate over yaml file to find a map matching
		// the provided condition and updating the provided value
		if len(selectors) != 0 {
			matchers = append(matchers, MapNodeContains(selectors))
		}
	}
	if len(matchers) == 0 {
		return nil, fmt.Errorf("need path or selectors or both")
	}
	maps := SearchNodes(root, ".", matchers...)

	// Build list of required value replacements. For now, the fields to be
	// updated must exist and the current value must be a string.
	var valueReplacements []valueReplacement
	for path, node := range maps {
		for name, value := range yamlValues {
			valueNode, ok := yamlMapFieldValueNode(node, name)
			if !ok {
				return nil, fmt.Errorf("map %q has no field %q", path, name)
			}
			// TODO(mgoodall): this may be too strict, it's probably ok to
			// replace any scalar
			if valueNode.Tag != "!!str" {
				return nil, fmt.Errorf("map %q has field %q with tag %s but expected !!str", path, name, valueNode.Tag)
			}
			valueReplacements = append(valueReplacements, valueReplacement{valueNode, value})
		}
	}

	// XXX the replacement loop below originally relied on the Node's Index &
	// EndIndex (populated from gopkg.in/yaml.v3's private yaml_mark_t type).
	//
	// Unfortunately, Index/EndIndex are affected by unicode chars that occur
	// before the replacement location.
	//
	// Fortunately, Line/Column and LineEnd/ColumnEnd are correct. To use those
	// we simply need to know the byte-position of the start of each line in the
	// yaml doc.
	lines, err := indexLines(doc)
	if err != nil {
		return nil, err
	}

	// Apply replacements in reverse location order to avoid affecting the
	// location of earlier content.
	sort.Sort(sort.Reverse(sortValueReplacementByNodeIndex(valueReplacements)))
	for _, replacement := range valueReplacements {
		node, value := replacement.node, replacement.value

		// TODO(mgoodall): this relies on a forked go-yaml.v3 that adds LineEnd
		// & ColumnEnd.
		start := lines[node.Line-1] + node.Column - 1
		end := lines[node.LineEnd-1] + node.ColumnEnd - 1

		// Replace value's content in the doc with the new value.
		// Silencing logging until we have better handling for log levels
		//log.Printf("replacing value %q from l%d:%d to l%d:%d with %q\n", node.Value, node.Line, node.Column, node.LineEnd, node.ColumnEnd, value)
		doc = append(doc[:start], append(value, doc[end:]...)...)
	}

	return doc, nil
}

// scanNewLine is a copy of `bufio.ScanLines` that does not strip any '\r'.
func scanNewLine(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// indexLines returns the 0-based index of the start of each line from the
// beginning of the doc.
func indexLines(doc []byte) ([]int, error) {
	index := []int{}

	scanner := bufio.NewScanner(bytes.NewBuffer(doc))
	scanner.Split(scanNewLine)

	offset := 0
	for scanner.Scan() {
		index = append(index, offset)
		offset += len(scanner.Bytes()) + 1
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return index, nil
}

// Parse the yaml doc and return the root node.
func parse(doc []byte) (*yaml.Node, error) {
	node := &yaml.Node{}
	if err := yaml.Unmarshal(doc, node); err != nil {
		return nil, err
	}

	// Sanity check we're starting with a document node containing a single
	// child, mostly because I (mgoodall) don't fully understand the API yet.
	if node.Kind != yaml.DocumentNode {
		return nil, fmt.Errorf("expected DocumentNode, got %d", node.Kind)
	}
	if n := len(node.Content); n != 1 {
		return nil, fmt.Errorf("expected 1 child on document node, got %d", n)
	}

	// The document node's only content node is the node (root?) with actual data.
	return node.Content[0], nil
}

// Find the named field's value in the map node.
func yamlMapFieldValueNode(node *yaml.Node, name string) (*yaml.Node, bool) {
	for i := 0; i < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if k.Value == name {
			return v, true
		}
	}
	return nil, false
}

type valueReplacement struct {
	node  *yaml.Node
	value []byte
}

type sortValueReplacementByNodeIndex []valueReplacement

func (rr sortValueReplacementByNodeIndex) Len() int {
	return len(rr)
}

func (rr sortValueReplacementByNodeIndex) Less(i, j int) bool {
	return rr[i].node.Index < rr[j].node.Index
}

func (rr sortValueReplacementByNodeIndex) Swap(i, j int) {
	rr[i], rr[j] = rr[j], rr[i]
}
