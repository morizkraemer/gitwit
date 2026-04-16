package main

import (
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) reloadChanges() {
	raw := loadChanges()
	m.changesRaw = raw
	m.changes = buildChangeTree(raw)
	m.diffAdded, m.diffRemoved = diffStat()
}

func editorName() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if _, err := exec.LookPath("nvim"); err == nil {
		return "nvim"
	}
	return "vim"
}

func openInEditor(file string) tea.Cmd {
	c := exec.Command(editorName(), file)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
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
			m.reloadChanges()
			m.branches = loadBranches()
			m.remoteBranches = loadRemoteBranches(m.branches)
			m.worktrees = loadWorktrees()
			m.commits = loadCommits(m.selectedBranch())
		}
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.statusMsg = "✗ editor: " + msg.err.Error()
		}
		if m.dirMode {
			m.dirEntries = buildDirTree(m.dirExpanded)
			if m.dirCursor >= len(m.dirEntries) {
				m.dirCursor = max(len(m.dirEntries)-1, 0)
			}
		}
		m.reloadChanges()
		return m, nil

	case mdRenderedMsg:
		if msg.err != nil {
			m.mdMode = false
			m.statusMsg = "✗ " + msg.err.Error()
		} else if msg.file == m.mdFile {
			m.mdLines = msg.lines
		}
		return m, nil

	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd())
		if !m.diffMode && !m.mdMode && !m.inputMode {
			branch := m.selectedBranch()
			cmds = append(cmds, func() tea.Msg {
				cur := currentBranch()
				raw := loadChanges()
				added, removed := diffStat()
				branches := loadBranches()
				return refreshMsg{
					currentBranch:  cur,
					changesRaw:     raw,
					changes:        buildChangeTree(raw),
					diffAdded:      added,
					diffRemoved:    removed,
					branches:       branches,
					remoteBranches: loadRemoteBranches(branches),
					worktrees:      loadWorktrees(),
					commits:        loadCommits(branch),
				}
			})
		}
		if !m.confirmMode {
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
		}
		return m, tea.Batch(cmds...)

	case refreshMsg:
		m.currentBranch = msg.currentBranch
		m.changesRaw = msg.changesRaw
		m.changes = msg.changes
		m.diffAdded = msg.diffAdded
		m.diffRemoved = msg.diffRemoved
		m.branches = msg.branches
		m.remoteBranches = msg.remoteBranches
		m.worktrees = msg.worktrees
		m.commits = msg.commits
		if m.dirMode {
			m.dirEntries = buildDirTree(m.dirExpanded)
			if m.dirCursor >= len(m.dirEntries) {
				m.dirCursor = max(len(m.dirEntries)-1, 0)
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Confirm mode (y/n)
		if m.confirmMode {
			switch msg.String() {
			case "y":
				m.confirmMode = false
				switch m.confirmAction {
				case "discard":
					file := m.confirmFile
					// Check if untracked
					status := ""
					for _, c := range m.changes {
						if c.filePath == file {
							status = c.status
							break
						}
					}
					if strings.HasPrefix(status, "?") {
						// Untracked file — remove it
						os.Remove(file)
					} else {
						// Tracked file — unstage first, then restore
						exec.Command("git", "reset", "HEAD", "--", file).Run()
						exec.Command("git", "checkout", "--", file).Run()
					}
					m.statusMsg = "Discarded " + file
					m.reloadChanges()
					if m.cursors[panelChanges] >= len(m.changes) {
						m.cursors[panelChanges] = max(len(m.changes)-1, 0)
					}
				}
			case "n", "esc":
				m.confirmMode = false
				m.statusMsg = ""
			}
			return m, nil
		}

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
			m.worktrees = loadWorktrees()
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
					m.reloadChanges()
					m.cursors[panelChanges] = 0
					m.offsets[panelChanges] = 0
					m.commits = loadCommits(m.selectedBranch())
					m.branches = loadBranches()
					m.remoteBranches = loadRemoteBranches(m.branches)
			m.worktrees = loadWorktrees()
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
			case "e":
				if m.diffFile != "" {
					m.diffMode = false
					return m, openInEditor(m.diffFile)
				}
				return m, nil
			}
			return m, nil
		}

		// Markdown preview mode keys
		if m.mdMode {
			switch msg.String() {
			case "q", "esc":
				m.mdMode = false
				return m, nil
			case "j", "down":
				if m.mdCursor < len(m.mdLines)-1 {
					m.mdCursor++
				}
				return m, nil
			case "k", "up":
				if m.mdCursor > 0 {
					m.mdCursor--
				}
				return m, nil
			case "d", "ctrl+d":
				jump := (m.height - 4) / 2
				m.mdCursor += jump
				if m.mdCursor >= len(m.mdLines) {
					m.mdCursor = max(len(m.mdLines)-1, 0)
				}
				return m, nil
			case "u", "ctrl+u":
				jump := (m.height - 4) / 2
				m.mdCursor -= jump
				if m.mdCursor < 0 {
					m.mdCursor = 0
				}
				return m, nil
			case "e":
				if m.mdFile != "" {
					m.mdMode = false
					return m, openInEditor(m.mdFile)
				}
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			visible := m.visiblePanels()
			for i, p := range visible {
				if p == m.activePanel {
					m.activePanel = visible[(i+1)%len(visible)]
					break
				}
			}
			m.statusMsg = ""
			return m, nil

		case "shift+tab":
			visible := m.visiblePanels()
			for i, p := range visible {
				if p == m.activePanel {
					m.activePanel = visible[(i+len(visible)-1)%len(visible)]
					break
				}
			}
			m.statusMsg = ""
			return m, nil

		case "enter":
			switch m.activePanel {
			case panelChanges:
				if m.dirMode {
					if len(m.dirEntries) > 0 && m.dirCursor < len(m.dirEntries) {
						entry := m.dirEntries[m.dirCursor]
						if entry.isDir {
							if m.dirExpanded[entry.filePath] {
								delete(m.dirExpanded, entry.filePath)
							} else {
								m.dirExpanded[entry.filePath] = true
							}
							m.dirEntries = buildDirTree(m.dirExpanded)
							if m.dirCursor >= len(m.dirEntries) {
								m.dirCursor = max(len(m.dirEntries)-1, 0)
							}
						} else if strings.HasSuffix(entry.filePath, ".md") {
							m.mdMode = true
							m.mdFile = entry.filePath
							m.mdLines = nil
							m.mdCursor = 0
							m.mdOffset = 0
							width := m.width - 4
							file := entry.filePath
							return m, func() tea.Msg {
								lines, err := renderMarkdown(file, width)
								return mdRenderedMsg{lines: lines, file: file, err: err}
							}
						} else {
							return m, openInEditor(entry.filePath)
						}
					}
					return m, nil
				}
				if len(m.changes) > 0 {
					entry := m.changes[m.cursors[panelChanges]]
					if !entry.isDir {
						if strings.HasSuffix(entry.filePath, ".md") {
							m.mdMode = true
							m.mdFile = entry.filePath
							m.mdLines = nil
							m.mdCursor = 0
							m.mdOffset = 0
							width := m.width - 4
							file := entry.filePath
							return m, func() tea.Msg {
								lines, err := renderMarkdown(file, width)
								return mdRenderedMsg{lines: lines, file: file, err: err}
							}
						}
						m.diffLines = loadDiff(entry.filePath, entry.status)
						m.diffFile = entry.filePath
						m.diffScroll = 0
						m.diffMode = true
					}
				}
				return m, nil

			case panelBranches:
				if m.branchTab == 1 {
					// Worktree: show path in status
					if len(m.worktrees) > 0 && m.worktreeCursor < len(m.worktrees) {
						wt := m.worktrees[m.worktreeCursor]
						m.statusMsg = "Worktree: " + wt.path
					}
					return m, nil
				}
				idx := m.cursors[panelBranches]
				if idx >= len(m.branches) {
					// Remote branch: checkout as local tracking branch
					ri := idx - len(m.branches)
					if ri < len(m.remoteBranches) {
						rb := m.remoteBranches[ri]
						cmd := exec.Command("git", "checkout", "-b", rb.branch, "--track", rb.name)
						out, err := cmd.CombinedOutput()
						if err != nil {
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
						m.worktrees = loadWorktrees()
						for i, b := range m.branches {
							if b.name == rb.branch {
								m.cursors[panelBranches] = i
								break
							}
						}
						m.reloadChanges()
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
				m.reloadChanges()
				m.branches = loadBranches()
				m.remoteBranches = loadRemoteBranches(m.branches)
				m.worktrees = loadWorktrees()
				m.commits = loadCommits(m.selectedBranch())
				return m, nil

			case panelCommits:
				idx := m.cursors[panelCommits]
				if len(m.commits) > 0 && idx < len(m.commits) {
					if m.expandedCommit == idx {
						m.expandedCommit = -1
						m.commitDetail = nil
					} else {
						hash := strings.SplitN(m.commits[idx], " ", 2)[0]
						m.expandedCommit = idx
						m.commitDetail = loadCommitDetail(hash)
					}
				}
				return m, nil
			}
			return m, nil

		case "j", "down":
			if m.activePanel == panelChanges && m.dirMode {
				if m.dirCursor < len(m.dirEntries)-1 {
					m.dirCursor++
				}
				return m, nil
			}
			if m.activePanel == panelBranches && m.branchTab == 1 {
				if m.worktreeCursor < len(m.worktrees)-1 {
					m.worktreeCursor++
				}
				return m, nil
			}
			if m.activePanel == panelBranches && m.branchTab == 0 {
				total := len(m.branches) + len(m.remoteBranches)
				if m.cursors[panelBranches] < total-1 {
					m.cursors[panelBranches]++
				}
				if m.cursors[panelBranches] < len(m.branches) {
					m.commits = loadCommits(m.selectedBranch())
					m.cursors[panelCommits] = 0
					m.offsets[panelCommits] = 0
				}
				return m, nil
			}
			items := m.panelItems(m.activePanel)
			if m.cursors[m.activePanel] < len(items)-1 {
				m.cursors[m.activePanel]++
			}
			return m, nil

		case "k", "up":
			if m.activePanel == panelChanges && m.dirMode {
				if m.dirCursor > 0 {
					m.dirCursor--
				}
				return m, nil
			}
			if m.activePanel == panelBranches && m.branchTab == 1 {
				if m.worktreeCursor > 0 {
					m.worktreeCursor--
				}
				return m, nil
			}
			if m.activePanel == panelBranches && m.branchTab == 0 {
				if m.cursors[panelBranches] > 0 {
					m.cursors[panelBranches]--
				}
				if m.cursors[panelBranches] < len(m.branches) {
					m.commits = loadCommits(m.selectedBranch())
					m.cursors[panelCommits] = 0
					m.offsets[panelCommits] = 0
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


		case "l", "right":
			if m.activePanel == panelChanges && m.dirMode && len(m.dirEntries) > 0 && m.dirCursor < len(m.dirEntries) {
				entry := m.dirEntries[m.dirCursor]
				if entry.isDir && !m.dirExpanded[entry.filePath] {
					m.dirExpanded[entry.filePath] = true
					m.dirEntries = buildDirTree(m.dirExpanded)
				}
				return m, nil
			}
			return m, nil

		case "h", "left":
			if m.activePanel == panelChanges && m.dirMode && len(m.dirEntries) > 0 && m.dirCursor < len(m.dirEntries) {
				entry := m.dirEntries[m.dirCursor]
				if entry.isDir && m.dirExpanded[entry.filePath] {
					delete(m.dirExpanded, entry.filePath)
					m.dirEntries = buildDirTree(m.dirExpanded)
					if m.dirCursor >= len(m.dirEntries) {
						m.dirCursor = max(len(m.dirEntries)-1, 0)
					}
				}
				return m, nil
			}
			return m, nil

		case "e":
			if m.activePanel == panelChanges && len(m.changes) > 0 {
				entry := m.changes[m.cursors[panelChanges]]
				if !entry.isDir {
					return m, openInEditor(entry.filePath)
				}
			}
			return m, nil

		case "d":
			if m.activePanel == panelCommits && m.expandedCommit >= 0 {
				hash := strings.SplitN(m.commits[m.expandedCommit], " ", 2)[0]
				m.diffLines = git("show", "--patch", hash)
				m.diffFile = hash
				m.diffScroll = 0
				m.diffMode = true
				return m, nil
			}
			if m.activePanel == panelChanges && !m.dirMode && len(m.changes) > 0 {
				entry := m.changes[m.cursors[panelChanges]]
				if !entry.isDir {
					m.confirmMode = true
					m.confirmAction = "discard"
					m.confirmFile = entry.filePath
					m.statusMsg = "Discard changes to " + entry.filePath + "? (y/n)"
				}
			}
			return m, nil

		case "r":
			m.reloadChanges()
			m.branches = loadBranches()
			m.remoteBranches = loadRemoteBranches(m.branches)
			m.worktrees = loadWorktrees()
			m.commits = loadCommits(m.selectedBranch())
			return m, nil

		case "v":
			if m.activePanel == panelChanges {
				m.dirMode = !m.dirMode
				if m.dirMode {
					if m.dirExpanded == nil {
						m.dirExpanded = make(map[string]bool)
					}
					m.dirEntries = buildDirTree(m.dirExpanded)
					m.dirCursor = 0
					m.dirOffset = 0
				}
			} else if m.activePanel == panelBranches {
				m.branchTab = (m.branchTab + 1) % 2
			}
			return m, nil

		case "1", "2", "3":
			panel := int(msg.String()[0] - '1')
			if m.showPanel[panel] {
				m.activePanel = panel
				m.statusMsg = ""
			}
			return m, nil

		case "!", "@", "#":
			panel := map[string]int{"!": 0, "@": 1, "#": 2}[msg.String()]
			if m.showPanel[panel] && m.visibleCount() <= 1 {
				return m, nil // don't hide the last panel
			}
			m.showPanel[panel] = !m.showPanel[panel]
			if !m.showPanel[panel] && m.activePanel == panel {
				m.activePanel = m.visiblePanels()[0]
			}
			return m, nil

		case " ":
			if m.activePanel == panelChanges && !m.dirMode && len(m.changes) > 0 {
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
					m.reloadChanges()
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
			if m.activePanel == panelChanges && !m.dirMode && len(m.changes) > 0 {
				// Stage all files
				exec.Command("git", "add", "-A").Run()
				m.reloadChanges()
				m.statusMsg = "Staged all files"
			}
			return m, nil

		case "c":
			if m.activePanel == panelChanges && !m.dirMode {
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
