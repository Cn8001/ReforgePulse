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
      <span class="absolute inset-x-0 top-0 h-10 bg-hover transition-transform duration-300 ease-[cubic-bezier(.2,.8,.2,1)]"
            :style="{ transform: `translateY(${index * 40}px)` }">
        <span class="absolute inset-y-0 left-0 w-[3px] bg-accent" />
      </span>
      <button v-for="(p, i) in pages" :key="p.id"
              class="group relative flex h-10 w-full items-center gap-3 px-4 text-left transition-colors"
              :class="page === p.id ? 'text-fg' : 'text-muted hover:text-fg'" @click="emit('go', p.id)">
        <span class="transition-transform duration-200 group-hover:scale-110" :class="page === p.id && 'text-accent'">
          <Icon :name="p.icon" />
        </span>
        <span class="flex-1" :class="page === p.id && 'font-semibold'">{{ p.label }}</span>
        <kbd class="font-mono text-[10px] text-muted opacity-0 transition-opacity group-hover:opacity-100">^{{ i + 1 }}</kbd>
      </button>
    </nav>
    <slot />
  </aside>
</template>
