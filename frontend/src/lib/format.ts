import { Availability, type Measurement } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

const missingLabel: Record<string, string> = {
  [Availability.NotMeasured]: 'Not measured',
  [Availability.Unsupported]: 'Unsupported',
  [Availability.Unknown]: 'Unknown',
}

export function formatBytes(n: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${n.toFixed(i >= 3 ? 1 : 0)} ${units[i]}`
}

export function formatDuration(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return d > 0 ? `${d}d ${h}h` : h > 0 ? `${h}h ${m}m` : `${m}m`
}

// A value that was not measured must never read as zero.
export function show(m: Measurement | undefined, fmt: (v: number) => string): string {
  if (!m || m.status !== Availability.Measured) return missingLabel[m?.status ?? ''] ?? 'Unknown'
  return fmt(m.value)
}

export function measured(...ms: (Measurement | undefined)[]): boolean {
  return ms.every((m) => m?.status === Availability.Measured)
}

export function since(iso: string): string {
  const s = (Date.now() - new Date(iso).getTime()) / 1000
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)} min ago`
  if (s < 86400) return `${Math.floor(s / 3600)} h ago`
  return new Date(iso).toLocaleDateString()
}

// A date and time short enough for a list row, in the user's locale.
export function when(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}
