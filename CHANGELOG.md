# Changelog

Versions follow [semantic versioning](https://semver.org). Before 1.0 a
minor version may change behaviour; the changes say so.

## 0.0.1 (2026-11-17)

The first release, for Windows 10 and 11 on x64.

### Issues

- Scan PC reads 30 days of Windows records: blue screens, unexpected
  restarts, WHEA hardware errors with the processor they came from, display
  driver resets, app crashes and hangs, failed services, disk and NTFS
  errors, firmware throttling, microcode, tuning drivers and devices
  without drivers.
- Each finding has a severity and a confidence, the evidence behind it and
  the other explanations that evidence allows.
- On an unstable PC, the processor, memory, storage, heat, power, drivers
  and software are each weighed by the evidence for and against them, with
  no verdict where the records cannot decide.
- Every scan is saved. Findings new since the scan before are marked, a
  finding can be marked resolved, and one that turns up again after that
  shows as "came back".

### Cleanup

- Temporary files, browser, app, developer and shader caches, Windows
  Update and error report leftovers, old logs and the Recycle Bin.
- Folders left behind by uninstalled programs, and downloads untouched for
  180 days, moved to the Recycle Bin.
- Programs not run for 90 days, removed by their own uninstaller.
- Startup apps and optional Windows features, switched off and back on.
- Deleting and moving to the Recycle Bin show their progress and can be
  stopped; the rest of the window stays usable meanwhile.

### Processes

- A process tree with CPU, GPU, private memory, ports and "not responding";
  search by name or port. End task refuses Windows' own processes.

### Tools

- Hardware and cores: board, BIOS, memory sticks, GPUs and disks; every
  logical processor with its live clock, P or E class and APIC ID.
- Core test: one core at a time, results compared across cores.
- Sensors (NVIDIA through NVML, CPU through HWiNFO), events, minidumps with
  `!analyze -v` when the Windows debugger is installed, a recorder that
  spots steady memory growth, driver problems with what Windows Update
  offers, system checks (SFC, DISM, CHKDSK, memory diagnostic) and a CPU
  frequency cap on every power plan.

### History

- Every change is recorded before it is made, with revert where it can be
  undone. Changes and saved scans can be deleted one by one or before a
  date. The log keeps its last 500 entries, and the last 30 scans are kept.

### The app

- Runs as Administrator and asks before every change, saying plainly what
  cannot be undone.
- Shows the terms and the MIT License to accept on first start, and again
  when either changes.
- Checks GitHub Releases for a new version at start (can be turned off) and
  from Settings, and asks before downloading or restarting.
- Reads that touch the hardware directly run in a child process, so a
  crash there does not close the window.
- Keeps its data in one SQLite database and writes crashes and errors to
  `reforgepulse.log`.
- `reforgepulse.exe mcp` is a read-only MCP server for Claude Code, Codex
  and OpenCode, set up from the Agents panel.
