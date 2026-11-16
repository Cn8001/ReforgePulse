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
import { loadLastScan, runScan, scan, scanning, severityTone } from '../lib/scan'
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
const top = computed(() => (scan.value?.findings ?? []).filter((f) => f.severity !== 'info').slice(0, 3))
const issues = computed(() => (scan.value?.findings ?? []).filter((f) => f.severity !== 'info').length)

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

const statusTone: Record<string, string> = {
  applied: 'text-ok',
  failed: 'text-bad',
  reverted: 'text-muted line-through',
  started: 'text-warn',
}
</script>

<template>
  <section class="flex flex-col gap-5 p-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="caption">status ·
          <span v-if="scanning" class="text-accent">scanning…</span>
          <span v-else-if="!scan" class="text-warn">not scanned</span>
          <span v-else-if="issues" class="text-warn">{{ issues }} issues need attention</span>
          <span v-else class="text-ok">no urgent issues found</span>
        </p>
        <h1 class="mt-1 text-2xl font-semibold">System overview</h1>
      </div>
      <div class="flex items-center gap-2">
        <button class="flex size-10 items-center justify-center border border-line bg-panel transition hover:bg-raised disabled:opacity-50"
                title="Refresh" :disabled="loading" @click="load">
          <Icon name="refresh" :class="loading && 'animate-spin'" />
        </button>
        <button :disabled="scanning" @click="runScan"
                class="h-10 bg-accent px-6 font-mono text-xs font-bold tracking-[.2em] text-white uppercase shadow-[0_0_24px_-6px_var(--accent)] transition hover:brightness-110 active:translate-y-px disabled:opacity-60">
          {{ scanning ? 'Scanning' : 'Scan PC' }}
        </button>
      </div>
    </header>

    <div v-if="overview && !overview.elevated" class="flex items-center gap-3 border border-warn/40 bg-warn/10 px-4 py-2.5 font-mono text-xs text-warn">
      <Icon name="shield" />
      WARN: running without administrator rights; event logs and repairs are limited.
    </div>

    <div v-if="top.length" class="flex flex-col border border-line bg-panel">
      <button v-for="(f, i) in top" :key="i" class="flex items-center gap-3 border-b border-line px-4 py-2.5 text-left transition last:border-0 hover:bg-hover"
              @click="emit('open', 'fix')">
        <span class="h-4 w-[3px]" :style="{ background: severityTone[f.severity] }" />
        <span class="flex-1 truncate">{{ f.title }}</span>
        <span class="font-mono text-[10px] text-muted uppercase">{{ f.severity }} · {{ f.confidence }}</span>
        <Icon name="chevron" class="size-3.5! text-muted" />
      </button>
    </div>

    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

    <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <Tile title="CPU" icon="cpu" tint="var(--accent)" :value="pct(now.cpu)" unit="%" class="cursor-pointer" @click="emit('open', 'processes', 'cpu')"
            :sub="overview ? `${overview.cpuModel || 'unknown'} · ${overview.logicalCpus} threads` : ''">
        <ClockTemp :clock="cpuClock" :temp="cpuTemp"
                   missing="Windows has no reliable CPU temperature. Run HWiNFO with Shared Memory Support on and it shows here." />
        <Sparkline :values="history.cpu" color="var(--accent)" />
      </Tile>
      <Tile title="GPU" icon="gpu" tint="var(--gpu)" :value="pct(now.gpu)" unit="%" :sub="overview?.gpuModel || 'unknown'"
            class="cursor-pointer" @click="emit('open', 'processes', 'gpu')">
        <ClockTemp :clock="gpuClock" :temp="gpuTemp" :missing="sensors?.nvml || 'No NVIDIA GPU sensors.'" />
        <Sparkline :values="history.gpu" color="var(--gpu)" />
      </Tile>
      <Tile title="Memory" icon="memory" tint="var(--mem)" :value="pct(now.memory)" unit="%" :sub="memoryDetail"
            class="cursor-pointer" @click="emit('open', 'processes', 'private')">
        <ClockTemp :clock="memClock" />
        <Sparkline :values="history.memory" color="var(--mem)" />
      </Tile>
      <Tile title="Disks" icon="disk" tint="var(--disk)" class="cursor-pointer md:col-span-2" @click="emit('open', 'cleanup')">
        <template #badge><span class="ml-auto font-mono text-[11px] text-muted">{{ drives.length }} fixed</span></template>
        <div class="flex flex-col gap-2.5">
          <div v-for="d in drives" :key="d.root" class="grid grid-cols-[2.5rem_1fr_3rem_9rem] items-center gap-3 font-mono text-xs">
            <span class="text-disk">{{ d.root.replace('\\', '') }}</span>
            <Bar :ratio="d.ratio" tint="var(--disk)" />
            <span class="text-right tabular-nums">{{ d.used }}%</span>
            <span class="truncate text-right text-muted">{{ d.detail }}</span>
          </div>
          <p v-if="!drives.length" class="font-mono text-xs text-muted">--</p>
        </div>
      </Tile>
      <Tile title="System" icon="system" tint="var(--muted)" :sub="overview?.os || ''">
        <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 font-mono text-xs">
          <dt class="text-muted">uptime</dt><dd>{{ overview ? show(overview.uptime, formatDuration) : '--' }}</dd>
          <dt class="text-muted">access</dt>
          <dd :class="overview?.elevated ? 'text-ok' : 'text-warn'">{{ overview ? (overview.elevated ? 'administrator' : 'limited') : '--' }}</dd>
          <dt class="text-muted">network</dt><dd class="text-ok">none used</dd>
        </dl>
      </Tile>
    </div>

    <article class="border border-line bg-panel">
      <header class="flex items-center gap-2 border-b border-line px-4 py-3">
        <Icon name="history" />
        <span class="caption">change log</span>
        <span class="ml-auto font-mono text-[11px] text-muted">{{ changes.length }} entries</span>
      </header>
      <p v-if="!changes.length" class="px-4 py-4 font-mono text-xs text-muted">// no changes made yet. every change is logged here, git style, before it happens.</p>
      <table v-else class="w-full font-mono text-xs">
        <tr v-for="c in changes.slice(0, 8)" :key="c.id" class="border-b border-line last:border-0 hover:bg-hover">
          <td class="w-24 px-4 py-2 text-accent">{{ c.id }}</td>
          <td class="truncate py-2" :title="c.before && c.after ? `${c.before} → ${c.after}` : ''">{{ c.summary }}</td>
          <td class="w-24 py-2" :class="statusTone[c.status]">{{ c.status }}</td>
          <td class="w-28 py-2 text-right text-muted">{{ since(c.at) }}</td>
          <td class="w-24 px-4 py-2 text-right">
            <button v-if="c.reversible && c.status === 'applied'" class="text-accent hover:underline" @click="reverting = c">revert</button>
          </td>
        </tr>
      </table>
    </article>
    <ConfirmDialog :open="!!reverting" :title="`Revert ${reverting?.id}?`" confirm="revert" :busy="busy" @yes="revert" @no="reverting = null">
      {{ reverting?.summary }}: goes back from <b>{{ reverting?.after }}</b> to <b>{{ reverting?.before }}</b>. The revert is logged as a new change.
    </ConfirmDialog>
  </section>
</template>
