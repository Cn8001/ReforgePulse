<script setup lang="ts">
import { ref } from 'vue'
import Sparkline from './Sparkline.vue'
import { Record } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/recorderservice'
import type { Recording } from '../../bindings/github.com/cn8001/reforgepulse/internal/recorder/models'

const minutes = ref(10)
const rec = ref<Recording | null>(null)
const running = ref(false)

async function start() {
  running.value = true
  rec.value = null
  try {
    rec.value = await Record(minutes.value)
  } finally {
    running.value = false
  }
}
const peak = (v?: number[] | null) => Math.round(Math.max(0, ...(v ?? [])))
const avg = (v?: number[] | null) => (v?.length ? Math.round(v.reduce((a, b) => a + b, 0) / v.length) : 0)
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel flex flex-wrap items-center gap-3 px-4 py-3">
      <p class="min-w-64 flex-1 text-muted">Record processor, graphics and memory load and every process's memory while you play or work, then see what happened.</p>
      <label class="flex items-center gap-2 text-muted">Minutes <input v-model.number="minutes" type="number" min="1" max="240" class="field w-20" /></label>
      <button class="btn-primary" :disabled="running" @click="start">{{ running ? 'Recording…' : 'Start recording' }}</button>
    </div>
    <p v-if="running" class="text-xs text-muted">Stop early from the status bar; what was recorded so far is kept.</p>

    <template v-if="rec">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <article v-for="[label, values] in [['Processor', rec.cpu], ['Graphics', rec.gpu], ['Memory', rec.memory]] as const" :key="label" class="panel p-4">
          <h2 class="label">{{ label }}</h2>
          <p class="mt-1"><b class="readout text-[26px]">{{ peak(values) }}%</b> <span class="text-muted">peak, {{ avg(values) }}% on average</span></p>
          <Sparkline :values="values ?? []" color="var(--accent)" :slots="Math.max(2, values?.length ?? 2)" />
        </article>
      </div>
      <article class="panel">
        <header class="border-b border-line px-4 py-3">
          <h2 class="text-[13px] font-semibold">Processes over {{ Math.round(rec.seconds / 60) || 1 }} min</h2>
          <p class="text-xs text-muted">By highest memory. A likely leak is memory that kept growing steadily, not a single jump.</p>
        </header>
        <table class="w-full text-[13px] tabular-nums">
          <tr class="text-muted"><th class="px-4 py-2 text-left font-medium">Process</th><th class="text-right font-medium">Start</th><th class="text-right font-medium">End</th>
            <th class="text-right font-medium">Highest</th><th class="text-right font-medium">Growth</th><th class="text-right font-medium">CPU</th><th class="px-4"></th></tr>
          <tr v-for="p in rec.top ?? []" :key="p.pid" class="border-t border-line/60">
            <td class="px-4 py-1.5">{{ p.name }} <span class="font-mono text-xs text-muted">{{ p.pid }}</span></td>
            <td class="text-right">{{ Math.round(p.startMb) }} MB</td><td class="text-right">{{ Math.round(p.endMb) }} MB</td><td class="text-right">{{ Math.round(p.peakMb) }} MB</td>
            <td class="text-right" :class="p.mbPerMinute > 5 ? 'text-warn' : 'text-muted'">{{ p.mbPerMinute.toFixed(1) }} MB/min</td>
            <td class="text-right">{{ p.avgCpu.toFixed(1) }}%</td>
            <td class="px-4 text-right"><span v-if="p.leak" class="chip border-bad/50 text-bad" :title="`Fits a steady rise, R² ${p.fit.toFixed(2)}`">Likely leak</span></td>
          </tr>
        </table>
      </article>
    </template>
  </div>
</template>
