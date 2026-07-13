#!/usr/bin/env bash
# Cut a new gitle release: pick the next version, tag it, and push the tag.
# Pushing a v* tag triggers the GitHub Actions workflow that builds and
# publishes the binaries (.github/workflows/release.yml), which is what the
# curl installer downloads.
#
# Usage:
#   scripts/release.sh v0.3.0    # explicit version
#   scripts/release.sh patch     # bump last tag's patch  (v0.2.3 -> v0.2.4)
#   scripts/release.sh minor     # bump minor             (v0.2.3 -> v0.3.0)
#   scripts/release.sh major     # bump major             (v0.2.3 -> v1.0.0)
set -euo pipefail

die() {
  echo "release: $*" >&2
  exit 1
}

[ $# -eq 1 ] || die "usage: scripts/release.sh <version|patch|minor|major>"

cd "$(git rev-parse --show-toplevel)" || die "not inside a git repository."

# Releases are cut from a clean main branch.
branch="$(git rev-parse --abbrev-ref HEAD)"
[ "$branch" = "main" ] || die "you're on '$branch'; releases are cut from 'main'."
[ -z "$(git status --porcelain)" ] || die "working tree has uncommitted changes; commit or stash first."

latest="$(git tag -l 'v*' --sort=-v:refname | head -n1)"

case "$1" in
  patch | minor | major)
    [ -n "$latest" ] || die "no existing vX.Y.Z tag to bump; pass an explicit version."
    core="${latest#v}" # strip leading v
    core="${core%%-*}" # strip any -beta / -rc suffix
    maj="${core%%.*}"
    rest="${core#*.}"
    min="${rest%%.*}"
    pat="${rest#*.}"
    case "$1" in
      patch) pat=$((pat + 1)) ;;
      minor)
        min=$((min + 1))
        pat=0
        ;;
      major)
        maj=$((maj + 1))
        min=0
        pat=0
        ;;
    esac
    version="v${maj}.${min}.${pat}"
    ;;
  v[0-9]*.[0-9]*.[0-9]*) version="$1" ;;
  [0-9]*.[0-9]*.[0-9]*) version="v$1" ;;
  *) die "invalid version '$1' (want vX.Y.Z, or patch/minor/major)." ;;
esac

git rev-parse "$version" >/dev/null 2>&1 && die "tag $version already exists."

echo "Latest tag : ${latest:-<none>}"
echo "New release: $version"
printf "Create and push this tag? [y/N] "
read -r reply
case "$reply" in
  y | Y | yes | YES) ;;
  *)
    echo "Aborted."
    exit 0
    ;;
esac

# Make sure main is pushed before the tag so the release matches the branch.
git push origin main
git tag -a "$version" -m "gitle $version"
git push origin "$version"

echo ""
echo "✓ Pushed tag $version — GitHub Actions is now building the release."
remote_url="$(git remote get-url origin 2>/dev/null || true)"
case "$remote_url" in
  *github.com*)
    slug="$(printf '%s' "$remote_url" | sed -E 's#.*github.com[:/]([^/]+/[^/.]+)(\.git)?$#\1#')"
    [ -n "$slug" ] && echo "  Watch it: https://github.com/$slug/actions"
    ;;
esac
