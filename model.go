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
	name       string
	upstream   string // e.g. "origin/main"
	ahead      int
	behind     int
	mainAhead  int // commits ahead of main
	mainBehind int // commits behind main
}

// remoteBranchEntry holds a remote branch reference
type remoteBranchEntry struct {
	name   string // e.g. "origin/feature-x"
	remote string // e.g. "origin"
	branch string // e.g. "feature-x"
}

// worktreeEntry holds a git worktree
type worktreeEntry struct {
	path   string // absolute path
	branch string // branch checked out
	bare   bool   // is bare repo
	head   string // short commit hash
}

// editorFinishedMsg is sent when an external editor process exits
type editorFinishedMsg struct {
	err error
}

// refreshMsg carries async-loaded data back to Update
type refreshMsg struct {
	currentBranch  string
	changesRaw     []string
	changes        []changeEntry
	diffAdded      int
	diffRemoved    int
	branches       []branchEntry
	remoteBranches []remoteBranchEntry
	worktrees      []worktreeEntry
	commits        []string
}

// mdRenderedMsg is sent when async markdown rendering completes
type mdRenderedMsg struct {
	lines []string
	file  string
	err   error
}

type model struct {
	changes        []changeEntry
	changesRaw     []string // raw porcelain lines for count
	diffAdded      int      // total lines added across all changes
	diffRemoved    int      // total lines removed across all changes
	branches       []branchEntry
	remoteBranches []remoteBranchEntry
	commits        []string
	currentBranch  string

	activePanel  int
	cursors      [3]int
	offsets      [3]int
	branchTab    int // 0=local, 1=remote, 2=worktree
	remoteCursor int
	remoteOffset int
	worktrees       []worktreeEntry
	worktreeCursor  int
	worktreeOffset  int

	// Directory browser mode (replaces changes panel)
	dirMode     bool
	dirEntries  []dirEntry
	dirExpanded map[string]bool
	dirCursor   int
	dirOffset   int

	// Panel visibility (at least one must remain true)
	showPanel [3]bool

	// Diff preview overlay
	diffMode   bool
	diffLines  []string
	diffFile   string
	diffScroll int

	// Markdown preview overlay
	mdMode   bool
	mdLines  []string
	mdFile   string
	mdCursor int
	mdOffset int

	// Commit detail (inline expansion)
	expandedCommit int // index of expanded commit, -1 = none
	commitDetail   []string

	// Confirm mode (e.g. discard changes)
	confirmMode   bool
	confirmAction string
	confirmFile   string

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
	idx := m.cursors[panelBranches]
	if idx >= len(m.branches) {
		idx = len(m.branches) - 1
	}
	return m.branches[idx].name
}

func (m model) visiblePanels() []int {
	var panels []int
	for i, show := range m.showPanel {
		if show {
			panels = append(panels, i)
		}
	}
	return panels
}

func (m model) visibleCount() int {
	n := 0
	for _, show := range m.showPanel {
		if show {
			n++
		}
	}
	return n
}

func (m model) panelItems(panel int) []string {
	switch panel {
	case panelChanges:
		if m.dirMode {
			items := make([]string, len(m.dirEntries))
			for i, d := range m.dirEntries {
				items[i] = d.display
			}
			return items
		}
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
