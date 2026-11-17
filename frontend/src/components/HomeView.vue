<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import Bar from './Bar.vue'
import Tile from './Tile.vue'
import Sparkline from './Sparkline.vue'
import ClockTemp from './ClockTemp.vue'
import { Loads } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/hardwareservice'
import { Read as ReadSensors } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/sensorservice'
import type { CoreLoad, SensorStatus } from '../../bindings/github.com/cn8001/reforgepulse/internal/platform/models'
import { facts, loadFacts } from '../lib/hardware'
import { GetLive, GetOverview } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/systemservice'
import { List, Revert } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/historyservice'
import ConfirmDialog from './ConfirmDialog.vue'
import { active, loadLastScan, runScan, scan, scanning, severityTone } from '../lib/scan'
import { Availability, type Change, type Measurement, type Overview } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes, formatDuration, measured, show, since } from '../lib/format'

const emit = defineEmits<{ elevated: [boolean]; open: [page: string, sort?: string] }>()
const overview = ref<Overview | null>(null)
const changes = ref<Change[]>([])
const error = ref('')
const loading = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [o, c] = await Promise.all([GetOverview(), List()])
    overview.value = o
    changes.value = c ?? []
    emit('elevated', o.elevated)
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

// One sample a second, kept for a minute. Polling pauses while the window
// is hidden so ReforgePulse costs nothing in the background.
const history = ref({ cpu: [] as number[], gpu: [] as number[], memory: [] as number[] })
const now = ref<Record<'cpu' | 'gpu' | 'memory', Measurement | null>>({ cpu: null, gpu: null, memory: null })
let timer: ReturnType<typeof setInterval> | undefined
// Clocks come with every sample; sensors every other one, which is plenty
// for temperatures and keeps NVML and HWiNFO reads rare.
const cores = ref<CoreLoad[]>([])
const sensors = ref<SensorStatus | null>(null)
let tick = 0
async function sample() {
  if (document.hidden) return
  Loads().then((l) => (cores.value = l ?? [])).catch(() => {})
  if (tick++ % 2 === 0) ReadSensors().then((st) => (sensors.value = st)).catch(() => {})
  try {
    const l = await GetLive()
    for (const k of ['cpu', 'gpu', 'memory'] as const) {
      now.value[k] = l[k]
      if (l[k].status === Availability.Measured) history.value[k] = [...history.value[k], l[k].value].slice(-60)
    }
  } catch {} // a missed sample just leaves a gap
}

onMounted(() => {
  load()
  loadFacts()
  loadLastScan()
  sample()
  timer = setInterval(sample, 1000)
})
onBeforeUnmount(() => clearInterval(timer))

const ghz = (mhz: number) => (mhz ? `${(mhz / 1000).toFixed(2)} GHz` : '')
const sensor = (pick: (s: NonNullable<SensorStatus['sensors']>[number]) => boolean) => sensors.value?.sensors?.find(pick)

// The fastest core is what "CPU speed" means to most people; the average
// hides a boosting core behind idle ones.
const cpuClock = computed(() => ghz(Math.max(0, ...cores.value.map((c) => c.mhz))))
const cpuTemp = computed(() => {
  const temps = (sensors.value?.sensors ?? []).filter((s) => s.source === 'hwinfo' && s.kind === 'temp' && /cpu/i.test(s.group))
  const pkg = temps.find((s) => /package|tctl|tdie/i.test(s.name))
  return pkg?.value ?? (temps.length ? Math.max(...temps.map((s) => s.value)) : null)
})
const gpuClock = computed(() => {
  const c = sensor((s) => s.source === 'nvml' && s.name === 'Core clock')
  return c ? ghz(c.value) : ''
})
const gpuTemp = computed(() => sensor((s) => s.kind === 'temp' && s.source === 'nvml')?.value ?? null)
const memClock = computed(() => {
  const m = facts.value?.Memory?.[0]
  return m ? `${m.ConfiguredMHz} MT/s` : ''
})

const pct = (m: Measurement | null) => (m?.status === Availability.Measured ? Math.round(m.value).toString() : '--')

const memoryDetail = computed(() => {
  const o = overview.value
  if (!o || !measured(o.memoryTotal, o.memoryUsed)) return show(o?.memoryTotal, formatBytes)
  return `${formatBytes(o.memoryUsed.value)} / ${formatBytes(o.memoryTotal.value)}`
})

const drives = computed(() =>
  (overview.value?.volumes ?? []).map((v) => {
    if (!measured(v.total, v.free)) return { root: v.root, ratio: null, used: '--', detail: show(v.total, formatBytes) }
    const ratio = (v.total.value - v.free.value) / v.total.value
    return { root: v.root, ratio, used: Math.round(ratio * 100).toString(), detail: `${formatBytes(v.free.value)} free / ${formatBytes(v.total.value)}` }
  }),
)

// Up to three findings worth acting on; info-only ones stay on Issues.
const top = computed(() => (scan.value?.findings ?? []).filter((f) => f.severity !== 'info' && active(f)).slice(0, 3))
const issues = computed(() => (scan.value?.findings ?? []).filter((f) => f.severity !== 'info' && active(f)).length)

const reverting = ref<Change | null>(null)
const busy = ref(false)
async function revert() {
  if (!reverting.value) return
  busy.value = true
  try {
    await Revert(reverting.value.id)
    changes.value = (await List()) ?? []
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    reverting.value = null
  }
}

const statusText: Record<string, [string, string]> = {
  applied: ['Applied', 'text-fg'],
  failed: ['Failed', 'text-bad'],
  reverted: ['Reverted', 'text-muted line-through'],
  started: ['Interrupted', 'text-warn'],
}

const headline = computed(() => {
  if (scanning.value) return 'Scanning…'
  if (!scan.value) return 'Not scanned yet'
  if (!issues.value) return 'No urgent issues found'
  return issues.value === 1 ? '1 issue needs attention' : `${issues.value} issues need attention`
})
</script>

<template>
  <section class="mx-auto flex max-w-6xl flex-col gap-5 p-6">
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="heading">{{ headline }}</h1>
        <p class="mt-1 text-muted">
          {{ scan ? `Last scan ${since(scan.at)}.` : 'A scan reads crash records, hardware errors, drives and memory. It changes nothing.' }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button class="btn size-9 px-0" title="Refresh" aria-label="Refresh" :disabled="loading" @click="load">
          <Icon name="refresh" class="size-4!" :class="loading && 'animate-spin'" />
        </button>
        <button class="btn-primary h-9 px-5" :disabled="scanning" @click="runScan">{{ scanning ? 'Scanning…' : 'Scan PC' }}</button>
      </div>
    </header>

    <div v-if="overview && !overview.elevated" class="notice flex items-center gap-3 border-warn/40 bg-warn/10">
      <Icon name="shield" class="text-warn" />
      ReforgePulse is running without administrator rights, so some event logs and every repair are unavailable.
    </div>

    <ul v-if="top.length" class="panel divide-y divide-line">
      <li v-for="(f, i) in top" :key="i">
        <button class="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-hover" @click="emit('open', 'fix')">
          <span class="size-2 shrink-0 rounded-full" :style="{ background: severityTone[f.severity] }" />
          <span class="flex-1 truncate">{{ f.title }}</span>
          <span class="text-xs text-muted capitalize">{{ f.severity }}</span>
          <Icon name="chevron" class="size-3.5! text-muted" />
        </button>
      </li>
      <li v-if="issues > top.length">
        <button class="w-full px-4 py-2 text-left text-xs text-muted transition hover:bg-hover hover:text-fg" @click="emit('open', 'fix')">
          {{ issues - top.length }} more in Issues
        </button>
      </li>
    </ul>

    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <Tile title="Processor" icon="cpu" :value="pct(now.cpu)" unit="%" :sub="overview ? overview.cpuModel || 'Unknown processor' : ''"
            class="cursor-pointer" @click="emit('open', 'processes', 'cpu')">
        <ClockTemp :clock="cpuClock" :temp="cpuTemp"
                   missing="Windows has no reliable CPU temperature. Run HWiNFO with Shared Memory Support on and it shows here." />
        <Sparkline :values="history.cpu" color="var(--accent)" />
      </Tile>
      <Tile title="Graphics" icon="gpu" :value="pct(now.gpu)" unit="%" :sub="overview?.gpuModel || 'Unknown graphics card'"
            class="cursor-pointer" @click="emit('open', 'processes', 'gpu')">
        <ClockTemp :clock="gpuClock" :temp="gpuTemp" :missing="sensors?.nvml || 'No graphics sensors found.'" />
        <Sparkline :values="history.gpu" color="var(--accent)" />
      </Tile>
      <Tile title="Memory" icon="memory" :value="pct(now.memory)" unit="%" :sub="memoryDetail"
            class="cursor-pointer" @click="emit('open', 'processes', 'private')">
        <ClockTemp :clock="memClock" />
        <Sparkline :values="history.memory" color="var(--accent)" />
      </Tile>

      <Tile title="Drives" icon="disk" class="cursor-pointer md:col-span-2" @click="emit('open', 'cleanup')">
        <table class="w-full text-[14px] tabular-nums">
          <tr v-for="d in drives" :key="d.root">
            <td class="w-12 py-1.5 font-mono">{{ d.root.replace('\\', '') }}</td>
            <td class="py-1.5"><Bar :ratio="d.ratio" /></td>
            <td class="w-14 py-1.5 text-right">{{ d.used }}%</td>
            <td class="w-44 py-1.5 text-right text-muted">{{ d.detail }}</td>
          </tr>
        </table>
        <p v-if="!drives.length" class="text-muted">No fixed drives found.</p>
      </Tile>
      <Tile title="System" icon="system" :sub="overview?.os || ''">
        <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-[14px]">
          <dt class="text-muted">Running for</dt><dd class="tabular-nums">{{ overview ? show(overview.uptime, formatDuration) : '–' }}</dd>
          <dt class="text-muted">Rights</dt>
          <dd :class="overview && !overview.elevated ? 'text-warn' : ''">{{ overview ? (overview.elevated ? 'Administrator' : 'Limited') : '–' }}</dd>
          <dt class="text-muted">Network</dt><dd>Updates and debug symbols</dd>
        </dl>
      </Tile>
    </div>

    <article class="panel">
      <header class="flex items-baseline justify-between border-b border-line px-4 py-3">
        <h2 class="text-[14px] font-semibold">Recent changes</h2>
        <button v-if="changes.length" class="text-xs text-muted hover:text-fg hover:underline" @click="emit('open', 'history', 'changes')">
          {{ changes.length > 5 ? `Show all ${changes.length}` : 'Open history' }}
        </button>
      </header>
      <p v-if="!changes.length" class="px-4 py-4 text-muted">Nothing has been changed yet. Every change ReforgePulse makes is recorded here before it happens.</p>
      <table v-else class="w-full text-[14px]">
        <tr v-for="c in changes.slice(0, 5)" :key="c.id" class="border-b border-line last:border-0 hover:bg-hover">
          <td class="w-24 px-4 py-2 font-mono text-xs text-muted">{{ c.id }}</td>
          <td class="truncate py-2" :title="c.before && c.after ? `${c.before} → ${c.after}` : ''">{{ c.summary }}</td>
          <td class="w-24 py-2" :class="statusText[c.status]?.[1]">{{ statusText[c.status]?.[0] ?? c.status }}</td>
          <td class="w-28 py-2 text-right text-muted">{{ since(c.at) }}</td>
          <td class="w-24 px-4 py-2 text-right">
            <button v-if="c.reversible && c.status === 'applied'" class="text-accent hover:underline" @click="reverting = c">Revert</button>
          </td>
        </tr>
      </table>
    </article>

    <ConfirmDialog :open="!!reverting" :title="`Revert “${reverting?.summary}”?`" confirm="Revert" :busy="busy" @yes="revert" @no="reverting = null">
      This goes back from <b class="text-fg">{{ reverting?.after }}</b> to <b class="text-fg">{{ reverting?.before }}</b>. The revert is recorded as a new change.
    </ConfirmDialog>
  </section>
</template>
