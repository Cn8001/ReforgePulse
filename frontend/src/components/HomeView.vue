<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import Bar from './Bar.vue'
import { GetOverview } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/systemservice'
import { List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/historyservice'
import type { Change, Overview } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
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
onMounted(load)

const memory = computed(() => {
  const o = overview.value
  if (!o || !measured(o.memoryTotal, o.memoryUsed)) return { ratio: null, detail: show(o?.memoryTotal, formatBytes) }
  return {
    ratio: o.memoryUsed.value / o.memoryTotal.value,
    detail: `${formatBytes(o.memoryUsed.value)} of ${formatBytes(o.memoryTotal.value)}`,
  }
})

const drives = computed(() =>
  (overview.value?.volumes ?? []).map((v) => {
    if (!measured(v.total, v.free)) return { root: v.root, ratio: null, detail: show(v.total, formatBytes) }
    const used = v.total.value - v.free.value
    return { root: v.root, ratio: used / v.total.value, detail: `${formatBytes(v.free.value)} free of ${formatBytes(v.total.value)}` }
  }),
)

const statusTone: Record<string, string> = {
  applied: 'text-accent',
  failed: 'text-[color:var(--color-red-500)]',
  reverted: 'text-muted line-through',
  started: 'text-[color:var(--color-amber-500)]',
}
</script>

<template>
  <section class="mx-auto flex max-w-3xl flex-col gap-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="text-xs font-medium tracking-wide text-muted uppercase">Not scanned yet</p>
        <h1 class="mt-1 text-2xl font-semibold tracking-tight">Let's see how your PC is doing</h1>
      </div>
      <button disabled title="Scanning arrives in the next version"
              class="rounded-xl bg-accent px-6 py-3 font-semibold text-white shadow-lg shadow-accent/30 transition hover:brightness-110 active:scale-[.98] disabled:cursor-not-allowed disabled:opacity-60">
        Scan PC
      </button>
    </header>

    <div v-if="overview && !overview.elevated"
         class="flex items-center gap-3 rounded-xl border border-[color:var(--color-amber-500)]/40 bg-[color:var(--color-amber-500)]/10 px-4 py-3">
      <Icon name="shield" />
      Running without administrator rights; some checks are unavailable.
    </div>

    <p v-if="error" class="card text-[color:var(--color-red-500)]">Could not read this PC: {{ error }}</p>

    <template v-else-if="overview">
      <div class="flex items-center gap-3 text-muted">
        <p class="min-w-0 flex-1 truncate">
          {{ overview.os || 'Windows' }} · {{ overview.cpuModel || 'Unknown CPU' }} · {{ overview.logicalCpus }} threads ·
          up {{ show(overview.uptime, formatDuration) }}
        </p>
        <button class="rounded-lg p-1.5 transition hover:bg-hover hover:text-fg disabled:opacity-50" title="Refresh"
                :disabled="loading" @click="load">
          <Icon name="refresh" :class="loading && 'animate-spin'" />
        </button>
      </div>

      <div class="card flex flex-col gap-5">
        <Bar label="Memory" :ratio="memory.ratio" :detail="memory.detail" />
        <Bar v-for="d in drives" :key="d.root" :label="`Drive ${d.root}`" :ratio="d.ratio" :detail="d.detail" />
      </div>

      <div class="card">
        <h2 class="mb-3 flex items-center gap-2 font-semibold"><Icon name="history" /> Recent changes</h2>
        <p v-if="!changes.length" class="text-muted">No changes made yet. Every change ReforgePulse makes will be listed here.</p>
        <ul v-else class="flex flex-col divide-y divide-line">
          <li v-for="c in changes.slice(0, 5)" :key="c.id" class="flex items-center gap-3 py-2">
            <code class="font-mono text-xs text-muted">{{ c.id }}</code>
            <span class="min-w-0 flex-1 truncate">{{ c.summary }}</span>
            <span class="text-xs" :class="statusTone[c.status]">{{ c.status }}</span>
            <span class="w-20 text-right text-xs text-muted">{{ since(c.at) }}</span>
          </li>
        </ul>
      </div>
    </template>

    <div v-else class="flex flex-col gap-4" aria-busy="true">
      <div class="h-4 w-2/3 animate-pulse rounded bg-hover" />
      <div class="h-32 animate-pulse rounded-xl bg-hover" />
    </div>
  </section>
</template>
