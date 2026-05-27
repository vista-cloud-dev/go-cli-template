# go-cli-template

**The shared Go-CLI scaffold for the m-cli Go toolchain.** Stage 0.1 of the
[implementation plan](../vista-dev-bridge/docs/m-cli-go-toolchain-implementation-plan.md) —
**built first**, so every Go repo in the toolchain (`m-cli`, `irissync`, `vista-meta`,
`kids-vc`, `m-dev-tools-mcp`) shares **one** command grammar, output contract, error/exit
ladder, and look-and-feel. Conventions: [`m-cli-go-toolchain-spec.md`](../vista-dev-bridge/docs/m-cli-go-toolchain-spec.md) §5 / §5.5.

This repo is also a **runnable reference CLI** (`hello`) that exercises every feature, so the
look-and-feel can be tuned here before any real tool is built on it.

## What it gives every CLI (the `clikit` package)

- **Kong** command grammar — the whole surface is one typed struct (the single source of truth).
- **TTY-gated [Lipgloss](https://github.com/charmbracelet/lipgloss) styling** — styled help/errors/tables on a terminal; **plain or JSON when piped**. Never blocks scripts or agents.
- **`--output {text|json|auto}`** — `auto` (default) = styled text on a TTY, JSON when piped/redirected.
- **A versioned JSON envelope** (`schemaVersion`/`command`/`ok`/`exit`/`data`/`diagnostics`/`error`).
- **Deterministic error objects + the exit-code ladder** — `0` ok · `1` runtime · `2` usage · `3` `--check`/findings · `4` engine-bound op refused.
- **`schema`** — reflects the Kong struct into the machine surface (so it can't drift from `--help`).
- **`version`**, **shell completions** (kongplete), styled help.

## Try it (tune the look-and-feel)

```sh
go run . greet Ada --greeting howdy --repeat 2   # styled on a TTY
go run . greet Ada -o json                        # the JSON envelope
go run . demo table                               # styled box table (TTY) / JSON rows (piped)
go run . demo diagnostics -o json                 # the diagnostics envelope (lint-style)
go run . demo fail --code 4; echo "exit=$?"       # deterministic error → exit 4
go run . schema | jq .                            # the machine surface (agent discovery)
go run . --help
NO_COLOR=1 go run . greet Ada -o text             # styling off
```

Tune colours/labels/layout in `clikit/context.go` (the `theme`) — every tool inherits the change.

## Build (static, reproducible — spec §10)

```sh
make build          # dist/hello, CGO_ENABLED=0 -trimpath, version-stamped
make dist           # cross-compile: linux/{amd64,arm64}, darwin/arm64, windows/amd64
make lint test schema
```

## Bootstrap a new toolchain repo

Two supported paths (the conventions stay single-sourced):

1. **Import `clikit`** (recommended) — `import "github.com/vista-cloud-dev/go-cli-template/clikit"`, define
   your command struct embedding `clikit.Globals`, and call `clikit.Run(...)` from `main`. Convention
   updates arrive via a `go get -u` bump.
2. **Scaffold-copy** — `gonew github.com/vista-cloud-dev/go-cli-template github.com/vista-cloud-dev/<repo>`,
   then replace the `hello` demo commands with the real ones and keep `clikit`, the Makefile, CI, and
   `.golangci.yml`.

The structure to keep: `clikit/` (conventions) · `main.go` (your command grammar) · `Makefile` ·
`.golangci.yml` · `.github/workflows/ci.yml`.

## License

**Apache-2.0** (Go binaries are Apache; the toolchain's VS Code extensions are MIT — the per-tier split in
[host-side-go-toolchain-adr.md](../vista-dev-bridge/docs/host-side-go-toolchain-adr.md) §3.2).
