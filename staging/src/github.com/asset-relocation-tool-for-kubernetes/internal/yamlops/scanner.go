// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamlops

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// NodeScanner performs a depth-first traversal of a yaml node tree, yielding
// each node with its jq-like path in the document.
type NodeScanner struct {
	rootNode *yaml.Node
	rootPath string

	stack []*stackItem
}

// NewNodeScanner returns a NodeScanner to traverse the yaml node graph starting
// at the root node and path.
func NewNodeScanner(root *yaml.Node, path string) *NodeScanner {
	return &NodeScanner{rootNode: root, rootPath: path}
}

type stackItem struct {
	node    *yaml.Node
	path    string
	content int
}

// Next moves the scanner to the next value node.
func (ns *NodeScanner) Next() bool {
	// Initialise stack with root node & path.
	if len(ns.stack) == 0 {
		ns.stack = []*stackItem{{ns.rootNode, ns.rootPath, 0}}
		return true
	}

	for len(ns.stack) > 0 {
		top := ns.stack[len(ns.stack)-1]

		// Push more "content" to stack and return.
		if top.content < len(top.node.Content) {
			switch top.node.Tag {
			case "!!map":
				name := top.node.Content[top.content]
				value := top.node.Content[top.content+1]
				// TODO(mgoodall): trim probably shouldn't happen, but is
				// currently needed to handle children of the root (`.`) node.
				ns.stack = append(ns.stack, &stackItem{value, strings.TrimRight(top.path, ".") + "." + name.Value, 0})
				top.content += 2
			case "!!seq":
				content := top.node.Content[top.content]
				ns.stack = append(ns.stack, &stackItem{content, fmt.Sprintf("%s[%d]", top.path, top.content), 0})
				top.content++
			default:
				// If triggered, tag type probably needs supporting/skipping.
				panic(top.node.Tag)
			}
			return true
		}

		// Unwind stack to check for more "content" on parent nodes.
		ns.stack = ns.stack[:len(ns.stack)-1]
	}

	return false
}

// Current returns the scanner's current node and path.
func (ns *NodeScanner) Current() (*yaml.Node, string) {
	current := ns.stack[len(ns.stack)-1]
	return current.node, current.path
}
