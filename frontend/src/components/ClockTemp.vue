<script setup lang="ts">
// One line under a tile's percentage: how fast it runs and how hot it is.
// A missing reading says why instead of showing a number it does not have.
defineProps<{ clock?: string; temp?: number | null; missing?: string }>()

const tone = (t: number) => (t >= 90 ? 'text-bad' : t >= 75 ? 'text-warn' : 'text-ok')
</script>

<template>
  <p class="flex items-center gap-4 font-mono text-xs">
    <span v-if="clock"><span class="text-muted">clock</span> {{ clock }}</span>
    <span v-if="temp != null"><span class="text-muted">temp</span> <b :class="tone(temp)">{{ Math.round(temp) }} °C</b></span>
    <span v-else-if="missing" class="text-muted" :title="missing">temp <span class="underline decoration-dotted">not measured</span></span>
  </p>
</template>
