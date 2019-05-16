// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"bytes"
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	"strings"
)

// Leaf is at the end of a branch
type Leaf struct {
	Attr  string
	Value string
}

// Node is at the root or branch of a tree
type Node struct {
	Name     string
	Children []*Node
	Leaves   []*Leaf
}

func (n Node) findNode(name string) *Node {
	var listID string
	if strings.Contains(name, "=") {
		nameParts := strings.Split(name, "=")
		name = nameParts[0]
		listID = nameParts[1]
	}
	for _, child := range n.Children {
		if child.Name == name {
			if listID == "" || child.findLeaf(listID) != nil {
				return child
			}
		}
	}
	return nil
}

func (n Node) findLeaf(value string) *Leaf {
	for _, leaf := range n.Leaves {
		if leaf.Value == value {
			return leaf
		}
	}
	return nil
}

func (n *Node) addLeaf(name string, value string) {
	n.Leaves = append(n.Leaves, &Leaf{name, value})
}

func (n *Node) addNode(name string) *Node {
	newNode := Node{name, make([]*Node, 0), make([]*Leaf, 0)}
	n.Children = append(n.Children, &newNode)
	return &newNode
}

// BuildTree is a function that takes an ordered array of ConfigValues and
// produces a structured formatted tree
func BuildTree(values []change.ConfigValue, pretty bool) ([]byte, error) {
	var buf bytes.Buffer

	root := Node{Name: "(root)", Children: make([]*Node, 0), Leaves: make([]*Leaf, 0)}

	for _, cv := range values {
		addPathToTree(cv.Path, cv.Value, &root)
	}

	jsonifyNodes(&root, &buf, pretty, 1)
	return buf.Bytes(), nil
}

func addPathToTree(path, value string, node *Node) error {
	pathelems := strings.Split(path, "/")

	var thisNode *Node
	if len(pathelems) == 2 && value != "" {
		// At the end of a line - this is the leaf
		node.addLeaf(pathelems[1], value)
		return nil
	} else if strings.Contains(pathelems[1], "=") {
		thisNode = node.findNode(pathelems[1])
		if thisNode == nil {
			listNameParts := strings.Split(pathelems[1], "=")
			thisNode = node.addNode(listNameParts[0])
			thisNode.addLeaf("id", listNameParts[1])
		}
	} else {
		thisNode = node.findNode(pathelems[1])
		if thisNode == nil {
			thisNode = node.addNode(pathelems[1])
		}
	}

	refinePath := strings.Join(pathelems[2:], "/")
	if refinePath == "" {
		return nil
	}

	refinePath = fmt.Sprintf("/%s", refinePath)
	err := addPathToTree(refinePath, value, thisNode)
	if err != nil {
		return err
	}

	return nil
}

func jsonifyNodes(n *Node, buf *bytes.Buffer, pretty bool, tablevel int) {
	var (
		newLine = ""
		tab = ""
	)
	if pretty {
		newLine = "\n"
		tabs := make([]string, tablevel)
		tab = strings.Join(tabs, "\t")
	}

	fmt.Fprintf(buf, "%s{\"%s\": [", tab, n.Name)
	var isFirst = true
	for _, c := range n.Children {
		if isFirst {
			isFirst = false
			fmt.Fprintf(buf, "%s", newLine)
		} else {
			fmt.Fprintf(buf, ",%s", newLine)
		}
		jsonifyNodes(c, buf, pretty, tablevel + 1)
	}
	if len(n.Leaves) > 0 {
		if isFirst {
			fmt.Fprintf(buf, "%s", newLine)
			isFirst = true
		} else {
			fmt.Fprintf(buf, ",%s", newLine)
		}
		fmt.Fprintf(buf, "%s{", tab + "\t")
		for _, l := range n.Leaves {
			if isFirst {
				isFirst = false
				fmt.Fprintf(buf, "%s", newLine)
			} else {
				fmt.Fprintf(buf, ",%s", newLine)
			}
			fmt.Fprintf(buf, "%s\"%s\":\"%s\"", tab + "\t", l.Attr, l.Value)
		}
		fmt.Fprintf(buf, "%s}%s", tab, newLine)
	}
	fmt.Fprintf(buf, "%s]}", tab)
}

func xmlifyNodes(n *Node, buf *bytes.Buffer) {
	if n.Name == "(root)" {
		fmt.Fprintf(buf, "<?xml version=\"1.0\"?><root>")
	} else {
		fmt.Fprintf(buf, "<%s>", n.Name)
	}
	for _, c := range n.Children {
		xmlifyNodes(c, buf)
	}
	if len(n.Leaves) > 0 {
		for _, l := range n.Leaves {
			fmt.Fprintf(buf, "<%s>%s</%s>", l.Attr, l.Value, l.Attr)
		}
	}
	if n.Name == "(root)" {
		fmt.Fprintf(buf, "</root>")
	} else {
		fmt.Fprintf(buf, "</%s>", n.Name)
	}
}
