# Convert gitle to a Bubble Tea UI

## Context

gitle today is a Cobra CLI (16 commands, ~2.5k lines) that prints through a
hand-rolled `internal/ui` package: raw ANSI escape codes, `bufio` reads on
stdin, and two bespoke raw-mode widgets (`Pick`, `Choose`) that redraw by
counting lines and emitting `\033[NA` cursor jumps. That code works, but it
owns problems that Bubble Tea solves for free: no resize handling, no
scrolling for long lists, no mouse, fragile cursor math, Windows raw-mode
gaps, and no way to render more than one thing at a time.

This plan moves gitle onto [Bubble Tea](https://github.com/charmbracelet/bubbletea)
in two directions at once, as agreed:

1. **Prompts** — `ui.Ask` / `ui.Confirm` / `ui.Pick` / `ui.Choose` are
   re-implemented on `huh`, keeping their exact signatures so the ~181
   existing call sites stay untouched.
2. **Dashboard** — bare `gitle` (no subcommand) launches a persistent
   full-screen TUI: status, changes, branches, history, conflicts, all
   driveable without typing a command.

**Non-interactive use keeps working.** Piped, redirected, CI, and
fully-flagged invocations (`gitle save --all "msg"`) never start a Bubble
Tea program and still emit plain lines. This constraint shapes the whole
design and is why Phase 3 exists.

## Dependencies

The Charm v2 stack is released and lives under the `charm.land` module path
(v1 remains at `github.com/charmbracelet/*`). Use v2 throughout — mixing v1
and v2 lipgloss in one binary means two independent colour profiles.

```
charm.land/bubbletea/v2  v2.0.8   // runtime
charm.land/bubbles/v2    v2.1.1   // list, textinput, viewport, spinner, table, help, key
charm.land/lipgloss/v2   v2.0.5   // styling, layout, borders
charm.land/huh/v2        v2.0.3   // forms — replaces Ask/Confirm/Pick/Choose
charm.land/glamour/v2    v2.0.1   // markdown — help text, explanation panes
```

`golang.org/x/term` drops out except for one `IsTerminal` call; Bubble Tea
owns raw mode from Phase 2 onward. Verify with `go mod why` after Phase 2 —
if `huh/v2` still pulls it transitively that's fine, it just leaves gitle's
own imports.

Binary size will grow substantially (glamour ships chroma + a syntax
corpus). Check `make build` output after Phase 6; if it matters, glamour is
the first thing to cut.

## Architecture

Three execution modes, one codebase:

| Mode | Trigger | Bubble Tea? | Output |
|---|---|---|---|
| **headless** | not a TTY, or every needed value supplied by flags/args | no | plain lines to stdout/stderr, as today |
| **prompted command** | `gitle save` on a TTY | short-lived — one `huh` form per prompt, exits between | plain lines around the forms |
| **dashboard** | bare `gitle` on a TTY | long-lived, alt-screen | rendered frames only |

The load-bearing rule: **a long-running Bubble Tea program owns the screen,
so nothing underneath it may call `fmt.Print` or `gitcmd.Run`** (which
streams git's stdout straight to the terminal, `internal/gitcmd/gitcmd.go:32`).
Today every command body is a print-as-you-go procedure, so none of them can
be reused by the dashboard as written. Phase 3 fixes that by extracting the
logic into an ops layer that *returns* results instead of printing them.

```
main.go
└── cmd/            cobra wiring — thin, decides mode, renders results
    └── internal/ops/      pure command logic → (Result, error), no printing
        └── internal/gitcmd/   unchanged (plus non-streaming Run variants)
    └── internal/tui/      dashboard: model, panes, keymap
    └── internal/ui/       prompts (huh) + plain-line printers
    └── internal/theme/    lipgloss styles, single source of colour
```

## Phase 0 — Scaffolding

- `go get` the five modules above; `go mod tidy`.
- Add `internal/theme` with a `lipgloss.Style` set replacing the ANSI
  constants in `internal/ui/ui.go:23-31`: `Success`, `Info`, `Warn`,
  `Error`, `Hint`, `Bold`, plus `Border`, `Title`, `PaneActive`,
  `PaneInactive` for the dashboard.
- Honour `NO_COLOR` via lipgloss's own detection instead of the manual
  `os.Getenv` check at `internal/ui/ui.go:16`.
- **Verify:** `go build ./...` and every existing command still behaves
  identically. Nothing user-visible changes in this phase.

## Phase 1 — Terminal detection

Today `ui.IsInteractive` (`internal/ui/ui.go:82`) stats **stdin only**. That
misreports `gitle save > log.txt`, where stdin is a TTY but rendering into a
file is wrong.

- Replace with `term.IsTerminal(stdin) && term.IsTerminal(stdout)`.
- Add `ui.Interactive()` as the single gate; keep `IsInteractive` as a
  deprecated alias so the 20-odd call sites migrate incrementally.
- Add `GITLE_NO_TUI=1` to force headless — needed for tests, and for users
  on terminals where the TUI misbehaves.
- **Verify:** `gitle save > /tmp/x` no longer prompts; `echo | gitle start`
  still takes defaults.

## Phase 2 — Prompts on huh

Rewrite the four widgets *behind their current signatures*. This deletes the
~160 lines of cursor arithmetic in `Pick` and `Choose` and is the cheapest
visible win in the plan.

```go
func Confirm(question string) bool                 // → huh.NewConfirm()
func ConfirmDefault(question string, def bool) bool
func Ask(question, def string) string              // → huh.NewInput()
func Pick(prompt string, items []string) []int     // → huh.NewMultiSelect[int]()
func Choose(prompt string, items []string) int     // → huh.NewSelect[int]()
```

Behaviour that must be preserved exactly — these are load-bearing today:

- Non-interactive `Pick` returns **all** indices (`ui.go:151`), preserving
  "save everything"; `Choose` returns `-1`; `Ask`/`Confirm` return the
  default. Gate before touching huh at all.
- `Pick` starts with **everything ticked** (`ui.go:162`). huh's multiselect
  defaults to none — set `.Value(&all)` explicitly or `gitle save` silently
  becomes a no-op.
- Ctrl-C: `Pick` currently selects nothing, `Choose` returns `-1`. huh
  returns `huh.ErrUserAborted` — map it to those, do **not** `os.Exit`.
- The shared `bufio` stdin reader (`ui.go:21`) and its read-ahead comment
  become obsolete; huh owns stdin. Delete it and confirm no queued-input
  regression in `gitle start`'s consecutive prompts.

Wins that come free: scrolling for long file lists (today a 200-file `save`
scrolls its own header off), filtering, resize, mouse, Windows.

Also add:
- `ui.Spinner(title string, fn func() error) error` — wraps the OpenRouter
  call in `suggestMessage` (`cmd/save.go:126`), which currently blocks with
  no feedback.
- `ui.Editor(...)` — `huh.NewText().Editor()` for multi-line commit
  descriptions, replacing the shell-out in `OpenEditor` (`ui.go:303`) where
  a full editor is overkill. Keep `OpenEditor` for conflict editing.

**Verify:** walk `gitle start` end to end, `gitle save` with 0/1/many files,
`gitle fix-conflicts` on a real conflict, each both on a TTY and piped.

## Phase 3 — Extract the ops layer

This is the largest phase and the one that makes the dashboard possible.
Command bodies today interleave git calls, decisions, and printing —
`cmd/save.go:32-119` is the canonical example. The dashboard needs the
decisions without the printing.

For each command, move the logic to `internal/ops` as a function that takes
explicit inputs and returns a typed result:

```go
package ops

type SaveInput struct {
    Paths   []string
    Message string
}

type SaveResult struct {
    Saved          []string
    Message        string
    LeftoverChange bool
    HasUpstream    bool
}

func Save(in SaveInput) (SaveResult, error)
```

Rules for `internal/ops`:

- **No printing, no prompting.** Anything needing a human decision is an
  input field or a returned "needs input" state. Cobra supplies it via
  `ui`; the dashboard supplies it via its own model.
- **No `gitcmd.Run`** — add `gitcmd.RunQuiet` (capture both streams) and use
  it. `Run`'s stdout streaming stays only for the handful of headless
  commands that genuinely want git's native output (`history`, `branches`).
- Errors stay errors; the caller decides how to show them.

`cmd/*.go` then becomes: parse flags → prompt via `ui` if interactive →
call `ops` → render the result with `ui.Success`/`Hint`. Command help text,
flags, and exit codes are untouched, so the CLI contract holds.

Suggested order (small and self-contained first, to prove the shape):
`status` → `changes` → `branches` / `history` → `save` → `send` / `grab` →
`switch` / `newbranch` → `undo` → `safety` → `gitignore` → `start` →
`fix-conflicts`.

**Verify:** this phase should be behaviour-neutral. Diff the output of every
command against the pre-refactor binary on a scratch repo; any difference is
a bug, not an improvement.

## Phase 4 — The dashboard

`internal/tui`, launched when `gitle` is run with no subcommand on a TTY
(headless or non-TTY keeps printing the current help screen from
`cmd/help.go`).

Layout — a `lipgloss` two-column split, bordered panes, active pane
highlighted:

```
┌ gitle ─ myproject ──────────── on main · 2 ahead ─┐
│ Changes (3)          │ Details                    │
│  ● src/auth.go       │  diff / file preview       │
│  ● README.md         │  (bubbles/viewport)        │
│  ○ .env              │                            │
│ Branches (4)         │                            │
│ History              │                            │
└ s save · g grab · p send · b branch · ? help ─────┘
```

Model:

```go
type Model struct {
    repo    repoState        // branch, ahead/behind, upstream — refreshed on demand
    panes   []pane           // changes, branches, history
    focus   int
    detail  viewport.Model
    help    help.Model
    keys    keyMap
    err     error
    busy    bool
}
```

Key pieces:

- **Async git.** Every `ops` call runs in a `tea.Cmd` returning a typed
  message (`savedMsg`, `pulledMsg`, `errMsg`). Git must never run on the
  update loop — `git push` over a slow link would freeze the UI.
- **Prompts inside the dashboard.** huh forms embed as a sub-model, not
  standalone programs — you cannot start a second `tea.Program` while one is
  running. Model a `mode` field: `modeBrowse` / `modeForm`, delegating
  `Update`/`View` to the form while it's active.
- **Refresh.** After any mutating op, re-read repo state via one batched
  `tea.Cmd`. Add a manual `r` refresh; skip filesystem watching for now.
- **Alt screen** (`tea.WithAltScreen`) so quitting restores the shell.
- **Keys** through `bubbles/key` with a `?` help overlay, so bindings are
  declared once and documented automatically.
- **Escape hatch:** a key that drops to a plain shell command
  (`tea.ExecProcess`) for anything the TUI doesn't cover — e.g. opening
  `$EDITOR`, which needs the terminal handed over properly.

Ship the dashboard **read-only first** (browse status/changes/branches/
history, no mutations), then add actions one at a time behind the same
`ops` calls the CLI uses. A read-only dashboard is useful on its own and
carries none of the risk.

## Phase 5 — Conflict resolver

`cmd/fixconflicts.go` is 421 lines and the strongest case for a real TUI:
today it walks files one at a time, prints hunks linearly, and asks a
question per section (`--advanced`). Rebuild as a dedicated full-screen
model:

- File list on the left, conflict hunks on the right in a `viewport`.
- Side-by-side "yours" / "theirs" panes using the existing plain-English
  labels from `sideLabels` (`cmd/fixconflicts.go:79`).
- Per-hunk keys: keep yours / keep theirs / keep both / edit in `$EDITOR`
  (via `tea.ExecProcess`).
- Progress indicator across files; abort maps to the existing `abortOp`.

Reuses the parsing already in `internal/gitcmd/conflict.go` — this is a view
layer over logic that exists.

Reachable both as `gitle fix-conflicts` and from the dashboard when the
repo is mid-merge (`gitcmd.CurrentOp() != OpNone`).

## Phase 6 — Polish

- **Help.** Render `Long` descriptions and `gitle help` through glamour as
  markdown. Optionally a searchable command palette in the dashboard.
- **Banner.** `ui.Banner` (`ui.go:345`) restyled with lipgloss; keep the
  same ASCII art.
- **Accessibility.** Wire `huh`'s accessible mode when `ACCESSIBLE=1` — it
  degrades forms to plain prompts for screen readers. Confirm `NO_COLOR`
  still fully strips styling.
- **Tests.** Add `teatest` (`bubbletea/v2/exp/teatest`) golden-file tests for
  the dashboard and the conflict resolver. `internal/ops` gets ordinary unit
  tests against a temp git repo — the first real test coverage in the
  project.
- **Docs.** README screenshots/VHS tape; note the new bare-`gitle`
  behaviour. `.goreleaser.yaml` needs no change.

## Risks

| Risk | Mitigation |
|---|---|
| Phase 3 is a wide refactor with no test suite to catch regressions | Do it command-by-command, diffing output against the old binary; write the `ops` unit tests as you go rather than in Phase 6 |
| Bare `gitle` changes meaning (help → dashboard) | Non-TTY still prints help; consider one release where the dashboard is `gitle ui` before promoting it to the default |
| `gitcmd.Run` streaming leaks into the dashboard and corrupts the frame | Ban `Run` inside `internal/ops` — grep in CI |
| Charm v2 + `charm.land` paths are recent; docs and examples online mostly show v1 | Pin exact versions; read the v2 migration notes before Phase 4, where the API differences (`Init`/`Update` signatures, cursor handling) actually bite |
| Interaction with the pending [translation-plan.md](translation-plan.md) | Both plans touch `internal/ui`. Translation wraps strings centrally inside `ui`; this plan changes `ui`'s implementation underneath. Land Phase 2 first, then translation, to avoid rewriting the same functions twice — or accept the overlap and do translation after Phase 3, when strings live in `ops` results and are easier to catalogue |

## Sequencing

Phases 0–2 are independently shippable and low-risk: same CLI, better
prompts. Phase 3 ships as a no-op refactor. Phases 4–5 are the new product
surface. Nothing after Phase 2 blocks a release.
