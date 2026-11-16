<p align="center"><img src="build/appicon.png" alt="" width="96"></p>

<h1 align="center">ReforgePulse</h1>

Crash and slowdown diagnostics for Windows, with cleanup that asks first:
a desktop app on Wails v3, Vue and Go that reads what Windows already
records (event logs, WHEA, minidumps, counters, sensors) and says which part
of the PC it points at. It runs elevated; every change goes through a
confirmation and into a git-style change log.

- **Issues**: Scan PC reads 30 days of stop errors (WER 1001), unexpected
  restarts (Kernel-Power 41), WHEA errors with their APIC IDs, display
  driver resets (4101), application and service failures, disk and NTFS
  errors, firmware throttling, microcode, tuning drivers (XTU, NTIOLib,
  WinRing0...) and devices without drivers. Findings have a severity and a
  confidence. On an unstable PC it weighs each suspect (CPU, memory, storage,
  heat, power, drivers, software) with the evidence for and against, and
  says "unmeasured" where Windows cannot see, rather than "fine".
- **Cleanup**: temp, browser, app, developer and shader caches; Windows
  Update and WER leftovers; old logs; the Recycle Bin. Leftover folders of
  uninstalled programs in Program Files, ProgramData and AppData, matched
  against installed programs, services, startup entries and running
  processes. Downloads untouched for 180 days. Programs Prefetch says have
  not run in 90 days, removed by their own uninstaller. Startup apps with
  the memory they use now, disabled through `StartupApproved` like Task
  Manager does. Optional Windows features (SMB 1.0, PowerShell 2.0...).
  Leftovers and old downloads go to the Recycle Bin.
- **Processes**: a tree under parents, with CPU, GPU (per-process 3D engine
  share), private memory, TCP/UDP ports and "not responding". Search `:8080`
  for a port. End task refuses Windows' own processes.
- **Tools**
  - Hardware & cores: board, BIOS, RAM sticks (rated vs configured speed),
    GPUs and drivers, disk health; every logical processor with its live
    effective clock (PDH `% Processor Performance`), P/E class and x2APIC ID
    from `CPUID`, so a WHEA "APIC 32" shows as LP 8, P-core 4.
  - Core test: runs deterministic SHA-256 and floating-point rounds on one
    pinned core at a time and compares results by majority vote.
  - Sensors: NVIDIA through `nvml.dll`; CPU temperatures and voltages from
    HWiNFO's shared memory (`HWiNFO_SENS_SM2`) when HWiNFO runs with Shared
    Memory Support. There is no driver of its own, so without HWiNFO the CPU
    temperature is not measured.
  - Events: System and Application, grouped by provider and ID with counts,
    messages through `EvtFormatMessage`, known noise (DCOM 10016...) hidden.
  - Minidumps: stop code and parameters from the `PAGEDU64` header;
    `!analyze -v` through cdb when WinDbg or the SDK debuggers are installed.
  - Recorder: load and per-process private bytes every 2 s; a linear fit
    flags steady growth as a likely leak.
  - Drivers: problem devices (ConfigManagerErrorCode) and what Windows
    Update offers, read through the Windows Update Agent. Installing is left
    to Windows Update and the makers' pages it links to.
  - System checks: `sfc /verifyonly`, `DISM /ScanHealth`, `chkdsk /scan`,
    Windows Memory Diagnostic.
  - CPU cap: `PROCFREQMAX` and `PROCFREQMAX1` on every power plan through
    `powrprof.dll`, old values kept for revert.
- **MCP**: `reforgepulse.exe mcp` is a read-only MCP server over stdio,
  running unelevated. The MCP button registers it with Claude Code, Codex or
  OpenCode. Agents can scan and read; changes stay in the window.

Long work (scans, cleanup, checks, the core test) runs as tasks in the
status bar, at most four at a time, each cancellable.

## Use

```
make run                # build bin/reforgepulse.exe and start it
make test               # go test -race ./internal/...
make vet                # go vet for GOOS=windows
make installer          # bin/reforgepulse-amd64-installer.exe (NSIS)
make release            # dist/: release assets and SHA256SUMS for the updater
make version V=1.2.3    # the version lives in build/config.yml only
```

Builds from Windows or WSL with Go, Node and `wails3`. The app is Windows
only; the packages under `internal/` other than `platform/windows` build and
test anywhere.

Data is in `%LOCALAPPDATA%\ReforgePulse`: `journal.jsonl` (the change
log), `webview/` and the debugger's `symbols/` cache. The only network use
is that symbol download, and only when you ask for a dump analysis.

## Layout

```
main.go, services.go   wiring: window, services, mcp mode, UAC relaunch
internal/
  domain       shared models
  platform     OS interfaces; windows/ implements them (Win32, PDH, WUA, NVML, wevtapi)
  inspect      read-only analysis: system, processes, startup, programs,
               leftovers, crashes, cores, events, dumps, features
  diagnosis    rules: evidence to findings and suspects
  cleanup      safe deletion in known locations
  coretest     the per-core stress test
  recorder     session recording and leak detection
  journal      the change log
  tasks        bounded, cancellable background work for the status bar
  agents       registers the MCP server with agent CLIs
  mcp          the MCP server
  app          services the window and MCP call
frontend/src   Vue shell, Tailwind, one component per page and tab
```

ReforgePulse is free to use and provided as is, without warranty: see
[TERMS.md](TERMS.md).
