# Claude Code Permissions Guide

A data-driven plan for cutting repetitive permission prompts without losing safety,
derived from an audit of your actual `.claude` configs and your permission-granting
history across `~/m-dev-tools` and `~/vista-cloud-dev`.

- **Generated:** 2026-05-27
- **Evidence base:** all `.claude/settings*.json` in both project roots + 63 session
  transcripts in `~/.claude/projects/` (1,353 Bash calls and ~2,180 total tool uses).
- **Verified against:** the official Claude Code permission docs
  ([permissions](https://code.claude.com/docs/en/permissions.md),
  [permission-modes](https://code.claude.com/docs/en/permission-modes.md),
  [settings](https://code.claude.com/docs/en/settings.md)).

> **TL;DR** — Most of your prompt fatigue is **read-only commands** (≈56% of all Bash
> calls are `ls`/`cat`/`grep`/`find`/read-only `git`). Pre-approve those once at the
> **user level** so they stop prompting in *every* repo. Keep a small **always-ask**
> list for destructive/outward-facing actions, and a hard **deny** list for the
> irreversible ones. Three concrete fixes you can do today are in
> [§9 Migration steps](#9-migration-steps) — including a setting in `vista-dev-bridge`
> that is currently a **no-op**.

---

## Contents

1. [How permissions actually work (the rules that justify this plan)](#1-how-permissions-actually-work)
2. [What your history shows](#2-what-your-history-shows)
3. [Audit of your current configs (findings)](#3-audit-of-your-current-configs)
4. [The plan: three layers](#4-the-plan-three-layers)
5. [The Allow list (pre-approved, safe, high-frequency)](#5-the-allow-list)
6. [The Always-Ask list (confirm every time)](#6-the-always-ask-list)
7. [The Deny list (never) + secret protection](#7-the-deny-list-and-secret-protection)
8. [Caveats — the limits of Bash allow-listing](#8-caveats)
9. [Migration steps](#9-migration-steps)
10. [Appendix: copy-paste configs & raw frequency data](#appendix)

---

## 1. How permissions actually work

Five facts drive every recommendation below. All are from the official docs.

### 1.1 Rule syntax

A rule is `Tool(specifier)`. For Bash the specifier is a command pattern:

| Form | Matches |
|---|---|
| `Bash(npm test)` | **only** the exact string `npm test` |
| `Bash(npm test:*)` | `npm test` **and** `npm test -- --watch` (bare command *and* any args) |
| `Bash(npm test *)` | `npm test --watch` but **not** bare `npm test` (space-`*` requires a trailing arg) |

**This guide uses the `:*` (colon-star) form everywhere**, because it matches the
command with *or without* arguments — exactly what you want for an allow rule. The
space-`*` form (`npm test *`) is what Claude Code writes when you click *"Yes, don't
ask again"*; it's equivalent for the with-arguments case but misses the bare command.

> Prefix rules are literal from the **start** of the command. `git log` and
> `git --no-pager log` are *different* prefixes — each needs its own rule. You use
> `git --no-pager …` a lot, so both variants are included below.

### 1.2 Chained commands can't smuggle past the rules ✅

Claude Code splits compound commands on `&&`, `||`, `;`, `|`, `|&`, `&`, and newlines,
then checks **each sub-command independently**. So if `ls:*` is allowed but `rm` is
not, `ls && rm x` still **prompts** — the `rm x` part isn't covered. This is the
property that makes a broad read-only allow list safe: an allowed prefix can't be used
as a wrapper to sneak a disallowed command through.

### 1.3 Precedence: deny → ask → allow, first match wins

For any action, Claude Code checks **deny** rules, then **ask**, then **allow**, and
stops at the first match. So:

- **Deny always beats allow.** Nothing can override a deny.
- **Ask always beats allow.** This lets you `allow` a broad prefix and `ask` on a
  dangerous sub-case — e.g. allow `git push:*` but ask on `git push --force:*`.

Rules from all settings files are **merged additively**. File precedence (high→low):
managed → command-line → `.claude/settings.local.json` → `.claude/settings.json` →
`~/.claude/settings.json`. But because deny/ask/allow precedence applies across the
*merged* set, a deny in your user settings still beats an allow in a project file.

### 1.4 `defaultMode` — the baseline behavior

| Mode | What it does |
|---|---|
| `default` | Prompt on first use of each tool. **Recommended** with a good allow list. |
| `acceptEdits` | Auto-approves edits + common fs commands (`mkdir`,`touch`,`mv`,`cp`,`rm`,`rmdir`,`sed`) within the working dir. |
| `plan` | Read-only exploration; no source edits. |
| `auto` | Auto-approve with a background safety classifier. **Requires Opus 4.6+/Sonnet 4.6+, and must live in `~/.claude/settings.json`** — a repo cannot grant itself `auto`. |
| `dontAsk` | Non-interactive: auto-*denies* everything except `permissions.allow` and read-only Bash. |
| `bypassPermissions` | Skips all prompts (except an `rm -rf /` / `rm -rf ~` circuit breaker). Isolated/throwaway envs only. |

### 1.5 Path rules for `Read`/`Edit`/`Write`

Gitignore-style globs with anchored roots:

| Pattern | Resolves to |
|---|---|
| `Read(//Users/rafael/x)` | the **absolute** path `/Users/rafael/x` (note the **double** slash) |
| `Read(/docs/**)` | `<project-root>/docs/**` (single slash = project root, *not* filesystem root) |
| `Read(~/.ssh/**)` | home dir |
| `Read(src/**)` / `Read(./src/**)` | `<cwd>/src/**` |
| `Read(.env)` | a file named `.env` at any depth (bare name ≡ `**/.env`) |

For **deny**, a rule matches if **either** a symlink **or** its target matches (so you
can't dodge a secret-file deny via a symlink).

---

## 2. What your history shows

Across 63 sessions, tool usage was:

```
1353  Bash      668  Edit      594  Read      201  Write
 122  WebSearch  78  WebFetch   66  TodoWrite  34  AskUserQuestion
```

Bash dominates the prompts. Breaking the 1,353 Bash calls down by what they actually
do (after stripping `cd … &&` wrappers):

```
 ~56%  read-only utilities   ls, cat, grep, find, head, tail, echo, which, wc, sort…
 large read-only git         status(156) log(101) diff(63) branch(38) rev-parse(30) show(24) ls-files(15)
 mutating-but-recoverable    git add(93) git commit(90) git checkout(23) mkdir(12) chmod(8)
 build/test/inspect          npm run(42) go build(18) make build(21) go test(8) go vet(7) pytest, ruff…
 container (dev sandbox)      podman exec(152) ps(29) images(29) run(30) cp(12) inspect, logs…
 outward / destructive        git push(74) gh api(42) rm(16) podman rm(22)/prune, brew install, kill(18)
```

**The headline:** more than half of every permission interruption is a command that
only *reads*. These are the prime candidates for a one-time, user-level pre-approval.
(Full tables in the [Appendix](#appendix).)

---

## 3. Audit of your current configs

What's in place today, and what to fix:

| # | Finding | Impact | Fix |
|---|---|---|---|
| 1 | **`vista-dev-bridge/.claude/settings.json` sets `"defaultMode": "auto"`** at the *shared-project* level. | **No-op.** Claude Code deliberately ignores a repo granting itself `auto`; only `~/.claude/settings.json` is honored (and only on Opus 4.6+/Sonnet 4.6+). You think you're in auto mode there but you aren't. | Decide deliberately: put `auto` in `~/.claude/settings.json` to get it everywhere, or drop the dead line. |
| 2 | **Two opposite models coexist.** `vista-iris` allows bare `"Bash"` (= *all* shell) and gates via ask/deny; `tree-sitter-m` uses `dontAsk` + deny (non-interactive). Others accumulate ad-hoc allow lists. | Inconsistent and hard to reason about. Bare `"Bash"` means anything *not* in your ask/deny lists runs silently — including `curl … \| sh`, `sudo …`, etc. | Standardize on the layered **allow + ask + deny** model (§4). Reserve `dontAsk` for genuinely sandboxed repos. |
| 3 | **`settings.local.json` files are full of one-shot noise** — e.g. a giant `printf` ObjectScript blob, an `awk '/Enterprise Search/…DPTLK7.m'`, specific `…/tasks/<id>.output` paths. These came from "always allow" saving up to 5 literal sub-command rules. | Clutter; they'll never match again. | Periodically prune local files to the genuinely reusable rules. |
| 4 | **Duplication across repos.** `WebSearch`, `WebFetch(domain:github.com)`, `git add *`, `git commit *`, `git push *` appear in 3–4 separate local files. | You re-grant the same things per repo. | Promote them **once** to `~/.claude/settings.json`; delete the copies. |
| 5 | **`go-cli-template/.gitignore` has no `.claude` entry**, so `settings.local.json` (personal/machine rules) would be committed. | Personal paths leak into the shared repo. | Add `.claude/settings.local.json` to `.gitignore`; keep `.claude/settings.json` committed for shared rules. |
| 6 | Your destructive **ask** list in `vista-iris` (rm/prune/clean) is genuinely good. | — | Promote it to user level so every repo inherits it (§6). |

---

## 4. The plan: three layers

Put each rule at the **broadest scope where it's still safe**, so you grant it once.

```
┌─ ~/.claude/settings.json  (USER — applies to every repo) ───────────────┐
│  • allow: read-only shell + read-only git + WebSearch + trusted domains │
│  • ask  : rm / prune / push --force / sudo / curl / installs (universal)│
│  • deny : catastrophic rm + secret files (never)                        │
│  • optional: defaultMode "auto" (if you want it; only works here)       │
├─ <repo>/.claude/settings.json  (SHARED, committed — per project) ───────┤
│  • allow: this stack's build/test/lint (go / node / python / podman)    │
│  • ask  : this project's destructive targets (make clean, mctl recreate)│
├─ <repo>/.claude/settings.local.json  (LOCAL, gitignored — personal) ────┤
│  • machine-specific one-offs; let the "always allow" button fill it     │
└─────────────────────────────────────────────────────────────────────────┘
```

Keep `defaultMode: "default"` (prompt) as the baseline. A strong allow list + this
prompt baseline gives you near-zero friction on safe work while still surfacing
anything new. Copy-paste blocks for all three layers are in the [Appendix](#appendix).

---

## 5. The Allow list

Pre-approve these — they're high-frequency and safe (read-only, or recoverable and
local). Rationale and observed frequency included.

### 5.1 Read-only shell utilities → **user level** (`~/.claude/settings.json`)

These are ~56% of your Bash prompts and only read/print.

| Rule | Why safe / freq |
|---|---|
| `Bash(ls:*)` `Bash(cat:*)` `Bash(head:*)` `Bash(tail:*)` `Bash(wc:*)` | read-only; ls(107) cat(43) head(11) tail(6) |
| `Bash(grep:*)` `Bash(rg:*)` `Bash(find:*)` `Bash(tree:*)` | search; grep(89) find(42) — see find caveat in §8 |
| `Bash(echo:*)` `Bash(printf:*)` `Bash(pwd)` `Bash(date:*)` | echo(420!) is almost all `echo "--- exit $? ---"` diagnostics |
| `Bash(which:*)` `Bash(command -v:*)` `Bash(type:*)` `Bash(file:*)` `Bash(stat:*)` | environment probing |
| `Bash(sort:*)` `Bash(uniq:*)` `Bash(cut:*)` `Bash(tr:*)` `Bash(column:*)` `Bash(jq:*)` `Bash(awk:*)` | text shaping; jq/awk pipelines |
| `Bash(diff:*)` `Bash(comm:*)` `Bash(realpath:*)` `Bash(dirname:*)` `Bash(basename:*)` `Bash(du:*)` `Bash(df:*)` | |

### 5.2 Read-only git → **user level**

| Rule | Freq |
|---|---|
| `Bash(git status:*)` `Bash(git --no-pager status:*)` | 156 |
| `Bash(git log:*)` `Bash(git --no-pager log:*)` | 101 |
| `Bash(git diff:*)` `Bash(git --no-pager diff:*)` | 63 |
| `Bash(git show:*)` `Bash(git --no-pager show:*)` | 24 |
| `Bash(git branch:*)` | 38 (note: `-D` deletes; recoverable via reflog) |
| `Bash(git rev-parse:*)` `Bash(git rev-list:*)` `Bash(git ls-files:*)` `Bash(git ls-remote:*)` | 30 / — / 15 |
| `Bash(git remote -v)` `Bash(git remote get-url:*)` `Bash(git describe:*)` `Bash(git blame:*)` `Bash(git shortlog:*)` `Bash(git tag -l:*)` | read-only |
| `Bash(git fetch:*)` `Bash(git check-ignore:*)` `Bash(git stash list:*)` `Bash(git config --get:*)` | fetch only downloads refs |

### 5.3 Web + skills → **user level**

| Rule | Why |
|---|---|
| `WebSearch` | read-only; you use it 122× |
| `WebFetch(domain:github.com)` `WebFetch(domain:raw.githubusercontent.com)` | already trusted across repos |
| `WebFetch(domain:pkg.go.dev)` `WebFetch(domain:code.claude.com)` `WebFetch(domain:docs.claude.com)` | docs you'll fetch repeatedly |
| `WebFetch(domain:docs.intersystems.com)` `WebFetch(domain:community.intersystems.com)` `WebFetch(domain:hub.docker.com)` | from your vista configs |
| `Skill(update-config)` | you've used it; it edits your own settings |

> Arbitrary `WebFetch(domain:*)` stays **prompted** — fetching attacker-controlled
> URLs is a prompt-injection vector, so only allow-list domains you trust.

### 5.4 Recoverable local file ops → **user level**

| Rule | Why safe |
|---|---|
| `Bash(mkdir:*)` `Bash(touch:*)` | create only; can't destroy data — freq mkdir(12) |
| `Bash(chmod +x:*)` | mark scripts executable; freq(8) |

### 5.5 Per-project build/test/inspect → **shared `.claude/settings.json`** (committed)

Pick the block(s) for each repo's stack. These compile/test/inspect; they don't deploy
or delete. (`make`/`npm run`/`node`/`python`/`uv run` execute *your repo's own code* —
fine for your own projects; see §8.) The inventory below is the rationale; **complete
copy-paste `.claude/settings.json` files for each stack are in
[Appendix B–G](#b-go-projects-go-cli-template-other-go-repos)** (Go, Node/TS, Python,
Containers read-only, Containers dev sandbox, Project CLIs).

**Go** (`go-cli-template`, and other Go repos):
```
Bash(go build:*)  Bash(go test:*)  Bash(go vet:*)  Bash(go run:*)  Bash(go fmt:*)
Bash(gofmt:*)  Bash(go generate:*)  Bash(go list:*)  Bash(go env:*)  Bash(go version)
Bash(go mod tidy)  Bash(go mod download)  Bash(go mod verify)  Bash(go mod why:*)
Bash(golangci-lint run:*)
Bash(make build:*)  Bash(make test:*)  Bash(make lint:*)  Bash(make run:*)
Bash(make tidy)  Bash(make schema:*)  Bash(make all)
```

**Node / TypeScript** (`vista-dev-bridge`):
```
Bash(npm run:*)  Bash(npm test:*)  Bash(npm ci)  Bash(node:*)  Bash(npx tsc:*)  Bash(tsc:*)
```

**Python** (`m-cli`):
```
Bash(pytest:*)  Bash(python3 -m pytest:*)  Bash(python3 -m py_compile:*)
Bash(ruff check:*)  Bash(ruff format --check:*)  Bash(mypy:*)  Bash(uv run:*)
```

**Containers — read-only** (`vista-iris`):
```
Bash(podman ps:*)  Bash(podman images:*)  Bash(podman image ls:*)  Bash(podman inspect:*)
Bash(podman info:*)  Bash(podman logs:*)  Bash(podman version)  Bash(podman --version)
Bash(podman machine list)  Bash(podman machine info:*)
Bash(docker ps:*)  Bash(docker images:*)  Bash(docker info:*)  Bash(docker version)
Bash(docker logs:*)  Bash(docker context ls)  Bash(docker context show)
```

**Containers — dev-sandbox actions** (allow with eyes open; the container is
disposable — see §8):
```
Bash(podman exec:*)   # 152 calls — runs cmds inside the ephemeral dev container
Bash(podman run:*)  Bash(podman cp:*)  Bash(podman build:*)  Bash(podman compose:*)
```

**Your own project CLIs** (mostly read/analysis tools):
```
Bash(vista:*)          # vista doc/doctor/list/search/matrix/where/risk… (vista init writes)
Bash(mctl doctor:*)  Bash(mctl status:*)  Bash(mctl version:*)  Bash(mctl vista:*)  Bash(mctl exec:*)
Bash(m lint:*)  Bash(claude-status:*)  Bash(git-update-repos:*)
Bash(make verify:*)  Bash(make check:*)  Bash(make preflight:*)  Bash(make help:*)
Bash(make gen-check:*)  Bash(make license:*)  Bash(make -n:*)
```

### 5.6 Recoverable mutations → **user level** (`~/.claude/settings.json`)

These change local state but are easily recovered (git history / working tree). You
already allow `git add`/`git commit`/`git checkout` in several repos, so they are
**promoted to the user level** — they live in the [Appendix A](#a-user-level--claudesettingsjson)
baseline below, not in per-repo local files:

| Rule | Note |
|---|---|
| `Bash(git add:*)` | freq 93; staging only |
| `Bash(git commit:*)` | freq 90; local, am//revertable |
| `Bash(git mv:*)` | tracked rename; recoverable |
| `Bash(git stash:*)` `Bash(git restore --staged:*)` | `git restore <file>` discards edits — keep the bare form prompted |
| `Bash(git checkout:*)` | freq 23 — **caveat:** `git checkout -- <file>` discards uncommitted edits. Prefer `git switch` for branches; allow only if comfortable. |
| `Bash(git push:*)` | freq 74; you already allow it everywhere. Pair with the **ask** rule on `--force` below. |
| `Bash(gh api:*)` | freq 42; allowed at user level per your request — see the trade-off note in §6.1. |

---

## 6. The Always-Ask list

`ask` rules **override allow** (§1.3), so they're how you allow a broad prefix while
still confirming the dangerous variant. Put the universal ones at **user level**.

### 6.1 Universal (→ `~/.claude/settings.json`)

**Filesystem destruction**
```
Bash(rm:*)  Bash(rmdir:*)  Bash(git rm:*)  Bash(git clean:*)
```

**Git history / force-push** (these pair with allowing `git push:*` and `git branch:*`)
```
Bash(git push --force:*)  Bash(git push -f:*)  Bash(git push --force-with-lease:*)
Bash(git reset --hard:*)  Bash(git rebase:*)  Bash(git merge:*)  Bash(git filter-branch:*)
Bash(git branch -D:*)  Bash(git tag -d:*)  Bash(git reflog expire:*)  Bash(git gc:*)
```

**Privilege / processes / perms**
```
Bash(sudo:*)  Bash(kill:*)  Bash(pkill:*)  Bash(killall:*)  Bash(chmod -R:*)  Bash(chown:*)
```

**Network fetch & pipe-to-shell** (can download/run code or exfiltrate)
```
Bash(curl:*)  Bash(wget:*)
```
> Tip: in a project that hits a *local* API a lot, allow just the loopback form there:
> `Bash(curl http://localhost:*)` / `Bash(curl http://127.0.0.1:*)`.

**Software installs (supply-chain surface)**
```
Bash(brew install:*)  Bash(brew tap:*)  Bash(brew uninstall:*)
Bash(npm install:*)  Bash(npm i:*)  Bash(npm uninstall:*)
Bash(pip install:*)  Bash(pip3 install:*)  Bash(uv pip install:*)  Bash(uv add:*)
Bash(go install:*)  Bash(go get:*)  Bash(cargo install:*)  Bash(gem install:*)  Bash(pipx install:*)
```

**Outward-facing GitHub mutations**
```
Bash(gh pr create:*)  Bash(gh pr merge:*)  Bash(gh pr close:*)
Bash(gh repo create:*)  Bash(gh repo delete:*)
Bash(gh release create:*)  Bash(gh release delete:*)  Bash(gh secret:*)
```

> **`gh api` — allowed at user level (your choice).** You ran it 42× (mostly GET), so
> `Bash(gh api:*)` is in the [Appendix A](#a-user-level--claudesettingsjson) **allow**
> list. Trade-off to accept: this also auto-approves `gh api -X POST/DELETE …`, which
> can create/delete repos or merge PRs — a path that bypasses the `gh repo delete` /
> `gh pr merge` ask rules above. It is deliberately *not* also in the ask list (a single
> rule can't be both allow and ask, and ask would override allow).

### 6.2 Per-project destructive (→ that repo's `.claude/settings.json`)

Promote your existing `vista-iris` ask list, plus project-specific destroyers:
```
# containers
Bash(podman rm:*)  Bash(podman rmi:*)  Bash(podman image rm:*)  Bash(podman image prune:*)
Bash(podman container prune:*)  Bash(podman system prune:*)  Bash(podman builder prune:*)
Bash(podman volume rm:*)  Bash(podman volume prune:*)  Bash(podman network prune:*)
Bash(podman machine rm:*)  Bash(podman machine stop:*)
Bash(docker rm:*)  Bash(docker rmi:*)  Bash(docker system prune:*)
Bash(docker image prune:*)  Bash(docker volume prune:*)
# project lifecycle (recreates/wipes environments)
Bash(make clean:*)  Bash(make fresh:*)  Bash(make trim:*)
Bash(mctl bootstrap:*)  Bash(mctl recreate:*)  Bash(mctl restart:*)
```

---

## 7. The Deny list and secret protection

Deny is the **hard floor** — it can't be overridden by any allow, and it catches the
`rm` part of `safe && rm -rf /` because of sub-command splitting (§1.2). Put it at
**user level**.

### 7.1 Catastrophic / irreversible
```
Bash(rm -rf /)  Bash(rm -rf /*)  Bash(rm -rf ~)  Bash(rm -rf ~/*)
Bash(rm -rf $HOME)  Bash(rm -rf $HOME/*)  Bash(rm --no-preserve-root:*)
Bash(sudo rm:*)  Bash(dd:*)  Bash(mkfs:*)  Bash(mkfs.*:*)  Bash(:(){:|:&};:)
```
(Everyday `rm` is in **ask** above — these denies are only the unrecoverable extremes.)

### 7.2 Secret protection (recommended add — you don't have this yet)

Stops Claude from reading credentials it could then leak via a web tool or paste. Deny
can't be overridden, and matches a file via symlink-or-target — so this is a real lock.
```
Read(**/.env*)
Read(**/*.pem)  Read(**/*.key)  Read(**/*.p12)
Read(**/id_rsa)  Read(**/id_ed25519)  Read(**/.npmrc)  Read(**/.pgpass)
Read(~/.ssh/**)  Read(~/.aws/**)  Read(~/.config/gcloud/**)
Read(**/secrets/**)  Read(**/credentials)
```
> **Broadened to `Read(**/.env*)` per your call.** This blocks *every* dotenv variant —
> `.env`, `.env.local`, `.env.production`, **and** the otherwise-harmless `.env.example`
> / `.env.sample` templates. Since deny can't be overridden by an allow, the consequence
> is real: if you ever need Claude to read a template, either rename it without the
> leading dot (e.g. `env.example`) or temporarily remove this rule. Add
> `Edit(**/.env*)` / `Write(**/.env*)` if you also want to block creating/clobbering
> dotenv files (note: that also stops Claude from helping you scaffold a new `.env`).

---

## 8. Caveats

Bash allow-listing is prefix matching, not semantic analysis. Be aware:

- **In-place / escape-hatch flags.** A prefix allow covers a command's *destructive*
  modes too: `find … -delete`/`-exec rm`, `sed -i` (in-place edit), `awk … > file`,
  `cp` overwrites. They're in §5 for frequency, but the safety net is the **deny** list
  (irreversible cases) + the fact that you're reviewing output. If you want to be
  stricter, drop `find`/`awk` from the allow list and accept the occasional prompt.
- **Redirection isn't analyzed.** `Bash(echo:*)` matches `echo x > important.txt` —
  the rule sees `echo …`, not the `>`-clobber. Same for any allowed command. This is
  why §7.2 protects secret *files* explicitly.
- **Sub-command splitting is the guardrail (§1.2),** but it operates on shell operators
  it recognizes; deeply obfuscated command substitution can still surprise it. Treat
  the allow list as "remove friction on the obviously-safe," and rely on **deny/ask**
  for the genuinely dangerous — never the reverse.
- **`make`/`npm run`/`node`/`python`/`uv run` run arbitrary code** from the project.
  Allowing them = trusting *that repo's* scripts. That's appropriate for your own repos;
  reconsider before allow-listing them in third-party clones.
- **`auto` mode** (`defaultMode: "auto"`) adds a background classifier on top of these
  rules, but only from `~/.claude/settings.json` and only on Opus 4.6+/Sonnet 4.6+. It
  complements, not replaces, the deny list.

---

## 9. Migration steps

Do these in order. Steps 1–3 are the high-value fixes; each lists the exact file and
action.

1. **Create the user baseline.**
   - **File:** `~/.claude/settings.json` (currently `{}`).
   - **Action:** replace its contents with [Appendix A](#a-user-level--claudesettingsjson) verbatim.
   - **Effect, in every repo:** read-only shell + read-only git, the recoverable git
     mutations from §5.6 (`git add/commit/checkout/mv/stash/push`), `WebSearch` +
     trusted `WebFetch` domains, `Bash(gh api:*)` (your choice — §6.1 trade-off), the
     full universal **ask** list (§6.1), and the **deny** floor incl. the broadened
     `Read(**/.env*)` secret guard (§7).
   - Afterward, run the `/fewer-permission-prompts` skill to top up from recent
     transcripts, or let the *"always allow"* button add stragglers to local files.

2. **Resolve the dead `auto` mode (Finding #1).**
   - **File:** `vista-cloud-dev/vista-dev-bridge/.claude/settings.json`.
   - **Action:** that file's `"defaultMode": "auto"` is silently ignored (a repo can't
     grant itself `auto`). Either **delete** the `"defaultMode": "auto"` line, **or** — if
     you actually want auto mode everywhere — add `"permissions": { "defaultMode": "auto" }`
     to `~/.claude/settings.json` (needs Opus 4.6+/Sonnet 4.6+).

3. **Gitignore local settings (Finding #5).**
   - **File:** `go-cli-template/.gitignore` (and any repo missing the entry).
   - **Action:** append:
     ```gitignore
     # Claude Code — personal/machine-local permissions
     .claude/settings.local.json
     ```
   - Keep `.claude/settings.json` committed so the team shares the project allow list.

4. **Add per-project shared rules** — paste the matching standalone file into each
   repo's `.claude/settings.json`:

   | Repo | Paste | Notes |
   |---|---|---|
   | `go-cli-template` (+ other Go) | [Appendix B](#b-go-projects-go-cli-template-other-go-repos) | as-is |
   | `vista-dev-bridge` | [Appendix C](#c-node--typescript-projects-vista-dev-bridge) | drop `"defaultMode": "auto"` here (step 2) |
   | `m-cli` | [Appendix D](#d-python-projects-m-cli) + merge [G](#g-project-clis-vista-mctl-m--eg-vista-iris-mctl-m-cli) `allow` | Python that also drives `m`/`vista` |
   | `vista-iris` | [Appendix F](#f-containers--dev-sandbox-vista-iris) + merge [G](#g-project-clis-vista-mctl-m--eg-vista-iris-mctl-m-cli) | see step 6 about its bare `"Bash"` |
   | `mctl` | [Appendix G](#g-project-clis-vista-mctl-m--eg-vista-iris-mctl-m-cli) | adds `mctl …` ask gates |
   | `tree-sitter-m` | leave as-is | its `dontAsk` + deny model is a fine sandbox |

5. **Dedupe & prune (Findings #3, #4).** In each `.claude/settings.local.json`, **delete**
   the rules now covered by the user baseline so they live in one place:
   - `WebSearch`, `WebFetch(domain:github.com)`, `WebFetch(domain:raw.githubusercontent.com)`
   - `git add *`, `git commit *`, `git push *`, and any read-only git (`git status/log/diff`)
   - the one-shot noise: the giant `printf '…ObjectScript…'`, `awk '/Enterprise Search/…DPTLK7.m'`,
     long `grep -nE '…'` literals, and `…/tasks/<id>.output` paths — these will never match again.

6. **Reconcile the broad-`Bash` repo (Finding #2).**
   - **File:** `vista-iris/.claude/settings.json` currently allows bare `"Bash"` (= every
     shell command runs silently unless caught by its ask/deny).
   - **Action:** replace the bare `"Bash"` entry with the explicit Appendix F + G `allow`
     lists, so anything unanticipated **prompts** instead of running unseen. Keep its
     existing `ask`/`deny` — they're good and are reflected in Appendix F.

---

## Appendix

### A. User level — `~/.claude/settings.json`

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(ls:*)", "Bash(cat:*)", "Bash(head:*)", "Bash(tail:*)", "Bash(wc:*)",
      "Bash(grep:*)", "Bash(rg:*)", "Bash(find:*)", "Bash(tree:*)",
      "Bash(echo:*)", "Bash(printf:*)", "Bash(pwd)", "Bash(date:*)",
      "Bash(which:*)", "Bash(command -v:*)", "Bash(type:*)", "Bash(file:*)", "Bash(stat:*)",
      "Bash(sort:*)", "Bash(uniq:*)", "Bash(cut:*)", "Bash(tr:*)", "Bash(column:*)",
      "Bash(jq:*)", "Bash(awk:*)", "Bash(diff:*)", "Bash(comm:*)",
      "Bash(realpath:*)", "Bash(dirname:*)", "Bash(basename:*)", "Bash(du:*)", "Bash(df:*)",
      "Bash(mkdir:*)", "Bash(touch:*)", "Bash(chmod +x:*)",

      "Bash(git status:*)", "Bash(git --no-pager status:*)",
      "Bash(git log:*)", "Bash(git --no-pager log:*)",
      "Bash(git diff:*)", "Bash(git --no-pager diff:*)",
      "Bash(git show:*)", "Bash(git --no-pager show:*)",
      "Bash(git branch:*)", "Bash(git rev-parse:*)", "Bash(git rev-list:*)",
      "Bash(git ls-files:*)", "Bash(git ls-remote:*)", "Bash(git remote -v)",
      "Bash(git remote get-url:*)", "Bash(git describe:*)", "Bash(git blame:*)",
      "Bash(git shortlog:*)", "Bash(git tag -l:*)", "Bash(git fetch:*)",
      "Bash(git check-ignore:*)", "Bash(git stash list:*)", "Bash(git config --get:*)",

      "Bash(git add:*)", "Bash(git commit:*)", "Bash(git mv:*)",
      "Bash(git stash:*)", "Bash(git restore --staged:*)",
      "Bash(git checkout:*)", "Bash(git push:*)",

      "Bash(gh api:*)",

      "WebSearch",
      "WebFetch(domain:github.com)", "WebFetch(domain:raw.githubusercontent.com)",
      "WebFetch(domain:pkg.go.dev)", "WebFetch(domain:code.claude.com)",
      "WebFetch(domain:docs.claude.com)", "WebFetch(domain:docs.intersystems.com)",
      "WebFetch(domain:community.intersystems.com)", "WebFetch(domain:hub.docker.com)",
      "Skill(update-config)"
    ],
    "ask": [
      "Bash(rm:*)", "Bash(rmdir:*)", "Bash(git rm:*)", "Bash(git clean:*)",
      "Bash(git push --force:*)", "Bash(git push -f:*)", "Bash(git push --force-with-lease:*)",
      "Bash(git reset --hard:*)", "Bash(git rebase:*)", "Bash(git merge:*)",
      "Bash(git filter-branch:*)", "Bash(git branch -D:*)", "Bash(git tag -d:*)",
      "Bash(git reflog expire:*)", "Bash(git gc:*)",
      "Bash(sudo:*)", "Bash(kill:*)", "Bash(pkill:*)", "Bash(killall:*)",
      "Bash(chmod -R:*)", "Bash(chown:*)",
      "Bash(curl:*)", "Bash(wget:*)",
      "Bash(brew install:*)", "Bash(brew tap:*)", "Bash(brew uninstall:*)",
      "Bash(npm install:*)", "Bash(npm i:*)", "Bash(npm uninstall:*)",
      "Bash(pip install:*)", "Bash(pip3 install:*)", "Bash(uv pip install:*)", "Bash(uv add:*)",
      "Bash(go install:*)", "Bash(go get:*)", "Bash(cargo install:*)",
      "Bash(gem install:*)", "Bash(pipx install:*)",
      "Bash(gh pr create:*)", "Bash(gh pr merge:*)", "Bash(gh pr close:*)",
      "Bash(gh repo create:*)", "Bash(gh repo delete:*)",
      "Bash(gh release create:*)", "Bash(gh release delete:*)", "Bash(gh secret:*)"
    ],
    "deny": [
      "Bash(rm -rf /)", "Bash(rm -rf /*)", "Bash(rm -rf ~)", "Bash(rm -rf ~/*)",
      "Bash(rm -rf $HOME)", "Bash(rm -rf $HOME/*)", "Bash(rm --no-preserve-root:*)",
      "Bash(sudo rm:*)", "Bash(dd:*)", "Bash(mkfs:*)", "Bash(mkfs.*:*)",
      "Read(**/.env*)",
      "Read(**/*.pem)", "Read(**/*.key)", "Read(**/*.p12)",
      "Read(**/id_rsa)", "Read(**/id_ed25519)", "Read(**/.npmrc)", "Read(**/.pgpass)",
      "Read(~/.ssh/**)", "Read(~/.aws/**)", "Read(~/.config/gcloud/**)",
      "Read(**/secrets/**)", "Read(**/credentials)"
    ]
  }
}
```

Each block below is a **complete, standalone `.claude/settings.json`** — drop it into
the repo's `.claude/` directory (commit it; it's the shared/team layer). It assumes the
user baseline (Appendix A) is already in place, so it only adds what's specific to that
stack. Where a repo spans two types (e.g. `m-cli` is Python *and* drives the project
CLIs), merge the relevant `allow`/`ask` arrays.

### B. Go projects (`go-cli-template`, other Go repos)

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(go build:*)", "Bash(go test:*)", "Bash(go vet:*)", "Bash(go run:*)",
      "Bash(go fmt:*)", "Bash(gofmt:*)", "Bash(go generate:*)", "Bash(go list:*)",
      "Bash(go env:*)", "Bash(go version)", "Bash(go mod tidy)",
      "Bash(go mod download)", "Bash(go mod verify)", "Bash(go mod why:*)",
      "Bash(golangci-lint run:*)",
      "Bash(make build:*)", "Bash(make test:*)", "Bash(make lint:*)",
      "Bash(make run:*)", "Bash(make tidy)", "Bash(make schema:*)", "Bash(make all)"
    ]
  }
}
```

### C. Node / TypeScript projects (`vista-dev-bridge`)

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(npm run:*)", "Bash(npm test:*)", "Bash(npm ci)",
      "Bash(node:*)", "Bash(tsc:*)", "Bash(npx tsc:*)",
      "Bash(npx vitest:*)", "Bash(npx jest:*)",
      "Bash(npx eslint:*)", "Bash(npx prettier --check:*)"
    ]
  }
}
```
> `npm ci` installs from the committed lockfile (deterministic); `npm install`/`npm i`
> stays in the user **ask** list because it can change the lockfile and runs install
> scripts. `node:*` runs arbitrary JS — fine for your own repo (see §8).

### D. Python projects (`m-cli`)

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(pytest:*)", "Bash(python3 -m pytest:*)", "Bash(python3 -m py_compile:*)",
      "Bash(ruff check:*)", "Bash(ruff format --check:*)", "Bash(mypy:*)",
      "Bash(uv run:*)", "Bash(uv sync)",
      "Bash(make check)", "Bash(make docs-check)",
      "Bash(make lint-vista)", "Bash(make lint-modern)"
    ]
  }
}
```
> `m-cli` also drives the `m`/`vista` CLIs — if so, merge in the **Appendix G** allow
> entries (`Bash(m lint:*)`, `Bash(vista:*)`, …).

### E. Containers — read-only / inspection only

For a repo where you only ever *inspect* containers (never create/destroy them).

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(podman ps:*)", "Bash(podman images:*)", "Bash(podman image ls:*)",
      "Bash(podman inspect:*)", "Bash(podman info:*)", "Bash(podman logs:*)",
      "Bash(podman version)", "Bash(podman --version)",
      "Bash(podman machine list)", "Bash(podman machine info:*)",
      "Bash(docker ps:*)", "Bash(docker images:*)", "Bash(docker inspect:*)",
      "Bash(docker info:*)", "Bash(docker version)", "Bash(docker logs:*)",
      "Bash(docker context ls)", "Bash(docker context show)"
    ]
  }
}
```

### F. Containers — dev sandbox (`vista-iris`)

The realistic file for a repo whose container *is* the dev environment: read-only
(Appendix E) **plus** exec/run/cp/build into the disposable container, plus build/verify
targets — and the destructive container/lifecycle ops gated behind `ask`. Merge this
with your existing `vista-iris` `deny` block (the `rm -rf` circuit breakers); keep that.

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(podman ps:*)", "Bash(podman images:*)", "Bash(podman image ls:*)",
      "Bash(podman inspect:*)", "Bash(podman info:*)", "Bash(podman logs:*)",
      "Bash(podman version)", "Bash(podman --version)",
      "Bash(podman machine list)", "Bash(podman machine info:*)",
      "Bash(podman exec:*)", "Bash(podman run:*)", "Bash(podman cp:*)",
      "Bash(podman build:*)", "Bash(podman compose:*)", "Bash(podman pull:*)",
      "Bash(docker ps:*)", "Bash(docker images:*)", "Bash(docker info:*)",
      "Bash(docker version)", "Bash(docker logs:*)", "Bash(docker context ls)",
      "Bash(make build:*)", "Bash(make verify:*)", "Bash(make check:*)",
      "Bash(make preflight:*)", "Bash(make help:*)", "Bash(make -n:*)"
    ],
    "ask": [
      "Bash(podman rm:*)", "Bash(podman rmi:*)", "Bash(podman image rm:*)",
      "Bash(podman image prune:*)", "Bash(podman container prune:*)",
      "Bash(podman system prune:*)", "Bash(podman builder prune:*)",
      "Bash(podman volume rm:*)", "Bash(podman volume prune:*)",
      "Bash(podman network prune:*)", "Bash(podman machine rm:*)",
      "Bash(podman machine stop:*)",
      "Bash(docker rm:*)", "Bash(docker rmi:*)", "Bash(docker system prune:*)",
      "Bash(docker image prune:*)", "Bash(docker volume prune:*)",
      "Bash(make clean:*)", "Bash(make fresh:*)", "Bash(make trim:*)"
    ],
    "deny": [
      "Bash(rm -rf /)", "Bash(rm -rf /*)", "Bash(rm -rf ~)", "Bash(rm -rf $HOME)"
    ]
  }
}
```

### G. Project CLIs (`vista`, `mctl`, `m` — e.g. `vista-iris`, `mctl`, `m-cli`)

Your own tooling — mostly read/analysis, with environment-recreating verbs gated.

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(vista:*)",
      "Bash(mctl doctor:*)", "Bash(mctl status:*)", "Bash(mctl version:*)",
      "Bash(mctl vista:*)", "Bash(mctl exec:*)",
      "Bash(m lint:*)", "Bash(claude-status:*)", "Bash(git-update-repos:*)",
      "Bash(make verify:*)", "Bash(make check:*)", "Bash(make preflight:*)",
      "Bash(make help:*)", "Bash(make gen-check:*)", "Bash(make license:*)",
      "Bash(make -n:*)"
    ],
    "ask": [
      "Bash(mctl bootstrap:*)", "Bash(mctl recreate:*)", "Bash(mctl restart:*)"
    ]
  }
}
```
> `Bash(vista:*)` is broad because `vista` is read-heavy analysis tooling; note
> `vista init` writes files. Narrow it to the read subcommands
> (`vista doc/doctor/list/search/matrix/where/risk/snapshot`) if you'd rather gate `init`.

### H. Local (`<repo>/.claude/settings.local.json`, gitignored)

Leave this for the *"Yes, don't ask again"* button. Periodically prune one-shots.
Machine-specific reads (like the cross-project reads currently in
`go-cli-template/.claude/settings.local.json`) belong here.

### I. Raw frequency data

Tool usage across 63 transcripts:
```
1353 Bash | 668 Edit | 594 Read | 201 Write | 122 WebSearch | 78 WebFetch
 66 TodoWrite | 34 AskUserQuestion | 27 ToolSearch | 13 Agent | 5 Skill
```

Top Bash signatures (after stripping `cd …&&` wrappers; % of 1,353):
```
echo 420(31%)  ls 107(8%)  grep 89(7%)  git add 60  cat 43  find 42  podman exec 27
python3 25  git push 24  npm run 23  git status 21  kill 18  rm 16  node 15
git commit 13  curl 13  mkdir 12  make build 12  gh api 12  sed 12  git checkout 10
```

Per-tool sub-command counts (every occurrence, incl. inside compound commands):
```
git:   status156 log101 add93 commit90 push74 diff63 branch38 rev-parse30 show24
       checkout23 remote16 ls-files15 mv10 merge9 rm7
gh:    api42 pr18 repo11 auth5 release4 search1
go:    build18 test8 vet7 run6 generate4 mod4 get2 version2 env1
make:  build21 verify5 gen-check5 license4 fresh3 preflight3 help2 check2
podman:exec152 machine35 run30 ps29 images29 rm22 image15 cp12 commit6 builder6 system6
docker:context15 info3 ps3 version2 build/compose/run…
npm:   run42 install3 test1
vista: doc8 doctor6 snapshot3 risk3 init3 search2 routine2 matrix2 where2 list2
mctl:  doctor13 vista9 exec7 bootstrap4 status4 version2 recreate2 restart1
```
Read-only utilities = ~753 of 1,353 Bash calls (**56%**).
```
