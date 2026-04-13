package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

// fitWidth truncates s to width display cells (ANSI-aware) and pads with spaces.
func fitWidth(s string, width int) string {
	s = ansi.Truncate(s, width, "")
	pad := width - lipgloss.Width(s)
	if pad > 0 {
		s += strings.Repeat(" ", pad)
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

func (m model) renderDiffView() string {
	contentWidth := m.width - 2
	viewHeight := m.height - 4 // outer border + title + help

	title := titleStyle.Render(fmt.Sprintf(" %s ", m.diffFile))
	help := dimStyle.Render("  q/esc: close · j/k: scroll · d/u: page down/up · e: open in editor")

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

func (m *model) renderMdView() string {
	contentWidth := m.width - 2
	viewHeight := m.height - 4

	title := titleStyle.Render(fmt.Sprintf(" %s ", m.mdFile))
	help := dimStyle.Render("  q/esc: close · j/k: navigate · d/u: page down/up · e: open in editor")

	if m.mdLines == nil {
		loading := dimStyle.Render("  Loading...")
		var pad []string
		for i := 0; i < viewHeight-1; i++ {
			pad = append(pad, "")
		}
		content := strings.Join(append([]string{loading}, pad...), "\n")
		inner := lipgloss.JoinVertical(lipgloss.Left, title, content, help)
		return outerBorderStyle.Width(contentWidth).Height(m.height - 2).Render(inner)
	}

	// Viewport follows cursor
	if m.mdCursor < m.mdOffset {
		m.mdOffset = m.mdCursor
	}
	if m.mdCursor >= m.mdOffset+viewHeight {
		m.mdOffset = m.mdCursor - viewHeight + 1
	}

	mdMarker := lipgloss.NewStyle().Background(lipgloss.Color("#3d3d5c")).Render(" ")

	var lines []string
	end := m.mdOffset + viewHeight
	if end > len(m.mdLines) {
		end = len(m.mdLines)
	}
	for i := m.mdOffset; i < end; i++ {
		line := m.mdLines[i]
		if i == m.mdCursor {
			padWidth := contentWidth - 2 - lipgloss.Width(line) - 2 // -2 for markers
			if padWidth < 0 {
				padWidth = 0
			}
			lines = append(lines, mdMarker+line+strings.Repeat(" ", padWidth)+mdMarker)
		} else {
			lines = append(lines, " "+line)
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

func (m model) renderBottomBar(width int) string {
	bg := lipgloss.Color("#2d2d4a")
	bar := lipgloss.NewStyle().Background(bg).Width(width)
	key := func(k string) string {
		return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#c0c0d0")).Bold(true).Render(k)
	}
	label := func(l string) string {
		return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#787890")).Render(l)
	}
	sep := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#555566")).Render(" │ ")
	dot := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#88cc88")).Render("●")

	if m.inputMode {
		prompt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#787890")).Render(m.inputPrompt)
		value := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#d4d4d4")).Render(m.inputValue + "█")
		return bar.Render(" " + prompt + value)
	}

	if m.statusMsg != "" {
		status := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#d4d4d4")).Render(m.statusMsg)
		return bar.Render(" " + status)
	}

	panelNames := [3]string{"changes", "branches", "commits"}
	shiftKeys := [3]string{"⇧1", "⇧2", "⇧3"}

	var parts []string
	parts = append(parts, key("1-3")+" "+label("select"))

	for i := 0; i < 3; i++ {
		p := key(shiftKeys[i]) + " " + label(panelNames[i])
		if m.showPanel[i] {
			p += " " + dot
		}
		parts = append(parts, p)
	}

	parts = append(parts, key("q")+" "+label("quit"))

	return bar.Render(" " + strings.Join(parts, sep))
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.diffMode {
		return m.renderDiffView()
	}

	if m.mdMode {
		return m.renderMdView()
	}

	// Outer border eats 2 cols and 2 rows
	contentWidth := m.width - 2
	innerWidth := contentWidth - 4 // panel borders

	// Chrome: per panel = title(1) + border(2) + help(1) = 4
	// Separators between panels: vc - 1
	// Bottom bar: 1, outer border: 2
	vc := m.visibleCount()
	chrome := vc*4 + (vc - 1) + 1 + 2
	available := m.height - chrome
	if available < 3 {
		available = 3
	}

	// Distribute height: first visible panel gets 50%, rest split evenly
	visible := m.visiblePanels()
	panelHeight := [3]int{}
	if len(visible) == 1 {
		panelHeight[visible[0]] = available
	} else {
		firstH := available / 2
		rest := available - firstH
		each := rest / (len(visible) - 1)
		panelHeight[visible[0]] = firstH
		for i := 1; i < len(visible); i++ {
			panelHeight[visible[i]] = each
		}
		// give remainder to last panel
		used := firstH + each*(len(visible)-1)
		panelHeight[visible[len(visible)-1]] += available - used
	}
	for i := range panelHeight {
		if m.showPanel[i] && panelHeight[i] < 2 {
			panelHeight[i] = 2
		}
	}

	borderFn := func(panel int, h int) lipgloss.Style {
		if panel == m.activePanel {
			return activeBorderStyle.Width(innerWidth).Height(h)
		}
		return inactiveBorderStyle.Width(innerWidth).Height(h)
	}

	// Build layout sections
	var sections []string
	first := true
	for _, p := range visible {
		if !first {
			sections = append(sections, "")
		}
		first = false

		h := panelHeight[p]
		switch p {
		case panelChanges:
			stagedCount := 0
			for _, line := range m.changesRaw {
				if len(line) >= 2 && strings.TrimSpace(line[:1]) != "" && line[:1] != "?" {
					stagedCount++
				}
			}
			var title string
			if m.dirMode {
				title = titleStyle.Render(fmt.Sprintf(" Files (%d) ", len(m.dirEntries)))
			} else {
				title = titleStyle.Render(fmt.Sprintf(" Changes (%d) · Staged (%d) ", len(m.changesRaw), stagedCount))
			}
			var help string
			if m.dirMode {
				help = dimStyle.Render("  enter: open/toggle · v: git changes")
			} else {
				help = dimStyle.Render("  space: stage/unstage · a: stage all · c: commit · enter: diff · e: edit · v: files")
			}
			view := m.renderPanel(panelChanges, innerWidth, h)
			sections = append(sections, title, borderFn(p, h).Render(view), help)

		case panelBranches:
			var title string
			if m.showRemote {
				title = titleStyle.Render(fmt.Sprintf(" Local (%d) │ Remote (%d) ", len(m.branches), len(m.remoteBranches)))
			} else {
				title = titleStyle.Render(fmt.Sprintf(" Branches (%d) ", len(m.branches)))
			}
			help := dimStyle.Render("  h/l: local/remote · enter: checkout · B: new · R: remote · f: fetch · p: pull · P: push")
			view := m.renderBranchesPanel(innerWidth, h)
			sections = append(sections, title, borderFn(p, h).Render(view), help)

		case panelCommits:
			commitLabel := m.selectedBranch()
			if commitLabel == "" {
				commitLabel = "none"
			}
			title := titleStyle.Render(fmt.Sprintf(" Commits · %s ", commitLabel))
			help := dimStyle.Render("  j/k: navigate")
			view := m.renderPanel(panelCommits, innerWidth, h)
			sections = append(sections, title, borderFn(p, h).Render(view), help)
		}
	}

	sections = append(sections, m.renderBottomBar(contentWidth))

	inner := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return outerBorderStyle.
		Width(contentWidth).
		Height(m.height - 2).
		Render(inner)
}

func (m *model) renderBranchesPanel(width, height int) string {
	isActive := m.activePanel == panelBranches
	var leftWidth, rightWidth int
	if m.showRemote {
		leftWidth = width / 2
		rightWidth = width - leftWidth - 1 // -1 for separator
		if rightWidth < 1 {
			rightWidth = 1
		}
	} else {
		leftWidth = width
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
			if b.mainAhead > 0 || b.mainBehind > 0 {
				suffix += " " + aheadStyle.Render(fmt.Sprintf("⇡%d", b.mainAhead)) +
					dimStyle.Render("⇣") + behindStyle.Render(fmt.Sprintf("%d", b.mainBehind))
			}
			var content string
			if b.name == m.currentBranch {
				content = branchCurrentStyle.Render("● "+b.name) + suffix
			} else {
				content = "  " + b.name + suffix
			}
			leftLines = append(leftLines, fitWidth(content, leftWidth))
		}
	}
	for len(leftLines) < height {
		leftLines = append(leftLines, strings.Repeat(" ", leftWidth))
	}

	if !m.showRemote {
		return strings.Join(leftLines, "\n")
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
			rightLines = append(rightLines, fitWidth(styled, rightWidth))
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
	offset := &m.offsets[panel]
	if panel == panelChanges && m.dirMode {
		cursor = m.dirCursor
		offset = &m.dirOffset
	}

	// Scroll offset
	if cursor < *offset {
		*offset = cursor
	}
	if cursor >= *offset+height {
		*offset = cursor - height + 1
	}

	var lines []string
	for i := *offset; i < len(items) && i < *offset+height; i++ {
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
	if b.mainAhead > 0 || b.mainBehind > 0 {
		s += fmt.Sprintf(" ⇡%d⇣%d", b.mainAhead, b.mainBehind)
	}
	return s
}

func (m model) plainLine(panel, idx int, line string) string {
	switch panel {
	case panelChanges:
		if m.dirMode {
			if idx < len(m.dirEntries) {
				return m.dirEntries[idx].display
			}
			return line
		}
		if idx < len(m.changes) {
			entry := m.changes[idx]
			if !entry.isDir {
				display := entry.display
				tag := statusTagPlain(entry.status)
				if lastConn := strings.LastIndex(display, "─ "); lastConn >= 0 {
					prefix := display[:lastConn+len("─ ")]
					name := display[lastConn+len("─ "):]
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
		if m.dirMode {
			if idx >= len(m.dirEntries) {
				return line
			}
			entry := m.dirEntries[idx]
			name := entry.display
			prefix := ""
			if lastConn := strings.LastIndex(name, "─ "); lastConn >= 0 {
				prefix = name[:lastConn+len("─ ")]
				name = name[lastConn+len("─ "):]
			}
			connPart := treeConnectorStyle.Render(prefix)
			if entry.isDir {
				return connPart + treeDirStyle.Render(name)
			}
			return connPart + name
		}
		if idx >= len(m.changes) {
			return line
		}
		entry := m.changes[idx]
		// Split connector prefix from the name
		name := entry.display
		prefix := ""
		if lastConn := strings.LastIndex(name, "─ "); lastConn >= 0 {
			prefix = name[:lastConn+len("─ ")]
			name = name[lastConn+len("─ "):]
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
