<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { followTasks, taskTitled } from '../lib/tasks'

// How far the update download is, read from its task so the dialog,
// Settings and the status bar all agree. The bar slides until the size is
// known.
onMounted(followTasks)
const task = computed(() => taskTitled('Downloading ReforgePulse'))
const known = computed(() => !!task.value && task.value.progress >= 0)
</script>

<template>
  <div class="flex flex-col gap-1.5" role="progressbar" aria-label="Update download"
       :aria-valuenow="known ? Math.round(task!.progress * 100) : undefined" aria-valuemin="0" aria-valuemax="100">
    <span class="relative h-1.5 overflow-hidden rounded-full bg-hover">
      <span v-if="known" class="absolute inset-y-0 left-0 rounded-full bg-accent transition-[width] duration-300"
            :style="{ width: `${task!.progress * 100}%` }" />
      <span v-else class="absolute inset-y-0 w-1/3 animate-[slide_1s_linear_infinite] rounded-full bg-accent" />
    </span>
    <p class="flex justify-between gap-3 text-xs text-muted">
      <span class="truncate">{{ task?.note || 'Starting the download' }}</span>
      <span class="shrink-0 tabular-nums">{{ known ? `${Math.round(task!.progress * 100)}%` : '' }}</span>
    </p>
  </div>
</template>

<style scoped>
@keyframes slide {
  from { left: -33%; }
  to { left: 100%; }
}
</style>
