package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

	var entries []changeEntry
	flattenTree(root, "", "", &entries)
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

type model struct {
	changes       []changeEntry
	changesRaw    []string // raw porcelain lines for count
	branches      []string
	commits       []string
	currentBranch string

	activePanel int
	cursors     [3]int
	offsets     [3]int

	// Diff preview overlay
	diffMode   bool
	diffLines  []string
	diffFile   string
	diffScroll int

	// Text input mode (e.g. new branch name)
	inputMode   bool
	inputPrompt string
	inputValue  string

	// Status message (shown briefly)
	statusMsg string

	width  int
	height int
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#c8c8c8")).
			Background(lipgloss.Color("#3b3b5c")).
			Padding(0, 1)

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7c7caa"))

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#4a4a4a"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0e0e0")).
			Background(lipgloss.Color("#3b4d6b"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a8aaa"))

	branchCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aab7a"))

	statusAddedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aab7a"))

	statusModifiedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c8a56e"))

	statusDeletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#b06060"))

	hashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c8a56e"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060"))

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aab7a"))

	diffDelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#b06060"))

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7c7caa"))

	diffHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#c8c8c8"))
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

func loadBranches() []string {
	raw := git("branch", "--format=%(refname:short)")
	return raw
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
		if b == cur {
			cursorIdx = i
			break
		}
	}
	raw := loadChanges()
	m := model{
		changesRaw:    raw,
		changes:       buildChangeTree(raw),
		branches:      branches,
		currentBranch: cur,
		activePanel:   panelChanges,
	}
	m.cursors[panelBranches] = cursorIdx
	m.commits = loadCommits(m.selectedBranch())
	return m
}

func (m model) selectedBranch() string {
	if len(m.branches) == 0 {
		return ""
	}
	return m.branches[m.cursors[panelBranches]]
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
		return m.branches
	case panelCommits:
		return m.commits
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.ClearScreen

	case tea.KeyMsg:
		// Text input mode
		if m.inputMode {
			switch msg.String() {
			case "esc":
				m.inputMode = false
				m.inputValue = ""
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.inputValue)
				m.inputMode = false
				m.inputValue = ""
				if name == "" {
					return m, nil
				}
				cmd := exec.Command("git", "checkout", "-b", name)
				out, err := cmd.CombinedOutput()
				if err != nil {
					m.statusMsg = "✗ " + strings.TrimSpace(string(out))
					return m, nil
				}
				m.currentBranch = name
				m.statusMsg = "Created & switched to " + name
				m.branches = loadBranches()
				for i, b := range m.branches {
					if b == name {
						m.cursors[panelBranches] = i
						break
					}
				}
				m.commits = loadCommits(name)
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
				m.commits = loadCommits(m.selectedBranch())
				return m, nil
			}
			return m, nil

		case "j", "down":
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
			if m.cursors[m.activePanel] > 0 {
				m.cursors[m.activePanel]--
			}
			if m.activePanel == panelBranches {
				m.commits = loadCommits(m.selectedBranch())
				m.cursors[panelCommits] = 0
				m.offsets[panelCommits] = 0
			}
			return m, nil

		case "r":
			raw := loadChanges()
			m.changesRaw = raw
			m.changes = buildChangeTree(raw)
			m.branches = loadBranches()
			m.commits = loadCommits(m.selectedBranch())
			return m, nil

		case "B":
			if m.activePanel == panelBranches {
				m.inputMode = true
				m.inputPrompt = "New branch name: "
				m.inputValue = ""
				m.statusMsg = ""
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

	// 3 titles (1 line each) + 3 panel borders (2 lines each) + 1 help line + 2 outer border = 12 lines of chrome
	available := m.height - 12
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
	branchesView := m.renderPanel(panelBranches, innerWidth, branchesHeight)
	commitsView := m.renderPanel(panelCommits, innerWidth, commitsHeight)

	// Titles
	changesTitle := titleStyle.Render(fmt.Sprintf(" Changes (%d) ", len(m.changesRaw)))
	branchesTitle := titleStyle.Render(fmt.Sprintf(" Branches (%d) ", len(m.branches)))

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

	var helpText string
	if m.inputMode {
		helpText = "  " + m.inputPrompt + m.inputValue + "█"
	} else if m.statusMsg != "" {
		helpText = "  " + m.statusMsg
	} else {
		helpText = "  tab: switch panel · j/k: navigate · enter: select · B: new branch · r: refresh · q: quit"
	}
	help := dimStyle.Render(helpText)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		changesTitle,
		borderFn(panelChanges, changesHeight).Render(changesView),
		branchesTitle,
		borderFn(panelBranches, branchesHeight).Render(branchesView),
		commitsTitle,
		borderFn(panelCommits, commitsHeight).Render(commitsView),
		help,
	)

	return outerBorderStyle.
		Width(contentWidth).
		Height(m.height - 2).
		Render(inner)
}

func truncate(s string, max int) string {
	w := 0
	for i, r := range s {
		rw := 1
		if r > 127 {
			rw = 2
		}
		if w+rw > max {
			return s[:i]
		}
		w += rw
	}
	return s
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

func (m model) plainLine(panel, idx int, line string) string {
	switch panel {
	case panelBranches:
		if line == m.currentBranch {
			return "● " + line
		}
		return "  " + line
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
		if entry.isDir {
			return dimStyle.Render(line)
		}
		status := entry.status
		switch {
		case strings.Contains(status, "A"), strings.Contains(status, "?"):
			return statusAddedStyle.Render(line)
		case strings.Contains(status, "D"):
			return statusDeletedStyle.Render(line)
		default:
			return statusModifiedStyle.Render(line)
		}

	case panelBranches:
		if line == m.currentBranch {
			return branchCurrentStyle.Render("● " + line)
		}
		return "  " + line

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
