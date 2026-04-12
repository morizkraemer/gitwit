package main

import (
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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
			raw := loadChanges()
			m.changesRaw = raw
			m.changes = buildChangeTree(raw)
			m.branches = loadBranches()
			m.remoteBranches = loadRemoteBranches(m.branches)
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
		raw := loadChanges()
		m.changesRaw = raw
		m.changes = buildChangeTree(raw)
		return m, nil

	case tickMsg:
		if !m.diffMode && !m.inputMode {
			m.currentBranch = currentBranch()
			raw := loadChanges()
			m.changesRaw = raw
			m.changes = buildChangeTree(raw)
			m.branches = loadBranches()
			m.remoteBranches = loadRemoteBranches(m.branches)
			m.commits = loadCommits(m.selectedBranch())
			if m.dirMode {
				m.dirEntries = buildDirTree(m.dirExpanded)
				if m.dirCursor >= len(m.dirEntries) {
					m.dirCursor = max(len(m.dirEntries)-1, 0)
				}
			}
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
			case "e":
				if m.diffFile != "" {
					m.diffMode = false
					return m, openInEditor(m.diffFile)
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
						} else {
							return m, openInEditor(entry.filePath)
						}
					}
					return m, nil
				}
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
			if m.activePanel == panelChanges && m.dirMode {
				if m.dirCursor < len(m.dirEntries)-1 {
					m.dirCursor++
				}
				return m, nil
			}
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
			if m.activePanel == panelChanges && m.dirMode {
				if m.dirCursor > 0 {
					m.dirCursor--
				}
				return m, nil
			}
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

		case "e":
			if m.activePanel == panelChanges && len(m.changes) > 0 {
				entry := m.changes[m.cursors[panelChanges]]
				if !entry.isDir {
					return m, openInEditor(entry.filePath)
				}
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
			if m.activePanel == panelChanges && !m.dirMode && len(m.changes) > 0 {
				// Stage all files
				exec.Command("git", "add", "-A").Run()
				raw := loadChanges()
				m.changesRaw = raw
				m.changes = buildChangeTree(raw)
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
