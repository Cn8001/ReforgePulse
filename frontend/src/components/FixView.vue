<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import FindingCard from './FindingCard.vue'
import Icon from './Icon.vue'
import SuspectList from './SuspectList.vue'
import { active, loadLastScan, loadReports, openReport, reports, runScan, scan, scanError, scanning, severityTone, viewed } from '../lib/scan'
import { since, when } from '../lib/format'

const emit = defineEmits<{ open: [page: string, arg?: string] }>()

onMounted(() => Promise.all([loadLastScan(), loadReports()]))

// What the page shows: an older report when one is open, else the latest.
const shown = computed(() => viewed.value ?? scan.value)
const current = computed(() => (shown.value?.findings ?? []).filter(active))
const resolved = computed(() => (shown.value?.findings ?? []).filter((f) => !active(f)))
const showResolved = ref(false)
const openError = ref('')

const order = ['critical', 'high', 'medium', 'low', 'info']
const counts = computed(() => {
  const c: Record<string, number> = {}
  for (const f of current.value) c[f.severity] = (c[f.severity] ?? 0) + 1
  return order.filter((s) => c[s]).map((s) => `${c[s]} ${s}`)
})

function summary(): string {
  if (counts.value.length) return `Found ${counts.value.join(', ')}.`
  if (resolved.value.length) return 'Everything found is marked resolved.'
  return 'Nothing in the last 30 days of records needs attention.'
}

async function pick(id: string) {
  openError.value = ''
  try {
    await openReport(id)
  } catch (e) {
    openError.value = String(e)
  }
}

function backToLatest() {
  viewed.value = null
}

function act(kind: string, arg?: string) {
  if (kind === 'cpu-cap') emit('open', 'tools', 'cpu-cap')
  else if (kind === 'open-tools') emit('open', 'tools', arg)
  else if (kind === 'open-processes') emit('open', 'processes', arg)
  else if (kind === 'open-cleanup') emit('open', 'cleanup', 'disk')
  else if (kind === 'open-startup') emit('open', 'cleanup', 'startup')
  else if (kind === 'open-programs') emit('open', 'cleanup', 'programs')
}
</script>

<template>
  <section class="mx-auto flex max-w-6xl flex-col gap-5 p-6">
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="heading">Issues</h1>
        <p class="mt-1 text-muted">
          <template v-if="shown">Scanned {{ since(shown.at) }}. {{ summary() }}</template>
          <template v-else>Crashes, hardware errors and slowdowns from the last 30 days of Windows records.</template>
        </p>
      </div>
      <button class="btn-primary h-9 px-5" :disabled="scanning" @click="runScan">{{ scanning ? 'Scanning…' : scan ? 'Scan again' : 'Scan PC' }}</button>
    </header>

    <p v-if="scanError" class="notice border-bad/40 bg-bad/10 text-bad">{{ scanError }}</p>
    <p v-if="openError" class="notice border-bad/40 bg-bad/10 text-bad">{{ openError }}</p>

    <div v-if="viewed" class="notice flex flex-wrap items-center justify-between gap-3 border-line bg-raised">
      <span>You are looking at the scan from {{ when(viewed.at) }}. Resolved marks show as they are now.</span>
      <button class="btn" @click="backToLatest">Back to the latest scan</button>
    </div>

    <div v-if="!shown && !scanning" class="panel px-5 py-10 text-center">
      <p class="text-[16px] font-semibold">Not scanned yet</p>
      <p class="mx-auto mt-1 max-w-lg text-muted">
        The scan reads blue screens, unexpected restarts, hardware errors, crashing apps and services, drives and memory. It changes nothing.
      </p>
      <button class="btn-primary mt-4" @click="runScan">Scan PC</button>
    </div>

    <SuspectList v-if="shown?.suspects?.length" :suspects="shown.suspects" />

    <div class="flex flex-col gap-3">
      <FindingCard v-for="f in current" :key="f.key" :f="f" @act="act" />
    </div>

    <div v-if="resolved.length" class="flex flex-col gap-3">
      <button class="flex items-center gap-2 self-start text-muted hover:text-fg" :aria-expanded="showResolved" @click="showResolved = !showResolved">
        <Icon name="chevron" class="size-3! transition-transform" :class="showResolved && 'rotate-90'" />
        Resolved ({{ resolved.length }})
      </button>
      <template v-if="showResolved">
        <FindingCard v-for="f in resolved" :key="f.key" :f="f" @act="act" />
      </template>
    </div>

    <div v-if="reports.length" class="panel">
      <div class="flex items-baseline justify-between border-b border-line px-5 py-3">
        <h2 class="text-[15px] font-semibold">Recent scans</h2>
        <span class="label">The last {{ reports.length }} are kept</span>
      </div>
      <ul>
        <li v-for="r in reports" :key="r.id" class="border-b border-line last:border-b-0">
          <button class="flex w-full items-center gap-3 px-5 py-2.5 text-left hover:bg-hover"
                  :class="r.id === shown?.id && 'bg-hover'" :aria-current="r.id === shown?.id" @click="pick(r.id)">
            <span class="size-2 shrink-0 rounded-full" :style="{ background: r.open ? severityTone[r.worst ?? 'info'] : 'var(--ok)' }" />
            <span class="w-36 shrink-0 tabular-nums">{{ when(r.at) }}</span>
            <span class="flex-1 text-muted">
              {{ r.open ? `${r.open} open` : 'Nothing open' }}<template v-if="r.resolved">, {{ r.resolved }} resolved</template>
            </span>
            <span v-if="r.id === scan?.id" class="chip border-line text-muted">Latest</span>
            <span v-if="r.id === shown?.id" class="label">Showing</span>
          </button>
        </li>
      </ul>
    </div>
  </section>
</template>
