<div align="center">

<pre>
       _ _   _      
  __ _(_) |_| | ___ 
 / _` | | __| |/ _ \
| (_| | | |_| |  __/
 \__, |_|\__|_|\___|
 |___/              
</pre>

### git, made friendly ✨

Plain-English version control for people who don't know git.

![Release](https://img.shields.io/github/v/release/edbarbera/gitle?color=2ea44f&label=release)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-2ea44f)
![Made with Go](https://img.shields.io/badge/made%20with-Go-2ea44f)
![Wraps git](https://img.shields.io/badge/wraps-git-2ea44f)

</div>

---

**gitle** wraps the `git` you already have and gives everyday tasks names that
make sense — `save`, `send`, `grab` — while keeping you on good habits
automatically. No jargon, no memorising commands, no fear of breaking things.

## 📦 Install

One line. No setup — just needs `git` on macOS or Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/edbarbera/gitle/main/install.sh | sh
```

Then set up your first project:

```sh
gitle start
```

That's it. 🎉

## ⚡ Commands

| You type | What it does |
| --- | --- |
| `gitle start` | Guided setup for a new project |
| `gitle save "message"` | Save a snapshot of your work |
| `gitle send` | Send your work online |
| `gitle grab` | Grab everyone's latest work |
| `gitle status` | See where you are right now |
| `gitle history` | See your saved points over time |
| `gitle branches` | List your lines of work |
| `gitle switch <name>` | Switch to another line of work |
| `gitle new-branch <name>` | Start a new line of work |
| `gitle undo` | Undo your last save (safely) |
| `gitle fix-conflicts` | Walk through conflicts step by step |
| `gitle help` | See everything, grouped and explained |

Run `gitle help` any time for the full, friendly list.

## 👀 A quick tour

```console
$ gitle save "add login page"

[?] Which changes do you want to save?
❯ ● New:     login.js
  ● Changed: app.js
  ↑/↓ move · space toggle · a all · n none · enter confirm

✓ Saved 2 file(s): "add login page"
  Send it online with gitle send.

$ gitle status
📦 my-app
   on branch main
✓ Everything is saved.
✓ Up to date with online.

$ gitle send
✓ Sent everything online.
```

## 🛟 Safety rails

gitle warns you — in plain English — before anything risky:

- 🔒 **Secrets** — spots `.env`, keys and credentials before you save them
- 📦 **Big files** — flags oversized files that would bloat your project
- 🌿 **Pushing to main** — nudges you to make a branch on shared projects
- 🔄 **Out of sync** — tells you to `gitle grab` when others pushed first
- 🧩 **Conflicts** — `gitle fix-conflicts` walks you through clashes file by file, no raw markers needed
- ⚠️ **Throwing away work** — always confirms before discarding changes

## 🔢 Version

```sh
gitle --version
```

---

<div align="center">

Built on top of `git` · Maintainer? See **[MAINTAINERS.md](MAINTAINERS.md)**

</div>
