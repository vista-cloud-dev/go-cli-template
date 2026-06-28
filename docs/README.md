# docs/ — documentation index

This repo follows the vista-cloud-dev standard `docs/` layout, so a tool forked
from this template is **born correct**. Use the same small folder vocabulary —
do not invent per-repo folders (`tracking/`, `plans/`, `prompts/`, `historical/`).

```
docs/
  README.md   # this index — the one navigation entry point
  guides/     # how-to for users of this tool (optional)
  design/     # this repo's own design notes / ADRs (optional)
  memory/     # auto-memory — DURABLE facts only (created when first written)
  archive/    # retired docs from THIS repo — git mv'd, never deleted
```

`modules/` (generated reference) is **stdlib repos only** — not used here.

Trackers for live work sit in `docs/` root as `<effort>-tracker.md` and move to
`docs/archive/` when the effort lands (they are Tier-D live status, not canon).

## Key docs

- [`guides/claude-code-permissions-guide.md`](guides/claude-code-permissions-guide.md)
  — two-layer Claude Code permissions blueprint + copy-paste templates.
