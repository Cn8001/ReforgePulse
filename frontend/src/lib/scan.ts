import { ref } from 'vue'
import { Last, Reopen, Report, Reports, Resolve, Scan } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/scanservice'
import type { Finding, ReportSummary, ScanResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

// One scan state for Home and Issues, so both show the same result. An older
// report opened on Issues goes in viewed, so Home keeps the latest.
export const scan = ref<ScanResult | null>(null)
export const viewed = ref<ScanResult | null>(null)
export const reports = ref<ReportSummary[]>([])
export const scanning = ref(false)
export const scanError = ref('')

export async function runScan() {
  if (scanning.value) return
  scanning.value = true
  scanError.value = ''
  try {
    scan.value = await Scan()
    viewed.value = null
    await loadReports()
  } catch (e) {
    scanError.value = String(e)
  } finally {
    scanning.value = false
  }
}

export async function loadLastScan() {
  scan.value = (await Last().catch(() => null)) ?? scan.value
}

export async function loadReports() {
  reports.value = (await Reports().catch(() => null)) ?? reports.value
}

export async function openReport(id: string) {
  viewed.value = id === scan.value?.id ? null : await Report(id)
}

// Still needing attention: open, or resolved and then seen again.
export const active = (f: Finding) => f.status !== 'resolved'

// Resolve and Reopen change every report's statuses, so all are reread.
export async function setResolved(f: Finding, resolved: boolean) {
  await (resolved ? Resolve(f.key, f.title) : Reopen(f.key))
  const id = viewed.value?.id
  await Promise.all([loadLastScan(), loadReports(), id ? Report(id).then((r) => (viewed.value = r)) : null])
}

// Red and amber are spent only on what needs attention; the signal colour
// stays for live data, so low and info findings are grey.
export const severityTone: Record<string, string> = {
  critical: 'var(--bad)',
  high: 'var(--bad)',
  medium: 'var(--warn)',
  low: 'var(--muted)',
  info: 'var(--muted)',
}
