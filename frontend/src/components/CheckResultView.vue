<script setup lang="ts">
import { ref } from 'vue'
import { ShowLog } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/checkservice'
import type { CheckResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

// A check's verdict. The full text stays in the log Windows keeps; only a
// run that just finished also has everything it printed.
defineProps<{ r: CheckResult }>()
const all = ref(false)
const error = ref('')
const show = (name: string) => ShowLog(name).catch((e) => (error.value = String(e)))
</script>

<template>
  <div class="flex flex-col gap-2">
    <pre class="font-mono text-[12px] whitespace-pre-wrap select-text">{{ r.summary || 'It printed no verdict.' }}</pre>
    <p class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted">
      <span v-if="r.fromWindows">Kept by Windows in its System event log.</span>
      <span v-else-if="r.exitCode !== 0" class="text-warn">Exit code {{ r.exitCode }}</span>
      <button v-if="r.log" class="text-accent hover:underline" :title="r.log" @click="show(r.name)">Show the full log</button>
      <button v-if="r.output" class="text-accent hover:underline" :aria-expanded="all" @click="all = !all">
        {{ all ? 'Hide' : 'Show' }} everything it printed
      </button>
    </p>
    <pre v-if="all" class="max-h-60 overflow-y-auto rounded border border-line bg-panel p-2 font-mono text-[12px] whitespace-pre-wrap select-text">{{ r.output }}</pre>
    <p v-if="error" class="text-xs text-bad">{{ error }}</p>
  </div>
</template>
