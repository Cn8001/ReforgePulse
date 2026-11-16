<script setup lang="ts">
import { ref } from 'vue'
import { Run } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/checkservice'
import type { CheckResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/models'

const checks = [
  { id: 'sfc', title: 'System files', what: 'sfc /verifyonly: checks Windows files against their known good copies. Reads only.', time: '5-15 min' },
  { id: 'dism', title: 'Windows image', what: 'DISM /ScanHealth: checks the component store Windows repairs itself from. Reads only.', time: '5-20 min' },
  { id: 'chkdsk', title: 'System drive', what: 'chkdsk /scan: checks the file system while Windows runs. Reports, does not repair.', time: '1-10 min' },
  { id: 'memtest', title: 'Memory test', what: 'Opens Windows Memory Diagnostic, which tests RAM after a restart you choose.', time: 'next restart' },
]
const results = ref<Record<string, CheckResult>>({})
const busy = ref('')
const error = ref('')

async function run(id: string) {
  busy.value = id
  error.value = ''
  try {
    results.value[id] = await Run(id)
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = ''
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <p v-if="error" class="font-mono text-xs text-bad">{{ error }}</p>
    <article v-for="c in checks" :key="c.id" class="border border-line bg-panel">
      <div class="flex items-center gap-4 px-4 py-3">
        <div class="min-w-0 flex-1">
          <p class="font-semibold">{{ c.title }} <span class="font-mono text-[11px] font-normal text-muted">· {{ c.time }}</span></p>
          <p class="text-xs text-muted">{{ c.what }}</p>
        </div>
        <button class="h-8 bg-accent px-4 font-mono text-xs font-bold text-white disabled:opacity-50" :disabled="!!busy" @click="run(c.id)">
          {{ busy === c.id ? 'running…' : 'run' }}</button>
      </div>
      <div v-if="results[c.id]" class="border-t border-line bg-raised px-4 py-3">
        <p class="font-mono text-[11px]" :class="results[c.id].exitCode === 0 ? 'text-ok' : 'text-warn'">exit code {{ results[c.id].exitCode }}</p>
        <pre class="mt-1 max-h-60 overflow-y-auto font-mono text-[11px] whitespace-pre-wrap text-muted select-text">{{ results[c.id].output }}</pre>
      </div>
    </article>
  </div>
</template>
