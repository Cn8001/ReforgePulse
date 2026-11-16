<script setup lang="ts">
import { computed, onMounted } from 'vue'
import FindingCard from './FindingCard.vue'
import SuspectList from './SuspectList.vue'
import { loadLastScan, runScan, scan, scanError, scanning } from '../lib/scan'
import { since } from '../lib/format'

const emit = defineEmits<{ open: [page: string, arg?: string] }>()

onMounted(loadLastScan)

const order = ['critical', 'high', 'medium', 'low', 'info']
const counts = computed(() => {
  const c: Record<string, number> = {}
  for (const f of scan.value?.findings ?? []) c[f.severity] = (c[f.severity] ?? 0) + 1
  return order.filter((s) => c[s]).map((s) => `${c[s]} ${s}`)
})

function act(kind: string, arg?: string) {
  if (kind === 'cpu-cap') emit('open', 'tools', 'cpu-cap')
  else if (kind === 'open-tools') emit('open', 'tools', arg)
  else if (kind === 'open-processes') emit('open', 'processes', arg)
  else if (kind === 'open-cleanup') emit('open', 'cleanup', 'disk')
  else if (kind === 'open-startup') emit('open', 'cleanup', 'startup')
  else if (kind === 'open-programs') emit('open', 'cleanup', 'programs')
}
</script>

<template>
  <section class="mx-auto flex max-w-6xl flex-col gap-5 p-6">
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="heading">Issues</h1>
        <p class="mt-1 text-muted">
          <template v-if="scan">Scanned {{ since(scan.at) }}. {{ counts.length ? `Found ${counts.join(', ')}.` : 'Nothing in the last 30 days of records needs attention.' }}</template>
          <template v-else>Crashes, hardware errors and slowdowns from the last 30 days of Windows records.</template>
        </p>
      </div>
      <button class="btn-primary h-9 px-5" :disabled="scanning" @click="runScan">{{ scanning ? 'Scanning…' : scan ? 'Scan again' : 'Scan PC' }}</button>
    </header>

    <p v-if="scanError" class="notice border-bad/40 bg-bad/10 text-bad">{{ scanError }}</p>

    <div v-if="!scan && !scanning" class="panel px-5 py-10 text-center">
      <p class="text-[15px] font-semibold">Not scanned yet</p>
      <p class="mx-auto mt-1 max-w-lg text-muted">
        The scan reads blue screens, unexpected restarts, hardware errors, crashing apps and services, drives and memory. It changes nothing.
      </p>
      <button class="btn-primary mt-4" @click="runScan">Scan PC</button>
    </div>

    <SuspectList v-if="scan?.suspects?.length" :suspects="scan.suspects" />

    <div class="flex flex-col gap-3">
      <FindingCard v-for="(f, i) in scan?.findings ?? []" :key="f.rule + i" :f="f" @act="act" />
    </div>
  </section>
</template>
