// Command hello is the go-cli-template reference CLI. It exercises every
// feature the template provides so the look-and-feel can be tuned before any
// real toolchain CLI is built on it: Kong commands/flags/enums/args, the
// TTY-gated Lipgloss styling, the --output text|json|auto contract, the JSON
// envelope + diagnostics, deterministic errors + the exit-code ladder, the
// reflected `schema`, `version`, and shell completions.
//
// Try:
//
//	hello greet Ada --greeting=howdy --repeat 2
//	hello greet Ada -o json
//	hello demo ui             # the full styling gallery (glyphs, badges, panels…)
//	hello demo table          # styled on a TTY; JSON rows when piped
//	hello demo diagnostics -o json
//	hello demo fail --code 4  # deterministic error → exit 4
//	hello schema | jq .       # the machine surface
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/willabides/kongplete"

	"github.com/vista-cloud-dev/go-cli-template/clikit"
)

// CLI is the root command grammar. The whole surface is one typed struct —
// the single source of truth Kong parses and `schema` reflects (spec §5).
type CLI struct {
	clikit.Globals

	Greet   greetCmd          `cmd:"" help:"Print a styled greeting — the look-and-feel sampler."`
	Demo    demoCmd           `cmd:"" help:"Feature demos: tables, diagnostics, deterministic errors."`
	Schema  clikit.SchemaCmd  `cmd:"" help:"Emit the command/flag/enum tree as JSON (agent discovery)."`
	Version clikit.VersionCmd `cmd:"" help:"Show version and build info."`

	InstallCompletions kongplete.InstallCompletions `cmd:"" help:"Install shell tab-completions."`
}

func main() {
	cli := &CLI{}
	os.Exit(clikit.Run(
		"hello",
		"go-cli-template — a reference CLI exercising every shared toolchain convention.",
		cli, &cli.Globals,
	))
}

// --- greet -------------------------------------------------------------------

type greetCmd struct {
	Name     string `arg:"" optional:"" default:"world" help:"Who to greet."`
	Greeting string `short:"g" enum:"hello,hi,howdy,salutations" default:"hello" help:"Greeting word."`
	Shout    bool   `help:"Uppercase the greeting."`
	Repeat   int    `short:"n" default:"1" help:"Repeat the greeting N times."`
}

type greetResult struct {
	Greeting string `json:"greeting"`
	Name     string `json:"name"`
	Message  string `json:"message"`
	Repeat   int    `json:"repeat"`
}

func (c *greetCmd) Run(cc *clikit.Context) error {
	if c.Repeat < 1 {
		return clikit.Fail(clikit.ExitUsage, "BAD_REPEAT", "--repeat must be >= 1", "")
	}
	msg := fmt.Sprintf("%s, %s!", capitalize(c.Greeting), c.Name)
	if c.Shout {
		msg = strings.ToUpper(msg)
	}
	return cc.Result(greetResult{c.Greeting, c.Name, msg, c.Repeat}, func() {
		cc.Title("greeting")
		for i := 0; i < c.Repeat; i++ {
			fmt.Fprintln(cc.Stdout, cc.Accent(msg))
		}
		if cc.Verbose {
			fmt.Fprintln(cc.Stderr, cc.Faint(fmt.Sprintf("greeting=%q shout=%v repeat=%d", c.Greeting, c.Shout, c.Repeat)))
		}
	})
}

// --- demo --------------------------------------------------------------------

type demoCmd struct {
	UI          uiCmd    `cmd:"" name:"ui" help:"Showcase the full styling toolkit (glyphs, badges, panels, tree, spinner)."`
	Table       tableCmd `cmd:"" help:"Render a styled table (TTY) or JSON rows (piped)."`
	Diagnostics diagCmd  `cmd:"" help:"Emit lint-style diagnostics in the --output json envelope."`
	Fail        failCmd  `cmd:"" help:"Trigger a deterministic error to show the error object + exit code."`
}

type tableCmd struct{}

type repoRow struct {
	Repo   string `json:"repo"`
	Tier   string `json:"tier"`
	Status string `json:"status"`
}

func (tableCmd) Run(cc *clikit.Context) error {
	rows := []repoRow{
		{"clikit", "Go", "done"},
		{"go-cli-template", "Go", "in progress"},
		{"m-parse", "Go", "to-do"},
		{"irissync", "Go", "to-do"},
		{"m-cli", "Go", "to-do"},
	}
	return cc.Result(rows, func() {
		cc.Title("repos")
		grid := make([][]string, 0, len(rows))
		for _, r := range rows {
			grid = append(grid, []string{r.Repo, r.Tier, statusBadge(cc, r.Status)})
		}
		cc.Table([]string{"Repo", "Tier", "Status"}, grid)
	})
}

// statusBadge maps a status string to a colored pill for table cells.
func statusBadge(cc *clikit.Context, status string) string {
	switch status {
	case "done":
		return cc.Badge("ok", status)
	case "in progress":
		return cc.Badge("info", status)
	default:
		return cc.Badge("neutral", status)
	}
}

type diagCmd struct{}

func (diagCmd) Run(cc *clikit.Context) error {
	diags := []clikit.Diagnostic{
		{File: "DGREG.mac", Line: 42, Col: 3, Rule: "M-MOD-036", Severity: "error", Message: "Tainted value reaches XECUTE."},
		{File: "XUSER.mac", Line: 7, Col: 1, Rule: "SAC-1.2", Severity: "warning", Message: "Line exceeds 245 columns."},
	}
	summary := map[string]int{"filesScanned": 412, "findings": len(diags)}
	return cc.Diagnostics(summary, diags, func() {
		cc.Title("diagnostics")
		for _, d := range diags {
			fmt.Fprintf(cc.Stdout, "%s  %s:%d:%d  %s  %s\n",
				cc.Severity(d.Severity), d.File, d.Line, d.Col, cc.Faint(d.Rule), d.Message)
		}
		cc.Rule("")
		fmt.Fprintf(cc.Stdout, "%s scanned   %s errors   %s warnings\n",
			cc.Badge("neutral", fmt.Sprintf("%d files", summary["filesScanned"])),
			cc.Badge("err", "1"), cc.Badge("warn", "1"))
	})
}

type failCmd struct {
	Code int `default:"4" help:"Exit code to simulate: 1 runtime, 2 usage, 3 check, 4 refused."`
}

func (c *failCmd) Run(_ *clikit.Context) error {
	switch c.Code {
	case clikit.ExitRuntime:
		return clikit.Fail(clikit.ExitRuntime, "RUNTIME", "simulated runtime/IO error", "")
	case clikit.ExitUsage:
		return clikit.Fail(clikit.ExitUsage, "USAGE", "simulated usage error", "run with --help")
	case clikit.ExitCheck:
		return clikit.Fail(clikit.ExitCheck, "FINDINGS", "simulated --check finding (drift detected)", "fix and re-run")
	default:
		return clikit.Fail(clikit.ExitRefused, "ENGINE_UNRESOLVED",
			"no engine resolved and no test substrate available",
			"pass --engine or set [engine] in .m-cli.toml")
	}
}

// --- demo ui (the styling gallery) -------------------------------------------

type uiCmd struct{}

type uiDoc struct {
	Palette    []string          `json:"palette"`
	Glyphs     map[string]string `json:"glyphs"`
	Components []string          `json:"components"`
}

func (uiCmd) Run(cc *clikit.Context) error {
	// In JSON mode the gallery is non-visual, so emit a description of the
	// toolkit (palette, active glyph set, components) as the machine surface.
	if cc.JSON() {
		g := cc.Glyphs()
		return cc.Result(uiDoc{
			Palette: []string{"indigo", "teal", "green", "amber", "red", "blue", "gray"},
			Glyphs: map[string]string{
				"ok": g.OK, "err": g.Err, "warn": g.Warn, "info": g.Info,
				"bullet": g.Bullet, "arrow": g.Arrow, "dot": g.Dot, "pending": g.Pending,
			},
			Components: []string{
				"title", "subtitle", "status", "badge", "kv", "list",
				"tree", "panel", "rule", "table", "spinner", "progress", "link",
			},
		}, nil)
	}

	cc.Title("clikit styling gallery")
	fmt.Fprintln(cc.Stdout, cc.Faint("the toolkit every toolchain CLI inherits — styled on a TTY, plain when piped"))

	cc.Rule("status lines")
	fmt.Fprintln(cc.Stdout, cc.Success("build passed — 0 errors"))
	fmt.Fprintln(cc.Stdout, cc.Warning("2 files exceed 245 columns"))
	fmt.Fprintln(cc.Stdout, cc.Failure("engine unresolved"))
	fmt.Fprintln(cc.Stdout, cc.Info("schema version 1.0"))

	cc.Rule("badges")
	fmt.Fprintln(cc.Stdout, strings.Join([]string{
		cc.Badge("ok", "PASS"), cc.Badge("warn", "FLAKY"), cc.Badge("err", "FAIL"),
		cc.Badge("info", "INFO"), cc.Badge("accent", "NEW"), cc.Badge("neutral", "SKIP"),
	}, " "))

	cc.Rule("key / value")
	cc.KV(
		[2]string{"tool", cc.Accent("hello")},
		[2]string{"schema", "1.0"},
		[2]string{"output", "auto · text on a TTY, json when piped"},
		[2]string{"spec", cc.Link("§5.5 conventions", "https://github.com/vista-cloud-dev/go-cli-template")},
	)

	cc.Rule("list")
	cc.List(
		"kong command grammar — one typed struct",
		"TTY-gated lipgloss styling",
		"versioned JSON envelope",
		"deterministic exit-code ladder",
	)

	cc.Rule("tree")
	cc.Tree(clikit.TreeNode{Label: cc.Accent("m-cli go toolchain"), Children: []clikit.TreeNode{
		{Label: "clikit " + cc.Faint("(shared conventions)")},
		{Label: "m-cli", Children: []clikit.TreeNode{{Label: "parse"}, {Label: "lint"}}},
		{Label: "irissync"},
	}})

	cc.Rule("panel")
	cc.Panel("build summary",
		cc.Success("compiled 5 packages"),
		cc.Info("3 / 5 cached"),
		cc.Faint("0.42s · CGO_ENABLED=0 · -trimpath"))

	cc.Rule("table")
	cc.Table([]string{"Repo", "Tier", "Status"}, [][]string{
		{"clikit", "Go", cc.Badge("ok", "done")},
		{"go-cli-template", "Go", cc.Badge("info", "in progress")},
		{"m-cli", "Go", cc.Badge("neutral", "to-do")},
	})

	cc.Rule("live: spinner + progress")
	uiLiveDemo(cc)

	cc.Rule("glyphs")
	g := cc.Glyphs()
	fmt.Fprintln(cc.Stdout, strings.Join([]string{
		cc.Success("ok"), cc.Failure("err"), cc.Warning("warn"), cc.Info("info"),
		cc.Accent(g.Bullet) + " bullet", cc.Accent(g.Arrow) + " arrow",
		cc.OK(g.Dot) + " active", cc.Faint(g.Pending) + " pending",
	}, "   "))
	return nil
}

// uiLiveDemo runs the spinner and progress bar. Both are no-ops off an
// interactive color TTY, so it prints a note in that case instead.
func uiLiveDemo(cc *clikit.Context) {
	if !cc.Color {
		fmt.Fprintln(cc.Stdout, cc.Faint("(spinner + progress bar animate only on an interactive color terminal)"))
		return
	}
	sp := cc.NewSpinner("resolving engine…")
	sp.Start()
	time.Sleep(650 * time.Millisecond)
	sp.Update("connecting to test substrate…")
	time.Sleep(550 * time.Millisecond)
	sp.Success("engine ready")

	files := []string{"DGREG.mac", "XUSER.mac", "ZVISTA.mac", "DIC.int", "DD.dat"}
	pb := cc.NewProgress(len(files))
	for i, f := range files {
		pb.Set(i+1, "scanning "+f)
		time.Sleep(180 * time.Millisecond)
	}
	pb.Done(fmt.Sprintf("scanned %d files", len(files)))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
