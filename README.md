# ReforgePulse

A simple Desktop app that helps bring a PC back to how it ran when it was new:

- **Cleanup** finds what takes up space and deletes what is not needed, only after you approve it.
- **Fix Issues** analyses crashes and bottlenecks and explains them in plain language.

It starts as Administrator (it asks through UAC), works offline, uploads nothing,
and logs every change it makes.

## Build

From WSL or Windows, with Go, Node and `wails3` installed:

```sh
make build   # bin/reforgepulse.exe
make run     # build and start it
make test    # Go tests under -race
make help    # every target
```

## Connect an AI agent

`reforgepulse.exe mcp` is a read-only MCP server over stdio. It runs as your
user, without asking for Administrator rights. The app's **Connect an agent**
panel shows the exact command for Claude Code, Codex and OpenCode.

## Terms

Free to use, provided as is, without warranty. See [TERMS.md](TERMS.md).
