<script setup lang="ts">
import PageLayout from './PageLayout.vue'
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import AnalysisView from './AnalysisView.vue'
import RecordingView from './RecordingView.vue'
import { DeleteReport, DeleteReportsBefore } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/scanservice'
import { Analyses, DeleteAnalysis } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/dumpservice'
import { DeleteRecording, Recording as GetRecording, Recordings } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/recorderservice'
import { DeleteResult, Results } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/checkservice'
import { checkTitle } from '../lib/checks'
import CheckResultView from './CheckResultView.vue'
import type { CheckResult, SavedAnalysis } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import type { Recording } from '../../bindings/github.com/cn8001/reforgepulse/internal/recorder/models'
import { loadLastScan, loadReports, openReport, reports, severityTone, viewed } from '../lib/scan'
import { when } from '../lib/format'

// What ReforgePulse found and measured, kept to look back at: scans, crash
// dump analyses and recorder sessions.
const props = defineProps<{ tab?: string }>()
const emit = defineEmits<{ open: [page: string, arg?: string] }>()

type Tab = 'scans' | 'dumps' | 'recordings' | 'checks'
const tab = ref<Tab>((['scans', 'dumps', 'recordings', 'checks'] as const).find((t) => t === props.tab) ?? 'scans')
const analyses = ref<SavedAnalysis[]>([])
const recordings = ref<Recording[]>([])
const checkResults = ref<CheckResult[]>([])
const opened = ref('') // the dump or recording shown in full
const openRec = ref<Recording | null>(null)
const error = ref('')
const busy = ref(false)
const before = ref('') // yyyy-mm-dd from the date field

async function load() {
  error.value = ''
  try {
    const [a, r, c] = await Promise.all([Analyses(), Recordings(), Results(), loadReports()])
    analyses.value = a ?? []
    recordings.value = r ?? []
    checkResults.value = c ?? []
  } catch (e) {
    error.value = String(e)
  }
}
onMounted(load)

const key = (a: SavedAnalysis) => `${a.name}@${a.dumped}`
function toggleDump(a: SavedAnalysis) {
  opened.value = opened.value === key(a) ? '' : key(a)
}
async function toggleRecording(r: Recording) {
  if (opened.value === r.id) {
    opened.value = ''
    return
  }
  try {
    openRec.value = await GetRecording(r.id!)
    opened.value = r.id!
  } catch (e) {
    error.value = String(e)
  }
}

async function showScan(id: string) {
  try {
    await openReport(id)
    emit('open', 'fix')
  } catch (e) {
    error.value = String(e)
  }
}

// The date field gives a local day; scans from its midnight on stay.
const cutoff = computed(() => {
  if (!before.value) return null
  const [y, m, d] = before.value.split('-').map(Number)
  return new Date(y, m - 1, d)
})

type Pending =
  | { kind: 'scan'; id: string; at: string }
  | { kind: 'scans-before'; count: number }
  | { kind: 'dump'; a: SavedAnalysis }
  | { kind: 'recording'; r: Recording }
  | { kind: 'check'; c: CheckResult }
const pending = ref<Pending | null>(null)

function askBefore() {
  const t = cutoff.value
  if (t) pending.value = { kind: 'scans-before', count: reports.value.filter((r) => new Date(r.at) < t).length }
}

const dialog = computed(() => {
  const p = pending.value
  if (!p) return { title: '', body: '' }
  switch (p.kind) {
    case 'scan':
      return { title: `Delete the scan from ${when(p.at)}?`, body: 'Only the saved report goes. Findings you marked resolved stay resolved.' }
    case 'scans-before':
      return {
        title: `Delete ${p.count} ${p.count === 1 ? 'scan' : 'scans'} from before ${cutoff.value?.toLocaleDateString()}?`,
        body: 'Only the saved reports go. Findings you marked resolved stay resolved.',
      }
    case 'dump':
      return { title: `Delete the analysis of ${p.a.name}?`, body: 'The dump file stays; you can analyse it again while Windows keeps it.' }
    case 'recording':
      return { title: `Delete the recording from ${when(p.r.started)}?`, body: '' }
    case 'check':
      return { title: `Delete the ${checkTitle(p.c.name).toLowerCase()} check from ${when(p.c.at)}?`, body: 'Only the saved result goes; nothing on the PC changes.' }
  }
})

async function confirmDelete() {
  const p = pending.value
  if (!p) return
  pending.value = null
  busy.value = true
  error.value = ''
  try {
    if (p.kind === 'scan') await DeleteReport(p.id)
    else if (p.kind === 'scans-before') await DeleteReportsBefore(cutoff.value!.toISOString())
    else if (p.kind === 'dump') await DeleteAnalysis(p.a.name, p.a.dumped)
    else if (p.kind === 'recording') await DeleteRecording(p.r.id!)
    else await DeleteResult(p.c.id!)
    if (p.kind === 'scan' || p.kind === 'scans-before') {
      if (viewed.value && (p.kind === 'scan' ? viewed.value.id === p.id : new Date(viewed.value.at) < cutoff.value!)) viewed.value = null
      await loadLastScan()
    }
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
  }
}

const leaks = (r: Recording) => (r.top ?? []).filter((t) => t.leak).length
const minutes = (r: Recording) => Math.max(1, Math.round(r.seconds / 60))
</script>

<template>
  <PageLayout>
    <template #head>
      <header>
        <h1 class="heading">Reports</h1>
        <p class="mt-1 text-muted">Scans, crash dump analyses, recordings and health checks, kept to look back at. The last 30 scans, 20 recordings and 50 check results are kept.</p>
      </header>
      <nav class="tabs" role="tablist">
        <button role="tab" class="tab" :class="tab === 'scans' && 'tab-on'" :aria-selected="tab === 'scans'" @click="tab = 'scans'">
          Scans <span class="text-xs text-muted">{{ reports.length }}</span>
        </button>
        <button role="tab" class="tab" :class="tab === 'dumps' && 'tab-on'" :aria-selected="tab === 'dumps'" @click="tab = 'dumps'">
          Crash dumps <span class="text-xs text-muted">{{ analyses.length }}</span>
        </button>
        <button role="tab" class="tab" :class="tab === 'recordings' && 'tab-on'" :aria-selected="tab === 'recordings'" @click="tab = 'recordings'">
          Recordings <span class="text-xs text-muted">{{ recordings.length }}</span>
        </button>
        <button role="tab" class="tab" :class="tab === 'checks' && 'tab-on'" :aria-selected="tab === 'checks'" @click="tab = 'checks'">
          Checks <span class="text-xs text-muted">{{ checkResults.length }}</span>
        </button>
      </nav>
    </template>

    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <template v-if="tab === 'scans'">
      <div class="flex flex-wrap items-center gap-2">
        <label for="before" class="text-muted">Delete scans before</label>
        <input id="before" v-model="before" type="date" class="field" />
        <button class="btn" :disabled="!before || busy" @click="askBefore"><Icon name="trash" class="size-3.5!" />Delete</button>
      </div>
      <article class="panel">
        <p v-if="!reports.length" class="px-4 py-6 text-center text-muted">No saved scans. Run Scan PC on Issues.</p>
        <ul v-else>
          <li v-for="r in reports" :key="r.id" class="flex items-center gap-3 border-b border-line px-4 py-2 last:border-b-0 hover:bg-hover">
            <span class="size-2 shrink-0 rounded-full" :style="{ background: r.open ? severityTone[r.worst ?? 'info'] : 'var(--ok)' }" />
            <span class="w-36 shrink-0 tabular-nums">{{ when(r.at) }}</span>
            <span class="flex-1 text-muted">
              {{ r.open ? `${r.open} open` : 'Nothing open' }}<template v-if="r.resolved">, {{ r.resolved }} resolved</template>
            </span>
            <button class="btn h-7" @click="showScan(r.id)">Open</button>
            <button class="rounded p-1 text-muted hover:bg-hover hover:text-bad" :aria-label="`Delete the scan from ${when(r.at)}`"
                    title="Delete this scan" @click="pending = { kind: 'scan', id: r.id, at: r.at }">
              <Icon name="trash" class="size-3.5!" />
            </button>
          </li>
        </ul>
      </article>
    </template>

    <article v-else-if="tab === 'dumps'" class="panel">
      <p v-if="!analyses.length" class="px-4 py-6 text-center text-muted">No analyses yet. Analyse a crash under Tools, Minidumps.</p>
      <div v-for="a in analyses" :key="key(a)" class="border-b border-line last:border-b-0">
        <div class="flex items-center gap-3 px-4 py-2 hover:bg-hover">
          <span class="w-36 shrink-0 tabular-nums">{{ when(a.dumped) }}</span>
          <span class="min-w-0 flex-1 truncate">
            <span class="font-mono text-xs text-muted">{{ a.name }}</span>
            <span class="ml-2">{{ a.image || a.module || 'No module named' }}</span>
          </span>
          <button class="btn h-7" :aria-expanded="opened === key(a)" @click="toggleDump(a)">{{ opened === key(a) ? 'Close' : 'Open' }}</button>
          <button class="rounded p-1 text-muted hover:bg-hover hover:text-bad" :aria-label="`Delete the analysis of ${a.name}`"
                  title="Delete this analysis" @click="pending = { kind: 'dump', a }">
            <Icon name="trash" class="size-3.5!" />
          </button>
        </div>
        <AnalysisView v-if="opened === key(a)" :a="a" class="px-4 pb-3" />
      </div>
    </article>

    <article v-else-if="tab === 'checks'" class="panel">
      <p v-if="!checkResults.length" class="px-4 py-6 text-center text-muted">No check results yet. Run one under Tools, Checks.</p>
      <div v-for="c in checkResults" :key="c.id" class="border-b border-line last:border-b-0">
        <div class="flex items-center gap-3 px-4 py-2 hover:bg-hover">
          <span class="w-36 shrink-0 tabular-nums">{{ when(c.at) }}</span>
          <span class="flex-1">{{ checkTitle(c.name) }}</span>
          <span class="chip" :class="c.exitCode === 0 ? 'border-line text-muted' : 'border-warn text-warn'">{{ c.exitCode === 0 ? 'Passed' : c.name === 'memtest' && c.exitCode === 2 ? 'Cancelled' : 'Needs a look' }}</span>
          <button class="btn h-7" :aria-expanded="opened === c.id" @click="opened = opened === c.id ? '' : c.id!">{{ opened === c.id ? 'Close' : 'Open' }}</button>
          <button v-if="!c.fromWindows" class="rounded p-1 text-muted hover:bg-hover hover:text-bad" :aria-label="`Delete the ${checkTitle(c.name)} result from ${when(c.at)}`"
                  title="Delete this result" @click="pending = { kind: 'check', c }">
            <Icon name="trash" class="size-3.5!" />
          </button>
          <span v-else class="w-[26px] shrink-0" aria-hidden="true" />
        </div>
        <CheckResultView v-if="opened === c.id" :r="c" class="mx-4 mb-3 rounded border border-line bg-raised p-3" />
      </div>
    </article>

    <article v-else class="panel">
      <p v-if="!recordings.length" class="px-4 py-6 text-center text-muted">No recordings yet. Start one under Tools, Recorder.</p>
      <div v-for="r in recordings" :key="r.id" class="border-b border-line last:border-b-0">
        <div class="flex items-center gap-3 px-4 py-2 hover:bg-hover">
          <span class="w-36 shrink-0 tabular-nums">{{ when(r.started) }}</span>
          <span class="flex-1 text-muted">
            {{ minutes(r) }} min<template v-if="leaks(r)">, <span class="text-warn">{{ leaks(r) }} likely {{ leaks(r) === 1 ? 'leak' : 'leaks' }}</span></template>
            <template v-else-if="r.top?.length">, highest memory {{ r.top[0].name }}</template>
          </span>
          <button class="btn h-7" :aria-expanded="opened === r.id" @click="toggleRecording(r)">{{ opened === r.id ? 'Close' : 'Open' }}</button>
          <button class="rounded p-1 text-muted hover:bg-hover hover:text-bad" :aria-label="`Delete the recording from ${when(r.started)}`"
                  title="Delete this recording" @click="pending = { kind: 'recording', r }">
            <Icon name="trash" class="size-3.5!" />
          </button>
        </div>
        <RecordingView v-if="opened === r.id && openRec" :rec="openRec" class="px-4 pb-4" />
      </div>
    </article>

    <ConfirmDialog :open="!!pending" :title="dialog.title" confirm="Delete" danger warning="This cannot be undone. Are you sure?"
                   @yes="confirmDelete" @no="pending = null">
      {{ dialog.body }}
    </ConfirmDialog>
  </PageLayout>
</template>
