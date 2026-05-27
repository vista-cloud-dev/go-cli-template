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
//	hello demo table          # styled on a TTY; JSON rows when piped
//	hello demo diagnostics -o json
//	hello demo fail --code 4  # deterministic error → exit 4
//	hello schema | jq .       # the machine surface
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/vista-cloud-dev/go-cli-template/clikit"
	"github.com/willabides/kongplete"
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
		{"go-cli-template", "Go", "in progress"},
		{"m-parse", "Go", "to-do"},
		{"irissync", "Go", "to-do"},
		{"m-cli", "Go", "to-do"},
	}
	return cc.Result(rows, func() {
		cc.Title("repos")
		grid := make([][]string, 0, len(rows))
		for _, r := range rows {
			grid = append(grid, []string{r.Repo, r.Tier, r.Status})
		}
		cc.Table([]string{"Repo", "Tier", "Status"}, grid)
	})
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
			fmt.Fprintf(cc.Stdout, "%s %s:%d:%d %s %s\n",
				cc.Severity(d.Severity), d.File, d.Line, d.Col, cc.Faint(d.Rule), d.Message)
		}
		fmt.Fprintln(cc.Stdout, cc.Faint(fmt.Sprintf("scanned %d files · %d findings", summary["filesScanned"], summary["findings"])))
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

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
