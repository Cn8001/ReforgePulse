<script setup lang="ts">
import { computed, useId } from 'vue'

// values are percentages, oldest first; the line always spans the full width.
const props = defineProps<{ values: number[]; color: string; slots?: number }>()
const id = useId()

const points = computed(() => {
  const n = props.slots ?? 60
  const vs = props.values.slice(-n)
  const offset = n - vs.length
  return vs.map((v, i) => `${((offset + i) / (n - 1)) * 100},${30 - (Math.min(100, Math.max(0, v)) / 100) * 28}`)
})
const line = computed(() => points.value.join(' '))
const area = computed(() => {
  const p = points.value
  return p.length ? `M${p[0].split(',')[0]},30 L${p.join(' L')} L100,30 Z` : ''
})
</script>

<template>
  <svg viewBox="0 0 100 30" preserveAspectRatio="none" class="h-12 w-full">
    <defs>
      <linearGradient :id="id" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" :stop-color="color" stop-opacity=".35" />
        <stop offset="1" :stop-color="color" stop-opacity="0" />
      </linearGradient>
    </defs>
    <path :d="area" :fill="`url(#${id})`" />
    <polyline :points="line" fill="none" :stroke="color" stroke-width="1.5" vector-effect="non-scaling-stroke" stroke-linejoin="round" />
  </svg>
</template>
