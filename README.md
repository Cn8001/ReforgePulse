<p align="center"><img src="build/appicon.png" width="96" alt=""></p>

<h1 align="center">ReforgePulse</h1>

<p align="center">Find out why your Windows PC crashes or crawls, and clean it up without guesswork.</p>

---

My PC kept restarting on its own. Event Viewer was a wall of noise, the
minidumps needed WinDbg, and the only way to see which core was throwing
hardware errors was a PowerShell script I wrote myself. ReforgePulse is
that script, and a few more like it, turned into one app.

It runs on Windows 11 (10 should work too), starts as Administrator, and does nothing
behind your back. Every change shows you what it will do first, waits for
a yes, and lands in a change log you can read later and, where possible,
undo.

## What it does

**Fix Issues** reads what Windows already recorded: blue screens,
restarts without a clean shutdown, WHEA hardware errors, GPU driver resets,
crashing apps and services, disk errors, firmware throttling, and the
drivers and updates installed right before things went wrong. Then it
doesn't jump to "your CPU is dead". It goes through processor, memory,
drives, temperature, power, drivers and software one by one, and for each
says what points there, what points away, and what to try next. Some of it
Windows simply can't see, like CPU voltage, and it says so instead of
guessing.

**Tools** is the part that replaces the programs you would otherwise juggle:

- every core live, with its clock, P or E type and APIC ID, so "WHEA error
  on APIC 32" turns into "P-core 4"
- a single-core stress test that runs each core alone at full boost and
  flags the ones that get the math wrong
- GPU sensors straight from the NVIDIA driver, and CPU temperatures and
  voltages from HWiNFO when it's running
- Event Viewer, grouped and explained, with the harmless spam hidden
- minidumps: stop code without any tools, and the driver to blame through
  WinDbg if you have it
- a recorder that watches CPU, GPU and per-process memory while you play,
  then points at anything that leaked
- devices missing a driver, what Windows Update offers for them, and the
  makers' own driver pages for your GPU and board
- sfc, DISM and chkdsk scans, and Windows Memory Diagnostic
- a CPU frequency cap you set yourself, applied to every power plan so a
  plan switch can't quietly undo it

**Cleanup** goes past temp files. It clears browser, app, developer and
shader caches, finds folders that uninstalled programs left behind, lists
programs Windows hasn't started in three months, digs out downloads nobody
touched in half a year, and shows what starts with Windows next to how much
memory it eats right now. Leftovers and old downloads go to the Recycle
Bin, not straight to oblivion.

**Processes** is a process tree with CPU, GPU, memory, open ports and a
"not responding" flag. Search `:8080` to find out who's holding the port.

## What it won't do

No "RAM booster", no registry cleaner, no one-click "optimize", and no
driver installs: it shows you what's missing and sends you to Windows
Update or the maker, who know their drivers best. Nothing is
uploaded and there is no account. The one time it touches the internet is
when you ask WinDbg to analyse a dump and it needs symbol files from
Microsoft, and it asks first.

## Use it with an AI agent

`reforgepulse.exe mcp` starts a small read-only MCP server. The **MCP**
button in the app registers it with Claude Code, Codex or OpenCode in one
click. The agent can run a scan, read crash evidence, hardware, processes
and the change log, and talk you through it. It can't change anything; that
stays with you.

## Building

You need Go, Node and `wails3`. It builds from Windows or from WSL.

```sh
make run           # build and start
make test          # Go tests with -race
make installer     # NSIS installer
make release       # dist/ laid out for the Wails updater
make version V=1.2.3
make help          # everything else
```

The version lives in one place, `build/config.yml`; `make version` copies
it everywhere else.

## License

Free to use, as is, no warranty. Read [TERMS.md](TERMS.md) before you let
it delete anything.
