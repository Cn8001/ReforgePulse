# ReforgePulse is a Windows app, so every build targets Windows, also from WSL.
# Go tests run on the host: Windows-only packages drop out by build tag.

.DEFAULT_GOAL := help
.PHONY: help build run dev test vet fmt tidy clean

## help: list the targets
help:
	@sed -n 's/^## \([^:]*\): /  make \1\t/p' $(MAKEFILE_LIST) | expand -t 16

## build: bin/reforgepulse.exe
build:
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

## clean: remove built binaries
clean:
	rm -rf bin
