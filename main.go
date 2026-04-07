package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

	activePanel    int
	cursors        [3]int
	offsets        [3]int
	branchSub      int // 0 = local, 1 = remote
	remoteCursor   int
	remoteOffset   int

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

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#d4d4d4")).
			Background(lipgloss.Color("#3b3b5c")).
			Padding(0, 1)

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#8888bb"))

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#505050"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f0f0f0")).
			Background(lipgloss.Color("#3d5278"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a0a0cc"))

	branchCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#88cc88"))

	statusAddedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#88cc88"))

	statusModifiedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ddbb66"))

	statusDeletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cc6666"))

	hashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ddbb66"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#787878"))

	treeDirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8899bb"))

	treeConnectorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666688"))

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88cc88"))

	diffDelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cc6666"))

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8888bb"))

	diffHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#d4d4d4"))

	aheadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88cc88"))

	behindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cc6666"))
)

func git(args ...string) []string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func currentBranch() string {
	lines := git("rev-parse", "--abbrev-ref", "HEAD")
	if len(lines) > 0 {
		return lines[0]
	}
	return ""
}

func loadChanges() []string {
	return git("status", "--porcelain")
}

func loadBranches() []branchEntry {
	raw := git("branch", "--format=%(refname:short)")
	var entries []branchEntry
	for _, name := range raw {
		ahead, behind := 0, 0
		// Check if branch has an upstream
		ab := git("rev-list", "--left-right", "--count", name+"..."+name+"@{upstream}")
		if len(ab) > 0 {
			fmt.Sscanf(ab[0], "%d\t%d", &ahead, &behind)
		}
		entries = append(entries, branchEntry{name: name, ahead: ahead, behind: behind})
	}
	return entries
}

func loadRemoteBranches(localBranches []branchEntry) []remoteBranchEntry {
	raw := git("branch", "-r", "--format=%(refname:short)")
	localSet := make(map[string]bool)
	for _, b := range localBranches {
		localSet[b.name] = true
	}
	var entries []remoteBranchEntry
	for _, name := range raw {
		// Skip HEAD pointers like "origin/HEAD"
		if strings.Contains(name, "/HEAD") {
			continue
		}
		// Extract remote and branch name
		parts := strings.SplitN(name, "/", 2)
		if len(parts) != 2 {
			continue
		}
		// Skip if a local branch with same name already exists
		if localSet[parts[1]] {
			continue
		}
		entries = append(entries, remoteBranchEntry{
			name:   name,
			remote: parts[0],
			branch: parts[1],
		})
	}
	return entries
}

func loadDiff(filePath, status string) []string {
	var args []string
	if strings.Contains(status, "?") {
		// Untracked file — just show the whole file content
		cmd := exec.Command("cat", filePath)
		out, err := cmd.Output()
		if err != nil {
			return []string{"(cannot read file)"}
		}
		lines := strings.Split(string(out), "\n")
		result := []string{fmt.Sprintf("=== new file: %s ===", filePath), ""}
		for i, l := range lines {
			result = append(result, fmt.Sprintf("+  %4d  %s", i+1, l))
		}
		return result
	}
	// Staged and unstaged diffs combined
	args = []string{"diff", "HEAD", "--", filePath}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		// Try just staged
		cmd = exec.Command("git", "diff", "--cached", "--", filePath)
		out, _ = cmd.Output()
	}
	if len(out) == 0 {
		// Try unstaged
		cmd = exec.Command("git", "diff", "--", filePath)
		out, _ = cmd.Output()
	}
	if len(out) == 0 {
		return []string{"(no diff available)"}
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n")
}

func switchBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}


func loadCommits(branch string) []string {
	if branch == "" {
		return nil
	}
	return git("log", branch, "--oneline", "-30")
}

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
		activePanel:    panelChanges,
	}
	m.cursors[panelBranches] = cursorIdx
	m.commits = loadCommits(m.selectedBranch())
	return m
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

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.ClearScreen

	case gitOpResult:
		if msg.err != nil {
			m.statusMsg = "✗ " + msg.op + ": " + strings.TrimSpace(msg.out)
		} else {
			m.statusMsg = "✓ " + msg.op + " complete"
			// Reload after successful git op
			raw := loadChanges()
			m.changesRaw = raw
			m.changes = buildChangeTree(raw)
			m.branches = loadBranches()
			m.remoteBranches = loadRemoteBranches(m.branches)
			m.commits = loadCommits(m.selectedBranch())
		}
		return m, nil

	case tickMsg:
		if !m.diffMode && !m.inputMode {
			raw := loadChanges()
			m.changesRaw = raw
			m.changes = buildChangeTree(raw)
			m.branches = loadBranches()
			m.remoteBranches = loadRemoteBranches(m.branches)
			m.commits = loadCommits(m.selectedBranch())
		}
		if m.statusMsg != "" {
			if m.statusTick >= 2 {
				m.statusMsg = ""
				m.statusTick = 0
			} else {
				m.statusTick++
			}
		} else {
			m.statusTick = 0
		}
		return m, tickCmd()

	case tea.KeyMsg:
		// Text input mode
		if m.inputMode {
			switch msg.String() {
			case "esc":
				m.inputMode = false
				m.inputValue = ""
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.inputValue)
				action := m.inputAction
				m.inputMode = false
				m.inputValue = ""
				if value == "" {
					return m, nil
				}
				switch action {
				case "branch":
					cmd := exec.Command("git", "checkout", "-b", value)
					out, err := cmd.CombinedOutput()
					if err != nil {
						m.statusMsg = "✗ " + strings.TrimSpace(string(out))
						return m, nil
					}
					m.currentBranch = value
					m.statusMsg = "Created & switched to " + value
					m.branches = loadBranches()
					m.remoteBranches = loadRemoteBranches(m.branches)
					for i, b := range m.branches {
						if b.name == value {
							m.cursors[panelBranches] = i
							break
						}
					}
					m.commits = loadCommits(value)
				case "commit":
					cmd := exec.Command("git", "commit", "-m", value)
					out, err := cmd.CombinedOutput()
					if err != nil {
						m.statusMsg = "✗ " + strings.TrimSpace(string(out))
						return m, nil
					}
					_ = out
					m.statusMsg = "✓ Committed: " + value
					raw := loadChanges()
					m.changesRaw = raw
					m.changes = buildChangeTree(raw)
					m.cursors[panelChanges] = 0
					m.offsets[panelChanges] = 0
					m.commits = loadCommits(m.selectedBranch())
					m.branches = loadBranches()
					m.remoteBranches = loadRemoteBranches(m.branches)
				}
				return m, nil
			case "backspace":
				if len(m.inputValue) > 0 {
					m.inputValue = m.inputValue[:len(m.inputValue)-1]
				}
				return m, nil
			default:
				// Only accept printable single chars
				k := msg.String()
				if len(k) == 1 && k[0] >= 32 && k[0] <= 126 {
					m.inputValue += k
				}
				return m, nil
			}
		}

		// Diff preview mode keys
		if m.diffMode {
			switch msg.String() {
			case "q", "esc":
				m.diffMode = false
				return m, nil
			case "j", "down":
				maxScroll := len(m.diffLines) - (m.height - 4)
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.diffScroll < maxScroll {
					m.diffScroll++
				}
				return m, nil
			case "k", "up":
				if m.diffScroll > 0 {
					m.diffScroll--
				}
				return m, nil
			case "d", "ctrl+d":
				jump := (m.height - 4) / 2
				maxScroll := len(m.diffLines) - (m.height - 4)
				if maxScroll < 0 {
					maxScroll = 0
				}
				m.diffScroll += jump
				if m.diffScroll > maxScroll {
					m.diffScroll = maxScroll
				}
				return m, nil
			case "u", "ctrl+u":
				jump := (m.height - 4) / 2
				m.diffScroll -= jump
				if m.diffScroll < 0 {
					m.diffScroll = 0
				}
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.activePanel = (m.activePanel + 1) % 3
			m.statusMsg = ""
			return m, nil

		case "shift+tab":
			m.activePanel = (m.activePanel + 2) % 3
			m.statusMsg = ""
			return m, nil

		case "enter":
			switch m.activePanel {
			case panelChanges:
				if len(m.changes) > 0 {
					entry := m.changes[m.cursors[panelChanges]]
					if !entry.isDir {
						m.diffLines = loadDiff(entry.filePath, entry.status)
						m.diffFile = entry.filePath
						m.diffScroll = 0
						m.diffMode = true
					}
				}
				return m, nil

			case panelBranches:
				if m.branchSub == 1 {
					// Remote branch: checkout as local tracking branch
					if len(m.remoteBranches) > 0 && m.remoteCursor < len(m.remoteBranches) {
						rb := m.remoteBranches[m.remoteCursor]
						cmd := exec.Command("git", "checkout", "-b", rb.branch, "--track", rb.name)
						out, err := cmd.CombinedOutput()
						if err != nil {
							// Maybe local branch already exists, try just checkout
							cmd2 := exec.Command("git", "checkout", "--track", rb.name)
							out2, err2 := cmd2.CombinedOutput()
							if err2 != nil {
								m.statusMsg = "✗ " + strings.TrimSpace(string(out))
								return m, nil
							}
							out = out2
							_ = out
						}
						m.currentBranch = rb.branch
						m.statusMsg = "Checked out " + rb.branch + " from " + rb.name
						m.branches = loadBranches()
						m.remoteBranches = loadRemoteBranches(m.branches)
						for i, b := range m.branches {
							if b.name == rb.branch {
								m.cursors[panelBranches] = i
								break
							}
						}
						m.branchSub = 0
						raw := loadChanges()
						m.changesRaw = raw
						m.changes = buildChangeTree(raw)
						m.commits = loadCommits(m.selectedBranch())
					}
					return m, nil
				}
				target := m.selectedBranch()
				if target == m.currentBranch {
					m.statusMsg = "Already on " + target
					return m, nil
				}
				err := switchBranch(target)
				if err != nil {
					m.statusMsg = "✗ " + err.Error()
					return m, nil
				}
				m.currentBranch = target
				m.statusMsg = "Switched to " + target
				// Reload everything
				raw := loadChanges()
				m.changesRaw = raw
				m.changes = buildChangeTree(raw)
				m.branches = loadBranches()
				m.remoteBranches = loadRemoteBranches(m.branches)
				m.commits = loadCommits(m.selectedBranch())
				return m, nil
			}
			return m, nil

		case "j", "down":
			if m.activePanel == panelBranches && m.branchSub == 1 {
				if m.remoteCursor < len(m.remoteBranches)-1 {
					m.remoteCursor++
				}
				return m, nil
			}
			items := m.panelItems(m.activePanel)
			if m.cursors[m.activePanel] < len(items)-1 {
				m.cursors[m.activePanel]++
			}
			if m.activePanel == panelBranches {
				m.commits = loadCommits(m.selectedBranch())
				m.cursors[panelCommits] = 0
				m.offsets[panelCommits] = 0
			}
			return m, nil

		case "k", "up":
			if m.activePanel == panelBranches && m.branchSub == 1 {
				if m.remoteCursor > 0 {
					m.remoteCursor--
				}
				return m, nil
			}
			if m.cursors[m.activePanel] > 0 {
				m.cursors[m.activePanel]--
			}
			if m.activePanel == panelBranches {
				m.commits = loadCommits(m.selectedBranch())
				m.cursors[panelCommits] = 0
				m.offsets[panelCommits] = 0
			}
			return m, nil

		case "h", "left":
			if m.activePanel == panelBranches && m.branchSub == 1 {
				m.branchSub = 0
			}
			return m, nil

		case "l", "right":
			if m.activePanel == panelBranches && m.branchSub == 0 {
				m.branchSub = 1
			}
			return m, nil

		case "r":
			raw := loadChanges()
			m.changesRaw = raw
			m.changes = buildChangeTree(raw)
			m.branches = loadBranches()
			m.remoteBranches = loadRemoteBranches(m.branches)
			m.commits = loadCommits(m.selectedBranch())
			return m, nil

		case " ":
			if m.activePanel == panelChanges && len(m.changes) > 0 {
				entry := m.changes[m.cursors[panelChanges]]
				if !entry.isDir {
					status := entry.status
					if strings.TrimSpace(status[:1]) != "" {
						// File has staged changes — unstage it
						exec.Command("git", "reset", "HEAD", "--", entry.filePath).Run()
					} else {
						// File is unstaged — stage it
						exec.Command("git", "add", "--", entry.filePath).Run()
					}
					raw := loadChanges()
					m.changesRaw = raw
					m.changes = buildChangeTree(raw)
					if m.cursors[panelChanges] >= len(m.changes) {
						m.cursors[panelChanges] = len(m.changes) - 1
					}
					if m.cursors[panelChanges] < 0 {
						m.cursors[panelChanges] = 0
					}
				}
			}
			return m, nil

		case "a":
			if m.activePanel == panelChanges && len(m.changes) > 0 {
				// Stage all files
				exec.Command("git", "add", "-A").Run()
				raw := loadChanges()
				m.changesRaw = raw
				m.changes = buildChangeTree(raw)
				m.statusMsg = "Staged all files"
			}
			return m, nil

		case "c":
			if m.activePanel == panelChanges {
				// Check if there are staged changes
				staged := git("diff", "--cached", "--name-only")
				if len(staged) == 0 {
					m.statusMsg = "✗ Nothing staged to commit"
					return m, nil
				}
				m.inputMode = true
				m.inputPrompt = "Commit message: "
				m.inputValue = ""
				m.inputAction = "commit"
				m.statusMsg = ""
			}
			return m, nil

		case "B":
			if m.activePanel == panelBranches {
				m.inputMode = true
				m.inputPrompt = "New branch name: "
				m.inputValue = ""
				m.inputAction = "branch"
				m.statusMsg = ""
			}
			return m, nil

		case "f":
			if m.activePanel == panelBranches {
				m.statusMsg = "Fetching..."
				return m, func() tea.Msg {
					cmd := exec.Command("git", "fetch", "--all", "--prune")
					out, err := cmd.CombinedOutput()
					return gitOpResult{op: "fetch", err: err, out: string(out)}
				}
			}
			return m, nil

		case "p":
			if m.activePanel == panelBranches {
				m.statusMsg = "Pulling..."
				return m, func() tea.Msg {
					cmd := exec.Command("git", "pull")
					out, err := cmd.CombinedOutput()
					return gitOpResult{op: "pull", err: err, out: string(out)}
				}
			}
			return m, nil

		case "P":
			if m.activePanel == panelBranches {
				m.statusMsg = "Pushing..."
				return m, func() tea.Msg {
					cmd := exec.Command("git", "push")
					out, err := cmd.CombinedOutput()
					return gitOpResult{op: "push", err: err, out: string(out)}
				}
			}
			return m, nil
		}
	}
	return m, nil
}

var outerBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#5a5a7a"))

func (m model) renderDiffView() string {
	contentWidth := m.width - 2
	viewHeight := m.height - 4 // outer border + title + help

	title := titleStyle.Render(fmt.Sprintf(" %s ", m.diffFile))
	help := dimStyle.Render("  q/esc: close · j/k: scroll · d/u: page down/up")

	var lines []string
	end := m.diffScroll + viewHeight
	if end > len(m.diffLines) {
		end = len(m.diffLines)
	}
	for i := m.diffScroll; i < end; i++ {
		line := m.diffLines[i]
		// Truncate to width
		if len(line) > contentWidth-2 {
			line = line[:contentWidth-2]
		}
		switch {
		case strings.HasPrefix(m.diffLines[i], "+"):
			lines = append(lines, diffAddStyle.Render(line))
		case strings.HasPrefix(m.diffLines[i], "-"):
			lines = append(lines, diffDelStyle.Render(line))
		case strings.HasPrefix(m.diffLines[i], "@@"):
			lines = append(lines, diffHunkStyle.Render(line))
		case strings.HasPrefix(m.diffLines[i], "diff"), strings.HasPrefix(m.diffLines[i], "index"), strings.HasPrefix(m.diffLines[i], "==="):
			lines = append(lines, diffHeaderStyle.Render(line))
		default:
			lines = append(lines, line)
		}
	}
	for len(lines) < viewHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	inner := lipgloss.JoinVertical(lipgloss.Left, title, content, help)
	return outerBorderStyle.
		Width(contentWidth).
		Height(m.height - 2).
		Render(inner)
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.diffMode {
		return m.renderDiffView()
	}

	// Outer border eats 2 cols and 2 rows
	contentWidth := m.width - 2
	innerWidth := contentWidth - 4 // panel borders

	// 3 titles (1 line each) + 3 panel borders (2 lines each) + 3 help lines + 3 blank lines + 1 status line + 2 outer border = 18 lines of chrome
	available := m.height - 18
	if available < 9 {
		available = 9
	}

	// Upper half: changes, next quarter: branches, bottom quarter: commits
	changesHeight := available / 2
	branchesHeight := available / 4
	commitsHeight := available - changesHeight - branchesHeight

	if changesHeight < 3 {
		changesHeight = 3
	}
	if branchesHeight < 2 {
		branchesHeight = 2
	}
	if commitsHeight < 2 {
		commitsHeight = 2
	}

	changesView := m.renderPanel(panelChanges, innerWidth, changesHeight)
	branchesView := m.renderBranchesPanel(innerWidth, branchesHeight)
	commitsView := m.renderPanel(panelCommits, innerWidth, commitsHeight)

	// Titles
	stagedCount := 0
	for _, line := range m.changesRaw {
		if len(line) >= 2 && strings.TrimSpace(line[:1]) != "" && line[:1] != "?" {
			stagedCount++
		}
	}
	changesTitle := titleStyle.Render(fmt.Sprintf(" Changes (%d) · Staged (%d) ", len(m.changesRaw), stagedCount))
	branchesTitle := titleStyle.Render(fmt.Sprintf(" Local (%d) │ Remote (%d) ", len(m.branches), len(m.remoteBranches)))

	commitLabel := m.selectedBranch()
	if commitLabel == "" {
		commitLabel = "none"
	}
	commitsTitle := titleStyle.Render(fmt.Sprintf(" Commits · %s ", commitLabel))

	borderFn := func(panel int, h int) lipgloss.Style {
		if panel == m.activePanel {
			return activeBorderStyle.Width(innerWidth).Height(h)
		}
		return inactiveBorderStyle.Width(innerWidth).Height(h)
	}

	changesHelp := dimStyle.Render("  space: stage/unstage · a: stage all · c: commit · enter: diff · r: refresh")
	branchesHelp := dimStyle.Render("  h/l: local/remote · enter: checkout · B: new · f: fetch · p: pull · P: push")
	commitsHelp := dimStyle.Render("  j/k: navigate")

	var statusLine string
	if m.inputMode {
		statusLine = dimStyle.Render("  " + m.inputPrompt + m.inputValue + "█")
	} else if m.statusMsg != "" {
		statusLine = "  " + m.statusMsg
	} else {
		statusLine = dimStyle.Render("  tab: switch panel · q: quit")
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		changesTitle,
		borderFn(panelChanges, changesHeight).Render(changesView),
		changesHelp,
		"",
		branchesTitle,
		borderFn(panelBranches, branchesHeight).Render(branchesView),
		branchesHelp,
		"",
		commitsTitle,
		borderFn(panelCommits, commitsHeight).Render(commitsView),
		commitsHelp,
		"",
		statusLine,
	)

	return outerBorderStyle.
		Width(contentWidth).
		Height(m.height - 2).
		Render(inner)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func (m *model) renderBranchesPanel(width, height int) string {
	isActive := m.activePanel == panelBranches
	leftWidth := width / 2
	rightWidth := width - leftWidth - 1 // -1 for separator
	if rightWidth < 1 {
		rightWidth = 1
	}
	if leftWidth < 1 {
		leftWidth = 1
	}

	// Render local branches (left side)
	var leftLines []string
	cursor := m.cursors[panelBranches]
	if cursor < m.offsets[panelBranches] {
		m.offsets[panelBranches] = cursor
	}
	if cursor >= m.offsets[panelBranches]+height {
		m.offsets[panelBranches] = cursor - height + 1
	}
	for i := m.offsets[panelBranches]; i < len(m.branches) && i < m.offsets[panelBranches]+height; i++ {
		b := m.branches[i]
		isSelected := i == cursor && isActive && m.branchSub == 0
		isCursor := i == cursor && !isSelected
		display := m.branchDisplay(i)
		prefix := "  "
		if b.name == m.currentBranch {
			prefix = "● "
		}
		plain := truncate(prefix+display, leftWidth)
		if isSelected {
			leftLines = append(leftLines, selectedStyle.Width(leftWidth).Render(plain))
		} else if isCursor {
			leftLines = append(leftLines, cursorStyle.Width(leftWidth).Render(plain))
		} else {
			// Build styled version
			suffix := ""
			if b.ahead > 0 {
				suffix += " " + aheadStyle.Render(fmt.Sprintf("↑%d", b.ahead))
			}
			if b.behind > 0 {
				suffix += " " + behindStyle.Render(fmt.Sprintf("↓%d", b.behind))
			}
			if b.name == m.currentBranch {
				leftLines = append(leftLines, branchCurrentStyle.Render(truncate("● "+b.name, leftWidth))+suffix)
			} else {
				leftLines = append(leftLines, "  "+truncate(b.name, leftWidth-2)+suffix)
			}
		}
	}
	for len(leftLines) < height {
		leftLines = append(leftLines, strings.Repeat(" ", leftWidth))
	}

	// Render remote branches (right side)
	var rightLines []string
	if m.remoteCursor < m.remoteOffset {
		m.remoteOffset = m.remoteCursor
	}
	if m.remoteCursor >= m.remoteOffset+height {
		m.remoteOffset = m.remoteCursor - height + 1
	}
	for i := m.remoteOffset; i < len(m.remoteBranches) && i < m.remoteOffset+height; i++ {
		rb := m.remoteBranches[i]
		isSelected := i == m.remoteCursor && isActive && m.branchSub == 1
		plain := truncate("  "+rb.name, rightWidth)
		if isSelected {
			rightLines = append(rightLines, selectedStyle.Width(rightWidth).Render(plain))
		} else {
			styled := "  " + dimStyle.Render(rb.remote+"/") + rb.branch
			rightLines = append(rightLines, truncate(styled, rightWidth+10)) // extra room for ANSI
		}
	}
	for len(rightLines) < height {
		rightLines = append(rightLines, strings.Repeat(" ", rightWidth))
	}

	// Combine left and right with separator
	sep := dimStyle.Render("│")
	var lines []string
	for i := 0; i < height; i++ {
		lines = append(lines, leftLines[i]+sep+rightLines[i])
	}

	return strings.Join(lines, "\n")
}

func (m *model) renderPanel(panel, width, height int) string {
	items := m.panelItems(panel)
	cursor := m.cursors[panel]

	// Scroll offset
	if cursor < m.offsets[panel] {
		m.offsets[panel] = cursor
	}
	if cursor >= m.offsets[panel]+height {
		m.offsets[panel] = cursor - height + 1
	}

	var lines []string
	for i := m.offsets[panel]; i < len(items) && i < m.offsets[panel]+height; i++ {
		line := truncate(items[i], width)
		isSelected := i == cursor && panel == m.activePanel
		isCursor := i == cursor && !isSelected
		if isSelected {
			plain := truncate(m.plainLine(panel, i, items[i]), width)
			rendered := selectedStyle.Width(width).Render(plain)
			lines = append(lines, rendered)
		} else if isCursor {
			plain := truncate(m.plainLine(panel, i, items[i]), width)
			rendered := cursorStyle.Width(width).Render(plain)
			lines = append(lines, rendered)
		} else {
			rendered := m.renderLine(panel, i, line, width)
			lines = append(lines, rendered)
		}
	}

	// Pad remaining lines
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

func (m model) branchDisplay(idx int) string {
	if idx >= len(m.branches) {
		return ""
	}
	b := m.branches[idx]
	s := b.name
	if b.ahead > 0 || b.behind > 0 {
		s += " "
		if b.ahead > 0 {
			s += fmt.Sprintf("↑%d", b.ahead)
		}
		if b.behind > 0 {
			s += fmt.Sprintf("↓%d", b.behind)
		}
	}
	return s
}

func statusTag(status string) string {
	if len(status) < 2 {
		return dimStyle.Render("?")
	}
	x, y := status[0], status[1]
	// Staged status (first char)
	if x == 'M' || x == 'A' || x == 'D' || x == 'R' || x == 'C' {
		return branchCurrentStyle.Render(string(x))
	}
	// Unstaged status (second char)
	if y == 'M' {
		return statusModifiedStyle.Render("M")
	}
	if y == 'D' {
		return statusDeletedStyle.Render("D")
	}
	if x == '?' {
		return statusAddedStyle.Render("?")
	}
	return dimStyle.Render(string(y))
}

func statusTagPlain(status string) string {
	if len(status) < 2 {
		return "?"
	}
	x, y := status[0], status[1]
	if x == 'M' || x == 'A' || x == 'D' || x == 'R' || x == 'C' {
		return string(x)
	}
	if y == 'M' || y == 'D' {
		return string(y)
	}
	if x == '?' {
		return "?"
	}
	return string(y)
}

func (m model) plainLine(panel, idx int, line string) string {
	switch panel {
	case panelChanges:
		if idx < len(m.changes) {
			entry := m.changes[idx]
			if !entry.isDir {
				display := entry.display
				tag := statusTagPlain(entry.status)
				if lastConn := strings.LastIndex(display, "── "); lastConn >= 0 {
					prefix := display[:lastConn+len("── ")]
					name := display[lastConn+len("── "):]
					return prefix + tag + " " + name
				}
				return tag + " " + display
			}
		}
		return line
	case panelBranches:
		display := m.branchDisplay(idx)
		if line == m.currentBranch {
			return "● " + display
		}
		return "  " + display
	case panelCommits:
		return line
	}
	return line
}

func (m model) renderLine(panel, idx int, line string, width int) string {
	switch panel {
	case panelChanges:
		if idx >= len(m.changes) {
			return line
		}
		entry := m.changes[idx]
		// Split connector prefix from the name
		name := entry.display
		prefix := ""
		if lastConn := strings.LastIndex(name, "── "); lastConn >= 0 {
			prefix = name[:lastConn+len("── ")]
			name = name[lastConn+len("── "):]
		}
		connPart := treeConnectorStyle.Render(prefix)
		if entry.isDir {
			return connPart + treeDirStyle.Render(name)
		}
		// Status indicator (e.g. M, A, D, ?) with color
		status := entry.status
		tag := statusTag(status)
		switch {
		case strings.Contains(status, "A"), strings.Contains(status, "?"):
			return connPart + tag + " " + statusAddedStyle.Render(name)
		case strings.Contains(status, "D"):
			return connPart + tag + " " + statusDeletedStyle.Render(name)
		default:
			return connPart + tag + " " + statusModifiedStyle.Render(name)
		}

	case panelBranches:
		if idx >= len(m.branches) {
			return line
		}
		b := m.branches[idx]
		suffix := ""
		if b.ahead > 0 {
			suffix += " " + aheadStyle.Render(fmt.Sprintf("↑%d", b.ahead))
		}
		if b.behind > 0 {
			suffix += " " + behindStyle.Render(fmt.Sprintf("↓%d", b.behind))
		}
		if line == m.currentBranch {
			return branchCurrentStyle.Render("● "+line) + suffix
		}
		return "  " + line + suffix

	case panelCommits:
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			return hashStyle.Render(parts[0]) + " " + parts[1]
		}
		return line
	}
	return line
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
