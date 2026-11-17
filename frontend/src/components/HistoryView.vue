<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Delete, DeleteBefore, List, Revert } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/historyservice'
import { DeleteReport, DeleteReportsBefore } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/scanservice'
import type { Change } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { loadLastScan, loadReports, openReport, reports, severityTone, viewed } from '../lib/scan'
import { when } from '../lib/format'

const props = defineProps<{ tab?: string }>()
const emit = defineEmits<{ open: [page: string, arg?: string] }>()

const tab = ref<'changes' | 'scans'>(props.tab === 'scans' ? 'scans' : 'changes')
const changes = ref<Change[]>([])
const error = ref('')
const busy = ref(false)
const before = ref('') // yyyy-mm-dd from the date field

async function load() {
  error.value = ''
  try {
    const [c] = await Promise.all([List(), loadReports()])
    changes.value = c ?? []
  } catch (e) {
    error.value = String(e)
  }
}
onMounted(load)

// The date field gives a local day; everything from its midnight on stays.
const cutoff = computed(() => {
  if (!before.value) return null
  const [y, m, d] = before.value.split('-').map(Number)
  return new Date(y, m - 1, d)
})

// A revert entry goes with the change it undid, as the log enforces.
const ids = computed(() => new Set(changes.value.map((c) => c.id)))
const deletable = (c: Change) => !c.reverts || !ids.value.has(c.reverts)
function withReverts(drop: Set<string>): Set<string> {
  for (let grew = true; grew; ) {
    grew = false
    for (const c of changes.value) {
      if (c.reverts && drop.has(c.reverts) && !drop.has(c.id)) {
        drop.add(c.id)
        grew = true
      }
    }
  }
  return drop
}

type Pending =
  | { kind: 'change'; change: Change; count: number; revertible: boolean }
  | { kind: 'changes-before'; count: number; revertible: boolean }
  | { kind: 'report'; id: string; at: string }
  | { kind: 'reports-before'; count: number }
const pending = ref<Pending | null>(null)

function askChange(c: Change) {
  const drop = withReverts(new Set([c.id]))
  pending.value = { kind: 'change', change: c, count: drop.size, revertible: !!c.reversible && c.status === 'applied' }
}
function askBefore() {
  const t = cutoff.value
  if (!t) return
  if (tab.value === 'changes') {
    const drop = withReverts(new Set(changes.value.filter((c) => new Date(c.at) < t).map((c) => c.id)))
    const revertible = changes.value.some((c) => drop.has(c.id) && c.reversible && c.status === 'applied')
    pending.value = { kind: 'changes-before', count: drop.size, revertible }
  } else {
    pending.value = { kind: 'reports-before', count: reports.value.filter((r) => new Date(r.at) < t).length }
  }
}

const dialog = computed(() => {
  const p = pending.value
  if (!p) return { title: '', warning: '', body: '' }
  const lost = 'The PC stays as it is, but deleted entries can no longer be reverted from here.'
  switch (p.kind) {
    case 'change':
      return {
        title: `Delete “${p.change.summary}” from the change log?`,
        warning: `This cannot be undone.${p.revertible ? ' This change could still be reverted; after this it cannot be, from here.' : ''} Are you sure?`,
        body: p.count > 1 ? `The ${p.count - 1} entries that reverted it go with it. ${lost}` : lost,
      }
    case 'changes-before':
      return {
        title: `Delete ${p.count} ${p.count === 1 ? 'entry' : 'entries'} from before ${cutoff.value?.toLocaleDateString()}?`,
        warning: `This cannot be undone.${p.revertible ? ' Some of these changes could still be reverted.' : ''} Are you sure?`,
        body: `Entries that revert them go too, even when newer. ${lost}`,
      }
    case 'report':
      return {
        title: `Delete the scan from ${when(p.at)}?`,
        warning: 'This cannot be undone. Are you sure?',
        body: 'Only the saved report goes. Findings you marked resolved stay resolved.',
      }
    case 'reports-before':
      return {
        title: `Delete ${p.count} ${p.count === 1 ? 'scan' : 'scans'} from before ${cutoff.value?.toLocaleDateString()}?`,
        warning: 'This cannot be undone. Are you sure?',
        body: 'Only the saved reports go. Findings you marked resolved stay resolved.',
      }
  }
})

async function confirmDelete() {
  const p = pending.value
  if (!p) return
  busy.value = true
  error.value = ''
  try {
    const t = cutoff.value?.toISOString() ?? ''
    if (p.kind === 'change') await Delete(p.change.id)
    else if (p.kind === 'changes-before') await DeleteBefore(t)
    else if (p.kind === 'report') await DeleteReport(p.id)
    else await DeleteReportsBefore(t)
    if (p.kind === 'report' || p.kind === 'reports-before') {
      if (viewed.value && (p.kind === 'report' ? viewed.value.id === p.id : new Date(viewed.value.at) < cutoff.value!)) viewed.value = null
      await loadLastScan()
    }
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    pending.value = null
  }
}

const reverting = ref<Change | null>(null)
async function revert() {
  if (!reverting.value) return
  busy.value = true
  try {
    await Revert(reverting.value.id)
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    reverting.value = null
  }
}

async function show(id: string) {
  try {
    await openReport(id)
    emit('open', 'fix')
  } catch (e) {
    error.value = String(e)
  }
}

const statusText: Record<string, [string, string]> = {
  applied: ['Applied', 'text-fg'],
  failed: ['Failed', 'text-bad'],
  reverted: ['Reverted', 'text-muted line-through'],
  started: ['Interrupted', 'text-warn'],
}
</script>

<template>
  <section class="mx-auto flex max-w-6xl flex-col gap-5 p-6">
    <header>
      <h1 class="heading">History</h1>
      <p class="mt-1 text-muted">
        What ReforgePulse changed on this PC and the scans it saved. The change log keeps its last 500 entries and the last 30 scans are kept.
      </p>
    </header>

    <nav class="tabs" role="tablist">
      <button role="tab" class="tab" :class="tab === 'changes' && 'tab-on'" :aria-selected="tab === 'changes'" @click="tab = 'changes'">
        Changes <span class="text-xs text-muted">{{ changes.length }}</span>
      </button>
      <button role="tab" class="tab" :class="tab === 'scans' && 'tab-on'" :aria-selected="tab === 'scans'" @click="tab = 'scans'">
        Scans <span class="text-xs text-muted">{{ reports.length }}</span>
      </button>
    </nav>

    <div class="flex flex-wrap items-center gap-2">
      <label for="before" class="text-muted">Delete everything before</label>
      <input id="before" v-model="before" type="date" class="field" />
      <button class="btn" :disabled="!before" @click="askBefore">
        <Icon name="trash" class="size-3.5!" />Delete
      </button>
    </div>

    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <article v-if="tab === 'changes'" class="panel">
      <p v-if="!changes.length" class="px-4 py-6 text-center text-muted">Nothing in the change log.</p>
      <table v-else class="w-full table-fixed text-[14px]">
        <tr v-for="c in changes" :key="c.id" class="border-b border-line last:border-0 hover:bg-hover">
          <td class="w-24 px-4 py-2 font-mono text-xs text-muted">{{ c.id }}</td>
          <td class="truncate py-2" :title="c.before && c.after ? `${c.before} → ${c.after}` : c.summary">
            {{ c.summary }}<span v-if="c.error" class="text-bad">: {{ c.error }}</span>
          </td>
          <td class="w-24 py-2" :class="statusText[c.status]?.[1]">{{ statusText[c.status]?.[0] ?? c.status }}</td>
          <td class="w-36 py-2 text-right text-muted tabular-nums">{{ when(c.at) }}</td>
          <td class="w-20 py-2 text-right">
            <button v-if="c.reversible && c.status === 'applied'" class="text-accent hover:underline" @click="reverting = c">Revert</button>
          </td>
          <td class="w-12 px-3 py-2 text-right">
            <button v-if="deletable(c)" class="rounded p-1 text-muted hover:bg-hover hover:text-bad" :aria-label="`Delete ${c.id} from the log`"
                    title="Delete from the log" @click="askChange(c)">
              <Icon name="trash" class="size-3.5!" />
            </button>
          </td>
        </tr>
      </table>
    </article>

    <article v-else class="panel">
      <p v-if="!reports.length" class="px-4 py-6 text-center text-muted">No saved scans.</p>
      <ul v-else>
        <li v-for="r in reports" :key="r.id" class="flex items-center gap-3 border-b border-line px-4 py-2 last:border-b-0 hover:bg-hover">
          <span class="size-2 shrink-0 rounded-full" :style="{ background: r.open ? severityTone[r.worst ?? 'info'] : 'var(--ok)' }" />
          <span class="w-36 shrink-0 tabular-nums">{{ when(r.at) }}</span>
          <span class="flex-1 text-muted">
            {{ r.open ? `${r.open} open` : 'Nothing open' }}<template v-if="r.resolved">, {{ r.resolved }} resolved</template>
          </span>
          <button class="btn h-7" @click="show(r.id)">Open</button>
          <button class="rounded p-1 text-muted hover:bg-hover hover:text-bad" :aria-label="`Delete the scan from ${when(r.at)}`"
                  title="Delete this scan" @click="pending = { kind: 'report', id: r.id, at: r.at }">
            <Icon name="trash" class="size-3.5!" />
          </button>
        </li>
      </ul>
    </article>

    <ConfirmDialog :open="!!pending" :title="dialog.title" confirm="Delete" danger :busy="busy" :warning="dialog.warning"
                   @yes="confirmDelete" @no="pending = null">
      {{ dialog.body }}
    </ConfirmDialog>
    <ConfirmDialog :open="!!reverting" :title="`Revert “${reverting?.summary}”?`" confirm="Revert" :busy="busy" @yes="revert" @no="reverting = null">
      This goes back from <b class="text-fg">{{ reverting?.after }}</b> to <b class="text-fg">{{ reverting?.before }}</b>. The revert is recorded as a new change.
    </ConfirmDialog>
  </section>
</template>
