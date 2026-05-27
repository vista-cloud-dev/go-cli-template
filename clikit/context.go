package clikit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// Envelope is the stable --output json wrapper for every command result
// (spec §5.5). One shape across the whole toolchain.
type Envelope struct {
	SchemaVersion string       `json:"schemaVersion"`
	Command       string       `json:"command,omitempty"`
	OK            bool         `json:"ok"`
	Exit          int          `json:"exit"`
	Data          any          `json:"data,omitempty"`
	Diagnostics   []Diagnostic `json:"diagnostics,omitempty"`
	Error         *Error       `json:"error,omitempty"`
}

// Diagnostic is one lint/diagnostic finding (the editor↔CI shared shape).
type Diagnostic struct {
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Col      int    `json:"col,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Context carries the resolved output mode + styling into a command's Run.
// Bind it via clikit.Run; command methods take Run(cc *clikit.Context) error.
type Context struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Format  OutputFormat
	Color   bool
	Verbose bool
	Command string
	th      theme
}

// NewContext resolves the format/color for this invocation from the globals
// and the TTY state of stdout.
func NewContext(g *Globals, command string) *Context {
	tty := term.IsTerminal(int(os.Stdout.Fd()))
	format := resolveFormat(g.Output, tty)
	color := format == FormatText && tty && !g.NoColor
	return &Context{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Format:  format,
		Color:   color,
		Verbose: g.Verbose,
		Command: command,
		th:      newTheme(),
	}
}

func resolveFormat(output string, tty bool) OutputFormat {
	switch output {
	case "json":
		return FormatJSON
	case "text":
		return FormatText
	default: // "auto": styled text on a TTY, JSON when piped/redirected (§3.3)
		if tty {
			return FormatText
		}
		return FormatJSON
	}
}

// JSON reports whether the machine-readable envelope should be emitted.
func (c *Context) JSON() bool { return c.Format == FormatJSON }

// Result renders a command's result: the JSON envelope in JSON mode, otherwise
// the human-facing text produced by the text closure.
func (c *Context) Result(data any, text func()) error {
	if c.JSON() {
		return c.emit(Envelope{SchemaVersion: SchemaVersion, Command: c.Command, OK: true, Exit: ExitOK, Data: data})
	}
	if text != nil {
		text()
	}
	return nil
}

// Diagnostics renders a result that carries lint-style findings.
func (c *Context) Diagnostics(data any, diags []Diagnostic, text func()) error {
	if c.JSON() {
		return c.emit(Envelope{SchemaVersion: SchemaVersion, Command: c.Command, OK: true, Exit: ExitOK, Data: data, Diagnostics: diags})
	}
	if text != nil {
		text()
	}
	return nil
}

// EmitJSON pretty-prints an arbitrary value as JSON (used by `schema`).
func (c *Context) EmitJSON(v any) error { return c.emit(v) }

func (c *Context) emit(v any) error { return writeJSON(c.Stdout, v) }

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// --- styling helpers (no-ops unless Color) ---

// Title prints a styled section heading.
func (c *Context) Title(s string) { fmt.Fprintln(c.Stdout, c.th.title.render(c.Color, "▸ "+s)) }

// Accent returns s styled as an accent (or unchanged when !Color).
func (c *Context) Accent(s string) string { return c.th.accent.render(c.Color, s) }

// Faint returns s styled faint/dim (or unchanged when !Color).
func (c *Context) Faint(s string) string { return c.th.faint.render(c.Color, s) }

// OK returns s styled as success.
func (c *Context) OK(s string) string { return c.th.ok.render(c.Color, s) }

// Severity returns an uppercased, colour-coded severity label.
func (c *Context) Severity(s string) string {
	label := strings.ToUpper(s)
	if !c.Color {
		return label
	}
	switch s {
	case "error":
		return c.th.errLabel.s.Render(label)
	case "warning":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(label)
	default:
		return c.th.faint.s.Render(label)
	}
}

// Table renders a styled box table on a TTY, or a clean tab-aligned table
// otherwise. (JSON mode never calls this — commands emit rows as data.)
func (c *Context) Table(headers []string, rows [][]string) {
	if c.Color {
		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(c.th.faint.s).
			Headers(headers...).
			Rows(rows...).
			StyleFunc(func(row, _ int) lipgloss.Style {
				if row == table.HeaderRow {
					return c.th.header.s.Padding(0, 1)
				}
				return lipgloss.NewStyle().Padding(0, 1)
			})
		fmt.Fprintln(c.Stdout, t.Render())
		return
	}
	tw := tabwriter.NewWriter(c.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	_ = tw.Flush()
}

// --- theme ---

type styled struct{ s lipgloss.Style }

func (x styled) render(color bool, s string) string {
	if !color {
		return s
	}
	return x.s.Render(s)
}

type theme struct {
	title    styled
	accent   styled
	faint    styled
	ok       styled
	errLabel styled
	hint     styled
	header   styled
}

func newTheme() theme {
	return theme{
		title:    styled{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))},
		accent:   styled{lipgloss.NewStyle().Foreground(lipgloss.Color("213"))},
		faint:    styled{lipgloss.NewStyle().Faint(true)},
		ok:       styled{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))},
		errLabel: styled{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))},
		hint:     styled{lipgloss.NewStyle().Faint(true).Italic(true)},
		header:   styled{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))},
	}
}
