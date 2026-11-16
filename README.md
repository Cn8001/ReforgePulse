# ReforgePulse

A Windows app that helps bring a PC back to how it ran when it was new. It
starts as Administrator (it asks through UAC), works offline, uploads
nothing, asks before every change and logs every change, git style, with
revert where a change can be undone.

- **Overview**: live CPU, GPU and memory load, disks and system facts.
- **Processes**: a live process tree with CPU, GPU, memory, open ports and
  "not responding" status; end a task after confirming.
- **Cleanup**
  - Disk space: temp files, browser, app, developer and shader caches,
    Windows Update and error report leftovers, old logs, the Recycle Bin.
  - Leftovers: folders uninstalled programs left in Program Files,
    ProgramData and AppData, moved to the Recycle Bin.
  - Old downloads: what sat in Downloads for 180 days.
  - Programs: installed programs Windows has not started in 90 days, with
    their uninstaller.
  - Startup apps: what starts at sign-in and the memory it uses now.
  - Features: Windows features most people are better off without.
- **Fix Issues**: Scan PC reads blue screens, unexpected restarts, WHEA
  hardware errors, GPU driver resets, app and service crashes, disk errors,
  microcode and tuning drivers, and explains them with severity, confidence
  and other possible causes. Includes a CPU frequency cap across all power
  plans.

## Build

From WSL or Windows, with Go, Node and `wails3` installed:

```sh
make build        # bin/reforgepulse.exe
make run          # build and start it
make test         # Go tests under -race
make installer    # NSIS installer with the terms page
make release      # dist/ with the asset names the Wails updater expects
make version V=…  # the version lives in build/config.yml only
make help         # every target
```

## Connect an AI agent

`reforgepulse.exe mcp` is a read-only MCP server over stdio. It runs as your
user without asking for Administrator rights. In the app, **MCP** in the top
bar connects Claude Code, Codex or OpenCode with one click. Agents can scan,
read crash evidence, processes, startup apps, programs, leftovers, power
plans and the change log; every change is still made by you in the window.

## Terms

Free to use, provided as is, without warranty. See [TERMS.md](TERMS.md).
