import { ref } from 'vue'
import { Facts } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/hardwareservice'
import type { HardwareFacts } from '../../bindings/github.com/cn8001/reforgepulse/internal/platform/models'

// Hardware facts take PowerShell a few seconds and do not change while the
// app runs, so they are read once and shared.
export const facts = ref<HardwareFacts | null>(null)
let loading: Promise<void> | null = null

export function loadFacts() {
  loading ??= Facts()
    .then((f) => void (facts.value = f))
    .catch(() => void (loading = null))
  return loading
}
