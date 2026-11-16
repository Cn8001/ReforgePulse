<p align="center"><img src="build/appicon.png" width="96" alt="ReforgePulse"></p>

<h1 align="center">ReforgePulse</h1>

<p align="center"><b>Diagnose. Explain. Fix. Verify.</b></p>

ReforgePulse is a Windows maintenance and diagnostics application. Its purpose
is to recover as much of a PC's original performance, stability and usable
storage as is realistically possible: it finds what changed, explains what is
wrong in plain language, offers safe corrective actions, and records every
change it makes.

It is not a system tweaker or a collection of administration tools. It uses
the evidence those tools expose to answer practical questions:

- Why does the PC crash or restart on its own?
- Why is it slower than it used to be?
- What is taking up storage, and what can safely be removed?
- Is the cause software, a driver, the configuration, heat, power or the
  hardware itself?

Software cannot repair failing hardware or restore performance lost to
physical wear. ReforgePulse states which findings are confirmed, which are
likely and which it cannot measure, and does not present a correlation as a
cause.

> **Status:** 0.0.1, early development. Windows 11 x64 is the primary target.

## Principles

| Principle | Meaning |
|---|---|
| Evidence first | Every finding names its source: event log records, crash dumps, sensors, performance counters. |
| No placebo | No RAM cleaner, registry cleaner, one-click optimiser or driver installer. |
| Consent | Nothing changes without a preview and an explicit confirmation. |
| Accountability | Every change is written to a local change log before it is applied, with the values before and after. Reversible changes can be reverted from the log. |
| Local only | No account, no telemetry, no uploads. |

## Features

### Issues

A single scan reads what Windows has recorded over the last 30 days: stop
errors, unexpected restarts, WHEA hardware errors, GPU driver resets,
application and service failures, disk and file system errors, firmware
throttling, the microcode revision, tuning drivers, devices without drivers,
and the drivers and updates installed before the first crash.

Each finding carries a severity and a confidence, kept separate. When the
system is unstable, the scan weighs every possible cause in turn (processor,
memory, storage, temperature, power, drivers and software), lists the
evidence for and against each, and names the next check to run.

### Cleanup

| Area | Scope | Removal |
|---|---|---|
| Disk space | Temporary files; browser, application, developer and shader caches; Windows Update and error report data; old logs; the Recycle Bin | Permanent; files in use are skipped |
| Leftovers | Folders in Program Files, ProgramData and AppData that no installed program, process, service or startup entry claims | Recycle Bin |
| Old downloads | Items in Downloads untouched for 180 days | Recycle Bin |
| Programs | Installed programs Windows has not started in 90 days | The program's own uninstaller |
| Startup apps | Programs that start at sign-in, with the memory they use now | Disabled as Task Manager does it; reversible |
| Windows features | Optional features that are rarely needed or carry known risk, such as SMB 1.0 | Reversible |

### Processes

A live process tree with CPU, GPU and memory per process, open TCP and UDP
ports, and applications that have stopped responding. Ending a process
requires confirmation and is logged.

### Tools

| Tool | Purpose |
|---|---|
| Hardware & cores | CPU, motherboard, BIOS, memory modules, GPUs and drives; every logical processor with its live clock, core type, APIC ID and WHEA error count |
| Core test | Stresses selected cores one at a time and reports cores that return wrong results |
| Sensors | GPU readings through the NVIDIA driver; CPU temperatures and voltages through HWiNFO's shared memory when HWiNFO is running |
| Events | System and Application logs grouped by event, counted and explained |
| Minidumps | Stop code and parameters read from each dump; full analysis through the Windows debugger engine when it is installed |
| Recorder | Records load and per-process memory over a session and flags processes with steady memory growth |
| Drivers | Devices with driver problems and the drivers Windows Update offers; installation is left to Windows Update and the manufacturers |
| System checks | `sfc /verifyonly`, `DISM /ScanHealth`, `chkdsk /scan` and Windows Memory Diagnostic |
| CPU cap | A user-defined P-core and E-core frequency limit, applied to every power plan |

## Privacy and network use

ReforgePulse works offline. It uses the network in one case only: when the
user asks the Windows debugger to analyse a crash dump, the debugger downloads
symbol files from Microsoft's public symbol server. The dump itself never
leaves the computer, and the application asks before the download.

All application data, including the change log, is stored in
`%LOCALAPPDATA%\ReforgePulse`.

## Requirements

- Windows 11 x64. Windows 10 is expected to work but is not tested.
- Administrator rights. The application requests elevation at start and runs
  with reduced functionality if it is declined.
- Optional: [HWiNFO](https://www.hwinfo.com/) with Shared Memory Support
  enabled, for CPU temperatures and voltages.
- Optional: WinDbg, for crash dump analysis.

## AI agent integration

`reforgepulse.exe mcp` starts a read-only [Model Context Protocol](https://modelcontextprotocol.io)
server over standard input and output, with the user's normal rights. It can
be registered with Claude Code, Codex or OpenCode from the MCP panel in the
application. Agents can run the scan and read hardware, crash evidence,
processes, startup apps, programs, leftovers, power plans and the change log.
They cannot make changes.

## Building from source

Requires Go, Node.js and the Wails v3 CLI (`wails3`). The project builds on
Windows and under WSL.

```sh
make build              # bin/reforgepulse.exe
make run                # build and start
make test               # Go tests with the race detector
make installer          # NSIS installer
make release            # release assets and SHA256SUMS for the Wails updater
make version V=1.2.3    # set the version in build/config.yml
```

## Project layout

```text
main.go, services.go   wiring: window, services, MCP mode, elevation
internal/domain        shared models
internal/platform      operating system interfaces; windows/ implements them
internal/inspect       read-only analysis: processes, crashes, cores, leftovers and more
internal/diagnosis     rules that turn evidence into findings and suspects
internal/app           services used by the window and the MCP server
internal/journal       the change log
internal/mcp           the MCP server
frontend/              Vue 3, TypeScript and Tailwind CSS
```

## License

Free to use. Provided as is, without warranty of any kind. See
[TERMS.md](TERMS.md).
