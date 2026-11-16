<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Read } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/sensorservice'
import type { SensorStatus } from '../../bindings/github.com/cn8001/reforgepulse/internal/platform/models'

const st = ref<SensorStatus | null>(null)
let timer: ReturnType<typeof setInterval> | undefined
async function read() {
  if (!document.hidden) st.value = await Read().catch(() => st.value)
}
onMounted(() => {
  read()
  timer = setInterval(read, 2000)
})
onBeforeUnmount(() => clearInterval(timer))

const groups = computed(() => {
  const g: Record<string, NonNullable<SensorStatus['sensors']>> = {}
  for (const s of st.value?.sensors ?? []) (g[s.group || s.source] ??= []).push(s)
  return Object.entries(g)
})

// Colours by how close a temperature is to where hardware starts protecting itself.
function tone(kind: string, v: number) {
  if (kind !== 'temp') return ''
  return v >= 90 ? 'text-bad' : v >= 75 ? 'text-warn' : ''
}
const order = ['temp', 'volt', 'power', 'clock', 'fan', 'load']
</script>

<template>
  <div class="flex flex-col gap-4">
    <div v-if="st?.hwinfo" class="notice border-warn/40 bg-warn/10">
      <p class="font-semibold">CPU temperatures and voltages are not measured</p>
      <p class="mt-1 text-muted">
        Windows has no reliable CPU temperature, and reading it directly takes a kernel driver that ReforgePulse does not install.
        Start <b class="text-fg">HWiNFO</b> (sensors-only mode is enough), open its settings and turn on <b class="text-fg">Shared Memory Support</b>.
        Every sensor it sees then appears here and in the scan.
      </p>
    </div>
    <p v-if="st?.nvml && !st.sensors?.some((s) => s.source === 'nvml')" class="text-muted">No NVIDIA graphics sensors: {{ st.nvml }}</p>

    <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
      <article v-for="[name, list] in groups" :key="name" class="panel">
        <header class="border-b border-line px-4 py-2.5"><h2 class="text-[13px] font-semibold">{{ name }}</h2></header>
        <table class="w-full text-[13px] tabular-nums">
          <tr v-for="(s, i) in [...list].sort((a, b) => order.indexOf(a.kind) - order.indexOf(b.kind))" :key="i" class="border-t border-line/60 first:border-0">
            <td class="px-4 py-1.5 text-muted">{{ s.name }}</td>
            <td class="py-1.5 text-right" :class="tone(s.kind, s.value)">{{ s.value.toFixed(s.kind === 'volt' ? 3 : 0) }} {{ s.unit }}</td>
            <td class="w-28 px-4 py-1.5 text-right text-xs text-muted">{{ s.max ? `Highest ${s.max.toFixed(s.kind === 'volt' ? 3 : 0)}` : '' }}</td>
          </tr>
        </table>
      </article>
    </div>
  </div>
</template>
