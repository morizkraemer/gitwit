# CLAUDE.md

## Project

gitwit is a minimal terminal UI for git, built with Go, Bubble Tea, and Lip Gloss.

## Build & Run

```sh
go build ./...
go run .
```

## Workflow

- Always review the diff before committing. Check for: unused variables/fields, incorrect logic order, accidental recursion, dead code.
- Tag releases after pushing (vX.Y.Z). GoReleaser builds binaries and updates the Homebrew tap automatically.

## Code Style

- Use `lipgloss.Width()` for display width calculations, never `len()` on strings that may contain multi-byte characters.
- Use the `withBg`/`defStyle`/`mutedStyle` pattern for selection-aware rendering — single code path, no duplication.
