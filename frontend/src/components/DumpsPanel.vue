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
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">{{ error }}</p>
    <p v-if="!debuggerOk" class="font-mono text-[11px] text-muted">// install WinDbg from the Microsoft Store to see which driver each crash blames; the stop codes below need nothing</p>

    <div class="border border-line bg-panel">
      <p v-if="!dumps" class="px-4 py-6 font-mono text-xs text-muted">// reading the Minidump folder…</p>
      <p v-else-if="!dumps.length" class="px-4 py-6 font-mono text-xs text-ok">// no crash dumps</p>
      <div v-for="d in dumps ?? []" :key="d.name" class="border-b border-line px-4 py-3 last:border-0">
        <div class="flex flex-wrap items-center gap-3">
          <span class="font-mono text-xs">{{ d.name }}</span>
          <span class="font-mono text-[11px] text-muted">{{ new Date(d.time).toLocaleString() }} · {{ formatBytes(d.size) }}</span>
          <span class="ml-auto font-mono text-xs"><b class="text-bad">{{ d.code || '?' }}</b> {{ d.codeName }}</span>
          <button class="h-7 bg-accent px-3 font-mono text-[11px] font-bold text-white transition hover:brightness-110 disabled:opacity-50"
                  :disabled="!debuggerOk || !!busy" @click="pending = d">{{ busy === d.name ? 'analysing…' : 'analyse' }}</button>
        </div>
        <p class="mt-1 font-mono text-[10px] text-muted">params {{ d.params?.join(' ') }}</p>
        <div v-if="results[d.name]" class="mt-2 grid grid-cols-[7rem_1fr] gap-y-1 border-l-2 border-accent pl-3 font-mono text-xs">
          <span class="text-muted">blames</span><span class="font-bold text-warn">{{ results[d.name].image || results[d.name].module || 'not named' }}</span>
          <span class="text-muted">stopped in</span><span class="select-text">{{ results[d.name].symbol || '--' }}</span>
          <span class="text-muted">process</span><span>{{ results[d.name].process || '--' }}</span>
          <span class="text-muted">bucket</span><span class="select-text">{{ results[d.name].bucket || '--' }}</span>
          <button class="col-span-2 mt-1 text-left text-accent hover:underline" @click="raw = results[d.name].output">raw output</button>
        </div>
      </div>
    </div>
    <p class="font-mono text-[11px] text-muted">// the module the debugger names is where the crash happened, which is usually but not always the cause; unstable hardware makes healthy drivers crash too.</p>

    <ConfirmDialog :open="!!pending" :title="`Analyse ${pending?.name}?`" confirm="analyse" @yes="analyze" @no="pending = null">
      The debugger downloads symbol files from Microsoft's symbol server the first time (tens of MB, cached afterwards).
      Only the names and versions of the modules in the dump are sent to fetch them; the dump itself stays on this PC.
    </ConfirmDialog>
    <div v-if="raw !== null" class="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6" @click.self="raw = null">
      <pre class="max-h-[80vh] w-[900px] max-w-full overflow-auto border border-line bg-panel p-4 font-mono text-[11px] whitespace-pre-wrap select-text">{{ raw }}</pre>
    </div>
  </div>
</template>
