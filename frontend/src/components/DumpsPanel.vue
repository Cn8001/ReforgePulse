<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Analyze, HasDebugger, List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/dumpservice'
import type { Dump, DumpAnalysis } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const dumps = ref<Dump[] | null>(null)
const debuggerOk = ref(false)
const pending = ref<Dump | null>(null)
const busy = ref('')
const results = ref<Record<string, DumpAnalysis>>({})
const error = ref('')
const raw = ref<string | null>(null)

onMounted(async () => {
  try {
    ;[dumps.value, debuggerOk.value] = await Promise.all([List().then((d) => d ?? []), HasDebugger()])
  } catch (e) {
    error.value = String(e)
  }
})

async function analyze() {
  const d = pending.value
  pending.value = null
  if (!d) return
  busy.value = d.name
  try {
    results.value[d.name] = await Analyze(d.name)
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = ''
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <p class="text-muted">
      Each blue screen leaves a minidump. Its stop code is read directly.
      <template v-if="debuggerOk">Analyse runs the Windows debugger to name the driver involved.</template>
      <template v-else>Install WinDbg from the Microsoft Store to also see which driver each crash blames.</template>
    </p>

    <div class="panel">
      <p v-if="!dumps" class="px-4 py-6 text-muted">Reading the Minidump folder…</p>
      <p v-else-if="!dumps.length" class="px-4 py-6 text-muted">No crash dumps. Windows has not written one, or the folder was emptied.</p>
      <div v-for="d in dumps ?? []" :key="d.name" class="border-b border-line px-4 py-3 last:border-0">
        <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
          <span class="font-semibold">{{ d.codeName || 'Unknown stop code' }}</span>
          <span class="font-mono text-xs text-bad">{{ d.code || '?' }}</span>
          <span class="text-xs text-muted">{{ new Date(d.time).toLocaleString() }}, {{ formatBytes(d.size) }}</span>
          <button class="btn ml-auto h-7" :disabled="!debuggerOk || !!busy" @click="pending = d">{{ busy === d.name ? 'Analysing…' : 'Analyse' }}</button>
        </div>
        <p class="mt-1 font-mono text-[12px] text-muted">{{ d.name }}, parameters {{ d.params?.join(' ') }}</p>
        <dl v-if="results[d.name]" class="mt-3 grid grid-cols-[8rem_1fr] gap-y-1 rounded border border-line bg-raised px-3 py-2 text-[14px]">
          <dt class="text-muted">Blames</dt><dd class="font-semibold text-warn">{{ results[d.name].image || results[d.name].module || 'Not named' }}</dd>
          <dt class="text-muted">Stopped in</dt><dd class="font-mono text-xs select-text">{{ results[d.name].symbol || '–' }}</dd>
          <dt class="text-muted">Process</dt><dd>{{ results[d.name].process || '–' }}</dd>
          <dt class="text-muted">Failure bucket</dt><dd class="font-mono text-xs select-text">{{ results[d.name].bucket || '–' }}</dd>
          <dd class="col-span-2 mt-1"><button class="text-accent hover:underline" @click="raw = results[d.name].output">Show the debugger output</button></dd>
        </dl>
      </div>
    </div>
    <p class="text-xs text-muted">The module the debugger names is where the crash happened. That is usually, not always, the cause: unstable hardware makes healthy drivers crash too.</p>

    <ConfirmDialog :open="!!pending" :title="`Analyse ${pending?.codeName || 'this crash'}?`" confirm="Analyse" @yes="analyze" @no="pending = null">
      The first time, the debugger downloads symbol files from Microsoft's symbol server, tens of MB that are kept afterwards.
      Only the names and versions of the modules in the dump are sent to fetch them; the dump itself stays on this PC.
    </ConfirmDialog>
    <div v-if="raw !== null" role="dialog" aria-modal="true" class="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6" @click.self="raw = null">
      <div class="panel flex max-h-[80vh] w-[900px] max-w-full flex-col">
        <header class="flex items-center justify-between border-b border-line px-4 py-2.5">
          <h2 class="text-[14px] font-semibold">Debugger output</h2><button class="btn h-7" @click="raw = null">Close</button>
        </header>
        <pre class="overflow-auto p-4 font-mono text-[12px] whitespace-pre-wrap select-text">{{ raw }}</pre>
      </div>
    </div>
  </div>
</template>
