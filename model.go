package main

import "time"

type tickMsg time.Time

type gitOpResult struct {
	op  string
	err error
	out string
}

// Panel indices
const (
	panelChanges  = 0
	panelBranches = 1
	panelCommits  = 2
)

// changeEntry is a flattened tree line for display
type changeEntry struct {
	display  string // pre-rendered tree line like "├── file.go"
	filePath string // actual file path relative to repo root
	status   string // git status code (e.g. "M ", "??")
	isDir    bool
}

// branchEntry holds branch name and remote tracking info
type branchEntry struct {
	name   string
	ahead  int
	behind int
}

// remoteBranchEntry holds a remote branch reference
type remoteBranchEntry struct {
	name   string // e.g. "origin/feature-x"
	remote string // e.g. "origin"
	branch string // e.g. "feature-x"
}

type model struct {
	changes        []changeEntry
	changesRaw     []string // raw porcelain lines for count
	branches       []branchEntry
	remoteBranches []remoteBranchEntry
	commits        []string
	currentBranch  string

	activePanel  int
	cursors      [3]int
	offsets      [3]int
	branchSub    int // 0 = local, 1 = remote
	remoteCursor int
	remoteOffset int

	// Diff preview overlay
	diffMode   bool
	diffLines  []string
	diffFile   string
	diffScroll int

	// Text input mode (e.g. new branch name, commit message)
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputAction string // "branch" or "commit"

	// Status message (shown briefly, cleared after one tick)
	statusMsg  string
	statusTick int

	width  int
	height int
}

func (m model) selectedBranch() string {
	if len(m.branches) == 0 {
		return ""
	}
	return m.branches[m.cursors[panelBranches]].name
}

func (m model) panelItems(panel int) []string {
	switch panel {
	case panelChanges:
		items := make([]string, len(m.changes))
		for i, c := range m.changes {
			items[i] = c.display
		}
		return items
	case panelBranches:
		items := make([]string, len(m.branches))
		for i, b := range m.branches {
			items[i] = b.name
		}
		return items
	case panelCommits:
		return m.commits
	}
	return nil
}
