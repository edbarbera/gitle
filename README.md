# gitle — git made friendly

`gitle` is a small command-line tool that wraps `git` in plain English. It's for
people who want the safety and sharing of version control without having to learn
git first. Under the hood it just runs real `git`, so it works with GitHub, your
existing credentials, and anything else you already use.

## Commands

| You type | What it does | Real git |
| --- | --- | --- |
| `gitle start` | Start tracking this folder (once, at the beginning) | `git init -b main` |
| `gitle save "message"` | Save a snapshot of all your work | `git add -A && git commit -m` |
| `gitle undo` | Undo your last save, keeping your changes | `git reset --soft HEAD~1` |
| `gitle send` | Send your saved work online | `git push` (sets upstream first time) |
| `gitle grab` | Grab everyone's latest work | `git pull --rebase` |
| `gitle status` | Plain-English summary of where you are | `git status` |
| `gitle history` | See your saved points over time | `git log --oneline --graph --decorate` |
| `gitle branches` | List separate lines of work | `git branch -a` |
| `gitle switch <name>` | Switch to an existing line of work | `git checkout <name>` |
| `gitle new-branch <name>` | Start a new line of work | `git checkout -b <name>` |

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

Then `gitle --help`. This downloads the right prebuilt binary and puts it on
your PATH (`/usr/local/bin`), so the command just works.

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

## Releasing (maintainer)

Binaries are built automatically by [GoReleaser](https://goreleaser.com) via
GitHub Actions. To publish a new version:

```sh
git tag v0.1.0
git push --tags
```

The workflow in `.github/workflows/release.yml` cross-compiles for macOS and
Linux (amd64 + arm64) and attaches the binaries to the GitHub Release, which is
what `install.sh` downloads.
