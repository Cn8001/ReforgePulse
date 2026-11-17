<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import AnalysisView from './AnalysisView.vue'
import { Analyze, HasDebugger, List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/dumpservice'
import type { Dump } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const dumps = ref<Dump[] | null>(null)
const debuggerOk = ref(false)
const pending = ref<Dump | null>(null)
const busy = ref('')
const error = ref('')

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
    d.analysis = await Analyze(d.name) // also saved, under Reports
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
          <button class="btn ml-auto h-7" :disabled="!debuggerOk || !!busy" @click="pending = d">{{ busy === d.name ? 'Analysing…' : d.analysis ? 'Analyse again' : 'Analyse' }}</button>
        </div>
        <p class="mt-1 font-mono text-[12px] text-muted">{{ d.name }}, parameters {{ d.params?.join(' ') }}</p>
        <AnalysisView v-if="d.analysis" :a="d.analysis" class="mt-3" />
      </div>
    </div>
    <p class="text-xs text-muted">The module the debugger names is where the crash happened. That is usually, not always, the cause: unstable hardware makes healthy drivers crash too.</p>

    <ConfirmDialog :open="!!pending" :title="`Analyse ${pending?.codeName || 'this crash'}?`" confirm="Analyse" @yes="analyze" @no="pending = null">
      The first time, the debugger downloads symbol files from Microsoft's symbol server, tens of MB that are kept afterwards.
      Only the names and versions of the modules in the dump are sent to fetch them; the dump itself stays on this PC.
    </ConfirmDialog>
  </div>
</template>
