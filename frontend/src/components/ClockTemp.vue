<script setup lang="ts">
// The line under a readout: how fast it runs and how hot it is. A missing
// reading says why instead of showing a number it does not have.
defineProps<{ clock?: string; temp?: number | null; missing?: string }>()

const tone = (t: number) => (t >= 90 ? 'text-bad' : t >= 75 ? 'text-warn' : '')
</script>

<template>
  <p class="flex items-center gap-4 text-[13px] tabular-nums">
    <span v-if="clock">{{ clock }}</span>
    <span v-if="temp != null" class="font-semibold" :class="tone(temp)">{{ Math.round(temp) }} °C</span>
    <span v-else-if="missing" class="text-muted" :title="missing">Temperature not measured</span>
  </p>
</template>
