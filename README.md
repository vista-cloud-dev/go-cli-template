# go-cli-template

**A batteries-included starting point for any Go command-line tool.** Clone it,
rename a couple of identifiers, replace the demo commands with yours, and you
have a CLI that already speaks one consistent command grammar, one output
contract (`text | json | auto`), one error/exit-code ladder, a polished
TTY-styling toolkit, shell completions, a reproducible build, CI, lint config,
and a curated set of Claude Code permissions for Go work.

> Originally the shared scaffold for the *m-cli Go toolchain* (so `m-cli`,
> `irissync`, `vista-meta`, … all behave identically), but it stands alone as a
> general Go-CLI template. The reference binary is called `hello`; it exercises
> every feature so you can see the look-and-feel before writing any code.

```sh
git clone https://github.com/vista-cloud-dev/go-cli-template.git
cd go-cli-template
go run . demo ui          # ← see the whole toolkit, right now
```

---

## Contents

- [Purpose](#purpose)
- [What you get](#what-you-get)
- [Quick start](#quick-start)
- [Architecture](#architecture)
- [Repository layout](#repository-layout)
- [The clikit package](#the-clikit-package)
- [Output contract and exit codes](#output-contract-and-exit-codes)
- [Styling toolkit](#styling-toolkit)
- [Bootstrap a new Go CLI](#bootstrap-a-new-go-cli)
- [Build and automation](#build-and-automation)
- [Go modules](#go-modules)
- [Claude Code permissions](#claude-code-permissions)
- [Continuous integration](#continuous-integration)
- [License](#license)

---

## Purpose

This repo solves the "every new CLI reinvents the same plumbing" problem. It
gives every Go tool built from it:

1. **One command grammar.** The entire surface is a single typed Go struct that
   [Kong](https://github.com/alecthomas/kong) parses — flags, args, enums,
   subcommands, help. No hand-rolled flag parsing.
2. **One output contract.** `--output text|json|auto`: styled human text on a
   terminal, a stable JSON envelope when piped or asked. Scripts and agents
   never have to scrape colored prose.
3. **One error/exit-code ladder.** Deterministic, machine-parseable errors with
   a fixed exit code per failure class.
4. **One look-and-feel.** A TTY-gated styling toolkit (adaptive palette, glyphs,
   badges, panels, tables, trees, spinner, progress) that degrades to plain
   text when not on a terminal.
5. **Turnkey tooling.** `make build/test/lint/dist/schema`, a pinned lint
   config, GitHub Actions CI, reproducible static builds, shell completions.
6. **Sane Claude Code defaults.** Go build/test/lint commands are pre-approved
   so the agent isn't constantly asking permission for routine work.

Use it two ways (see [Bootstrap a new Go CLI](#bootstrap-a-new-go-cli)):
**import `clikit`** as a library (recommended — convention updates arrive via
`go get -u`), or **scaffold-copy** the whole repo as a starting skeleton.

## What you get

- **Kong** command grammar — the whole CLI surface is one source-of-truth struct.
- **TTY-gated [Lipgloss](https://github.com/charmbracelet/lipgloss) styling** —
  styled on a terminal; **plain or JSON when piped**. Never blocks scripts/agents.
- **A professional styling toolkit** (`clikit/style.go`) — an adaptive, semantic
  palette (light/dark-aware; downsamples truecolor → 256 → 16), a glyph set with
  an ASCII fallback (`✓ ✗ ⚠ ℹ • → ▸ ●`), and composable primitives: titles,
  status lines, badges, key/value lists, lists, rules, panels, trees, tables —
  plus a spinner + progress bar (`clikit/spinner.go`). Every primitive is a
  no-op off a color TTY, so JSON/piped output stays clean.
- **`--output {text|json|auto}`** — `auto` = styled text on a TTY, JSON when piped.
- **A versioned JSON envelope** (`schemaVersion`/`command`/`ok`/`exit`/`data`/`diagnostics`/`error`).
- **Deterministic errors + exit-code ladder** — `0` ok · `1` runtime · `2` usage
  · `3` `--check`/findings · `4` engine-bound op refused.
- **`schema`** — reflects the Kong struct into JSON so the machine surface can't
  drift from `--help` (agent discovery).
- **`version`** (ldflags build stamp), **shell completions** (kongplete), styled help.

## Quick start

**Prerequisites:** Go **1.26+** (see `go.mod`). Optional: `golangci-lint` for
`make lint`. Nothing else — builds are pure-Go and `CGO_ENABLED=0`.

```sh
go run . demo ui                                  # the full styling gallery (glyphs, badges, panels, tree, spinner…)
go run . greet Ada --greeting howdy --repeat 2    # styled greeting on a TTY
go run . greet Ada -o json                        # the JSON envelope
go run . demo table                               # styled box table (TTY) / JSON rows (piped)
go run . demo diagnostics -o json                 # lint-style diagnostics envelope
go run . demo fail --code 4; echo "exit=$?"       # deterministic error → exit 4
go run . schema | jq .                            # the machine surface (agent discovery)
go run . version                                  # build metadata
go run . --help                                   # styled help
NO_COLOR=1 go run . greet Ada -o text             # styling off
```

Run `demo ui` on a real terminal to see colour; pipe it (`demo ui | cat`) and it
falls back to plain glyphs and tab-aligned layout. Tune the palette, glyphs, and
primitives in `clikit/style.go` and the live elements in `clikit/spinner.go` —
every tool built on `clikit` inherits the change.

## Architecture

The flow from process args to rendered output. Your code is just `main.go`
(the grammar + command bodies); everything else is the shared `clikit` layer.

```
   $ hello greet Ada --repeat 2 -o auto
                │  os.Args
                ▼
   ┌──────────────────────────────────────────────────────────────┐
   │ main.go — YOUR CLI grammar (one typed Kong struct)             │
   │   type CLI struct {                                            │
   │     clikit.Globals          // --output, --no-color, -v        │
   │     Greet   greetCmd   `cmd:""`   ┐                            │
   │     Demo    demoCmd    `cmd:""`   ├─ your commands             │
   │     Schema  clikit.SchemaCmd      │                            │
   │     Version clikit.VersionCmd     ┘  + reusable schema/version │
   │   }   func (c *greetCmd) Run(cc *clikit.Context) error { … }   │
   └───────────────────────────┬──────────────────────────────────┘
                               │  clikit.Run("hello", desc, &cli, &cli.Globals)
                               ▼
   ┌──────────────────────────────────────────────────────────────┐
   │ clikit/run.go — the ONE entry point every CLI shares           │
   │   1. kong.New(cli)        build grammar from the struct        │
   │   2. kongplete.Complete   shell tab-completion                 │
   │   3. parser.Parse(args)   → kctx.Command()                     │
   │   4. NewContext(globals)  resolve output format + color        │
   │   5. kctx.Run(ctx)        dispatch to your cmd.Run(cc)         │
   │   6. RenderError + exit   deterministic exit-code ladder       │
   └─────────┬─────────────────────────────────────────┬──────────┘
     success │                                          │ error
             ▼                                          ▼
   ┌─────────────────────────────────┐   ┌────────────────────────────┐
   │ clikit/context.go               │   │ clikit/errors.go            │
   │   Format: text | json | auto    │   │   Fail(exit,code,msg,hint)  │
   │   Color : TTY && !NO_COLOR      │   │   ladder: 0·1·2·3·4         │
   │   Result() / Diagnostics()      │   └────────────────────────────┘
   └──────┬───────────────────┬──────┘
   text   │                   │  json / piped / -o json
   (TTY)  ▼                   ▼
   ┌────────────────────┐   ┌─────────────────────────────────────────┐
   │ clikit/style.go    │   │ JSON envelope — the stable machine        │
   │ clikit/spinner.go  │   │ surface for agents & CI:                  │
   │   palette + glyphs │   │ { schemaVersion, command, ok, exit,       │
   │   Title Badge KV   │   │   data, diagnostics[], error }            │
   │   Panel Tree Table │   └─────────────────────────────────────────┘
   │   Spinner Progress │
   └────────────────────┘

   side commands:  clikit/schema.go  → `schema`  (reflects the grammar to JSON)
                   clikit/version.go → `version` (ldflags build metadata)
                   clikit/globals.go → shared --output / --no-color / --verbose
```

## Repository layout

```
go-cli-template/
├── main.go                     # YOUR CLI: the Kong command grammar + command Run() bodies
├── clikit/                     # the shared convention layer (import or copy this)
│   ├── globals.go              #   Globals: --output, --no-color, --verbose; OutputFormat
│   ├── run.go                  #   Run(): the single entry point (Kong + completion + dispatch)
│   ├── context.go              #   Context: resolved format/color; Result()/Diagnostics(); JSON envelope
│   ├── errors.go               #   Error object, Fail(), RenderError(), the exit-code ladder
│   ├── style.go                #   styling toolkit: palette, glyphs, Title/Badge/KV/Panel/Tree/Table…
│   ├── spinner.go              #   live elements: Spinner + Progress (TTY-only, no extra deps)
│   ├── schema.go               #   `schema` subcommand: reflects the Kong grammar → JSON
│   └── version.go              #   `version` subcommand: ldflags-stamped build metadata
├── Makefile                    # build / run / lint / test / tidy / schema / dist / clean
├── .golangci.yml               # pinned lint config (the single source of truth)
├── .github/workflows/ci.yml    # CI: lint + race tests + schema contract + cross-compile matrix
├── .claude/
│   ├── settings.json           # committed: Go build/test/lint/run + make targets pre-approved
│   └── settings.local.json     # (gitignored) personal/machine-local permission overrides
├── docs/
│   └── claude-permissions-guide.md   # deep dive on Claude Code permission layering
├── go.mod / go.sum             # module path + pinned dependencies
├── LICENSE / NOTICE            # Apache-2.0
└── .gitignore                  # build output, editor cruft, settings.local.json
```

## The clikit package

`clikit` is the convention layer. A minimal CLI is just this:

```go
package main

import (
	"os"

	"github.com/vista-cloud-dev/go-cli-template/clikit"
)

type CLI struct {
	clikit.Globals                       // --output, --no-color, --verbose

	Greet   GreetCmd          `cmd:"" help:"Say hello."`
	Schema  clikit.SchemaCmd  `cmd:"" help:"Emit the command tree as JSON."`
	Version clikit.VersionCmd `cmd:"" help:"Show version + build info."`
}

type GreetCmd struct {
	Name string `arg:"" default:"world" help:"Who to greet."`
}

// Every command implements Run(cc *clikit.Context) error.
func (c *GreetCmd) Run(cc *clikit.Context) error {
	return cc.Result(
		map[string]string{"name": c.Name},          // → data in the JSON envelope
		func() { cc.Title("greeting"); /* styled text path */ },
	)
}

func main() {
	cli := &CLI{}
	os.Exit(clikit.Run("mytool", "what mytool does.", cli, &cli.Globals))
}
```

Key idea: a command returns its result through **`cc.Result(data, textFn)`**.
In JSON/piped mode `data` is emitted in the envelope; on a TTY the `textFn`
closure renders styled output. You write the data once and the format is handled
for you. Use **`cc.Diagnostics(...)`** for lint-style findings and
**`clikit.Fail(exit, code, msg, hint)`** to return a deterministic error.

## Output contract and exit codes

`--output` resolves the render mode (`clikit/globals.go`, `clikit/context.go`):

| `--output` | On a TTY            | Piped / redirected |
|------------|---------------------|--------------------|
| `auto` (default) | styled text    | JSON envelope      |
| `text`     | styled text (color if TTY) | plain text  |
| `json`     | JSON envelope       | JSON envelope      |

`NO_COLOR=1` (or `--no-color`) disables ANSI even on a TTY.

**JSON envelope** — one stable shape for every command (`clikit/context.go`):

```json
{ "schemaVersion": "1.0", "command": "greet", "ok": true, "exit": 0,
  "data": { … }, "diagnostics": [ … ], "error": { … } }
```

**Exit-code ladder** (`clikit/errors.go`) — agents and CI branch on `code`+`exit`,
never on prose:

| Exit | Constant      | Meaning                                            |
|------|---------------|----------------------------------------------------|
| `0`  | `ExitOK`      | success                                            |
| `1`  | `ExitRuntime` | runtime error (IO / engine / parse)                |
| `2`  | `ExitUsage`   | usage error (bad flags/args)                       |
| `3`  | `ExitCheck`   | `--check`/lint found findings or drift             |
| `4`  | `ExitRefused` | engine-bound op refused (no engine / substrate)    |

## Styling toolkit

All styling is methods on `*clikit.Context` and a **no-op unless on a color TTY**,
so it never leaks ANSI into pipes or JSON. Defined in `clikit/style.go` /
`clikit/spinner.go`; see them all live with `go run . demo ui`.

| Primitive | Method | Notes |
|-----------|--------|-------|
| Section heading | `Title(s)` / `Subtitle(s)` | `▸`-prefixed, indigo |
| Status line | `Success` `Warning` `Failure` `Info` | glyph + colored label |
| Diagnostic severity | `Severity(s)` | `✗ ERROR` / `⚠ WARNING` / `ℹ INFO` |
| Inline pill | `Badge(kind, label)` | `ok/warn/err/info/accent/neutral` |
| Aligned pairs | `KV(pairs…)` | keys padded to a common width |
| Bulleted list | `List(items…)` | |
| Divider | `Rule(label)` | width-aware, optional centered label |
| Bordered box | `Panel(title, lines…)` | rounded border |
| Tree | `Tree(TreeNode{…})` | `├─ └─ │` connectors |
| Table | `Table(headers, rows)` | rounded border + zebra striping |
| Hyperlink | `Link(text, url)` | OSC-8; falls back to `text (url)` |
| Inline text | `Accent` `Faint` `Muted` `OK` | |
| Spinner | `NewSpinner(msg)` → `Start/Update/Success/Fail` | braille, stderr, TTY-only |
| Progress | `NewProgress(total)` → `Set(n,label)/Done` | `█`/`░` bar, stderr, TTY-only |

The palette is **adaptive** (`lipgloss.AdaptiveColor`): it picks light- or
dark-background inks automatically and downsamples to the terminal's color
profile (truecolor → 256 → 16). Glyphs use a Unicode set on UTF-8 locales and an
ASCII fallback (`+ x ! i * -> >`) otherwise — detected from `LC_ALL`/`LC_CTYPE`/`LANG`.

## Bootstrap a new Go CLI

Pick one path. Both keep the conventions single-sourced.

### Option 1 — import `clikit` (recommended)

Your repo depends on this module; convention updates arrive via `go get -u`.

```sh
go mod init github.com/you/mytool
go get github.com/vista-cloud-dev/go-cli-template/clikit
```

Then write your `main.go` as shown in [The clikit package](#the-clikit-package)
and call `clikit.Run(...)`. Copy the `Makefile`, `.golangci.yml`,
`.github/workflows/ci.yml`, and `.claude/settings.json` for the same tooling.

### Option 2 — scaffold-copy the whole repo

Start from this skeleton and edit in place:

```sh
# clone (or: gonew github.com/vista-cloud-dev/go-cli-template github.com/you/mytool)
git clone https://github.com/vista-cloud-dev/go-cli-template.git mytool
cd mytool && rm -rf .git && git init
```

**Rename checklist** (so everything keeps working):

1. **Module path** — `go mod edit -module github.com/you/mytool`, then update the
   `clikit` import paths in `main.go` (and any others) to the new module path.
2. **`Makefile`** — set `BIN` (output binary name) and `PKG` (module path; `LDPKG`
   and the ldflags version stamp derive from it).
3. **`main.go`** — change `clikit.Run("hello", "…description…", …)` to your tool's
   name + description; replace the `greet`/`demo` commands with your real ones.
4. **`docs/`, `NOTICE`, `LICENSE`** — update names/copyright as needed.
5. `go mod tidy && make all` — confirm it builds, lints, and tests.

CI (`.github/workflows/ci.yml`) and the `schema` contract use `.` / `go run .`,
so they keep working without edits.

## Build and automation

Every target works with just Go installed (`make lint` also needs
`golangci-lint`). Builds are static (`CGO_ENABLED=0`), `-trimpath`, and
version-stamped via `-ldflags`.

| Target | What it does |
|--------|--------------|
| `make build` | `dist/$(BIN)`, static + trimmed + version-stamped |
| `make run ARGS="greet Ada"` | build, then run with `ARGS` |
| `make test` | `go test -race -cover ./...` |
| `make lint` | `golangci-lint run ./...` |
| `make tidy` | `go mod tidy` |
| `make schema` | build + emit the JSON schema (a CI conformance artifact) |
| `make dist` | cross-compile the matrix → `dist/` |
| `make all` | `lint test build` |
| `make clean` | remove `dist/` |

**Version stamping** — `make` injects build metadata at link time:

```sh
go build -trimpath -ldflags \
  "-s -w -X <module>/clikit.Version=$VER -X …/clikit.Commit=$SHA -X …/clikit.Date=$DATE" .
```

`VERSION`/`COMMIT`/`DATE` default to `git describe` / `git rev-parse` / UTC date,
so a tagged release self-stamps. **Cross-compile matrix** (`make dist`):
`linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64`.

## Go modules

Pinned in `go.mod`; `go.sum` carries the checksums. Run `make tidy` after adding
imports.

**Direct dependencies** — the four that define the CLI:

| Module | Version | Role |
|--------|---------|------|
| [`github.com/alecthomas/kong`](https://github.com/alecthomas/kong) | `v1.15.0` | Struct-tag CLI parser — the command/flag/arg/enum grammar |
| [`github.com/charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) | `v1.1.0` | Terminal styling & layout (color, borders, tables, trees) |
| [`github.com/willabides/kongplete`](https://github.com/willabides/kongplete) | `v0.4.0` | Shell tab-completion for Kong CLIs |
| [`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term) | `v0.43.0` | TTY detection & terminal size |

**Indirect (transitive) dependencies** — pulled in by the four above:

| Module | Version | Pulled in by / role |
|--------|---------|---------------------|
| `github.com/charmbracelet/colorprofile` | `v0.2.3-…` | lipgloss — color-profile detection |
| `github.com/charmbracelet/x/ansi` | `v0.8.0` | lipgloss — ANSI sequence handling |
| `github.com/charmbracelet/x/cellbuf` | `v0.0.13-…` | lipgloss — terminal cell buffer |
| `github.com/charmbracelet/x/term` | `v0.2.1` | lipgloss — terminal utilities |
| `github.com/muesli/termenv` | `v0.16.0` | lipgloss — env detection & color degradation |
| `github.com/lucasb-eyer/go-colorful` | `v1.2.0` | lipgloss — color math |
| `github.com/aymanbagabas/go-osc52/v2` | `v2.0.1` | termenv — OSC 52 clipboard sequences |
| `github.com/xo/terminfo` | `…` | termenv — terminfo parsing |
| `github.com/mattn/go-isatty` | `v0.0.20` | termenv — isatty checks |
| `github.com/mattn/go-runewidth` | `v0.0.16` | lipgloss — rune display width |
| `github.com/rivo/uniseg` | `v0.4.7` | runewidth — Unicode segmentation |
| `github.com/posener/complete` | `v1.2.3` | kongplete — completion engine |
| `github.com/riywo/loginshell` | `…` | kongplete — login-shell detection |
| `github.com/hashicorp/go-multierror` | `v1.1.1` | error aggregation |
| `github.com/hashicorp/errwrap` | `v1.1.0` | go-multierror — error wrapping |
| `golang.org/x/sys` | `v0.44.0` | low-level syscalls (term/isatty) |

> No heavy TUI runtime: the spinner/progress bar are hand-rolled
> (goroutine + carriage-return repaint), so the dependency set stays small.

## Claude Code permissions

This template ships a committed `.claude/settings.json` so that routine Go work
is **pre-approved** and the agent doesn't interrupt to ask. It allow-lists the
common read/build/test commands:

```
go build / test / vet / run / fmt / generate / list / env / version
go mod tidy / download / verify / why
gofmt · golangci-lint run
make build / test / lint / run / tidy / schema / all
```

These are all recoverable, non-destructive operations. Anything outside the list
still prompts. The model is layered:

- **`~/.claude/settings.json`** (user, global) — read-only utilities, git reads,
  recoverable mutations you trust everywhere.
- **`<repo>/.claude/settings.json`** (committed, shared) — this file: per-project
  build/test/inspect commands, shared by everyone who clones the repo.
- **`<repo>/.claude/settings.local.json`** (gitignored) — your personal/machine
  overrides; never committed (see `.gitignore`).

The full rationale — rule syntax, precedence (`deny → ask → allow`),
`defaultMode`, secret protection, and per-language allow lists — is in
[`docs/claude-permissions-guide.md`](docs/claude-permissions-guide.md).

## Continuous integration

`.github/workflows/ci.yml` runs on every push to `main` and every PR:

- **lint** — `golangci-lint` (config in `.golangci.yml`).
- **test** — `go test -race -cover ./...`.
- **schema contract** — `go run . schema` must emit valid JSON (guards against
  the machine surface drifting from the grammar).
- **build matrix** — cross-compiles `linux/{amd64,arm64}`, `darwin/arm64`,
  `windows/amd64` with `CGO_ENABLED=0`.

> Note: `.golangci.yml` uses the golangci-lint **v1** config schema. If your CI
> resolves `golangci-lint@latest` to **v2.x**, either pin a v1 release in the
> action or migrate the config (`golangci-lint migrate`).

## License

**Apache-2.0** — see [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE). Go binaries
built from this template are Apache-2.0.
