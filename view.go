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

	dimSelectedColor = lipgloss.Color("#9090a0")

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

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9090b0"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#d4d4d4")).
			Background(lipgloss.Color("#3b3b5c")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#606080")).
				Padding(0, 1)
)

func renderTabBar(tabs []string, active int, info string, width int, globalHints string) string {
	var parts []string
	for i, tab := range tabs {
		if i == active {
			parts = append(parts, activeTabStyle.Render(tab))
		} else {
			parts = append(parts, inactiveTabStyle.Render(tab))
		}
	}
	left := strings.Join(parts, "")
	if info != "" {
		left += " " + info
	}
	right := globalHints
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// helpBar renders key/action pairs with brighter keys.
// Each pair is "key action", separated by " · ".
func helpBar(pairs ...string) string {
	var parts []string
	for _, p := range pairs {
		// Split on first space: key + action
		if idx := strings.Index(p, " "); idx >= 0 {
			key := p[:idx]
			action := p[idx:]
			parts = append(parts, helpKeyStyle.Render(key)+helpBarStyle.Render(action))
		} else {
			parts = append(parts, helpKeyStyle.Render(p))
		}
	}
	return " " + strings.Join(parts, helpBarStyle.Render(" · "))
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

func statusTag(status string, withBg func(lipgloss.Style) lipgloss.Style) string {
	if len(status) < 2 {
		return withBg(dimStyle).Render("?")
	}
	x, y := status[0], status[1]
	// Staged status (first char)
	if x == 'M' || x == 'A' || x == 'D' || x == 'R' || x == 'C' {
		return withBg(branchCurrentStyle).Render(string(x))
	}
	// Unstaged status (second char)
	if y == 'M' {
		return withBg(statusModifiedStyle).Render("M")
	}
	if y == 'D' {
		return withBg(statusDeletedStyle).Render("D")
	}
	if x == '?' {
		return withBg(statusAddedStyle).Render("?")
	}
	return withBg(dimStyle).Render(string(y))
}

func (m model) renderDiffView() string {
	contentWidth := m.width - 2 // border
	viewHeight := m.height - 4 // title + border + help inside

	title := titleStyle.Render(fmt.Sprintf(" %s ", m.diffFile))
	help := helpBar("q/esc close", "j/k scroll", "d/u page", "e edit")

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
	help := helpBar("q/esc close", "j/k navigate", "d/u page", "e edit")

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

	if m.inputMode {
		prompt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#787890")).Render(m.inputPrompt)
		value := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#d4d4d4")).Render(m.inputValue + "█")
		return bar.Render(" " + prompt + value)
	}

	if m.statusMsg != "" {
		status := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#d4d4d4")).Render(m.statusMsg)
		return bar.Render(" " + status)
	}

	quit := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("#787890")).Render("q quit")
	return bar.Render(" " + quit)
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

	// Chrome: per panel = title(1) + border(2) = 3; help is inside border height
	// Bottom bar: always 1 (reserved for status/input)
	vc := m.visibleCount()
	chrome := vc*3 + 1
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
	panelKeys := [3]string{
		helpBar("1 select", "! toggle"),
		helpBar("2 select", "@ toggle"),
		helpBar("3 select", "# toggle"),
	}
	var sections []string
	for _, p := range visible {
		hints := panelKeys[p]
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
			diffInfo := dimStyle.Render("v switch")
			if m.diffAdded > 0 || m.diffRemoved > 0 {
				diffInfo = aheadStyle.Render(fmt.Sprintf("+%d", m.diffAdded)) +
					dimStyle.Render("/") +
					behindStyle.Render(fmt.Sprintf("-%d", m.diffRemoved)) +
					dimStyle.Render(" · v switch")
			}
			tabs := renderTabBar(
				[]string{
					fmt.Sprintf("Changes (%d/%d)", stagedCount, len(m.changesRaw)),
					fmt.Sprintf("Files (%d)", len(m.dirEntries)),
				},
				activeTab,
				diffInfo,
				contentWidth,
				hints,
			)
			var help string
			if m.dirMode {
				help = helpBar("return open/toggle")
			} else {
				help = helpBar("space stage", "a all", "c commit", "d discard", "return diff", "e edit")
			}
			view := m.renderPanel(panelChanges, innerWidth, contentH)
			content := view + "\n" + fitWidth(help, innerWidth)
			sections = append(sections, tabs, borderFn(p, h).Render(content))

		case panelBranches:
			total := len(m.branches) + len(m.remoteBranches)
			tabs := renderTabBar(
				[]string{
					fmt.Sprintf("Branches (%d)", total),
					fmt.Sprintf("Worktrees (%d)", len(m.worktrees)),
				},
				m.branchTab,
				dimStyle.Render("v switch"),
				contentWidth,
				hints,
			)
			var help string
			var view string
			switch m.branchTab {
			case 0:
				help = helpBar("return checkout", "B new", "f fetch", "p pull", "P push")
				view = m.renderBranches(innerWidth, contentH)
			case 1:
				help = helpBar("return show path")
				view = m.renderWorktrees(innerWidth, contentH)
			}
			content := view + "\n" + fitWidth(help, innerWidth)
			sections = append(sections, tabs, borderFn(p, h).Render(content))

		case panelCommits:
			commitLabel := m.selectedBranch()
			if commitLabel == "" {
				commitLabel = "none"
			}
			tabs := renderTabBar([]string{"Commits"}, 0, dimStyle.Render(commitLabel), contentWidth, hints)
			help := ""
			view := m.renderPanel(panelCommits, innerWidth, contentH)
			content := view + "\n" + fitWidth(help, innerWidth)
			sections = append(sections, tabs, borderFn(p, h).Render(content))
		}
	}

	if bar := m.renderBottomBar(contentWidth); bar != "" {
		sections = append(sections, bar)
	}

	result := lipgloss.JoinVertical(lipgloss.Left, sections...)
	// Pad to fill terminal height
	lines := strings.Split(result, "\n")
	for len(lines) < m.height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderBranches(width, height int) string {
	isActive := m.activePanel == panelBranches && m.branchTab == 0
	total := len(m.branches) + len(m.remoteBranches)

	cursor := m.cursors[panelBranches]
	if cursor < m.offsets[panelBranches] {
		m.offsets[panelBranches] = cursor
	}
	if cursor >= m.offsets[panelBranches]+height {
		m.offsets[panelBranches] = cursor - height + 1
	}


	// Measure column widths for local branches (display width, not byte count)
	maxName := 0
	maxUpstream := 0
	maxSync := 0
	maxMainAhead := 0
	maxMainBehind := 0
	for _, b := range m.branches {
		nameLen := lipgloss.Width(b.name) + 2 // +2 for prefix ("● " or "  ")
		if nameLen > maxName {
			maxName = nameLen
		}
		if lipgloss.Width(b.upstream) > maxUpstream {
			maxUpstream = lipgloss.Width(b.upstream)
		}
		sync := ""
		if b.ahead > 0 {
			sync += fmt.Sprintf("↑%d", b.ahead)
		}
		if b.behind > 0 {
			sync += fmt.Sprintf("↓%d", b.behind)
		}
		if lipgloss.Width(sync) > maxSync {
			maxSync = lipgloss.Width(sync)
		}
		if b.mainAhead > 0 || b.mainBehind > 0 {
			aStr := fmt.Sprintf("+%d", b.mainAhead)
			bStr := fmt.Sprintf("-%d", b.mainBehind)
			if len(aStr) > maxMainAhead {
				maxMainAhead = len(aStr)
			}
			if len(bStr) > maxMainBehind {
				maxMainBehind = len(bStr)
			}
		}
	}
	// Remote branch names also contribute to maxName
	for _, rb := range m.remoteBranches {
		nameLen := lipgloss.Width(rb.name) + 2
		if nameLen > maxName {
			maxName = nameLen
		}
	}

	var lines []string
	for i := m.offsets[panelBranches]; i < total && i < m.offsets[panelBranches]+height; i++ {
		isSelected := i == cursor && isActive
		isCursor := i == cursor && !isSelected

		// Determine styles based on selection state
		var bg lipgloss.TerminalColor
		hasBg := isSelected || isCursor
		if isSelected {
			bg = selectedStyle.GetBackground()
		} else if isCursor {
			bg = cursorStyle.GetBackground()
		}
		withBg := func(s lipgloss.Style) lipgloss.Style {
			if hasBg {
				return s.Background(bg)
			}
			return s
		}
		defaultFg := lipgloss.NewStyle()
		mutedFg := dimStyle
		if isSelected {
			defaultFg = defaultFg.Foreground(selectedStyle.GetForeground())
			mutedFg = lipgloss.NewStyle().Foreground(dimSelectedColor)
		} else if isCursor {
			defaultFg = defaultFg.Foreground(cursorStyle.GetForeground())
			mutedFg = lipgloss.NewStyle().Foreground(dimSelectedColor)
		}

		if i < len(m.branches) {
			// Local branch
			b := m.branches[i]
			prefix := "  "
			if b.name == m.currentBranch {
				prefix = "● "
			}

			namePad := maxName - lipgloss.Width(prefix) - lipgloss.Width(b.name)
			if namePad < 0 {
				namePad = 0
			}

			upstreamPad := ""
			if maxUpstream > 0 {
				pad := maxUpstream - lipgloss.Width(b.upstream)
				if pad < 0 {
					pad = 0
				}
				upstreamPad = strings.Repeat(" ", pad)
			}

			plainSync := ""
			if b.ahead > 0 {
				plainSync += fmt.Sprintf("↑%d", b.ahead)
			}
			if b.behind > 0 {
				plainSync += fmt.Sprintf("↓%d", b.behind)
			}
			syncPad := maxSync - lipgloss.Width(plainSync)
			if syncPad < 0 {
				syncPad = 0
			}

			// Name column
			var nameCol string
			if b.name == m.currentBranch {
				nameCol = withBg(branchCurrentStyle).Render(prefix) + withBg(defaultFg).Render(b.name) + withBg(defaultFg).Render(strings.Repeat(" ", namePad))
			} else {
				nameCol = withBg(defaultFg).Render(prefix+b.name) + withBg(defaultFg).Render(strings.Repeat(" ", namePad))
			}

			// Upstream column
			upstreamCol := ""
			if maxUpstream > 0 {
				upstreamCol = withBg(defaultFg).Render(" ") + withBg(mutedFg).Render(b.upstream) + withBg(defaultFg).Render(upstreamPad)
			}

			// Sync column
			syncCol := ""
			if maxSync > 0 {
				s := ""
				if b.ahead > 0 {
					s += withBg(aheadStyle).Render(fmt.Sprintf("↑%d", b.ahead))
				}
				if b.behind > 0 {
					s += withBg(behindStyle).Render(fmt.Sprintf("↓%d", b.behind))
				}
				syncCol = withBg(defaultFg).Render(" ") + s + withBg(defaultFg).Render(strings.Repeat(" ", syncPad))
			}

			// Main divergence column
			mainCol := ""
			if b.mainAhead > 0 || b.mainBehind > 0 {
				aStr := fmt.Sprintf("+%d", b.mainAhead)
				bStr := fmt.Sprintf("-%d", b.mainBehind)
				mainCol = withBg(mutedFg).Render(" main ") +
					withBg(aheadStyle).Render(fmt.Sprintf("%*s", maxMainAhead, aStr)) +
					withBg(mutedFg).Render("/") +
					withBg(behindStyle).Render(fmt.Sprintf("%*s", maxMainBehind, bStr))
			}

			content := nameCol + upstreamCol + syncCol + mainCol
			if hasBg {
				cw := lipgloss.Width(content)
				if cw < width {
					content += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-cw))
				}
			} else {
				content = fitWidth(content, width)
			}
			lines = append(lines, content)
		} else {
			// Remote-only branch
			ri := i - len(m.branches)
			rb := m.remoteBranches[ri]
			content := withBg(defaultFg).Render("  ") + withBg(mutedFg).Render(rb.remote+"/") + withBg(defaultFg).Render(rb.branch)
			if hasBg {
				cw := lipgloss.Width(content)
				if cw < width {
					content += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-cw))
				}
			} else {
				content = fitWidth(content, width)
			}
			lines = append(lines, content)
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderWorktrees(width, height int) string {
	isActive := m.activePanel == panelBranches && m.branchTab == 1

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
		withBg := func(s lipgloss.Style) lipgloss.Style { return s }
		defStyle := lipgloss.NewStyle()
		mutedFg := dimStyle
		var bg lipgloss.TerminalColor
		hasBg := false
		if isSelected {
			bg = selectedStyle.GetBackground()
			hasBg = true
			withBg = func(s lipgloss.Style) lipgloss.Style { return s.Background(bg) }
			defStyle = lipgloss.NewStyle().Foreground(selectedStyle.GetForeground()).Background(bg)
			mutedFg = lipgloss.NewStyle().Foreground(dimSelectedColor)
		}

		content := defStyle.Render("  "+label) + withBg(mutedFg).Render(" "+wt.path)
		if hasBg {
			cw := lipgloss.Width(content)
			if cw < width {
				content += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-cw))
			}
		} else {
			content = fitWidth(content, width)
		}
		lines = append(lines, content)
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

		withBg := func(s lipgloss.Style) lipgloss.Style { return s }
		defStyle := lipgloss.NewStyle()
		mutedStyle := dimStyle
		var bg lipgloss.TerminalColor
		hasBg := false

		if isSelected {
			bg = selectedStyle.GetBackground()
			hasBg = true
			withBg = func(s lipgloss.Style) lipgloss.Style { return s.Background(bg) }
			defStyle = lipgloss.NewStyle().Foreground(selectedStyle.GetForeground()).Background(bg)
			mutedStyle = lipgloss.NewStyle().Foreground(dimSelectedColor)
		} else if isCursor {
			defStyle = lipgloss.NewStyle().Foreground(cursorStyle.GetForeground())
			mutedStyle = lipgloss.NewStyle().Foreground(dimSelectedColor)
		}

		rendered := m.renderLine(panel, i, line, width, withBg, defStyle, mutedStyle)
		if hasBg {
			cw := lipgloss.Width(rendered)
			if cw < width {
				rendered += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-cw))
			}
		}
		lines = append(lines, rendered)
	}

	// Pad remaining lines
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

func (m model) renderLine(panel, idx int, line string, width int, withBg func(lipgloss.Style) lipgloss.Style, defStyle, mutedStyle lipgloss.Style) string {
	switch panel {
	case panelChanges:
		if m.dirMode {
			if idx >= len(m.dirEntries) {
				return defStyle.Render(line)
			}
			entry := m.dirEntries[idx]
			name := entry.display
			prefix := ""
			if lastConn := strings.LastIndex(name, "─ "); lastConn >= 0 {
				prefix = name[:lastConn+len("─ ")]
				name = name[lastConn+len("─ "):]
			}
			connPart := withBg(mutedStyle).Render(prefix)
			if entry.isDir {
				return connPart + withBg(treeDirStyle).Render(name)
			}
			return connPart + defStyle.Render(name)
		}
		if idx >= len(m.changes) {
			return defStyle.Render(line)
		}
		entry := m.changes[idx]
		name := entry.display
		prefix := ""
		if lastConn := strings.LastIndex(name, "─ "); lastConn >= 0 {
			prefix = name[:lastConn+len("─ ")]
			name = name[lastConn+len("─ "):]
		}
		connPart := withBg(mutedStyle).Render(prefix)
		if entry.isDir {
			return connPart + withBg(treeDirStyle).Render(name)
		}
		tag := statusTag(entry.status, withBg)
		switch {
		case strings.Contains(entry.status, "A"), strings.Contains(entry.status, "?"):
			return connPart + tag + defStyle.Render(" ") + withBg(statusAddedStyle).Render(name)
		case strings.Contains(entry.status, "D"):
			return connPart + tag + defStyle.Render(" ") + withBg(statusDeletedStyle).Render(name)
		default:
			return connPart + tag + defStyle.Render(" ") + withBg(statusModifiedStyle).Render(name)
		}

	case panelCommits:
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			return withBg(hashStyle).Render(parts[0]) + defStyle.Render(" "+parts[1])
		}
		return defStyle.Render(line)
	}
	return defStyle.Render(line)
}
