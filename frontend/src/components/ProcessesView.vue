<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
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
const open = ref(new Set<number>()) // expanded rows, by PID

function toggle(pid: number) {
  const next = new Set(open.value)
  next.has(pid) ? next.delete(pid) : next.add(pid)
  open.value = next
}

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

// While a row is open the order holds still, so the row being read does
// not jump away on the next refresh; only the numbers change. Sorting or
// searching on purpose re-orders. Not reactive: it is a cache of the last order.
let frozen: Map<number, number> | null = null

function sortBy(k: Key) {
  frozen = null
  desc.value = sortKey.value === k ? !desc.value : k !== 'name'
  sortKey.value = k
}
watch(query, () => (frozen = null))

type Row = { p: Process; depth: number; kids: number; cpu: number; gpu: number; mem: number }

// The tree: children under their parent, each row carrying its subtree's
// totals so a browser with forty helpers sorts by what it costs in all.
// A search flattens it, so matches deep in the tree still show.
const rows = computed<Row[]>(() => {
  const q = query.value.trim().toLowerCase()
  const port = Number(q.replace(/^:/, ''))
  const flat = q !== '' || onlyPorts.value
  let list = procs.value.filter((p) => !onlyPorts.value || p.ports?.length)
  if (q) {
    list = list.filter((p) =>
      p.name.toLowerCase().includes(q) || String(p.pid) === q || p.path?.toLowerCase().includes(q) ||
      (port > 0 && p.ports?.some((x) => x.port === port)),
    )
  }

  const byPid = new Map(list.map((p) => [p.pid, p]))
  const kids = new Map<number, Process[]>()
  for (const p of list) {
    if (!flat && p.parentPid !== p.pid && byPid.has(p.parentPid)) {
      kids.set(p.parentPid, [...(kids.get(p.parentPid) ?? []), p])
    }
  }
  // Totals are computed once per process. A reused PID can make a parent look
  // like its own descendant; visiting stops that loop.
  const agg = new Map<number, Row>()
  const visiting = new Set<number>()
  const total = (p: Process): Row => {
    const done = agg.get(p.pid)
    if (done) return done
    visiting.add(p.pid)
    const r: Row = { p, depth: 0, kids: 0, cpu: p.cpu, gpu: p.gpu, mem: p.private }
    for (const c of kids.get(p.pid) ?? []) {
      if (visiting.has(c.pid)) continue
      const t = total(c)
      r.kids++
      r.cpu += t.cpu
      r.gpu += t.gpu
      r.mem += t.mem
    }
    visiting.delete(p.pid)
    agg.set(p.pid, r)
    return r
  }
  const k = sortKey.value
  const dir = desc.value ? -1 : 1
  const value = (r: Row) => (k === 'cpu' ? r.cpu : k === 'gpu' ? r.gpu : k === 'private' ? r.mem : r.p[k])
  const hold = open.value.size > 0 && frozen
  const at = (r: Row) => frozen?.get(r.p.pid) ?? Number.MAX_SAFE_INTEGER // new processes go last
  const sort = (rs: Row[]) =>
    hold
      ? rs.sort((a, b) => at(a) - at(b))
      : rs.sort((a, b) => (value(a) < value(b) ? -dir : value(a) > value(b) ? dir : 0))

  const roots = list.filter((p) => flat || !byPid.has(p.parentPid) || p.parentPid === p.pid)
  const out: Row[] = []
  const shown = new Set<number>()
  const walk = (r: Row, depth: number) => {
    if (shown.has(r.p.pid)) return
    shown.add(r.p.pid)
    out.push({ ...r, depth })
    if (!open.value.has(r.p.pid)) return
    for (const c of sort((kids.get(r.p.pid) ?? []).map(total))) walk(c, depth + 1)
  }
  for (const r of sort(roots.map(total))) walk(r, 0)
  if (!hold) frozen = new Map(out.map((r, i) => [r.p.pid, i]))
  return out
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
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="heading">Processes</h1>
        <p class="mt-1 text-muted">
          {{ totals.count }} running<template v-if="totals.hung">, <span class="text-bad">{{ totals.hung }} not responding</span></template>,
          {{ totals.ports }} open ports. Child processes sit under their parent; totals include them.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button class="btn" :class="onlyPorts && 'border-accent text-accent'" :aria-pressed="onlyPorts" @click="onlyPorts = !onlyPorts">With ports</button>
        <label class="flex h-8 w-72 items-center gap-2 rounded border border-line bg-raised px-2.5 focus-within:border-accent">
          <Icon name="search" class="size-4! text-muted" />
          <input v-model="query" aria-label="Filter processes" placeholder="Name, PID, path or :port" spellcheck="false"
                 class="min-w-0 flex-1 bg-transparent text-[13px] outline-none placeholder:text-muted" />
        </label>
      </div>
    </header>

    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <div class="panel min-h-0 flex-1 overflow-auto">
      <table class="w-full table-fixed text-[13px] tabular-nums">
        <thead class="sticky top-0 z-10 bg-panel">
          <tr>
            <th v-for="c in columns" :key="c.key" :class="c.cls" :aria-sort="sortKey === c.key ? (desc ? 'descending' : 'ascending') : 'none'"
                class="border-b border-line px-3 py-2 font-medium whitespace-nowrap text-muted">
              <button class="hover:text-fg" @click="sortBy(c.key)">
                {{ c.label }}<span class="ml-1 text-fg">{{ sortKey === c.key ? (desc ? '↓' : '↑') : '' }}</span>
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-for="{ p, depth, kids, cpu, gpu, mem } in rows" :key="p.pid">
            <tr class="cursor-pointer border-b border-line/60 hover:bg-hover" :class="open.has(p.pid) && 'bg-hover'" @click="toggle(p.pid)">
              <td class="truncate px-3 py-1.5">
                <span class="flex items-center gap-2" :style="{ paddingLeft: `${depth * 18}px` }">
                  <Icon name="chevron" class="size-3! shrink-0 text-muted transition-transform" :class="open.has(p.pid) && 'rotate-90'" />
                  <Icon v-if="!p.access" name="lock" class="size-3.5! shrink-0 text-muted" />
                  <span class="truncate" :class="!p.access && 'text-muted'">{{ p.name }}</span>
                  <span v-if="kids" class="text-xs text-muted">{{ kids }}</span>
                  <span v-if="p.hung" class="chip border-bad/50 text-bad">Not responding</span>
                </span>
              </td>
              <td class="px-3 py-1.5 text-right font-mono text-xs text-muted">{{ p.pid }}</td>
              <td class="px-3 py-1.5">
                <span class="flex items-center justify-end gap-2">
                  <span class="h-1 w-12 overflow-hidden rounded-full bg-hover"><span class="block h-full rounded-full bg-accent transition-[width] duration-500" :style="{ width: `${Math.min(100, cpu)}%` }" /></span>
                  <span class="w-10 text-right" :class="cpu >= 25 ? 'text-warn' : ''">{{ p.access || kids ? cpu.toFixed(1) : '–' }}</span>
                </span>
              </td>
              <td class="px-3 py-1.5 text-right" :class="gpu < 1 && 'text-muted'">{{ gpu ? gpu.toFixed(1) : '0' }}</td>
              <td class="px-3 py-1.5 text-right">{{ p.access || kids ? formatBytes(mem) : '–' }}</td>
              <td class="px-3 py-1.5 text-right text-muted">{{ p.threads }}</td>
            </tr>
            <tr v-if="open.has(p.pid)" class="border-b border-line bg-raised">
              <td colspan="6" class="py-4 pr-6" :style="{ paddingLeft: `${30 + depth * 18}px` }">
                <div class="flex flex-wrap items-start gap-x-10 gap-y-3">
                  <dl class="grid min-w-0 flex-1 grid-cols-[7rem_1fr] gap-y-1.5">
                    <dt class="text-muted">Path</dt><dd class="truncate font-mono text-xs select-text" :title="p.path">{{ p.path || 'Hidden by Windows (protected process)' }}</dd>
                    <dt class="text-muted">Parent</dt>
                    <dd>{{ procs.find((x) => x.pid === p.parentPid)?.name ?? 'No longer running' }} <span class="font-mono text-xs text-muted">{{ p.parentPid }}</span></dd>
                    <template v-if="kids"><dt class="text-muted">Children</dt><dd>{{ kids }}, listed below</dd></template>
                    <dt class="text-muted">Working set</dt><dd>{{ p.access ? formatBytes(p.workingSet) : '–' }}</dd>
                    <dt class="text-muted">State</dt><dd :class="p.hung && 'text-bad'">{{ p.hung ? 'Not responding' : 'Responding' }}</dd>
                  </dl>
                  <button class="btn border-bad/50 text-bad hover:bg-bad hover:text-white" @click.stop="ending = p">End task</button>
                </div>
                <table v-if="p.ports?.length" class="mt-4 w-full max-w-2xl text-[13px]">
                  <tr class="text-muted"><th class="w-16 pb-1 text-left font-medium">Protocol</th><th class="pb-1 text-left font-medium">Local</th>
                    <th class="pb-1 text-left font-medium">Remote</th><th class="pb-1 text-left font-medium">State</th></tr>
                  <tr v-for="(x, i) in p.ports" :key="i" class="border-t border-line/60">
                    <td class="py-1 text-muted uppercase">{{ x.proto }}</td>
                    <td class="py-1 font-mono text-xs select-text">{{ x.local }}:{{ x.port }}</td>
                    <td class="py-1 font-mono text-xs text-muted">{{ x.remote || '–' }}</td>
                    <td class="py-1 capitalize" :class="x.state !== 'listen' && 'text-muted'">{{ x.state || 'Bound' }}</td>
                  </tr>
                </table>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <ConfirmDialog :open="!!ending" :title="`End ${ending?.name}?`" confirm="End task" danger
                   warning="Unsaved work in it will be lost. Are you sure?" :busy="busy" @yes="endTask" @no="ending = null">
      Process {{ ending?.pid }} stops at once and any unsaved work in it is lost{{ ending?.ports?.length ? '. The ports it holds are freed' : '' }}.
    </ConfirmDialog>
  </section>
</template>
