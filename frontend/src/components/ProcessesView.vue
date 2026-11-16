<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import { List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/processservice'
import type { Process } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const procs = ref<Process[]>([])
const error = ref('')
const query = ref('')
type Key = 'name' | 'pid' | 'cpu' | 'private' | 'workingSet' | 'threads'
const sortKey = ref<Key>('cpu')
const desc = ref(true)

// Every two seconds while the window is visible: often enough to watch a
// spike, rare enough not to become one.
let timer: ReturnType<typeof setInterval> | undefined
async function refresh() {
  if (document.hidden) return
  try {
    procs.value = (await List()) ?? []
    error.value = ''
  } catch (e) {
    error.value = String(e)
  }
}
onMounted(() => {
  refresh()
  timer = setInterval(refresh, 2000)
})
onBeforeUnmount(() => clearInterval(timer))

function sortBy(k: Key) {
  desc.value = sortKey.value === k ? !desc.value : k !== 'name'
  sortKey.value = k
}

const rows = computed(() => {
  const q = query.value.trim().toLowerCase()
  const list = q
    ? procs.value.filter((p) => p.name.toLowerCase().includes(q) || String(p.pid) === q || p.path?.toLowerCase().includes(q))
    : [...procs.value]
  const k = sortKey.value
  const dir = desc.value ? -1 : 1
  return list.sort((a, b) => (a[k] < b[k] ? -dir : a[k] > b[k] ? dir : 0))
})

const totals = computed(() => ({
  count: procs.value.length,
  threads: procs.value.reduce((n, p) => n + p.threads, 0),
  cpu: procs.value.reduce((n, p) => n + p.cpu, 0),
  denied: procs.value.filter((p) => !p.access).length,
}))

const columns: { key: Key; label: string; cls: string }[] = [
  { key: 'name', label: 'Name', cls: 'text-left' },
  { key: 'pid', label: 'PID', cls: 'w-20 text-right' },
  { key: 'cpu', label: 'CPU', cls: 'w-36 text-right' },
  { key: 'private', label: 'Private', cls: 'w-28 text-right' },
  { key: 'workingSet', label: 'Working set', cls: 'w-28 text-right' },
  { key: 'threads', label: 'Threads', cls: 'w-20 text-right' },
]
</script>

<template>
  <section class="flex h-full flex-col gap-4 p-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="caption">live · every 2 s</p>
        <h1 class="mt-1 text-2xl font-semibold">Processes</h1>
      </div>
      <label class="flex h-10 w-72 items-center gap-2 border border-line bg-panel px-3 focus-within:border-accent">
        <Icon name="search" class="text-muted" />
        <input v-model="query" placeholder="Filter by name, PID or path" spellcheck="false"
               class="min-w-0 flex-1 bg-transparent font-mono text-xs outline-none placeholder:text-muted" />
      </label>
    </header>

    <div class="grid grid-cols-4 border border-line bg-panel font-mono">
      <div class="border-r border-line px-4 py-3"><p class="caption">processes</p><p class="mt-1 text-xl text-accent tabular-nums">{{ totals.count }}</p></div>
      <div class="border-r border-line px-4 py-3"><p class="caption">threads</p><p class="mt-1 text-xl tabular-nums">{{ totals.threads }}</p></div>
      <div class="border-r border-line px-4 py-3"><p class="caption">cpu, all</p><p class="mt-1 text-xl tabular-nums">{{ Math.min(100, totals.cpu).toFixed(1) }}%</p></div>
      <div class="px-4 py-3" title="Protected processes Windows will not open, even for an administrator">
        <p class="caption">protected</p><p class="mt-1 text-xl text-muted tabular-nums">{{ totals.denied }}</p>
      </div>
    </div>

    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

    <div class="min-h-0 flex-1 overflow-auto border border-line bg-panel">
      <table class="w-full table-fixed font-mono text-xs">
        <thead class="sticky top-0 z-10 bg-raised">
          <tr>
            <th v-for="c in columns" :key="c.key" :class="c.cls"
                class="cursor-pointer border-b border-line px-3 py-2 font-normal whitespace-nowrap text-muted uppercase transition hover:text-fg"
                @click="sortBy(c.key)">
              {{ c.label }}<span class="ml-1 text-accent">{{ sortKey === c.key ? (desc ? '▾' : '▴') : '' }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in rows" :key="p.pid" class="border-b border-line/60 hover:bg-hover" :title="p.path || p.name">
            <td class="truncate px-3 py-1.5">
              <span class="flex items-center gap-2">
                <Icon v-if="!p.access" name="lock" class="size-3.5! text-muted" />
                <span class="truncate" :class="!p.access && 'text-muted'">{{ p.name }}</span>
              </span>
            </td>
            <td class="px-3 py-1.5 text-right text-muted tabular-nums">{{ p.pid }}</td>
            <td class="px-3 py-1.5">
              <span class="flex items-center justify-end gap-2">
                <span class="h-1.5 w-14 bg-hover"><span class="block h-full bg-accent transition-[width] duration-500" :style="{ width: `${Math.min(100, p.cpu)}%` }" /></span>
                <span class="w-12 text-right tabular-nums" :class="p.cpu >= 10 ? 'text-warn' : ''">{{ p.access ? p.cpu.toFixed(1) : '--' }}</span>
              </span>
            </td>
            <td class="px-3 py-1.5 text-right tabular-nums">{{ p.access ? formatBytes(p.private) : '--' }}</td>
            <td class="px-3 py-1.5 text-right text-muted tabular-nums">{{ p.access ? formatBytes(p.workingSet) : '--' }}</td>
            <td class="px-3 py-1.5 text-right text-muted tabular-nums">{{ p.threads }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
