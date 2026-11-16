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
    <div class="flex flex-wrap items-center gap-3 border border-line bg-panel px-4 py-3 font-mono text-xs">
      <span class="text-muted">record CPU, GPU, memory and every process while you play or work, then see what happened</span>
      <label class="ml-auto flex items-center gap-2">minutes <input v-model.number="minutes" type="number" min="1" max="240" class="h-8 w-20 border border-line bg-raised px-2" /></label>
      <button class="h-8 bg-accent px-4 font-bold text-white disabled:opacity-60" :disabled="running" @click="start">{{ running ? 'recording… stop from the status bar' : 'start' }}</button>
    </div>

    <template v-if="rec">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <article v-for="[label, values, color] in [['CPU', rec.cpu, 'var(--accent)'], ['GPU', rec.gpu, 'var(--gpu)'], ['Memory', rec.memory, 'var(--mem)']] as const"
                 :key="label" class="border border-line bg-panel p-4">
          <p class="caption">{{ label }} · {{ rec.seconds }} s</p>
          <p class="mt-1 font-mono text-sm">peak <b :style="{ color }">{{ peak(values) }}%</b> · average {{ avg(values) }}%</p>
          <Sparkline :values="values ?? []" :color="color" :slots="Math.max(2, values?.length ?? 2)" />
        </article>
      </div>
      <article class="border border-line bg-panel">
        <header class="border-b border-line px-4 py-2.5"><p class="caption">processes by peak memory · leaks first</p></header>
        <table class="w-full font-mono text-xs">
          <tr class="text-muted"><td class="px-4 py-2">process</td><td>start</td><td>end</td><td>peak</td><td>growth</td><td>avg cpu</td><td></td></tr>
          <tr v-for="p in rec.top ?? []" :key="p.pid" class="border-t border-line/60">
            <td class="px-4 py-1.5">{{ p.name }} <span class="text-muted">{{ p.pid }}</span></td>
            <td>{{ Math.round(p.startMb) }} MB</td><td>{{ Math.round(p.endMb) }} MB</td><td>{{ Math.round(p.peakMb) }} MB</td>
            <td :class="p.mbPerMinute > 5 ? 'text-warn' : 'text-muted'">{{ p.mbPerMinute.toFixed(1) }} MB/min</td>
            <td>{{ p.avgCpu.toFixed(1) }}%</td>
            <td class="pr-4"><span v-if="p.leak" class="border border-bad/50 px-1.5 text-[10px] text-bad" :title="`steady growth, fit ${p.fit.toFixed(2)}`">likely leak</span></td>
          </tr>
        </table>
      </article>
    </template>
  </div>
</template>
