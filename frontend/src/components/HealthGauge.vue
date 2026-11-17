<script setup lang="ts">
import { computed } from 'vue'

// The health score as a ring: the number is what to read, the ring and its
// colour only say how far from 100 it is.
const props = defineProps<{ score: number | null }>()
const r = 26
const length = 2 * Math.PI * r
const tone = computed(() => {
  const s = props.score
  if (s === null) return 'var(--muted)'
  return s >= 80 ? 'var(--ok)' : s >= 50 ? 'var(--warn)' : 'var(--bad)'
})
</script>

<template>
  <span class="relative grid size-16 shrink-0 place-items-center">
    <svg viewBox="0 0 64 64" class="absolute inset-0 -rotate-90" aria-hidden="true">
      <circle cx="32" cy="32" :r="r" fill="none" stroke="var(--line)" stroke-width="5" />
      <circle v-if="score !== null" cx="32" cy="32" :r="r" fill="none" :stroke="tone" stroke-width="5" stroke-linecap="round"
              :stroke-dasharray="length" :stroke-dashoffset="length * (1 - score / 100)" class="transition-[stroke-dashoffset] duration-500" />
    </svg>
    <span class="readout text-[22px]" :style="{ color: score === null ? 'var(--muted)' : undefined }">{{ score ?? '–' }}</span>
  </span>
</template>
