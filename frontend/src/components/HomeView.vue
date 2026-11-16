<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import Bar from './Bar.vue'
import Tile from './Tile.vue'
import Sparkline from './Sparkline.vue'
import { GetLive, GetOverview } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/systemservice'
import { List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/historyservice'
import { Availability, type Change, type Measurement, type Overview } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes, formatDuration, measured, show, since } from '../lib/format'

const emit = defineEmits<{ elevated: [boolean] }>()
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
async function sample() {
  if (document.hidden) return
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
  sample()
  timer = setInterval(sample, 1000)
})
onBeforeUnmount(() => clearInterval(timer))

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
    return { root: v.root, ratio, used: Math.round(ratio * 100).toString(), detail: `${formatBytes(v.free.value)} free of ${formatBytes(v.total.value)}` }
  }),
)

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
        <p class="caption">status · <span class="text-warn">not scanned</span></p>
        <h1 class="mt-1 text-2xl font-semibold">System overview</h1>
      </div>
      <div class="flex items-center gap-2">
        <button class="flex size-10 items-center justify-center border border-line bg-panel transition hover:bg-raised disabled:opacity-50"
                title="Refresh" :disabled="loading" @click="load">
          <Icon name="refresh" :class="loading && 'animate-spin'" />
        </button>
        <button disabled title="Scanning arrives in the next version"
                class="h-10 bg-accent px-6 font-mono text-xs font-bold tracking-[.2em] text-white uppercase shadow-[0_0_24px_-6px_var(--accent)] transition hover:brightness-110 active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60">
          Scan PC
        </button>
      </div>
    </header>

    <div v-if="overview && !overview.elevated" class="flex items-center gap-3 border border-warn/40 bg-warn/10 px-4 py-2.5 font-mono text-xs text-warn">
      <Icon name="shield" />
      WARN: running without administrator rights; event logs and repairs are limited.
    </div>

    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

    <div class="grid grid-cols-[repeat(auto-fill,minmax(15rem,1fr))] gap-3">
      <Tile title="CPU" icon="cpu" tint="var(--accent)" :value="pct(now.cpu)" unit="%"
            :sub="overview ? `${overview.cpuModel || 'unknown'} · ${overview.logicalCpus} threads` : ''">
        <Sparkline :values="history.cpu" color="var(--accent)" />
      </Tile>
      <Tile title="GPU" icon="gpu" tint="var(--gpu)" :value="pct(now.gpu)" unit="%" :sub="overview?.gpuModel || 'unknown'">
        <Sparkline :values="history.gpu" color="var(--gpu)" />
      </Tile>
      <Tile title="Memory" icon="memory" tint="var(--mem)" :value="pct(now.memory)" unit="%" :sub="memoryDetail">
        <Sparkline :values="history.memory" color="var(--mem)" />
      </Tile>
      <Tile v-for="d in drives" :key="d.root" :title="`Disk ${d.root}`" icon="disk" tint="var(--disk)"
            :value="d.used" unit="% used" :sub="d.detail">
        <Bar :ratio="d.ratio" tint="var(--disk)" class="mt-auto" />
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
          <td class="truncate py-2">{{ c.summary }}</td>
          <td class="w-24 py-2" :class="statusTone[c.status]">{{ c.status }}</td>
          <td class="w-28 px-4 py-2 text-right text-muted">{{ since(c.at) }}</td>
        </tr>
      </table>
    </article>
  </section>
</template>
