// What the Features page shows. Images are files next to the built site in
// docs/, the same ones the README uses; every screenshot shows a made-up PC.

export interface Shot {
  src: string;
  label: string;
  alt: string;
}

export interface Feature {
  id: string;
  title: string;
  body: string[];
  facts?: string[];
  shots: Shot[];
}

export interface Group {
  title: string;
  features: Feature[];
}

export const groups: Group[] = [
  {
    title: "Find the problem",
    features: [
      {
        id: "overview",
        title: "Overview",
        body: [
          "The first screen shows a health score out of 100, the issues that deserve your attention and the live load on the processor, graphics card and memory.",
          "The app follows the Windows accent colour and offers a light theme as well as the dark one.",
        ],
        shots: [
          { src: "overview.png", label: "Dark", alt: "Overview: a health score of 87, three issues with their severity, and live processor, graphics and memory load" },
          { src: "overview-light.png", label: "Light", alt: "The same overview in the light theme" },
        ],
      },
      {
        id: "issues",
        title: "Crashes and restarts",
        body: [
          "Scan PC reviews the last 30 days of crashes, unexpected restarts, hardware errors (WHEA), graphics driver resets, disk errors, failing services and devices without drivers.",
          "When a PC is unstable, it weighs each possible cause, such as the processor, memory, storage, heat, power, drivers or software, against the evidence for and against it. Where Windows cannot measure something, it says so rather than calling it fine.",
        ],
        facts: ["WER 1001", "Kernel-Power 41", "WHEA-Logger", "Display 4101"],
        shots: [{ src: "issues.png", label: "Issues", alt: "Issues: each finding with its severity and confidence, with buttons to mark it resolved or delete it" }],
      },
      {
        id: "events",
        title: "Events",
        body: [
          "The System and Application logs, grouped by source and event ID with a count, the first and last time and the message Windows wrote.",
          "Entries known to be harmless, such as DCOM 10016, are hidden unless you ask for them.",
        ],
        facts: ["EvtFormatMessage"],
        shots: [{ src: "events.png", label: "Events", alt: "Events: grouped log entries with how many times each happened and when" }],
      },
      {
        id: "minidumps",
        title: "Minidumps",
        body: [
          "Every blue screen leaves a minidump. ReforgePulse reads the stop code and its parameters straight from the file, so you see what happened even without a debugger.",
          "If WinDbg or the Windows SDK debuggers are installed, Analyse runs the debugger to name the driver involved. The symbols it needs are downloaded during that analysis.",
        ],
        facts: ["PAGEDU64 header", "cdb !analyze -v"],
        shots: [{ src: "minidumps.png", label: "Minidumps", alt: "Minidumps: two crash dumps with their stop codes and parameters, one analysed and naming nvlddmkm.sys" }],
      },
      {
        id: "hardware",
        title: "Hardware and sensors",
        body: [
          "The motherboard, BIOS, memory modules (rated against actual speed), graphics card and drives with their health, plus live temperatures, clocks, power and fan speeds with the highest value seen.",
          "Each logical processor is listed with its live speed and APIC ID, so a hardware error that names APIC 32 is shown as logical processor 8, performance core 4. A core test checks the cores one at a time.",
        ],
        facts: ["CPUID", "NVML", "HWiNFO shared memory"],
        shots: [
          { src: "tools.png", label: "Hardware", alt: "Hardware: processor, motherboard, graphics, drives and memory modules in detail" },
          { src: "sensors.png", label: "Sensors", alt: "Sensors: graphics card and processor temperatures, clocks, voltages, power and fan speeds with their highest values" },
        ],
      },
      {
        id: "recorder",
        title: "Recorder",
        body: [
          "Records processor, graphics and memory load and the memory of every process every 2 seconds while you work or play, then shows what happened.",
          "A process whose memory kept growing steadily is flagged as a likely leak. A single jump is not.",
        ],
        shots: [{ src: "recorder.png", label: "Recording", alt: "A 30-minute recording with load graphs and a process flagged as a likely leak" }],
      },
      {
        id: "checks",
        title: "Checks",
        body: [
          "Runs the health checks built into Windows: system files, the Windows image, the system drive and memory. They only read and report; none of them repairs anything.",
          "Only the verdict is kept. The full log stays where Windows writes it, one click away.",
        ],
        facts: ["sfc /verifyonly", "DISM /ScanHealth", "chkdsk /scan", "Windows Memory Diagnostic"],
        shots: [{ src: "checks.png", label: "Checks", alt: "Checks: system files and Windows image checks with their last results, and the drive and memory checks ready to run" }],
      },
    ],
  },
  {
    title: "Clean up and tune",
    features: [
      {
        id: "cleanup",
        title: "Cleanup",
        body: [
          "Temporary files, browser and app caches, shader caches, leftover Windows Update files and old logs, each with its size before anything is removed.",
          "Folders left behind by programs you removed are matched against what is still installed. Old downloads and leftover folders go to the Recycle Bin, so you can still get them back.",
        ],
        shots: [
          { src: "cleanup.png", label: "Disk space", alt: "Cleanup: 2.8 GB can be freed, with 2.4 GB selected across temporary files, caches and update leftovers" },
          { src: "leftovers.png", label: "Leftovers", alt: "Leftovers: 2.7 GB of folders left behind by removed programs, three selected for the Recycle Bin" },
        ],
      },
      {
        id: "startup",
        title: "Startup apps",
        body: [
          "Every app that starts with Windows, with the memory it uses right now.",
          "Turning one off works the way Task Manager does it, and it can be turned back on at any time.",
        ],
        facts: ["StartupApproved"],
        shots: [{ src: "startup.png", label: "Startup apps", alt: "Startup apps: nine apps start at sign-in and use 1.4 GB of memory, each with a switch to turn it off" }],
      },
      {
        id: "processes",
        title: "Processes",
        body: [
          "Every running program grouped under the one that started it, with its processor, graphics and memory use and any network ports it holds.",
          "To find the program that uses a port, type it into the search box, for example :8080. Windows' own processes are protected from End task.",
        ],
        shots: [{ src: "processes.png", label: "Processes", alt: "Processes: a tree of running programs, with Steam selected showing its path, open ports and child processes" }],
      },
      {
        id: "drivers",
        title: "Drivers",
        body: [
          "Devices with a problem, and the drivers Windows Update offers for this PC.",
          "ReforgePulse does not install drivers itself. It opens Windows Update or the maker's site, who know their drivers best.",
        ],
        facts: ["Windows Update Agent"],
        shots: [{ src: "drivers.png", label: "Drivers", alt: "Drivers: one device without a driver, Windows Update offers and links to the makers' driver pages" }],
      },
      {
        id: "cpu-cap",
        title: "CPU cap",
        body: [
          "Limits the highest clock of the performance and efficiency cores on every power plan, so switching plans cannot undo it.",
          "Use it to test whether lower clocks stop the crashes. The old values are kept, so the cap can be reverted from History.",
        ],
        facts: ["PROCFREQMAX", "PROCFREQMAX1"],
        shots: [{ src: "cpucap.png", label: "CPU cap", alt: "CPU cap: performance cores capped at 4800 MHz and efficiency cores at 3900 MHz on every power plan" }],
      },
    ],
  },
  {
    title: "Keep track",
    features: [
      {
        id: "history",
        title: "Change log",
        body: [
          "Every change is recorded before and after it is made, like a git history. Where a change can be undone, the log offers a Revert button, and the revert is recorded as a change of its own.",
          "The log keeps its last 500 entries.",
        ],
        shots: [{ src: "history.png", label: "History", alt: "History: a list of changes with short IDs, Revert buttons on changes that can be undone, and one change already reverted" }],
      },
      {
        id: "reports",
        title: "Reports",
        body: [
          "Every scan is saved as a report, along with crash dump analyses, recordings and check results.",
          "Each one opens again, downloads as a Markdown file or can be deleted. The last 30 scans, 20 recordings and 50 check results are kept.",
        ],
        shots: [{ src: "reports.png", label: "Reports", alt: "Reports: eight saved scans with their open and resolved findings, each to open, download or delete" }],
      },
      {
        id: "agents",
        title: "Coding agents",
        body: [
          "ReforgePulse includes a read-only MCP server for Claude Code, Codex and OpenCode. Connect one from the Agents panel and see each tool it called.",
          "Agents can run a scan and read the results, but any change still has to be approved by you in the ReforgePulse window. The server runs without administrator rights.",
        ],
        facts: ["reforgepulse.exe mcp"],
        shots: [{ src: "agents.png", label: "Agents", alt: "The Connect an AI agent panel: Claude Code connected with its recent tool calls, Codex and OpenCode ready to connect" }],
      },
    ],
  },
];

export const allFeatures = groups.flatMap((g) => g.features);

export const repo = "https://github.com/Cn8001/ReforgePulse";
export const installer = `${repo}/releases/latest/download/reforgepulse-windows-amd64-installer.exe`;
export const version = __APP_VERSION__;
