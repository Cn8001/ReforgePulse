<script setup lang="ts">
import { ref } from 'vue'
import { Run } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/checkservice'
import type { CheckResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/models'

const checks = [
  { id: 'sfc', title: 'System files', what: 'Compares Windows files against their known good copies (sfc /verifyonly).', time: '5 to 15 min' },
  { id: 'dism', title: 'Windows image', what: 'Checks the store Windows repairs itself from (DISM /ScanHealth).', time: '5 to 20 min' },
  { id: 'chkdsk', title: 'System drive', what: 'Checks the file system while Windows runs (chkdsk /scan).', time: '1 to 10 min' },
  { id: 'memtest', title: 'Memory', what: 'Opens Windows Memory Diagnostic, which tests the RAM after a restart you choose.', time: 'one restart' },
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
    <p class="text-muted">Windows' own health checks. They only read and report; none of them repairs anything.</p>
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <article v-for="c in checks" :key="c.id" class="panel">
      <div class="flex items-center gap-4 px-4 py-3">
        <div class="min-w-0 flex-1">
          <p class="font-semibold">{{ c.title }} <span class="text-xs font-normal text-muted">takes {{ c.time }}</span></p>
          <p class="text-[13px] text-muted">{{ c.what }}</p>
        </div>
        <button class="btn-primary" :disabled="!!busy" @click="run(c.id)">{{ busy === c.id ? 'Running…' : 'Run' }}</button>
      </div>
      <div v-if="results[c.id]" class="border-t border-line bg-raised px-4 py-3">
        <p class="text-xs" :class="results[c.id].exitCode === 0 ? 'text-muted' : 'text-warn'">Finished with exit code {{ results[c.id].exitCode }}</p>
        <pre class="mt-1 max-h-60 overflow-y-auto font-mono text-[11px] whitespace-pre-wrap select-text">{{ results[c.id].output }}</pre>
      </div>
    </article>
  </div>
</template>
