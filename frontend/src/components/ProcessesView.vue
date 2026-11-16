<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { End, List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/processservice'
import type { Process } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

type Key = 'name' | 'pid' | 'cpu' | 'gpu' | 'private' | 'threads'
const props = defineProps<{ sort?: Key }>()

const procs = ref<Process[]>([])
const error = ref('')
const query = ref('')
const onlyPorts = ref(false)
const sortKey = ref<Key>(props.sort ?? 'cpu')
const desc = ref(sortKey.value !== 'name')
const open = ref<number | null>(null) // the expanded row's PID

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

// ":8080" or a bare number also matches ports, so "who holds 8080" is one search.
const rows = computed(() => {
  const q = query.value.trim().toLowerCase()
  const port = Number(q.replace(/^:/, ''))
  let list = procs.value.filter((p) => !onlyPorts.value || p.ports?.length)
  if (q) {
    list = list.filter((p) =>
      p.name.toLowerCase().includes(q) || String(p.pid) === q || p.path?.toLowerCase().includes(q) ||
      (port > 0 && p.ports?.some((x) => x.port === port)),
    )
  }
  const k = sortKey.value
  const dir = desc.value ? -1 : 1
  return [...list].sort((a, b) => (a[k] < b[k] ? -dir : a[k] > b[k] ? dir : 0))
})

const totals = computed(() => ({
  count: procs.value.length,
  hung: procs.value.filter((p) => p.hung).length,
  ports: procs.value.reduce((n, p) => n + (p.ports?.filter((x) => x.state === 'listen' || x.proto.startsWith('udp')).length ?? 0), 0),
  denied: procs.value.filter((p) => !p.access).length,
}))

const columns: { key: Key; label: string; cls: string }[] = [
  { key: 'name', label: 'Name', cls: 'text-left' },
  { key: 'pid', label: 'PID', cls: 'w-20 text-right' },
  { key: 'cpu', label: 'CPU', cls: 'w-32 text-right' },
  { key: 'gpu', label: 'GPU', cls: 'w-20 text-right' },
  { key: 'private', label: 'Memory', cls: 'w-28 text-right' },
  { key: 'threads', label: 'Threads', cls: 'w-20 text-right' },
]

const ending = ref<Process | null>(null)
const busy = ref(false)
async function endTask() {
  if (!ending.value) return
  busy.value = true
  try {
    await End(ending.value.pid, ending.value.name)
    open.value = null
    await refresh()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    ending.value = null
  }
}
</script>

<template>
  <section class="flex h-full flex-col gap-4 p-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="caption">live · every 2 s · click a row for details</p>
        <h1 class="mt-1 text-2xl font-semibold">Processes</h1>
      </div>
      <div class="flex items-center gap-2">
        <button class="h-10 border px-3 font-mono text-xs transition"
                :class="onlyPorts ? 'border-accent bg-accent/15 text-accent' : 'border-line bg-panel text-muted hover:text-fg'"
                @click="onlyPorts = !onlyPorts">ports only</button>
        <label class="flex h-10 w-72 items-center gap-2 border border-line bg-panel px-3 focus-within:border-accent">
          <Icon name="search" class="text-muted" />
          <input v-model="query" placeholder="name, PID, path or :port" spellcheck="false"
                 class="min-w-0 flex-1 bg-transparent font-mono text-xs outline-none placeholder:text-muted" />
        </label>
      </div>
    </header>

    <div class="grid grid-cols-4 border border-line bg-panel font-mono">
      <div class="border-r border-line px-4 py-3"><p class="caption">processes</p><p class="mt-1 text-xl text-accent tabular-nums">{{ totals.count }}</p></div>
      <div class="border-r border-line px-4 py-3"><p class="caption">not responding</p>
        <p class="mt-1 text-xl tabular-nums" :class="totals.hung ? 'text-bad' : 'text-ok'">{{ totals.hung }}</p></div>
      <div class="border-r border-line px-4 py-3"><p class="caption">open ports</p><p class="mt-1 text-xl tabular-nums">{{ totals.ports }}</p></div>
      <div class="px-4 py-3" title="Protected processes Windows will not open, even for an administrator">
        <p class="caption">protected</p><p class="mt-1 text-xl text-muted tabular-nums">{{ totals.denied }}</p></div>
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
          <template v-for="p in rows" :key="p.pid">
            <tr class="cursor-pointer border-b border-line/60 hover:bg-hover" :class="open === p.pid && 'bg-hover'"
                @click="open = open === p.pid ? null : p.pid">
              <td class="truncate px-3 py-1.5">
                <span class="flex items-center gap-2">
                  <Icon name="chevron" class="size-3! text-muted transition-transform" :class="open === p.pid && 'rotate-90'" />
                  <Icon v-if="!p.access" name="lock" class="size-3.5! text-muted" />
                  <span class="truncate" :class="!p.access && 'text-muted'">{{ p.name }}</span>
                  <span v-if="p.hung" class="shrink-0 border border-bad/50 bg-bad/10 px-1.5 text-[10px] text-bad">not responding</span>
                  <span v-if="p.ports?.length" class="shrink-0 text-[10px] text-accent">{{ p.ports.length }} ports</span>
                </span>
              </td>
              <td class="px-3 py-1.5 text-right text-muted tabular-nums">{{ p.pid }}</td>
              <td class="px-3 py-1.5">
                <span class="flex items-center justify-end gap-2">
                  <span class="h-1.5 w-12 bg-hover"><span class="block h-full bg-accent transition-[width] duration-500" :style="{ width: `${Math.min(100, p.cpu)}%` }" /></span>
                  <span class="w-10 text-right tabular-nums" :class="p.cpu >= 10 ? 'text-warn' : ''">{{ p.access ? p.cpu.toFixed(1) : '--' }}</span>
                </span>
              </td>
              <td class="px-3 py-1.5 text-right tabular-nums" :class="p.gpu >= 1 ? 'text-gpu' : 'text-muted'">{{ p.gpu ? p.gpu.toFixed(1) : '0' }}</td>
              <td class="px-3 py-1.5 text-right tabular-nums" :class="p.private >= 2 ** 30 ? 'text-mem' : ''">{{ p.access ? formatBytes(p.private) : '--' }}</td>
              <td class="px-3 py-1.5 text-right text-muted tabular-nums">{{ p.threads }}</td>
            </tr>
            <tr v-if="open === p.pid" class="border-b border-line bg-raised">
              <td colspan="6" class="px-6 py-4">
                <div class="flex flex-wrap items-start gap-x-10 gap-y-3">
                  <dl class="grid min-w-0 flex-1 grid-cols-[6rem_1fr] gap-y-1">
                    <dt class="text-muted">path</dt><dd class="truncate select-text" :title="p.path">{{ p.path || 'unknown (protected)' }}</dd>
                    <dt class="text-muted">parent</dt><dd>{{ p.parentPid }}</dd>
                    <dt class="text-muted">working set</dt><dd>{{ p.access ? formatBytes(p.workingSet) : '--' }}</dd>
                    <dt class="text-muted">status</dt><dd :class="p.hung ? 'text-bad' : 'text-ok'">{{ p.hung ? 'not responding' : 'responding' }}</dd>
                  </dl>
                  <button class="flex h-8 items-center gap-2 border border-bad/50 px-3 text-bad transition hover:bg-bad hover:text-white"
                          @click.stop="ending = p"><Icon name="close" class="size-3.5!" />end task</button>
                </div>
                <table v-if="p.ports?.length" class="mt-4 w-full max-w-2xl">
                  <tr class="text-muted"><td class="w-16 pb-1">proto</td><td class="pb-1">local</td><td class="pb-1">remote</td><td class="pb-1">state</td></tr>
                  <tr v-for="(x, i) in p.ports" :key="i" class="border-t border-line/60">
                    <td class="py-1 text-muted">{{ x.proto }}</td>
                    <td class="py-1 text-accent select-text">{{ x.local }}:{{ x.port }}</td>
                    <td class="py-1 text-muted">{{ x.remote || '--' }}</td>
                    <td class="py-1" :class="x.state === 'listen' ? 'text-ok' : 'text-muted'">{{ x.state || 'bound' }}</td>
                  </tr>
                </table>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <ConfirmDialog :open="!!ending" :title="`End ${ending?.name} (PID ${ending?.pid})?`" confirm="end task" danger :busy="busy"
                   @yes="endTask" @no="ending = null">
      The process stops at once. Unsaved work in it is lost{{ ending?.ports?.length ? ', and the ports it holds are freed' : '' }}.
    </ConfirmDialog>
  </section>
</template>
