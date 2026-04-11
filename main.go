package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func initialModel() model {
	branches := loadBranches()
	cur := currentBranch()
	cursorIdx := 0
	for i, b := range branches {
		if b.name == cur {
			cursorIdx = i
			break
		}
	}
	raw := loadChanges()
	m := model{
		changesRaw:     raw,
		changes:        buildChangeTree(raw),
		branches:       branches,
		remoteBranches: loadRemoteBranches(branches),
		currentBranch:  cur,
		activePanel: panelChanges,
		showPanel:   [3]bool{true, true, true},
	}
	m.cursors[panelBranches] = cursorIdx
	m.commits = loadCommits(m.selectedBranch())
	return m
}

func main() {
	// Check we're in a git repo
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Not a git repository")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
