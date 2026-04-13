package main

import (
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

func boolPtr(b bool) *bool    { return &b }
func uintPtr(u uint) *uint    { return &u }
func strPtr(s string) *string { return &s }

var cleanStyle = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "\n",
			BlockSuffix: "\n",
		},
		Indent: uintPtr(2),
		Margin: uintPtr(0),
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: strPtr("228"),
			Bold:  boolPtr(true),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: strPtr("228"),
			Bold:  boolPtr(true),
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: strPtr("39"),
			Bold:  boolPtr(true),
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: strPtr("39"),
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: strPtr("39"),
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  strPtr("35"),
			Faint: boolPtr(true),
		},
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
		Indent:         uintPtr(2),
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			Indent: uintPtr(2),
		},
		LevelIndent: 2,
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
	},
	Emph:          ansi.StylePrimitive{Italic: boolPtr(true)},
	Strong:        ansi.StylePrimitive{Bold: boolPtr(true)},
	Strikethrough: ansi.StylePrimitive{CrossedOut: boolPtr(true)},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: strPtr("203"),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			Indent: uintPtr(4),
			Margin: uintPtr(0),
		},
		Theme: "dracula",
	},
	BlockQuote: ansi.StyleBlock{
		Indent:      uintPtr(2),
		IndentToken: strPtr("│ "),
	},
	Link: ansi.StylePrimitive{
		Color:     strPtr("30"),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color: strPtr("35"),
		Bold:  boolPtr(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  strPtr("240"),
		Format: "\n--------\n",
	},
	Task: ansi.StyleTask{
		Ticked:   "☑ ",
		Unticked: "☐ ",
	},
	Table: ansi.StyleTable{},
}

func renderMarkdown(filePath string, width int) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(cleanStyle),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return nil, err
	}

	rendered, err := r.Render(string(content))
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimRight(rendered, "\n"), "\n"), nil
}
