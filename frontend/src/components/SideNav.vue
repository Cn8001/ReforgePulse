<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'
import type { IconName } from '../lib/icons'

const props = defineProps<{ pages: { id: string; label: string; icon: IconName; group?: string }[]; page: string }>()
const emit = defineEmits<{ go: [id: string] }>()

// Rows are a fixed 40px, so the indicator can slide by index.
const index = computed(() => props.pages.findIndex((p) => p.id === props.page))
</script>

<template>
  <aside class="flex w-56 shrink-0 flex-col border-r border-line bg-panel/70 py-2">
    <nav class="relative">
      <span class="active absolute inset-x-0 top-0 h-10 transition-transform duration-300 ease-[cubic-bezier(.2,.8,.2,1)]"
            :style="{ transform: `translateY(${index * 40}px)` }">
        <span class="bar absolute inset-y-0 left-0 w-[3px]" />
      </span>
      <button v-for="p in pages" :key="p.id"
              class="group relative flex h-10 w-full items-center gap-3 px-4 text-left transition-colors"
              :class="page === p.id ? 'text-fg' : 'text-muted hover:text-fg'" @click="emit('go', p.id)">
        <span class="transition-transform duration-200 group-hover:scale-110" :class="page === p.id && 'text-accent'">
          <Icon :name="p.icon" />
        </span>
        <span class="flex-1" :class="page === p.id && 'font-semibold'">{{ p.label }}</span>
      </button>
    </nav>
    <slot />
  </aside>
</template>

<style scoped>
/* The active row: a black, blue and orange sweep that drifts slowly, the
   colours of the pulse trace and the bolt. */
.active {
  background: linear-gradient(90deg, color-mix(in srgb, var(--accent) 22%, transparent), color-mix(in srgb, #ff8a00 10%, transparent) 55%, transparent);
  background-size: 200% 100%;
  animation: drift 6s ease-in-out infinite alternate;
}
.bar {
  background: linear-gradient(180deg, #0b0e13, var(--accent), #ff8a00, var(--accent), #0b0e13);
  background-size: 100% 300%;
  animation: flow 2.4s linear infinite;
  box-shadow: 0 0 10px color-mix(in srgb, var(--accent) 70%, transparent);
}
@keyframes drift {
  to { background-position: 100% 0; }
}
@keyframes flow {
  to { background-position: 0 300%; }
}
</style>
