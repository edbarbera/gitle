# Maintaining gitle

Everything you need to develop, build, and release gitle. User-facing docs live
in [README.md](README.md).

## Requirements

- [Go](https://go.dev/dl/) 1.25+
- `git`
- Optional: [GitHub CLI](https://cli.github.com) (`gh`) for testing the
  repo-create flow in `gitle send`

## Project layout

```
main.go                     entry point + version resolution
cmd/                        one file per command (cobra)
  root.go                   root command, Execute(), shared guards
  save.go grab.go send.go   ... the commands
  changes.go                shared git-status parsing
  safety.go                 secret / large-file / protected-branch rails
  help.go                   grouped `gitle help` overview
internal/gitcmd/            thin wrapper around the git binary
internal/ui/                terminal output, prompts, arrow-key picker
install.sh                  curl installer (downloads release binaries)
.goreleaser.yaml            cross-compile + release config
.github/workflows/release.yml   builds + publishes on tag push
scripts/release.sh          cut a release locally
```

**Design rule:** gitle never reimplements git — it shells out via
`internal/gitcmd` so users get their own config, credentials, and hooks.

## Everyday commands

```sh
make build       # go build -o gitle .
make install     # go install .
make version     # build + print gitle --version
make vet         # go vet ./...
make fmt         # gofmt -w .
```

Or use the underlying tools directly (`go build -o gitle .`, etc.).

Before committing, keep it clean:

```sh
go build ./... && go vet ./... && gofmt -l .
```

## How versioning works

`gitle --version` resolves in this order (see `main.go`):

1. **`-ldflags "-X main.version=..."`** — set by GoReleaser for release
   binaries. This is the source of truth for published versions.
2. **Go build info** — `go install ...@vX` records the module version, read via
   `runtime/debug.ReadBuildInfo()`.
3. **`dev`** — plain local `go build`; Go stamps a `-dirty` VCS pseudo-version.

## Releasing

Releases are fully automated. Pushing a `v*` tag triggers
`.github/workflows/release.yml`, which runs [GoReleaser](https://goreleaser.com)
to cross-compile (macOS + Linux, amd64 + arm64) and attach the binaries to a
GitHub Release. `install.sh` downloads those binaries.

**Don't tag by hand — use the script:**

```sh
scripts/release.sh patch     # v0.2.3 -> v0.2.4
scripts/release.sh minor     # v0.2.3 -> v0.3.0
scripts/release.sh major     # v0.2.3 -> v1.0.0
scripts/release.sh v0.5.0    # explicit version

# or via make
make release BUMP=patch
```

The script refuses to run unless you're on a clean `main`, bumps from the latest
tag, creates an annotated tag, pushes `main` then the tag, and prints the
Actions URL to watch.

### Prerequisites for releases to work

- The GitHub repo must be **public** (so `install.sh` and the checksum database
  can fetch anonymously).
- The first tagged run needs the GoReleaser Action to succeed — check the
  **Actions** tab after your first `scripts/release.sh`.

### Stable vs pre-release

`install.sh` installs the latest **stable** release by default. Tags with a
`-beta` / `-rc` suffix are marked as pre-releases by GoReleaser (`prerelease:
auto`) and are skipped by the default installer. To install a pre-release for
testing:

```sh
curl -fsSL https://raw.githubusercontent.com/edbarbera/gitle/main/install.sh | GITLE_PRERELEASE=1 sh
```

## The installer

`install.sh` detects OS/arch, resolves the newest matching release via the
GitHub API, downloads `gitle_<os>_<arch>`, and installs it to `/usr/local/bin`
(using `sudo` only if needed). Overrides:

- `GITLE_PRERELEASE=1` — include pre-releases
- `GITLE_VERSION=vX.Y.Z` — pin an exact version

## Testing interactive flows

The arrow-key picker and wizards need a real TTY. Drive them with `expect`:

```sh
expect <<'EOF'
spawn gitle save
expect "confirm"
send " \r"
expect "Describe"
send "my change\r"
expect eof
EOF
```

Piped/no-TTY runs fall back to safe defaults (save-all, message required).
