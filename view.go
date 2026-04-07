package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

	outerBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#5a5a7a"))
)

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
