# gitle — git made friendly

`gitle` is a small command-line tool that wraps `git` in plain English. It's for
people who want the safety and sharing of version control without having to learn
git first. Under the hood it just runs real `git`, so it works with GitHub, your
existing credentials, and anything else you already use.

## Commands

| You type | What it does | Real git |
| --- | --- | --- |
| `gitle start` | Guided setup: track the folder, name yourself, add a .gitignore, first save, connect to GitHub | `git init` + config, and more |
| `gitle save ["message"]` | Arrow-key checklist to pick which files to include, then save a snapshot (`--all` saves everything) | `git add` (chosen files) `&& git commit` |
| `gitle undo` | Undo your last save, keeping your changes (`--hard` discards uncommitted changes) | `git reset --soft HEAD~1` |
| `gitle send` | Send your saved work online (offers to create a GitHub repo if there isn't one) | `git push` (sets upstream first time) |
| `gitle grab` | Grab everyone's latest work | `git pull --rebase` |
| `gitle status` | Friendly summary: project, branch, ahead/behind main, colour-coded changes, online sync | `git status` + more |
| `gitle help` | Aesthetic overview of all commands, grouped by task | — |
| `gitle history` | See your saved points over time | `git log --oneline --graph --decorate` |
| `gitle branches` | List separate lines of work | `git branch -a` |
| `gitle switch <name>` | Switch to an existing line of work | `git checkout <name>` |
| `gitle new-branch <name>` | Start a new line of work | `git checkout -b <name>` |

## Friendly onboarding

`gitle start` is a short, guided wizard that gets a brand-new folder ready:

1. **Start tracking** — sets up version control (`git init`).
2. **Who are you?** — asks your name and email so your saves are signed
   (skips this if git already knows you). Prevents git's confusing
   "who are you?" error on your first save.
3. **Keep junk and secrets out** — detects your project type (Node, Python,
   Go, Rust, Ruby, Java, .NET, PHP, Swift, Elixir, Dart/Flutter) and offers a
   fitting `.gitignore`, always including common secret files like `.env`.
4. **First save** — offers to make your very first snapshot right away.
5. **Connect to GitHub** — optionally links a repo so `gitle send` works.

If you later run `gitle send` without an online home, it offers to create a
GitHub repo for you (using the free [`gh`](https://cli.github.com) tool) and
push in one step.

It's safe to run again — each step detects what's already done and skips it.
When run without a terminal (piped/scripted), it uses safe defaults instead of
prompting.

## Picking what to save

Run `gitle save` in a terminal and it shows a checklist of everything that
changed — all selected by default:

```
[?] Which changes do you want to save?
❯ ● New:     notes.txt
  ● Changed: main.go
  ○ Removed: old.txt
  ↑/↓ move · space toggle · a all · n none · enter confirm
```

Move with the **↑/↓ arrow keys**, press **Space** to toggle a file on (`●`) or
off (`○`), then **Enter** to save. Only the selected files go in — the rest stay
as unsaved changes for next time.

- `gitle save "message"` — skips the description prompt.
- `gitle save --all` (or `-a`) — skips the checklist and saves everything;
  still asks for a description if you didn't pass one.
- Piped or scripted (no terminal) — saves everything, message required.

## Knowing where you are

`gitle status` gives a friendly, colour-coded snapshot:

```
📦 myproject
   on branch feature-login
   compared to main: 2 ahead

! You have unsaved changes:
  New:     notes.txt
  Changed: main.go
  Save these with gitle save "...".
✓ Up to date with online.
```

It shows your project name, which line of work you're on, how far ahead/behind
`main` you are, exactly what's changed, and whether you're in sync with the
online copy. (The online line reflects your last `gitle grab` — status stays
fast and doesn't reach over the network.)

## Safety rails

gitle steps in with a plain-English warning before anything risky:

- **Secrets** — `gitle save` spots files that look like passwords or keys
  (`.env`, `*.pem`, `id_rsa`, `credentials.json`, …) and asks before saving
  them, so you don't leak credentials.
- **Big files** — `gitle save` flags files over 10 MB that would bloat your
  project, and asks before including them.
- **Pushing to main** — `gitle send` warns when you're sending straight to a
  shared branch like `main`/`master` and suggests making a branch first.
- **Someone else pushed first** — if `gitle send` is rejected because there's
  newer work online, gitle explains it and tells you to `gitle grab` first
  (instead of a wall of git errors).
- **Throwing away work** — `gitle undo --hard` lists exactly which files it
  will discard, warns that it can't be undone, and refuses to run without a
  confirmation.

## Good habits, built in

- **`save` never loses changes** — it always stages everything, so nothing is
  forgotten.
- **`undo` is safe** — it keeps your file changes and only removes the saved
  point. It asks before doing anything.
- **`grab` uses rebase** — keeping history tidy and avoiding confusing merge
  commits, and it refuses to run over unsaved changes.
- **`send` sets up the online link** for you on the first push.
- Friendly errors point you at the next step instead of showing raw git output.

## Install (for everyone)

One command. No Go, no setup — just needs `git` and macOS or Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/edbarbera/gitle/main/install.sh | sh
```

Then `gitle --help`. This downloads the latest **stable** release, puts it on
your PATH (`/usr/local/bin`), so the command just works.

Maintainer/testing overrides (pass as env vars before the command):

```sh
# Install the newest pre-release (beta / rc) instead of stable
curl -fsSL https://raw.githubusercontent.com/edbarbera/gitle/main/install.sh | GITLE_PRERELEASE=1 sh

# Pin an exact version
curl -fsSL https://raw.githubusercontent.com/edbarbera/gitle/main/install.sh | GITLE_VERSION=v0.2.0 sh
```

## Install (for Go developers)

```sh
go install github.com/edbarbera/gitle@latest
```

Note: this needs `~/go/bin` (see `go env GOPATH`) on your `PATH`.

## Build from source

Requires [Go](https://go.dev/dl/) 1.22+ and `git`.

```sh
go build -o gitle .
./gitle --help
```

## Version

```sh
gitle --version
```

Released binaries report their Git tag (e.g. `gitle v0.2.0`) — the tag is
embedded at build time by GoReleaser, and `go install`ed copies read it from
Go's build info. Local `go build`s show a `-dirty` development version.

## Releasing (maintainer)

Binaries are built automatically by [GoReleaser](https://goreleaser.com) via
GitHub Actions whenever a `v*` tag is pushed. Instead of tagging by hand, use
the release script — it bumps the version, checks you're on a clean `main`,
tags, and pushes (which kicks off the build):

```sh
scripts/release.sh patch     # v0.2.3 -> v0.2.4
scripts/release.sh minor     # v0.2.3 -> v0.3.0
scripts/release.sh major     # v0.2.3 -> v1.0.0
scripts/release.sh v0.5.0    # or an explicit version

# or via make
make release BUMP=patch
```

The workflow in `.github/workflows/release.yml` cross-compiles for macOS and
Linux (amd64 + arm64) and attaches the binaries to the GitHub Release, which is
what `install.sh` downloads.
