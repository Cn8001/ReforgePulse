<script setup lang="ts">
import { onMounted, ref } from 'vue'

const props = defineProps<{ label: string; detail: string; ratio: number | null }>()
// Starts at zero so the bar grows in when the page opens.
const width = ref(0)
onMounted(() => requestAnimationFrame(() => (width.value = Math.min(1, Math.max(0, props.ratio ?? 0)))))
</script>

<template>
  <div>
    <div class="mb-1.5 flex items-baseline justify-between gap-3">
      <span class="font-medium">{{ label }}</span>
      <span class="text-xs text-muted tabular-nums">{{ detail }}</span>
    </div>
    <div class="h-1.5 overflow-hidden rounded-full bg-hover">
      <div v-if="ratio !== null"
           class="h-full rounded-full transition-[width] duration-700 ease-out"
           :class="width > 0.9 ? 'bg-[color:var(--color-red-500)]' : 'bg-accent'"
           :style="{ width: `${width * 100}%` }" />
    </div>
  </div>
</template>
