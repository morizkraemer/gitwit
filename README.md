# gitwit

A minimal terminal UI for git — like lazygit, but only the parts you actually need.

![Go](https://img.shields.io/badge/Go-1.26-blue)
![License](https://img.shields.io/badge/License-MIT-green)

## Features

- **Changes panel** — file tree of uncommitted changes, color-coded by git status
- **Directory browser** — browse all repo files, open/collapse folders, open files in your editor
- **Branches panel** — local and remote branches with ahead/behind tracking (upstream + main)
- **Commits panel** — recent commits for the selected branch
- **Diff preview** — full-screen color-coded diff for any changed file
- **Markdown viewer** — rendered markdown preview with cursor navigation
- **Panel management** — hide/show any panel with keyboard shortcuts
- **Worktrees** — view and navigate git worktrees
- **Editor integration** — opens files in `$EDITOR`, auto-detects `nvim`, falls back to `vim`
- **Auto-refresh** — syncs with external git changes every 2 seconds

## Install

### Homebrew

```sh
brew install morizkraemer/tap/gitwit
```

### Go

```sh
go install github.com/morizkraemer/gitwit@latest
```

### From source

```sh
git clone https://github.com/morizkraemer/gitwit.git
cd gitwit
go build -o gitwit .
```

## Usage

Run `gitwit` inside any git repository.

### Global

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Cycle between visible panels |
| `1` `2` `3` | Jump to changes / branches / commits panel |
| `!` `@` `#` | Toggle visibility of each panel |
| `q` / `ctrl+c` | Quit |

### Changes panel

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `space` | Stage / unstage file |
| `a` | Stage all |
| `c` | Commit (opens message prompt) |
| `enter` | Open diff preview |
| `e` | Open file in editor |
| `v` | Switch to directory browser |
| `r` | Refresh |

### Directory browser

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `enter` | Open/close folder, open file in editor (`.md` files open formatted) |
| `v` | Switch back to git changes |

### Branches panel

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `enter` | Checkout branch |
| `B` | Create new branch |
| `f` | Fetch all remotes |
| `p` | Pull |
| `P` | Push |
| `v` | Switch to worktrees |

### Worktrees

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `v` | Switch back to branches |

### Commits panel

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |

### Diff / Markdown viewer

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll / navigate |
| `d` / `u` | Page down / up |
| `e` | Open file in editor |
| `q` / `esc` | Close |

## Built with

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Glamour](https://github.com/charmbracelet/glamour) — markdown rendering

## License

[MIT](LICENSE)
