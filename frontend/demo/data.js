// The made-up PC the demo build shows: every value here is invented.
export const GB = 1024 ** 3, MB = 1024 ** 2
export const ago = (s) => new Date(Date.now() - s * 1000).toISOString()
export const m = (value, unit = 'percent') => ({ value, unit, status: 'measured', source: 'mock', at: new Date().toISOString() })
export const jitter = (v, d) => Math.max(0, v + (Math.random() * 2 - 1) * d)

let pid = 1000
export const P = (name, parent, o = {}) => ({
  pid: o.pid ?? (pid += 4 + Math.floor(Math.random() * 40) * 4), parentPid: parent, name,
  path: o.path ?? '', threads: o.threads ?? 12, cpu: o.cpu ?? 0, workingSet: (o.mem ?? 20) * MB * 1.3,
  private: (o.mem ?? 20) * MB, gpu: o.gpu ?? 0, hung: !!o.hung, ports: o.ports ?? null,
  access: o.access ?? true, ours: o.ours, protected: o.protected,
})
export const port = (proto, local, p, remote, state) => ({ proto, local, port: p, remote, state })
export const procs = []
export const add = (...a) => { const p = P(...a); procs.push(p); return p }
export const sys = add('System', 0, { pid: 4, access: false, protected: true, threads: 280, cpu: 0.6 })
add('Registry', 4, { pid: 172, access: false, protected: true })
add('smss.exe', 4, { pid: 584, access: false, protected: true })
add('Memory Compression', 4, { pid: 2412, access: false, protected: true })
export const wininit = add('wininit.exe', 0, { pid: 912, path: 'C:\\Windows\\System32\\wininit.exe', protected: true, mem: 2 })
export const services = add('services.exe', 912, { pid: 1032, path: 'C:\\Windows\\System32\\services.exe', protected: true, mem: 8 })
add('lsass.exe', 912, { pid: 1052, path: 'C:\\Windows\\System32\\lsass.exe', protected: true, mem: 14, ports: [port('tcp', '0.0.0.0', 49664, '', 'listen')] })
export const svcs = [['svchost.exe', 0.2, 18, [port('tcp', '0.0.0.0', 135, '', 'listen')]], ['svchost.exe', 0, 9], ['svchost.exe', 0.4, 42], ['svchost.exe', 0, 6], ['svchost.exe', 0.1, 24], ['svchost.exe', 0, 11], ['svchost.exe', 0, 7], ['spoolsv.exe', 0, 9], ['MsMpEng.exe', 1.2, 310], ['SearchIndexer.exe', 0.1, 38], ['nvcontainer.exe', 0, 22], ['WmiPrvSE.exe', 0.3, 15]]
for (const [n, c, mem, ports] of svcs) add(n, 1032, { path: `C:\\Windows\\System32\\${n}`, cpu: c, mem, ports, protected: n === 'svchost.exe' || n === 'MsMpEng.exe', access: n !== 'MsMpEng.exe' })
add('winlogon.exe', 0, { pid: 1104, path: 'C:\\Windows\\System32\\winlogon.exe', protected: true, mem: 3 })
add('dwm.exe', 1104, { path: 'C:\\Windows\\System32\\dwm.exe', protected: true, cpu: 1.4, gpu: 6.2, mem: 140, threads: 22 })
export const explorer = add('explorer.exe', 7712, { pid: 7840, path: 'C:\\Windows\\explorer.exe', cpu: 0.4, mem: 182, threads: 96 })
export const U = 'C:\\Users\\Alex\\AppData\\Local'
export const chrome = add('chrome.exe', 7840, { path: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe', cpu: 1.1, gpu: 0, mem: 310, threads: 48,
  ports: [port('tcp', '192.168.1.24', 51322, '203.0.113.14:443', 'established'), port('udp', '0.0.0.0', 5353)] })
export const cw = [[3.2, 0.8, 412], [0.6, 4.1, 260], [0.4, 0, 188], [0.9, 0, 236], [0.2, 0, 92], [0, 0, 64], [0.1, 0, 140], [0, 0, 48], [0.3, 0, 120], [0, 0, 36], [0, 0, 22]]
for (const [c, g, mem] of cw) add('chrome.exe', chrome.pid, { path: chrome.path, cpu: c, gpu: g, mem })
export const code = add('Code.exe', 7840, { path: `${U}\\Programs\\Microsoft VS Code\\Code.exe`, cpu: 0.8, gpu: 0.4, mem: 210, threads: 40 })
for (const [c, mem] of [[1.6, 380], [0.3, 160], [0.2, 96], [0, 54], [0.4, 220]]) add('Code.exe', code.pid, { path: code.path, cpu: c, mem })
export const term = add('WindowsTerminal.exe', 7840, { path: 'C:\\Program Files\\WindowsApps\\Microsoft.WindowsTerminal\\WindowsTerminal.exe', cpu: 0.2, mem: 96 })
export const pwsh = add('pwsh.exe', term.pid, { path: 'C:\\Program Files\\PowerShell\\7\\pwsh.exe', mem: 88 })
add('node.exe', pwsh.pid, { path: 'C:\\Program Files\\nodejs\\node.exe', cpu: 2.4, mem: 236, ports: [port('tcp', '127.0.0.1', 5173, '', 'listen'), port('tcp', '0.0.0.0', 3000, '', 'listen')] })
export const steam = add('steam.exe', 7840, { pid: 9316, path: 'C:\\Program Files (x86)\\Steam\\steam.exe', cpu: 0.6, mem: 148, threads: 74,
  ports: [port('tcp', '0.0.0.0', 27036, '', 'listen'), port('udp', '0.0.0.0', 27036), port('tcp', '192.168.1.24', 51410, '203.0.113.80:27019', 'established')] })
for (const [c, g, mem] of [[0.8, 1.2, 168], [0.2, 0, 92], [0.1, 0, 64], [0, 0, 44]]) add('steamwebhelper.exe', steam.pid, { path: 'C:\\Program Files (x86)\\Steam\\bin\\cef\\cef.win7x64\\steamwebhelper.exe', cpu: c, gpu: g, mem })
add('Launcher.exe', steam.pid, { path: 'D:\\SteamLibrary\\steamapps\\common\\Example Game\\Launcher.exe', cpu: 6.8, gpu: 21.4, mem: 2860, threads: 64 })
add('Discord.exe', 7840, { path: `${U}\\Discord\\app-1.0.9188\\Discord.exe`, cpu: 0.5, gpu: 0.3, mem: 290, threads: 52 })
add('Spotify.exe', 7840, { path: 'C:\\Users\\Alex\\AppData\\Roaming\\Spotify\\Spotify.exe', cpu: 0.4, mem: 210 })
export const rp = add('reforgepulse.exe', 7840, { path: 'C:\\Program Files\\ReforgePulse\\reforgepulse.exe', cpu: 0.9, gpu: 0.2, mem: 64, ours: true, protected: true })
add('reforgepulse.exe', rp.pid, { path: rp.path, cpu: 0.3, mem: 18, ours: true, protected: true })
add('OneDrive.exe', 7840, { path: `${U}\\Microsoft\\OneDrive\\OneDrive.exe`, mem: 72 })
add('SecurityHealthSystray.exe', 7840, { path: 'C:\\Windows\\System32\\SecurityHealthSystray.exe', mem: 4 })

export const findings = [
  { rule: 'gpu-resets', key: 'gpu-resets', title: 'The graphics driver stopped responding 3 times', severity: 'medium', confidence: 'confirmed', new: true, status: 'open',
    summary: 'Windows reset the display driver to recover. Repeats usually mean a driver problem, an unstable GPU overclock, or power to the card.',
    evidence: ['2026-11-15 22:41 Display 4101 nvlddmkm', '2026-11-12 21:07 Display 4101 nvlddmkm', '2026-11-04 23:15 Display 4101 nvlddmkm'],
    alternatives: 'A game pushing the card past a stable overclock can cause the same resets.', latest: ago(2 * 86400) },
  { rule: 'app-crash', key: 'app-crash:Launcher.exe', title: 'Launcher.exe in UnityPlayer.dll crashed 4 times', severity: 'low', confidence: 'confirmed', status: 'open',
    summary: 'Updating or reinstalling it usually helps. If many different apps crash, look at system stability instead.', evidence: ['2026-11-15 22:40 WER 1000 Launcher.exe UnityPlayer.dll'] },
  { rule: 'startup', key: 'startup', title: '9 apps start with Windows', severity: 'low', confidence: 'likely', status: 'open',
    summary: 'Together they use 1.4 GB of memory right now and slow sign-in.', evidence: [], action: { label: 'Review startup apps', kind: 'open-startup' } },
  { rule: 'service', key: 'service:Printer Helper', title: 'The Printer Helper service failed 3 times', severity: 'low', confidence: 'confirmed', status: 'open',
    summary: 'It crashed or did not start. If it belongs to a program you removed, its leftovers may still register it.', evidence: ['2026-11-16 08:02 Service Control Manager 7031'] },
  { rule: 'memtest', key: 'memtest', title: 'Windows Memory Diagnostic found no errors', severity: 'info', confidence: 'confirmed', status: 'open',
    summary: 'The last memory test, on 2026-10-30, passed.', evidence: [] },
]
export const suspects = [
  { area: 'drivers', title: 'Drivers', verdict: 'some', for: ['3 display driver resets (4101) in 30 days'], against: [], checks: ['Install the newest graphics driver from Tools > Drivers'] },
  { area: 'power', title: 'Power and voltage', verdict: 'none', for: [], against: ['No unexpected restarts (Kernel-Power 41)'], checks: [] },
  { area: 'heat', title: 'Temperature', verdict: 'none', for: [], against: ['CPU package peaked at 81 °C'], checks: [] },
]
export const scan = () => ({ id: 'r-today', at: ago(41 * 60), findings, suspects, elevated: true,
  health: { score: 87, deductions: [{ key: 'gpu-resets', title: findings[0].title, points: 8 }, { key: 'app-crash:Launcher.exe', title: findings[1].title, points: 2 }, { key: 'startup', title: findings[2].title, points: 2 }, { key: 'service:Printer Helper', title: findings[3].title, points: 1 }] } })

export const changes = () => [
  { id: 'e41b07d', at: ago(3 * 60), action: 'startup', summary: 'Revert: Disable at startup: Spotify', target: 'run:Spotify', before: 'disabled', after: 'enabled', reversible: true, reverts: 'a3f9c21', status: 'applied' },
  { id: 'a3f9c21', at: ago(6 * 60), action: 'startup', summary: 'Disable at startup: Spotify', target: 'run:Spotify', before: 'enabled', after: 'disabled', reversible: true, status: 'reverted' },
  { id: '7be0d14', at: ago(9 * 60), action: 'startup', summary: 'Disable at startup: Steam', target: 'run:Steam', before: 'enabled', after: 'disabled', reversible: true, status: 'applied' },
  { id: '5c82e9f', at: ago(14 * 60), action: 'cleanup', summary: 'Clean up Temporary files, Browser caches', target: '2.4 GB in 18,240 files', reversible: false, status: 'applied' },
  { id: '19c4e8a', at: ago(26 * 3600), action: 'recycle', summary: 'Move 3 leftover folders to the Recycle Bin', target: 'C:\\ProgramData\\Example Tuner', reversible: false, status: 'applied' },
  { id: 'd06a3b2', at: ago(28 * 3600), action: 'cpu-cap', summary: 'Cap CPU frequency: P-cores 4.8 GHz, E-cores 3.9 GHz', target: 'PROCFREQMAX, PROCFREQMAX1', before: 'no cap', after: '4800 / 3900 MHz', reversible: true, status: 'applied' },
  { id: '8f1e5c7', at: ago(3 * 86400), action: 'feature', summary: 'Disable Windows feature SMB1Protocol', target: 'SMB1Protocol', before: 'enabled', after: 'disabled', reversible: true, status: 'applied' },
  { id: '2b9d460', at: ago(4 * 86400), action: 'agent-connect', summary: 'Connect claude to the ReforgePulse MCP server', target: 'claude', status: 'applied' },
  { id: 'f7a0c18', at: ago(5 * 86400), action: 'uninstall', summary: 'Uninstall Example Toolbar', target: 'C:\\Program Files\\Example Toolbar\\uninstall.exe', before: 'version 2.3.1', status: 'applied' },
]

export const nv = 'NVIDIA GeForce RTX 4070'
export const cpuG = 'CPU [#0]: Intel Core i7-13700K: DTS'
export const cpuE = 'CPU [#0]: Intel Core i7-13700K: Enhanced'
export const board = 'Example Boards Z790 GAMING'
export const sensors = () => ({ nvml: '', graphics: '', hwinfo: '', sensors: [
  { source: 'nvml', group: nv, name: 'GPU temperature', kind: 'temp', value: jitter(54, 1), unit: '°C', max: 71 },
  { source: 'nvml', group: nv, name: 'Core clock', kind: 'clock', value: 2475, unit: 'MHz', max: 2790 },
  { source: 'nvml', group: nv, name: 'Memory clock', kind: 'clock', value: 10501, unit: 'MHz', max: 10501 },
  { source: 'nvml', group: nv, name: 'Fan', kind: 'fan', value: 38, unit: '%', max: 61 },
  { source: 'nvml', group: nv, name: 'Board power', kind: 'power', value: jitter(112, 4), unit: 'W', max: 196 },
  { source: 'nvml', group: nv, name: 'GPU load', kind: 'load', value: 34, unit: '%', max: 99 },
  { source: 'hwinfo', group: cpuG, name: 'CPU Package', kind: 'temp', value: jitter(62, 1), unit: '°C', max: 81 },
  { source: 'hwinfo', group: cpuG, name: 'P-core max', kind: 'temp', value: 64, unit: '°C', max: 84 },
  { source: 'hwinfo', group: cpuG, name: 'E-core max', kind: 'temp', value: 55, unit: '°C', max: 72 },
  { source: 'hwinfo', group: cpuE, name: 'Vcore', kind: 'volt', value: 1.284, unit: 'V', max: 1.412 },
  { source: 'hwinfo', group: cpuE, name: 'CPU Package Power', kind: 'power', value: jitter(78, 3), unit: 'W', max: 212 },
  { source: 'hwinfo', group: cpuE, name: 'IA Cores Power', kind: 'power', value: 64, unit: 'W', max: 196 },
  { source: 'hwinfo', group: cpuE, name: 'P-core effective clock', kind: 'clock', value: 5179, unit: 'MHz', max: 5390 },
  { source: 'hwinfo', group: board, name: 'Motherboard', kind: 'temp', value: 38, unit: '°C', max: 44 },
  { source: 'hwinfo', group: board, name: 'VRM', kind: 'temp', value: 52, unit: '°C', max: 68 },
  { source: 'hwinfo', group: board, name: 'CPU fan', kind: 'fan', value: 1150, unit: 'RPM', max: 1720 },
  { source: 'hwinfo', group: board, name: 'System fan 1', kind: 'fan', value: 840, unit: 'RPM', max: 1100 },
  { source: 'hwinfo', group: board, name: '+12V', kind: 'volt', value: 12.096, unit: 'V', max: 12.168 },
  { source: 'hwinfo', group: 'Example NVMe 2TB', name: 'Drive temperature', kind: 'temp', value: 41, unit: '°C', max: 56 },
] })

export const facts = {
  CPU: { Name: 'Intel(R) Core(TM) i7-13700K', Cores: 16, Threads: 24, MaxMHz: 3400 },
  GPUs: [{ Name: nv, DriverVersion: '32.0.15.6614', DriverDate: '2026-10-02' }],
  Memory: [{ Slot: 'DIMM A2', Maker: 'Example', Part: 'EX5-5600-16G', CapacityBytes: 16 * GB, RatedMHz: 5600, ConfiguredMHz: 5600 },
           { Slot: 'DIMM B2', Maker: 'Example', Part: 'EX5-5600-16G', CapacityBytes: 16 * GB, RatedMHz: 5600, ConfiguredMHz: 5600 }],
  Disks: [{ Name: 'Example NVMe 2TB', Media: 'SSD', Health: 'Healthy', Status: 'OK', TempC: 41, TempMaxC: 56, Wear: 3, ReadErrors: 0, WriteErrors: 0, PowerOnHours: 4180 }],
  Hypervisor: false, BIOS: 'Example BIOS 1.40', BIOSDate: '2026-08-12', Board: board,
}

export const areas = [
  { id: 'user-temp', name: 'Temporary files', note: 'Files apps left in your temp folder. Only files untouched for a day.', files: 6120, bytes: 0.82 * GB },
  { id: 'windows-temp', name: 'Windows temporary files', note: 'Leftovers from installers and Windows. Only files untouched for a day.', files: 940, bytes: 0.31 * GB },
  { id: 'browser-cache', name: 'Browser caches', note: 'Chrome, Edge, Brave and Firefox page caches. Logins, history and bookmarks stay. An open browser keeps its files.', files: 8410, bytes: 0.44 * GB },
  { id: 'app-cache', name: 'App caches', note: 'Discord, Slack, Teams, Spotify and VS Code caches. The apps download them again as needed.', files: 2210, bytes: 0.19 * GB },
  { id: 'dev-cache', name: 'Developer caches', note: 'npm, Yarn, pip and Go build caches. The next build downloads or rebuilds what it needs.', files: 0, bytes: 0 },
  { id: 'shader-cache', name: 'GPU shader caches', note: 'Compiled shaders. Games rebuild them, so the first launch after may stutter briefly.', files: 1630, bytes: 0.27 * GB },
  { id: 'error-reports', name: 'Old error reports', note: 'Windows Error Reporting files older than a week. Recent ones stay for Issues.', files: 38, bytes: 0.06 * GB },
  { id: 'windows-logs', name: 'Old Windows logs', note: 'Servicing and setup logs older than a month.', files: 112, bytes: 0.04 * GB },
  { id: 'update-cache', name: 'Windows Update downloads', note: 'Installed update packages. Windows downloads one again if it still needs it.', files: 214, bytes: 0.21 * GB },
  { id: 'delivery-cache', name: 'Delivery Optimization cache', note: 'Update pieces shared with other PCs. Rebuilt on demand.', files: 66, bytes: 0.08 * GB },
  { id: 'recycle-bin', name: 'Recycle Bin', note: 'Everything in your Recycle Bin, on every drive. Deleted for good.', files: 27, bytes: 0.35 * GB },
].map((a) => ({ ...a, paths: [], bytes: Math.round(a.bytes) }))

export const startup = [
  { id: 's1', name: 'Steam', command: '"C:\\Program Files (x86)\\Steam\\steam.exe" -silent', source: 'Run', machine: false, enabled: true, running: true, memory: 470 * MB },
  { id: 's2', name: 'Discord', command: `${U}\\Discord\\Update.exe --processStart Discord.exe`, source: 'Run', machine: false, enabled: true, running: true, memory: 290 * MB },
  { id: 's3', name: 'Spotify', command: 'C:\\Users\\Alex\\AppData\\Roaming\\Spotify\\Spotify.exe /minimized', source: 'Run', machine: false, enabled: true, running: true, memory: 210 * MB },
  { id: 's4', name: 'Microsoft OneDrive', command: `${U}\\Microsoft\\OneDrive\\OneDrive.exe /background`, source: 'Run', machine: false, enabled: true, running: true, memory: 72 * MB },
  { id: 's5', name: 'Example RGB Control', command: 'C:\\Program Files\\Example RGB\\rgbctl.exe --tray', source: 'Run', machine: true, enabled: true, running: true, memory: 184 * MB },
  { id: 's6', name: 'NVIDIA Broadcast', command: 'C:\\Program Files\\NVIDIA Corporation\\NVIDIA Broadcast\\NVIDIA Broadcast UI.exe', source: 'Run', machine: true, enabled: true, running: true, memory: 96 * MB },
  { id: 's7', name: 'Windows Security notification icon', command: '%windir%\\system32\\SecurityHealthSystray.exe', source: 'Run', machine: true, enabled: true, running: true, memory: 4 * MB },
  { id: 's8', name: 'Example Game Launcher', command: 'C:\\Program Files\\Example Launcher\\launcher.exe --autostart', source: 'Startup folder', machine: false, enabled: true, running: true, memory: 118 * MB },
  { id: 's9', name: 'Zoom', command: `${U}\\Zoom\\bin\\Zoom.exe --autostart`, source: 'Run', machine: false, enabled: true, running: false, memory: 0 },
  { id: 's10', name: 'Adobe Updater', command: 'C:\\Program Files (x86)\\Common Files\\Adobe\\ARM\\AdobeARM.exe', source: 'Task', machine: true, enabled: false, running: false, memory: 0 },
]

export const leftovers = [
  { path: 'C:\\ProgramData\\Example Tuner', root: 'ProgramData', size: 412 * MB, files: 318, modified: ago(240 * 86400) },
  { path: 'C:\\Program Files\\Example Toolbar', root: 'Program Files', size: 96 * MB, files: 41, modified: ago(160 * 86400) },
  { path: 'C:\\Users\\Alex\\AppData\\Roaming\\OldChat', root: 'AppData (Roaming)', size: 188 * MB, files: 2210, modified: ago(410 * 86400) },
  { path: `${U}\\ExampleCapture`, root: 'AppData (Local)', size: 1.3 * GB, files: 64, modified: ago(132 * 86400) },
  { path: 'C:\\Program Files (x86)\\Example Benchmark 2024', root: 'Program Files (x86)', size: 640 * MB, files: 1180, modified: ago(300 * 86400) },
  { path: `${U}\\Temp\\ExampleSetup_x64`, root: 'AppData (Local)', size: 72 * MB, files: 12, modified: ago(95 * 86400) },
].map((l) => ({ ...l, size: Math.round(l.size) }))
export const downloads = [
  { path: 'C:\\Users\\Alex\\Downloads\\ExampleGame_Setup_2.1.exe', root: 'Downloads', size: 1.8 * GB, files: 1, modified: ago(380 * 86400) },
  { path: 'C:\\Users\\Alex\\Downloads\\drivers_32.0.15.5585.exe', root: 'Downloads', size: 690 * MB, files: 1, modified: ago(260 * 86400) },
  { path: 'C:\\Users\\Alex\\Downloads\\photos-2025-summer.zip', root: 'Downloads', size: 2.4 * GB, files: 1, modified: ago(420 * 86400) },
].map((l) => ({ ...l, size: Math.round(l.size) }))

export const reports = [
  { id: 'r-today', at: ago(41 * 60), open: 4, resolved: 0, worst: 'medium' },
  { id: 'r2', at: ago(2 * 86400 + 3 * 3600), open: 4, resolved: 1, worst: 'medium' },
  { id: 'r3', at: ago(6 * 86400), open: 5, resolved: 0, worst: 'high' },
  { id: 'r4', at: ago(9 * 86400), open: 3, resolved: 0, worst: 'medium' },
  { id: 'r5', at: ago(14 * 86400), open: 2, resolved: 0, worst: 'low' },
  { id: 'r6', at: ago(17 * 86400), open: 3, resolved: 0, worst: 'medium' },
  { id: 'r7', at: ago(21 * 86400), open: 2, resolved: 1, worst: 'low' },
  { id: 'r8', at: ago(26 * 86400), open: 1, resolved: 0, worst: 'low' },
]
export const analyses = [
  { name: '111526-11203-01.dmp', dumped: ago(2 * 86400), at: ago(2 * 86400 - 600), module: 'nvlddmkm.sys', image: 'nvlddmkm.sys', symbol: 'nvlddmkm+0x1a2b3c', process: 'Launcher.exe',
    bucket: 'AV_nvlddmkm!unknown_function', output: 'BUGCHECK_CODE:  d1\nMODULE_NAME: nvlddmkm\nIMAGE_NAME:  nvlddmkm.sys\nPROCESS_NAME:  Launcher.exe' },
]
export const checkResults = [
  { id: 'k1', name: 'sfc', at: ago(26 * 3600), exitCode: 0, summary: 'No integrity violations.', log: 'C:\\Windows\\Logs\\CBS\\CBS.log' },
  { id: 'k2', name: 'dism', at: ago(26 * 3600 - 900), exitCode: 0, summary: 'No component store corruption detected.', log: 'C:\\Windows\\Logs\\DISM\\dism.log' },
  { id: 'k3', name: 'coretest', at: ago(3 * 86400), exitCode: 0, summary: 'All 24 logical processors agreed on every round. No new WHEA errors.' },
  { id: 'k4', name: 'memtest', at: ago(18 * 86400), exitCode: 0, summary: 'Windows Memory Diagnostic found no errors.', fromWindows: true },
]
export const series = (n, base, d) => Array.from({ length: n }, (_, i) => Math.max(0, base + Math.sin(i / 5) * d + (Math.random() - 0.5) * d))
export const recordings = [
  { id: 'rec1', started: ago(5 * 3600), seconds: 1800, sampleSeconds: 2, cpu: series(900, 18, 8), gpu: series(900, 30, 12), memory: series(900, 44, 2),
    top: [{ name: 'Launcher.exe', pid: 11204, startMb: 2100, endMb: 2860, peakMb: 2890, avgCpu: 6.8, mbPerMinute: 25.3, fit: 0.94, leak: true },
          { name: 'chrome.exe', pid: 8120, startMb: 410, endMb: 430, peakMb: 470, avgCpu: 3.1, mbPerMinute: 0.7, fit: 0.2, leak: false }] },
]
export const agents = [
  { agent: 'claude', installed: true, registered: true, live: true, events: [
    { at: ago(70), pid: 14220, client: 'claude-code', event: 'start' },
    { at: ago(64), pid: 14220, client: 'claude-code', event: 'call', tool: 'get_crash_evidence', millis: 840 },
    { at: ago(40), pid: 14220, client: 'claude-code', event: 'call', tool: 'list_startup_apps', millis: 120 }] },
  { agent: 'codex', installed: true, registered: false, live: false, events: null },
  { agent: 'opencode', installed: false, registered: false, live: false, note: 'opencode was not found on PATH.', events: null },
]

