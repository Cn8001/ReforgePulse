<script setup lang="ts">
import PageLayout from './PageLayout.vue'
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Delete, DeleteBefore, List, Revert } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/historyservice'
import type { Change } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { when } from '../lib/format'

const changes = ref<Change[]>([])
const error = ref('')
const busy = ref(false)
const before = ref('') // yyyy-mm-dd from the date field

async function load() {
  error.value = ''
  try {
    const c = await List()
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
const pending = ref<Pending | null>(null)

function askChange(c: Change) {
  const drop = withReverts(new Set([c.id]))
  pending.value = { kind: 'change', change: c, count: drop.size, revertible: !!c.reversible && c.status === 'applied' }
}
function askBefore() {
  const t = cutoff.value
  if (!t) return
  const drop = withReverts(new Set(changes.value.filter((c) => new Date(c.at) < t).map((c) => c.id)))
  const revertible = changes.value.some((c) => drop.has(c.id) && c.reversible && c.status === 'applied')
  pending.value = { kind: 'changes-before', count: drop.size, revertible }
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
    else await DeleteBefore(t)
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

const statusText: Record<string, [string, string]> = {
  applied: ['Applied', 'text-fg'],
  failed: ['Failed', 'text-bad'],
  reverted: ['Reverted', 'text-muted line-through'],
  started: ['Interrupted', 'text-warn'],
}
</script>

<template>
  <PageLayout>
    <template #head>
      <header>
        <h1 class="heading">History</h1>
        <p class="mt-1 text-muted">
          Everything ReforgePulse changed on this PC, newest first. Reverting a change and deleting an entry both start here. The log keeps its last 500 entries.
        </p>
      </header>
    </template>


    <div class="flex flex-wrap items-center gap-2">
      <label for="before" class="text-muted">Delete everything before</label>
      <input id="before" v-model="before" type="date" class="field" />
      <button class="btn" :disabled="!before" @click="askBefore">
        <Icon name="trash" class="size-3.5!" />Delete
      </button>
    </div>

    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <article class="panel">
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


    <ConfirmDialog :open="!!pending" :title="dialog.title" confirm="Delete" danger :busy="busy" :warning="dialog.warning"
                   @yes="confirmDelete" @no="pending = null">
      {{ dialog.body }}
    </ConfirmDialog>
    <ConfirmDialog :open="!!reverting" :title="`Revert “${reverting?.summary}”?`" confirm="Revert" :busy="busy" @yes="revert" @no="reverting = null">
      This goes back from <b class="text-fg">{{ reverting?.after }}</b> to <b class="text-fg">{{ reverting?.before }}</b>. The revert is recorded as a new change.
    </ConfirmDialog>
  </PageLayout>
</template>
