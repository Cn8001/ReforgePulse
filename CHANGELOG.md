# Changelog

Versions follow [semantic versioning](https://semver.org). Before 1.0 a
minor version may change behaviour; the changes say so.

## 0.0.3

Graphics temperature, fan and power for every vendor, and fixes for two
crashes, one of which closed the app after it had been open for a day.
Data from 0.0.2 carries over.

### New

- Graphics temperature, fan speed and power are read from the graphics
  driver, so AMD, Intel and NVIDIA cards without NVML show them too.

### Improved

- Recordings can run for up to 24 hours instead of 4, so a whole night
  fits in one session.
- Turning off a Windows feature closes the dialog at once; the row and the
  status bar show the work while Windows finishes it.
- The restart notice for Windows features names the features still waiting
  for a restart, and goes away once Windows has restarted.

### Fixed

- The app no longer closes after it has been open for many hours. Each
  process list used up one of about 2000 slots the app can never give
  back, so a long session or a long recording ran out.
- The hardware probe no longer crashes on NVIDIA cards while reading
  graphics sensors.
- A recording length outside the allowed range now shows a message instead
  of doing nothing.

## 0.0.2

A health score on Overview, processes as a tree with program icons, system
check results kept in Reports, reports to download, and an Agents panel
that shows whether each agent is connected. Data from 0.0.1 carries over;
the database is upgraded on first start.

### New

- Overview scores system health out of 100 from the last scan's open
  findings and shows what took points off. Resolved findings, installed
  tuning tools and checks that could not run do not lower it.
- Processes shows the tree on the left, beside the opened process and its
  children.
- Program icons beside each process, loaded in the background so the list
  never waits for them.
- ReforgePulse's own processes show in green.
- System check results are kept: the verdict and where Windows keeps the
  full log, which opens from the result. Core test results are kept with
  them, as Processor cores. Both are in a new Checks tab in Reports.
- Any report downloads as a Markdown file.
- The Agents panel shows whether each agent is set up and whether its
  ReforgePulse server is running now, and what that server did.
- Leftovers can be selected all at once.

### Improved

- The process tree and table sort by the chosen column, by name until one
  is picked, and only the opened process holds its place.
- End task is hidden on processes Windows protects, where it could only
  fail.
- A dump's saved analysis opens and closes.
- An agent's setup command shows when asked for.
- An agent's server started while the window is open stops when the window
  closes.
- Page titles and tabs stay in place; only the content under them scrolls.
- Cards and rows that open something show an arrow.
- Tools comes before Reports in the sidebar.

### Fixed

- Memory diagnostic results are read from the event log, and a cancelled
  test no longer counts as a pass.
- Checking and connecting agents no longer flashes console windows.
- Long names and paths stay inside their boxes.
- Severity labels are written in sentence case.

## 0.0.1

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
  spots steady memory growth and stays the same size however long it runs, driver problems with what Windows Update
  offers, system checks (SFC, DISM, CHKDSK, memory diagnostic) and a CPU
  frequency cap on every power plan.

### Reports

- Saved scans (the last 30), crash dump analyses and recorder sessions (the
  last 20) stay to be opened again or deleted. A dump analysis is kept once
  it has run, so it does not have to run again.

### History

- Every change is recorded before it is made, with revert where it can be
  undone. Entries can be deleted one by one or before a date; the log keeps
  its last 500.
- Settings can delete history and reports older than 10, 30 or a chosen
  number of days, at once and at every start. Off unless turned on.

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
