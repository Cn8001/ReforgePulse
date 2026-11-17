<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { Cancel } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/taskservice'
import { followTasks, taskTitled } from '../lib/tasks'

// A running task shown where it was started: what it is doing, how far it
// is, and a way to stop it. The rest of the window stays usable meanwhile.
const props = defineProps<{ title: string }>()
onMounted(followTasks)
const task = computed(() => taskTitled(props.title))
</script>

<template>
  <div class="panel flex items-center gap-4 px-4 py-3" role="status" aria-live="polite">
    <div class="flex min-w-0 flex-1 flex-col gap-2">
      <p class="flex items-baseline justify-between gap-3">
        <span class="font-semibold">{{ task?.title ?? title }}</span>
        <span class="truncate text-xs text-muted">{{ task?.note }}</span>
      </p>
      <span class="relative h-1.5 overflow-hidden rounded-full bg-hover">
        <span v-if="task && task.progress >= 0" class="absolute inset-y-0 left-0 rounded-full bg-accent transition-[width] duration-300"
              :style="{ width: `${task.progress * 100}%` }" />
        <span v-else class="absolute inset-y-0 w-1/3 animate-[slide_1s_linear_infinite] rounded-full bg-accent" />
      </span>
    </div>
    <span class="w-10 shrink-0 text-right tabular-nums">{{ task && task.progress >= 0 ? `${Math.round(task.progress * 100)}%` : '' }}</span>
    <button class="btn shrink-0" :disabled="!task" @click="task && Cancel(task.id)">Cancel</button>
  </div>
</template>

<style scoped>
@keyframes slide {
  from { left: -33%; }
  to { left: 100%; }
}
</style>
