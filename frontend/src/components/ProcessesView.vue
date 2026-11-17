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
const expanded = ref(new Set<number>()) // tree nodes showing their children
const selected = ref<number | null>(null) // the process on the right; null is the top level

function toggle(pid: number) {
  const next = new Set(expanded.value)
  next.has(pid) ? next.delete(pid) : next.add(pid)
  expanded.value = next
}

// Selecting opens the way to it in the tree, so the left always shows where
// the right is.
function select(pid: number | null) {
  selected.value = pid
  if (pid === null) return
  const next = new Set(expanded.value)
  for (const a of ancestors.value) next.add(a.pid)
  expanded.value = next
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

function sortBy(k: Key) {
  desc.value = sortKey.value === k ? !desc.value : k !== 'name'
  sortKey.value = k
}

type Node = { p: Process; kids: Process[]; count: number; cpu: number; gpu: number; mem: number }

// The tree with each process's subtree totals, so a browser with forty
// helpers shows what it costs in all. A search or the ports filter
// flattens it, so matches deep in the tree still show.
const flat = computed(() => query.value.trim() !== '' || onlyPorts.value)
const tree = computed(() => {
  const q = query.value.trim().toLowerCase()
  const port = Number(q.replace(/^:/, ''))
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
    if (!flat.value && p.parentPid !== p.pid && byPid.has(p.parentPid)) kids.set(p.parentPid, [...(kids.get(p.parentPid) ?? []), p])
  }
  // A reused PID can make a parent look like its own descendant; visiting
  // stops that loop.
  const nodes = new Map<number, Node>()
  const visiting = new Set<number>()
  const total = (p: Process): Node => {
    const done = nodes.get(p.pid)
    if (done) return done
    visiting.add(p.pid)
    const n: Node = { p, kids: [], count: 0, cpu: p.cpu, gpu: p.gpu, mem: p.private }
    for (const c of kids.get(p.pid) ?? []) {
      if (visiting.has(c.pid)) continue
      const t = total(c)
      n.kids.push(c)
      n.count += 1 + t.count
      n.cpu += t.cpu
      n.gpu += t.gpu
      n.mem += t.mem
    }
    visiting.delete(p.pid)
    nodes.set(p.pid, n)
    return n
  }
  list.forEach(total)
  const roots = list.filter((p) => flat.value || !byPid.has(p.parentPid) || p.parentPid === p.pid)
  return { nodes, roots, byPid }
})

const value = (n: Node) => {
  const k = sortKey.value
  return k === 'cpu' ? n.cpu : k === 'gpu' ? n.gpu : k === 'private' ? n.mem : n.p[k]
}
const order = (ns: Node[]) => {
  const dir = desc.value ? -1 : 1
  return ns.sort((a, b) => (value(a) < value(b) ? -dir : value(a) > value(b) ? dir : 0))
}

// The left pane: the tree, opened where the user opened it.
const treeRows = computed(() => {
  const { nodes, roots } = tree.value
  const out: { n: Node; depth: number }[] = []
  const shown = new Set<number>()
  const walk = (n: Node, depth: number) => {
    if (shown.has(n.p.pid)) return
    shown.add(n.p.pid)
    out.push({ n, depth })
    if (expanded.value.has(n.p.pid)) for (const c of order(n.kids.map((k) => nodes.get(k.pid)!))) walk(c, depth + 1)
  }
  for (const r of order(roots.map((p) => nodes.get(p.pid)!))) walk(r, 0)
  return out
})

const current = computed(() => (selected.value === null ? null : tree.value.nodes.get(selected.value) ?? null))
// The way from the top to the selection, for the breadcrumb.
const ancestors = computed(() => {
  const out: Process[] = []
  const byPid = new Map(procs.value.map((p) => [p.pid, p]))
  let p = selected.value === null ? undefined : byPid.get(selected.value)
  const seen = new Set<number>()
  while (p && byPid.has(p.parentPid) && p.parentPid !== p.pid && !seen.has(p.parentPid)) {
    seen.add(p.parentPid)
    p = byPid.get(p.parentPid)!
    out.unshift(p)
  }
  return out
})

// The right pane: the selection's children, or the top level. While the
// pointer is over the table the order holds still, so the row under it does
// not move on the next refresh; only the numbers change.
const hovering = ref(false)
let held: number[] = []
const tableRows = computed(() => {
  const { nodes, roots } = tree.value
  const base = current.value ? current.value.kids : roots
  const ns = base.map((p) => nodes.get(p.pid)!).filter(Boolean)
  if (hovering.value && held.length) {
    const at = new Map(held.map((pid, i) => [pid, i]))
    return ns.sort((a, b) => (at.get(a.p.pid) ?? 1e9) - (at.get(b.p.pid) ?? 1e9))
  }
  const sorted = order(ns)
  held = sorted.map((n) => n.p.pid)
  return sorted
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
  { key: 'cpu', label: 'CPU', cls: 'w-28 text-right' },
  { key: 'gpu', label: 'GPU', cls: 'w-14 text-right' },
  { key: 'private', label: 'Memory', cls: 'w-24 text-right' },
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
          {{ totals.ports }} open ports. Pick one in the tree to see it and what it started; totals include its children.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button class="btn" :class="onlyPorts && 'border-accent text-accent'" :aria-pressed="onlyPorts" @click="onlyPorts = !onlyPorts">With ports</button>
        <label class="flex h-8 w-72 items-center gap-2 rounded border border-line bg-raised px-2.5 focus-within:border-accent">
          <Icon name="search" class="size-4! text-muted" />
          <input v-model="query" aria-label="Filter processes" placeholder="Name, PID, path or :port" spellcheck="false"
                 class="min-w-0 flex-1 bg-transparent text-[14px] outline-none placeholder:text-muted" />
        </label>
      </div>
    </header>

    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <div class="flex min-h-0 flex-1 gap-4">
      <!-- The tree: only who started whom. Picking a process shows it on the right. -->
      <nav class="panel flex w-64 shrink-0 flex-col overflow-hidden" aria-label="Process tree">
        <button class="flex items-center gap-2 border-b border-line px-3 py-2 text-left hover:bg-hover"
                :class="selected === null && 'bg-hover font-semibold'" @click="select(null)">
          {{ flat ? `${tree.roots.length} matches` : 'All processes' }}
        </button>
        <ul class="min-h-0 flex-1 overflow-y-auto py-1 [scrollbar-gutter:stable]">
          <li v-for="{ n, depth } in treeRows" :key="n.p.pid" class="flex items-center" :style="{ paddingLeft: `${depth * 14 + 4}px` }">
            <button v-if="n.kids.length" class="grid size-6 shrink-0 place-items-center rounded text-muted hover:text-fg"
                    :aria-label="`${expanded.has(n.p.pid) ? 'Collapse' : 'Expand'} ${n.p.name}`" :aria-expanded="expanded.has(n.p.pid)" @click="toggle(n.p.pid)">
              <Icon name="chevron" class="size-3! transition-transform" :class="expanded.has(n.p.pid) && 'rotate-90'" />
            </button>
            <span v-else class="size-6 shrink-0" />
            <button class="flex min-w-0 flex-1 items-center gap-2 rounded px-1.5 py-1 text-left hover:bg-hover"
                    :class="selected === n.p.pid && 'bg-hover'" :aria-current="selected === n.p.pid" @click="select(n.p.pid)">
              <span class="min-w-0 truncate" :class="[!n.p.access && 'text-muted', selected === n.p.pid && 'text-fg']" :title="n.p.name">{{ n.p.name }}</span>
              <span v-if="n.count" class="text-xs text-muted">{{ n.count }}</span>
              <span v-if="n.p.hung" class="size-1.5 shrink-0 rounded-full bg-bad" title="Not responding" />
              <span class="ml-auto shrink-0 text-xs tabular-nums" :class="n.cpu >= 25 ? 'text-warn' : 'text-muted'">{{ n.cpu >= 0.1 ? n.cpu.toFixed(1) : '' }}</span>
            </button>
          </li>
        </ul>
      </nav>

      <div class="flex min-w-0 flex-1 flex-col gap-4">
        <!-- The selection: where it sits, what it is, and its own ports. -->
        <article v-if="current" class="panel shrink-0 px-4 py-3">
          <p class="flex flex-wrap items-center gap-1 text-xs text-muted">
            <button class="hover:text-fg hover:underline" @click="select(null)">All processes</button>
            <template v-for="a in ancestors" :key="a.pid">
              <Icon name="chevron" class="size-3!" /><button class="max-w-48 truncate hover:text-fg hover:underline" :title="a.name" @click="select(a.pid)">{{ a.name }}</button>
            </template>
          </p>
          <div class="mt-2 flex flex-wrap items-start gap-x-10 gap-y-3">
            <div class="min-w-0 flex-1">
              <h2 class="flex min-w-0 flex-wrap items-center gap-2 text-[16px] font-semibold break-all">
                {{ current.p.name }} <span class="font-mono text-xs font-normal text-muted">{{ current.p.pid }}</span>
                <span v-if="current.p.hung" class="chip border-bad/50 text-bad">Not responding</span>
              </h2>
              <dl class="mt-2 grid grid-cols-[7rem_minmax(0,1fr)] gap-y-1">
                <dt class="text-muted">Path</dt><dd class="truncate font-mono text-xs select-text" :title="current.p.path">{{ current.p.path || 'Hidden by Windows (protected process)' }}</dd>
                <dt class="text-muted">Itself</dt>
                <dd class="tabular-nums">{{ current.p.access ? `${current.p.cpu.toFixed(1)}% CPU, ${formatBytes(current.p.private)} memory, ${current.p.threads} threads` : 'Windows does not let it be measured' }}</dd>
                <template v-if="current.count"><dt class="text-muted">With children</dt>
                  <dd class="tabular-nums">{{ current.cpu.toFixed(1) }}% CPU, {{ formatBytes(current.mem) }} memory, {{ current.count }} processes</dd></template>
              </dl>
            </div>
            <button class="btn border-bad/50 text-bad hover:bg-bad hover:text-white" @click="ending = current.p">End task</button>
          </div>
          <table v-if="current.p.ports?.length" class="mt-3 w-full max-w-2xl table-fixed text-[14px]">
            <tr class="text-muted"><th class="w-16 pb-1 text-left font-medium">Protocol</th><th class="pb-1 text-left font-medium">Local</th>
              <th class="pb-1 text-left font-medium">Remote</th><th class="w-28 pb-1 text-left font-medium">State</th></tr>
            <tr v-for="(x, i) in current.p.ports" :key="i" class="border-t border-line/60">
              <td class="py-1 text-muted uppercase">{{ x.proto }}</td>
              <td class="py-1 pr-2 font-mono text-xs break-all select-text">{{ x.local }}:{{ x.port }}</td>
              <td class="py-1 pr-2 font-mono text-xs break-all text-muted">{{ x.remote || '–' }}</td>
              <td class="py-1 capitalize" :class="x.state !== 'listen' && 'text-muted'">{{ x.state || 'Bound' }}</td>
            </tr>
          </table>
        </article>

        <div class="panel min-h-0 flex-1 overflow-auto [scrollbar-gutter:stable]" @mouseenter="hovering = true" @mouseleave="hovering = false">
          <p v-if="current && !tableRows.length" class="px-4 py-6 text-center text-muted">{{ current.p.name }} has started no other processes.</p>
          <table v-else class="w-full table-fixed text-[14px] tabular-nums">
            <caption class="sr-only">{{ current ? `Processes started by ${current.p.name}` : 'Top-level processes' }}</caption>
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
              <tr v-for="n in tableRows" :key="n.p.pid" class="cursor-pointer border-b border-line/60 hover:bg-hover" @click="select(n.p.pid)">
                <td class="truncate px-3 py-1.5">
                  <span class="flex items-center gap-2">
                    <Icon v-if="!n.p.access" name="lock" class="size-3.5! shrink-0 text-muted" />
                    <span class="min-w-0 truncate" :class="!n.p.access && 'text-muted'" :title="n.p.name">{{ n.p.name }}</span>
                    <span v-if="n.count" class="text-xs text-muted">{{ n.count }}</span>
                    <span v-if="n.p.hung" class="chip border-bad/50 text-bad">Not responding</span>
                  </span>
                </td>
                <td class="px-3 py-1.5 text-right font-mono text-xs text-muted">{{ n.p.pid }}</td>
                <td class="px-3 py-1.5">
                  <span class="flex items-center justify-end gap-2">
                    <span class="h-1 w-8 overflow-hidden rounded-full bg-hover"><span class="block h-full rounded-full bg-accent transition-[width] duration-500" :style="{ width: `${Math.min(100, n.cpu)}%` }" /></span>
                    <span class="w-10 text-right" :class="n.cpu >= 25 ? 'text-warn' : ''">{{ n.p.access || n.count ? n.cpu.toFixed(1) : '–' }}</span>
                  </span>
                </td>
                <td class="px-3 py-1.5 text-right" :class="n.gpu < 1 && 'text-muted'">{{ n.gpu ? n.gpu.toFixed(1) : '0' }}</td>
                <td class="px-3 py-1.5 text-right">{{ n.p.access || n.count ? formatBytes(n.mem) : '–' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <ConfirmDialog :open="!!ending" :title="`End ${ending?.name}?`" confirm="End task" danger
                   warning="Unsaved work in it will be lost. Are you sure?" :busy="busy" @yes="endTask" @no="ending = null">
      Process {{ ending?.pid }} stops at once and any unsaved work in it is lost{{ ending?.ports?.length ? '. The ports it holds are freed' : '' }}.
    </ConfirmDialog>
  </section>
</template>
