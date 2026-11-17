#!/bin/sh
# Publishes the version in build/config.yml as a GitHub release: tags it,
# builds dist/, pushes master and the tag, and uploads dist/ with that
# version's CHANGELOG section as the notes. It stops at the first thing that
# is not right, before anything leaves this machine.
set -eu
cd "$(dirname "$0")/.."

v=$(sh scripts/version.sh | cut -d' ' -f2)
tag="v$v"

[ "$(git rev-parse --abbrev-ref HEAD)" = master ] || { echo "publish from master" >&2; exit 1; }
git diff --quiet && git diff --cached --quiet || { echo "commit or stash your changes first" >&2; exit 1; }

notes=$(awk -v v="$v" '
	$0 ~ "^## " v "( |$)" { on = 1; next }
	on && /^## / { exit }
	on { print }
' CHANGELOG.md)
[ -n "$(echo "$notes" | tr -d '[:space:]')" ] || { echo "CHANGELOG.md has no section for $v" >&2; exit 1; }

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
	[ "$(git rev-list -n1 "$tag")" = "$(git rev-parse HEAD)" ] || { echo "$tag already points at another commit" >&2; exit 1; }
else
	git -c tag.gpgSign=false tag -a "$tag" -m "ReforgePulse $v"
fi
if gh release view "$tag" >/dev/null 2>&1; then
	echo "release $tag already exists on GitHub" >&2
	exit 1
fi

make release
git push origin master
git push origin "$tag"
echo "$notes" | gh release create "$tag" dist/* --title "ReforgePulse $v" --notes-file -
