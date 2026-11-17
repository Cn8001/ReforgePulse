<script setup lang="ts">
import { ref } from 'vue'
import type { DumpAnalysis } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { when } from '../lib/format'

// What the debugger said about one crash dump.
defineProps<{ a: DumpAnalysis }>()
const raw = ref(false)
</script>

<template>
  <div>
  <dl class="grid grid-cols-[8rem_minmax(0,1fr)] gap-y-1 rounded border border-line bg-raised px-3 py-2 text-[14px]">
    <dt class="text-muted">Blames</dt><dd class="font-semibold text-warn">{{ a.image || a.module || 'Not named' }}</dd>
    <dt class="text-muted">Stopped in</dt><dd class="font-mono text-xs break-all select-text">{{ a.symbol || '–' }}</dd>
    <dt class="text-muted">Process</dt><dd>{{ a.process || '–' }}</dd>
    <dt class="text-muted">Failure bucket</dt><dd class="font-mono text-xs break-all select-text">{{ a.bucket || '–' }}</dd>
    <dt v-if="a.at" class="text-muted">Analysed</dt><dd v-if="a.at">{{ when(a.at) }}</dd>
    <dd class="col-span-2 mt-1"><button class="text-accent hover:underline" @click="raw = true">Show the debugger output</button></dd>
  </dl>
  <div v-if="raw" role="dialog" aria-modal="true" class="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6" @click.self="raw = false">
    <div class="panel flex max-h-[80vh] w-[900px] max-w-full flex-col">
      <header class="flex items-center justify-between border-b border-line px-4 py-2.5">
        <h2 class="text-[14px] font-semibold">Debugger output</h2><button class="btn h-7" @click="raw = false">Close</button>
      </header>
      <pre class="overflow-auto p-4 font-mono text-[12px] whitespace-pre-wrap select-text">{{ a.output }}</pre>
    </div>
  </div>
  </div>
</template>
