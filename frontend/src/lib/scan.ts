import { ref } from 'vue'
import { Last, Scan } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/scanservice'
import type { ScanResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

// One scan state for Home and Fix Issues, so both show the same result.
export const scan = ref<ScanResult | null>(null)
export const scanning = ref(false)
export const scanError = ref('')

export async function runScan() {
  if (scanning.value) return
  scanning.value = true
  scanError.value = ''
  try {
    scan.value = await Scan()
  } catch (e) {
    scanError.value = String(e)
  } finally {
    scanning.value = false
  }
}

export async function loadLastScan() {
  scan.value = (await Last().catch(() => null)) ?? scan.value
}

export const severityTone: Record<string, string> = {
  critical: 'var(--bad)',
  high: '#ff8a00',
  medium: 'var(--warn)',
  low: 'var(--accent)',
  info: 'var(--muted)',
}
