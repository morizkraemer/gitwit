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

	helpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606080"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#d4d4d4")).
			Background(lipgloss.Color("#3b3b5c")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#606080")).
				Padding(0, 1)
)

func renderTabBar(tabs []string, active int, info string) string {
	var parts []string
	for i, tab := range tabs {
		if i == active {
			parts = append(parts, activeTabStyle.Render(tab))
		} else {
			parts = append(parts, inactiveTabStyle.Render(tab))
		}
	}
	bar := strings.Join(parts, "")
	if info != "" {
		bar += " " + dimStyle.Render(info)
	}
	return bar
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
	contentWidth := m.width - 2 // border
	viewHeight := m.height - 4 // title + border + help inside

	title := titleStyle.Render(fmt.Sprintf(" %s ", m.diffFile))
	help := helpBarStyle.Render(" q/esc close · j/k scroll · d/u page · e edit")

	var lines []string
	end := m.diffScroll + viewHeight
	if end > len(m.diffLines) {
		end = len(m.diffLines)
	}
	for i := m.diffScroll; i < end; i++ {
		line := m.diffLines[i]
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
	inner := content + "\n" + fitWidth(help, contentWidth)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		activeBorderStyle.Width(contentWidth).Height(viewHeight+1).Render(inner),
	)
}

func (m *model) renderMdView() string {
	contentWidth := m.width - 2 // border
	viewHeight := m.height - 4 // title + border + help inside

	title := titleStyle.Render(fmt.Sprintf(" %s ", m.mdFile))
	help := helpBarStyle.Render(" q/esc close · j/k navigate · d/u page · e edit")

	if m.mdLines == nil {
		loading := dimStyle.Render("  Loading...")
		var pad []string
		for i := 0; i < viewHeight; i++ {
			pad = append(pad, "")
		}
		content := strings.Join(append([]string{loading}, pad...), "\n")
		inner := content + "\n" + fitWidth(help, contentWidth)
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			activeBorderStyle.Width(contentWidth).Height(viewHeight+1).Render(inner),
		)
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
			padWidth := contentWidth - 2 - lipgloss.Width(line) - 2
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
	inner := content + "\n" + fitWidth(help, contentWidth)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		activeBorderStyle.Width(contentWidth).Height(viewHeight+1).Render(inner),
	)
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

	contentWidth := m.width
	innerWidth := contentWidth - 2 // panel borders

	// Chrome: per panel = title(1) + border(2) + helpInside(1) = 4
	// Bottom bar: 1
	vc := m.visibleCount()
	chrome := vc*4 + 1
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
	for _, p := range visible {
		h := panelHeight[p]
		// Content gets h-1 lines, last line is help
		contentH := h - 1
		if contentH < 1 {
			contentH = 1
		}

		switch p {
		case panelChanges:
			stagedCount := 0
			for _, line := range m.changesRaw {
				if len(line) >= 2 && strings.TrimSpace(line[:1]) != "" && line[:1] != "?" {
					stagedCount++
				}
			}
			activeTab := 0
			if m.dirMode {
				activeTab = 1
			}
			tabs := renderTabBar(
				[]string{
					fmt.Sprintf("Changes (%d/%d)", stagedCount, len(m.changesRaw)),
					fmt.Sprintf("Files (%d)", len(m.dirEntries)),
				},
				activeTab,
				"v switch",
			)
			var help string
			if m.dirMode {
				help = helpBarStyle.Render(" ⏎ open/toggle")
			} else {
				help = helpBarStyle.Render(" spc stage · a all · c commit · ⏎ diff · e edit")
			}
			view := m.renderPanel(panelChanges, innerWidth, contentH)
			content := view + "\n" + fitWidth(help, innerWidth)
			sections = append(sections, tabs, borderFn(p, h).Render(content))

		case panelBranches:
			var info string
			switch m.branchTab {
			case 0:
				info = fmt.Sprintf("%d branches", len(m.branches))
			case 1:
				info = fmt.Sprintf("%d remotes", len(m.remoteBranches))
			case 2:
				info = fmt.Sprintf("%d worktrees", len(m.worktrees))
			}
			tabs := renderTabBar(
				[]string{"Local", "Remote", "Worktrees"},
				m.branchTab,
				info+" · v switch",
			)
			var help string
			var view string
			switch m.branchTab {
			case 0:
				help = helpBarStyle.Render(" ⏎ checkout · B new · f fetch · p pull · P push")
				view = m.renderLocalBranches(innerWidth, contentH)
			case 1:
				help = helpBarStyle.Render(" ⏎ checkout · f fetch · p pull · P push")
				view = m.renderRemoteBranches(innerWidth, contentH)
			case 2:
				help = helpBarStyle.Render(" j/k navigate")
				view = m.renderWorktrees(innerWidth, contentH)
			}
			content := view + "\n" + fitWidth(help, innerWidth)
			sections = append(sections, tabs, borderFn(p, h).Render(content))

		case panelCommits:
			commitLabel := m.selectedBranch()
			if commitLabel == "" {
				commitLabel = "none"
			}
			tabs := renderTabBar([]string{"Commits"}, 0, commitLabel)
			help := helpBarStyle.Render(" j/k navigate")
			view := m.renderPanel(panelCommits, innerWidth, contentH)
			content := view + "\n" + fitWidth(help, innerWidth)
			sections = append(sections, tabs, borderFn(p, h).Render(content))
		}
	}

	sections = append(sections, m.renderBottomBar(contentWidth))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *model) renderLocalBranches(width, height int) string {
	isActive := m.activePanel == panelBranches && m.branchTab == 0

	var lines []string
	cursor := m.cursors[panelBranches]
	if cursor < m.offsets[panelBranches] {
		m.offsets[panelBranches] = cursor
	}
	if cursor >= m.offsets[panelBranches]+height {
		m.offsets[panelBranches] = cursor - height + 1
	}
	for i := m.offsets[panelBranches]; i < len(m.branches) && i < m.offsets[panelBranches]+height; i++ {
		b := m.branches[i]
		isSelected := i == cursor && isActive
		isCursor := i == cursor && !isSelected
		display := m.branchDisplay(i)
		prefix := "  "
		if b.name == m.currentBranch {
			prefix = "● "
		}
		plain := truncate(prefix+display, width)
		if isSelected {
			lines = append(lines, selectedStyle.Width(width).Render(plain))
		} else if isCursor {
			lines = append(lines, cursorStyle.Width(width).Render(plain))
		} else {
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
			lines = append(lines, fitWidth(content, width))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderRemoteBranches(width, height int) string {
	isActive := m.activePanel == panelBranches && m.branchTab == 1

	var lines []string
	if m.remoteCursor < m.remoteOffset {
		m.remoteOffset = m.remoteCursor
	}
	if m.remoteCursor >= m.remoteOffset+height {
		m.remoteOffset = m.remoteCursor - height + 1
	}
	for i := m.remoteOffset; i < len(m.remoteBranches) && i < m.remoteOffset+height; i++ {
		rb := m.remoteBranches[i]
		isSelected := i == m.remoteCursor && isActive
		plain := truncate("  "+rb.name, width)
		if isSelected {
			lines = append(lines, selectedStyle.Width(width).Render(plain))
		} else {
			styled := "  " + dimStyle.Render(rb.remote+"/") + rb.branch
			lines = append(lines, fitWidth(styled, width))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderWorktrees(width, height int) string {
	isActive := m.activePanel == panelBranches && m.branchTab == 2

	var lines []string
	if m.worktreeCursor < m.worktreeOffset {
		m.worktreeOffset = m.worktreeCursor
	}
	if m.worktreeCursor >= m.worktreeOffset+height {
		m.worktreeOffset = m.worktreeCursor - height + 1
	}
	for i := m.worktreeOffset; i < len(m.worktrees) && i < m.worktreeOffset+height; i++ {
		wt := m.worktrees[i]
		isSelected := i == m.worktreeCursor && isActive
		label := wt.branch
		if label == "" {
			label = wt.head
		}
		if wt.bare {
			label = "(bare)"
		}
		detail := dimStyle.Render(" " + wt.path)
		prefix := "  "
		if wt.branch == m.currentBranch {
			prefix = "● "
		}
		plain := truncate(prefix+label+" "+wt.path, width)
		if isSelected {
			lines = append(lines, selectedStyle.Width(width).Render(plain))
		} else {
			var content string
			if wt.branch == m.currentBranch {
				content = branchCurrentStyle.Render("● "+label) + detail
			} else {
				content = "  " + label + detail
			}
			lines = append(lines, fitWidth(content, width))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
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
