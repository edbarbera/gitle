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

## Build it

Requires [Go](https://go.dev/dl/) 1.22+ and `git` on your machine.

```sh
go build -o gitle .
./gitle --help
```

## Install for a friend

Once this is pushed to GitHub, anyone with Go can install it with one line:

```sh
go install github.com/edbarbera/gitle@latest
```

That puts a `gitle` command on their machine. (You can also send them the
compiled binary directly — it's a single file.)
