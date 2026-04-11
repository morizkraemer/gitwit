package main

import (
	"path/filepath"
	"sort"
	"strings"
)

// dirEntry is a flattened directory tree line for display
type dirEntry struct {
	display  string // pre-rendered tree line
	filePath string // actual file path relative to repo root
	isDir    bool
}

func buildDirTree(expanded map[string]bool) []dirEntry {
	files := git("ls-files", "-co", "--exclude-standard")
	if files == nil {
		return nil
	}

	root := newTreeNode("", true)

	for _, file := range files {
		parts := strings.Split(filepath.ToSlash(file), "/")
		node := root
		for i, part := range parts {
			isLast := i == len(parts)-1
			if _, ok := node.children[part]; !ok {
				node.children[part] = newTreeNode(part, !isLast)
			}
			node = node.children[part]
		}
	}

	var entries []dirEntry
	flattenDirTree(root, "", "", expanded, &entries)
	return entries
}

func flattenDirTree(node *treeNode, prefix, pathPrefix string, expanded map[string]bool, entries *[]dirEntry) {
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

		connector := "├─ "
		childPrefix := prefix + "│  "
		if isLast {
			connector = "└─ "
			childPrefix = prefix + "   "
		}

		fullPath := name
		if pathPrefix != "" {
			fullPath = pathPrefix + "/" + name
		}

		display := prefix + connector
		if child.isDir {
			if expanded[fullPath] {
				display += "▾ " + name + "/"
			} else {
				display += "▸ " + name + "/"
			}
		} else {
			display += name
		}

		*entries = append(*entries, dirEntry{
			display:  display,
			filePath: fullPath,
			isDir:    child.isDir,
		})

		if child.isDir && expanded[fullPath] {
			flattenDirTree(child, childPrefix, fullPath, expanded, entries)
		}
	}
}
