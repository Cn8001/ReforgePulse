#!/bin/sh
# The app's version lives in one place: info.version in build/config.yml.
# This copies it into the files each platform's packaging reads, which Wails
# generated once and never updates. The Makefile runs it before every
# build; `make version V=1.2.3` sets a new version and runs it.
#
#   sh scripts/version.sh           sync the platform files to config.yml
#   sh scripts/version.sh 1.2.3     set config.yml to 1.2.3, then sync
set -eu
cd "$(dirname "$0")/.."

config=build/config.yml
if [ $# -gt 0 ]; then
	case "$1" in
	*[!0-9.]* | "" | .* | *. | *..*)
		echo "version must be numbers and dots, such as 1.2.3: $1" >&2
		exit 1
		;;
	esac
	# Only the first version: line under info:, not the commented examples.
	awk -v v="$1" '
		/^info:/ { inInfo = 1 }
		/^[^ #]/ && !/^info:/ { inInfo = 0 }
		inInfo && !done && /^  version:/ { sub(/"[^"]*"/, "\"" v "\""); done = 1 }
		{ print }
	' "$config" >"$config.tmp" && mv "$config.tmp" "$config"
fi

v=$(awk '/^info:/ { i = 1; next } /^[^ #]/ { i = 0 } i && /^  version:/ { match($0, /"[^"]*"/); print substr($0, RSTART + 1, RLENGTH - 2); exit }' "$config")
if [ -z "$v" ]; then
	echo "no info.version in $config" >&2
	exit 1
fi
# MSIX wants four numbers.
quad=$(echo "$v.0.0.0" | cut -d. -f1-4)

edit() { sed "$1" "$2" >"$2.tmp" && mv "$2.tmp" "$2"; }

edit "s/\"file_version\": \"[^\"]*\"/\"file_version\": \"$v\"/; s/\"ProductVersion\": \"[^\"]*\"/\"ProductVersion\": \"$v\"/" build/windows/info.json
edit "s/!define INFO_PRODUCTVERSION \"[^\"]*\"/!define INFO_PRODUCTVERSION \"$v\"/" build/windows/nsis/wails_tools.nsh
# The app's own identity only; the manifest also names Windows' controls.
edit "s/\(<assemblyIdentity type=\"win32\" name=\"[^\"]*reforgepulse\" version=\"\)[^\"]*\"/\1$v\"/" build/windows/wails.exe.manifest
for f in build/windows/msix/app_manifest.xml build/windows/msix/template.xml; do
	edit "s/^\([[:space:]]*Version=\"\)[0-9.]*\"/\1$quad\"/" "$f"
done
edit "s/^version: \"[^\"]*\"/version: \"$v\"/" build/linux/nfpm/nfpm.yaml
for f in build/darwin/Info.plist build/darwin/Info.dev.plist; do
	[ -f "$f" ] || continue
	edit "/<key>CFBundleShortVersionString<\/key>/{n;s/<string>[0-9.]*\(-dev\)\{0,1\}<\/string>/<string>$v\1<\/string>/;}
/<key>CFBundleVersion<\/key>/{n;s/<string>[^<]*<\/string>/<string>$v<\/string>/;}" "$f"
done
echo "version $v"
