# ReforgePulse is a Windows app, so every build targets Windows, also from WSL.
# Go tests run on the host: Windows-only packages drop out by build tag.

.DEFAULT_GOAL := help
.PHONY: help build run dev test vet fmt tidy clean installer release publish version site

## help: list the targets
help:
	@sed -n 's/^## \([^:]*\): /  make \1\t/p' $(MAKEFILE_LIST) | expand -t 16

## build: bin/reforgepulse.exe
build:
	@sh scripts/version.sh >/dev/null
	GOOS=windows wails3 task build

## run: build, then start the app (works from WSL too)
run: build
	./bin/reforgepulse.exe

## dev: hot-reload dev mode (on Windows)
dev:
	wails3 dev -config ./build/config.yml

## test: Go tests under -race
test:
	go test -race ./internal/...

## vet: go vet for the Windows target (needs frontend/dist, so build first)
vet:
	GOOS=windows go vet ./...

## fmt: gofmt our Go files (the generated build/ files are left alone)
fmt:
	gofmt -w *.go internal

## tidy: go mod tidy
tidy:
	go mod tidy

## installer: bin/reforgepulse-amd64-installer.exe (needs NSIS: apt install nsis)
installer:
	@command -v makensis >/dev/null || { echo "The installer needs NSIS: sudo apt-get install nsis, or winget install NSIS.NSIS"; exit 1; }
	# The licence page shows TERMS.md; NSIS reads UTF-16LE with a BOM and CRLF.
	{ printf '\377\376'; sed 's/^#\{1,\} //' TERMS.md | sed 's/$$/\r/' | iconv -f UTF-8 -t UTF-16LE; } > build/windows/nsis/license.txt
	GOOS=windows wails3 task windows:package

ARCH := amd64

# The Wails updater (GitHub provider) picks the release asset whose name holds
# both the OS and the architecture, skips "-installer." names, and checks the
# download against SHA256SUMS. These names follow that.
## release: installer, then dist/ with the assets and SHA256SUMS the Wails updater reads
release: installer
	rm -rf dist && mkdir dist
	cp bin/reforgepulse.exe dist/reforgepulse-windows-$(ARCH).exe
	cp bin/reforgepulse-$(ARCH)-installer.exe dist/reforgepulse-windows-$(ARCH)-installer.exe
	cd dist && sha256sum reforgepulse-* > SHA256SUMS

## publish: tag the version, build dist/, push and create the GitHub release
publish:
	@sh scripts/publish.sh

## version: V=1.2.3 sets the version in build/config.yml, the one place it lives
version:
	@sh scripts/version.sh $(V)

# GitHub Pages serves docs/ from master. The screenshots and icon in docs/
# are source files the README uses too, so only the build's own output is
# cleared.
## site: build the website into docs/
site:
	@[ -d site/node_modules ] || (cd site && npm ci)
	rm -rf docs/assets
	cd site && npm run build

## clean: remove built binaries and release assets
clean:
	rm -rf bin dist
