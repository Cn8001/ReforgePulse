<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import FindingCard from './FindingCard.vue'
import CpuCapPanel from './CpuCapPanel.vue'
import { loadLastScan, runScan, scan, scanError, scanning } from '../lib/scan'
import { since } from '../lib/format'

const emit = defineEmits<{ open: [page: string, arg?: string] }>()
const capPanel = ref<HTMLElement>()

onMounted(loadLastScan)

const counts = computed(() => {
  const c: Record<string, number> = {}
  for (const f of scan.value?.findings ?? []) c[f.severity] = (c[f.severity] ?? 0) + 1
  return c
})

function act(kind: string, arg?: string) {
  if (kind === 'cpu-cap') capPanel.value?.scrollIntoView({ behavior: 'smooth' })
  else if (kind === 'open-processes') emit('open', 'processes', arg)
  else if (kind === 'open-cleanup') emit('open', 'cleanup', 'disk')
  else if (kind === 'open-startup') emit('open', 'cleanup', 'startup')
  else if (kind === 'open-programs') emit('open', 'cleanup', 'programs')
}
</script>

<template>
  <section class="flex flex-col gap-5 p-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="caption">crashes · hardware errors · bottlenecks · last 30 days</p>
        <h1 class="mt-1 text-2xl font-semibold">Fix Issues</h1>
      </div>
      <button class="flex h-10 items-center gap-2 bg-accent px-5 font-mono text-xs font-bold tracking-[.15em] text-white uppercase shadow-[0_0_24px_-6px_var(--accent)] transition hover:brightness-110 disabled:opacity-60"
              :disabled="scanning" @click="runScan">
        <Icon name="refresh" class="size-3.5!" :class="scanning && 'animate-spin'" />{{ scanning ? 'scanning' : scan ? 'scan again' : 'scan pc' }}
      </button>
    </header>

    <p v-if="scanError" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ scanError }}</p>

    <div v-if="scan" class="flex flex-wrap items-center gap-x-6 gap-y-1 border border-line bg-panel px-5 py-3 font-mono text-xs">
      <span class="text-muted">scanned {{ since(scan.at) }}</span>
      <span v-for="s in ['critical', 'high', 'medium', 'low', 'info']" :key="s" v-show="counts[s]">
        <b>{{ counts[s] }}</b> <span class="text-muted">{{ s }}</span>
      </span>
      <span v-if="!scan.findings?.length" class="text-ok">no problems found in the evidence Windows keeps</span>
    </div>
    <div v-else-if="!scanning" class="border border-line bg-panel px-5 py-8 text-center">
      <p class="font-semibold">Not scanned yet</p>
      <p class="mt-1 text-muted">The scan reads Windows' own records: blue screens, unexpected restarts, hardware errors, crashing apps and services, disks and memory. It changes nothing.</p>
    </div>

    <TransitionGroup tag="div" class="flex flex-col gap-3" enter-from-class="opacity-0 translate-y-1" enter-active-class="transition duration-300">
      <FindingCard v-for="(f, i) in scan?.findings ?? []" :key="f.rule + i" :f="f" @act="act" />
    </TransitionGroup>

    <div ref="capPanel" class="mt-4 flex flex-col gap-3">
      <p class="caption">tools</p>
      <CpuCapPanel />
    </div>
  </section>
</template>
