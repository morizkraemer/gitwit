package main

import (
	"path/filepath"
	"sort"
	"strings"
)

// treeNode builds the file tree
type treeNode struct {
	name     string
	status   string // only for files
	children map[string]*treeNode
	isDir    bool
}

func newTreeNode(name string, isDir bool) *treeNode {
	return &treeNode{name: name, isDir: isDir, children: make(map[string]*treeNode)}
}

func buildChangeTree(porcelain []string) []changeEntry {
	root := newTreeNode("", true)

	for _, line := range porcelain {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])
		parts := strings.Split(filepath.ToSlash(file), "/")

		node := root
		for i, part := range parts {
			isLast := i == len(parts)-1
			if _, ok := node.children[part]; !ok {
				node.children[part] = newTreeNode(part, !isLast)
			}
			node = node.children[part]
			if isLast {
				node.status = status
			}
		}
	}

	// Skip single-child directory wrapper nodes at the top
	pathPrefix := ""
	collapsed := root
	for len(collapsed.children) == 1 {
		var only *treeNode
		var onlyName string
		for n, v := range collapsed.children {
			only = v
			onlyName = n
		}
		if !only.isDir {
			break
		}
		if pathPrefix == "" {
			pathPrefix = onlyName
		} else {
			pathPrefix = pathPrefix + "/" + onlyName
		}
		collapsed = only
	}

	var entries []changeEntry
	flattenTree(collapsed, "", pathPrefix, &entries)
	return entries
}

func flattenTree(node *treeNode, prefix string, pathPrefix string, entries *[]changeEntry) {
	names := make([]string, 0, len(node.children))
	for name := range node.children {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		ci := node.children[names[i]]
		cj := node.children[names[j]]
		if ci.isDir != cj.isDir {
			return ci.isDir
		}
		return names[i] < names[j]
	})

	for i, name := range names {
		child := node.children[name]
		isLast := i == len(names)-1

		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		fullPath := name
		if pathPrefix != "" {
			fullPath = pathPrefix + "/" + name
		}

		display := prefix + connector + name
		if child.isDir {
			display += "/"
		}

		*entries = append(*entries, changeEntry{
			display:  display,
			filePath: fullPath,
			status:   child.status,
			isDir:    child.isDir,
		})

		if child.isDir {
			flattenTree(child, childPrefix, fullPath, entries)
		}
	}
}
