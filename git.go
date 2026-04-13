package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func git(args ...string) []string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	s := strings.TrimRight(string(out), "\n\r")
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

func mainBranch() string {
	if exec.Command("git", "rev-parse", "--verify", "main").Run() == nil {
		return "main"
	}
	if exec.Command("git", "rev-parse", "--verify", "master").Run() == nil {
		return "master"
	}
	return ""
}

func commitsVsMain(mainRef, branch string) (int, int) {
	ab := git("rev-list", "--left-right", "--count", mainRef+"..."+branch)
	if len(ab) > 0 {
		var behind, ahead int
		fmt.Sscanf(ab[0], "%d\t%d", &behind, &ahead)
		return ahead, behind
	}
	return 0, 0
}

func loadBranches() []branchEntry {
	raw := git("branch", "--format=%(refname:short)")
	main := mainBranch()
	var entries []branchEntry
	for _, name := range raw {
		ahead, behind := 0, 0
		// Check if branch has an upstream
		ab := git("rev-list", "--left-right", "--count", name+"..."+name+"@{upstream}")
		if len(ab) > 0 {
			fmt.Sscanf(ab[0], "%d\t%d", &ahead, &behind)
		}
		e := branchEntry{name: name, ahead: ahead, behind: behind}
		if main != "" && name != main {
			e.mainAhead, e.mainBehind = commitsVsMain(main, name)
		}
		entries = append(entries, e)
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

func loadWorktrees() []worktreeEntry {
	lines := git("worktree", "list", "--porcelain")
	var entries []worktreeEntry
	var current worktreeEntry
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = worktreeEntry{path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			h := strings.TrimPrefix(line, "HEAD ")
			if len(h) > 7 {
				h = h[:7]
			}
			current.head = h
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			current.bare = true
		case line == "":
			if current.path != "" {
				entries = append(entries, current)
			}
			current = worktreeEntry{}
		}
	}
	if current.path != "" {
		entries = append(entries, current)
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
