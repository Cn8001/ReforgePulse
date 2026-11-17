// A stand-in for the Go services, so the real frontend runs in a browser for
// the website's live demo. Keys are "<service file>.<method>" from the
// generated bindings; the build maps them to the method IDs the runtime sends.
// Nothing here touches a PC: changes only update the made-up state below.
import * as d from './data.js'

const { GB, MB, ago, jitter } = d
const T = (x) => () => x
const later = (ms, v) => new Promise((r) => setTimeout(() => r(typeof v === 'function' ? v() : v), ms))

const startup = d.startup.map((a) => ({ ...a }))
const history = d.changes()
let seq = 0x1c3a
const record = (summary, o = {}) => {
  const id = (seq++).toString(16).padStart(4, '0') + 'e7a'
  history.unshift({ id, at: new Date().toISOString(), summary, status: 'applied', ...o })
  return id
}

const dumps = [
  { name: '111526-11203-01.dmp', time: ago(2 * 86400), size: 1.9 * MB, code: '0x000000D1', codeName: 'DRIVER_IRQL_NOT_LESS_OR_EQUAL',
    params: ['0xFFFFB80C2A4D1010', '0x0000000000000002', '0x0000000000000000', '0xFFFFF8053A1A2B3C'], analysis: d.analyses[0] },
  { name: '102926-9875-01.dmp', time: ago(19 * 86400), size: 2.3 * MB, code: '0x00000124', codeName: 'WHEA_UNCORRECTABLE_ERROR',
    params: ['0x0000000000000000', '0xFFFF9A8C3E7F8028', '0x00000000BE000000', '0x00000000800400F0'], analysis: null },
]

const plans = [
  { guid: '381b4222-f694-41f0-9685-ff5bb260df2e', name: 'Balanced', active: true, pCoreAc: 4800, pCoreDc: 4800, eCoreAc: 3900, eCoreDc: 3900 },
  { guid: '8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c', name: 'High performance', active: false, pCoreAc: 4800, pCoreDc: 4800, eCoreAc: 3900, eCoreDc: 3900 },
  { guid: 'a1841308-3541-4fab-bc81-f71556f20b4a', name: 'Power saver', active: false, pCoreAc: 4800, pCoreDc: 4800, eCoreAc: 3900, eCoreDc: 3900 },
]

const events = [
  { channel: 'System', provider: 'Display', id: 4101, level: 3, count: 3, first: ago(13 * 86400), last: ago(2 * 86400),
    message: 'Display driver nvlddmkm stopped responding and has successfully recovered.', explain: 'Windows reset the graphics driver to recover from a hang.', noise: false },
  { channel: 'System', provider: 'Microsoft-Windows-WHEA-Logger', id: 19, level: 3, count: 6, first: ago(20 * 86400), last: ago(9 * 86400),
    message: 'A corrected hardware error has occurred. Component: Processor Core. APIC ID: 32.', explain: 'The processor fixed an error itself. Many of them on one core can point to an unstable core.', noise: false },
  { channel: 'System', provider: 'Service Control Manager', id: 7031, level: 2, count: 3, first: ago(6 * 86400), last: ago(1 * 86400),
    message: 'The Printer Helper service terminated unexpectedly. It has done this 3 time(s).', noise: false },
  { channel: 'Application', provider: 'Application Error', id: 1000, level: 2, count: 4, first: ago(12 * 86400), last: ago(2 * 86400),
    message: 'Faulting application name: Launcher.exe, faulting module name: UnityPlayer.dll', noise: false },
  { channel: 'System', provider: 'Microsoft-Windows-Kernel-General', id: 1, level: 4, count: 40, first: ago(30 * 86400), last: ago(3600),
    message: 'The system time has changed.', noise: false },
  { channel: 'System', provider: 'Microsoft-Windows-DistributedCOM', id: 10016, level: 3, count: 212, first: ago(30 * 86400), last: ago(600),
    message: 'The application-specific permission settings do not grant Local Activation permission.', noise: true },
]

const cores = Array.from({ length: 24 }, (_, i) => i < 16
  ? { index: i, core: i >> 1, kind: 'P', apic: i * 4 + (i & 1) * 1, sibling: i ^ 1, whea: i === 8 ? 6 : 0 }
  : { index: i, core: 8 + (i - 16), kind: 'E', apic: 64 + (i - 16) * 2, sibling: -1, whea: 0 })

const problems = [
  { name: 'PCI Simple Communications Controller', class: 'Other devices', hardwareId: 'PCI\\VEN_8086&DEV_7AE8', deviceId: 'PCI\\VEN_8086&DEV_7AE8&SUBSYS_00000000\\3&11583659&0&B0', errorCode: 28 },
]
const offers = [
  { id: 'u1', title: 'Intel - System - 2442.1.4.0', manufacturer: 'Intel', model: 'Management Engine Interface', class: 'System', date: '2026-09-14', sizeMb: 3.1, hardwareId: 'PCI\\VEN_8086&DEV_7AE8' },
  { id: 'u2', title: 'Realtek Semiconductor Corp. - Extension - 6.0.9768.1', manufacturer: 'Realtek', model: 'Realtek USB Audio', class: 'Extension', date: '2026-08-02', sizeMb: 18.4, hardwareId: 'USB\\VID_0BDA&PID_4014' },
]

const features = [
  { name: 'SMB1Protocol', title: 'SMB 1.0 file sharing', note: 'An old file-sharing protocol. Only very old NAS boxes and printers need it.', enabled: false, pending: '', suggest: false, protected: false },
  { name: 'MicrosoftWindowsPowerShellV2Root', title: 'Windows PowerShell 2.0', note: 'An old PowerShell kept for compatibility. Scripts run on newer versions.', enabled: true, pending: '', suggest: true, protected: false },
  { name: 'Microsoft-Hyper-V-All', title: 'Hyper-V', note: 'Runs virtual machines. It also changes how some games and anti-cheat tools behave.', enabled: false, pending: '', suggest: false, protected: false },
  { name: 'Printing-XPSServices-Features', title: 'XPS services', note: 'Prints to XPS files. Rarely used.', enabled: true, pending: '', suggest: true, protected: false },
]

const programs = [
  { id: 'p1', name: 'Example Benchmark 2024', publisher: 'Example Labs', version: '1.4.2', size: 1.2 * GB, location: 'C:\\Program Files\\Example Benchmark 2024', lastRun: ago(210 * 86400), daysIdle: 210, forgotten: true, basis: 'Prefetch', uninstaller: true },
  { id: 'p2', name: 'Example Photo Editor', publisher: 'Example Software', version: '8.0.1', size: 860 * MB, location: 'C:\\Program Files\\Example Photo Editor', lastRun: ago(140 * 86400), daysIdle: 140, forgotten: true, basis: 'Prefetch', uninstaller: true },
  { id: 'p3', name: 'Zoom', publisher: 'Zoom Video Communications', version: '6.2.5', size: 310 * MB, location: '', lastRun: ago(98 * 86400), daysIdle: 98, forgotten: true, basis: 'Prefetch', uninstaller: true },
]

export const backend = {
  'aboutservice.Accepted': T(true),
  'aboutservice.Accept': T(null),
  'aboutservice.Info': T({ version: __APP_VERSION__, dataDir: 'C:\\Users\\Alex\\AppData\\Local\\ReforgePulse', terms: '', license: '', source: 'https://github.com/Cn8001/ReforgePulse', wording: '' }),
  'themeservice.Accent': T(''),
  'updateservice.CheckOnStart': T(false),
  'updateservice.OnStart': T(null),
  'updateservice.Check': T(null),
  'taskservice.List': T([]),
  'retentionservice.Days': T(0),
  'logservice.Frontend': T(null),

  'systemservice.GetOverview': () => ({ os: 'Windows 11 Pro (build 26100)', cpuModel: d.facts.CPU.Name, gpuModel: d.nv, logicalCpus: 24, elevated: true,
    uptime: d.m(2 * 86400 + 5 * 3600 + 51 * 60, 'seconds'), memoryTotal: d.m(32 * GB, 'bytes'), memoryUsed: d.m(jitter(13.4, 0.1) * GB, 'bytes'),
    volumes: [{ root: 'C:\\', total: d.m(1000 * GB, 'bytes'), free: d.m(412 * GB, 'bytes') }, { root: 'D:\\', total: d.m(2000 * GB, 'bytes'), free: d.m(1180 * GB, 'bytes') }] }),
  'systemservice.GetLive': () => ({ cpu: d.m(jitter(15, 6)), gpu: d.m(jitter(34, 9)), memory: d.m(jitter(43, 0.4)) }),

  'hardwareservice.Facts': T(d.facts),
  'hardwareservice.Loads': () => Array.from({ length: 24 }, (_, i) => ({ index: i, load: jitter(i < 16 ? 18 : 8, 8),
    mhz: i === 4 ? Math.round(jitter(5179, 40)) : i < 16 ? Math.round(jitter(4700, 300)) : Math.round(jitter(3900, 200)) })),
  'hardwareservice.Cores': T(cores),
  'hardwareservice.TestCores': (cpus, seconds) => later(Math.min(seconds, 6) * 1000, () => ({ newWhea: 0, seconds,
    results: (cpus ?? cores.map((c) => c.index)).map((cpu) => ({ cpu, rounds: 40, mismatches: 0, ok: true })) })),
  'sensorservice.Read': d.sensors,

  'scanservice.Last': d.scan,
  'scanservice.Report': d.scan,
  'scanservice.Reports': T(d.reports),
  'scanservice.Scan': () => later(2500, d.scan),
  'scanservice.Resolve': T(null),
  'scanservice.Reopen': T(null),

  'historyservice.List': () => history,
  'historyservice.Revert': (id) => {
    const c = history.find((h) => h.id === id)
    if (c) { c.status = 'reverted'; record(`Revert: ${c.summary}`, { reverts: id, action: c.action }) }
    return null
  },

  'processservice.List': () => d.procs.map((p) => ({ ...p, cpu: p.cpu ? jitter(p.cpu, p.cpu * 0.15) : 0 })),
  'processservice.Icons': T({}),
  'processservice.End': T(null),

  'cleanupservice.Analyze': () => later(600, d.areas),
  'cleanupservice.Clean': (ids) => {
    const picked = d.areas.filter((a) => (ids ?? []).includes(a.id))
    const freed = picked.reduce((s, a) => s + a.bytes, 0)
    const changeId = record(`Clean up ${picked.map((a) => a.name).join(', ')}`, { action: 'cleanup', reversible: false })
    return later(1500, { changeId, deleted: picked.reduce((s, a) => s + a.files, 0), freed, skipped: 0 })
  },
  'startupservice.List': () => startup,
  'startupservice.SetEnabled': (id, name, enabled) => {
    const a = startup.find((s) => s.id === id)
    if (a) a.enabled = enabled
    record(`${enabled ? 'Enable' : 'Disable'} at startup: ${name}`, { action: 'startup', reversible: true })
    return null
  },
  'leftoverservice.List': T(d.leftovers),
  'leftoverservice.Recycle': T(null),
  'oldfilesservice.List': T(d.downloads),
  'oldfilesservice.Recycle': T(null),
  'programservice.List': T(programs),
  'featureservice.List': T(features),

  'dumpservice.List': T(dumps),
  'dumpservice.HasDebugger': T(true),
  'dumpservice.Analyses': T(d.analyses),
  'dumpservice.Analyze': (name) => later(3000, () => ({ ...d.analyses[0], name })),
  'eventservice.List': (days, warnings) => events.filter((e) => warnings || e.level <= 2),
  'driverservice.Problems': T(problems),
  'driverservice.Offers': () => later(800, offers),
  'checkservice.Results': T(d.checkResults),
  'checkservice.Run': (name) => later(4000, () => ({ id: `k-${name}`, name, at: new Date().toISOString(), exitCode: 0,
    summary: name === 'sfc' ? 'No integrity violations.' : name === 'dism' ? 'No component store corruption detected.' : 'No problems found.' })),
  'cpucapservice.Plans': () => plans,
  'cpucapservice.Apply': (p, e) => {
    for (const pl of plans) { pl.pCoreAc = pl.pCoreDc = p; pl.eCoreAc = pl.eCoreDc = e }
    record(`Cap CPU frequency: P-cores ${(p / 1000).toFixed(1)} GHz, E-cores ${(e / 1000).toFixed(1)} GHz`, { action: 'cpu-cap', reversible: true })
    return null
  },
  'recorderservice.Recordings': T(d.recordings),
  'recorderservice.Recording': (id) => d.recordings.find((r) => r.id === id) ?? d.recordings[0],

  'agentservice.Status': T(d.agents),
  'agentservice.Executable': T('C:\\Program Files\\ReforgePulse\\reforgepulse.exe'),
  'agentservice.Connect': T(''),
}
