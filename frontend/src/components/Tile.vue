<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from '../lib/icons'

// A readout panel: the label, one big number, the detail under it.
// clickable makes it a button with a ">" in its corner, so a card that
// opens a page says so before it is hovered. The extra bottom room keeps
// the ">" on a line of its own, clear of a graph at the card's end.
defineProps<{ title: string; icon: IconName; value?: string; unit?: string; sub?: string; clickable?: boolean }>()
const emit = defineEmits<{ click: [] }>()
</script>

<template>
  <article class="panel group relative flex min-w-0 flex-col gap-3 p-4 transition-colors"
           :class="clickable && 'cursor-pointer pb-9 hover:bg-raised focus-visible:outline-2 focus-visible:outline-accent'"
           :role="clickable ? 'button' : undefined" :tabindex="clickable ? 0 : undefined"
           @click="clickable && emit('click')" @keydown.enter.space.prevent="clickable && emit('click')">
    <header class="flex items-center gap-2 text-muted">
      <Icon :name="icon" class="size-4!" />
      <span class="text-[14px] font-medium text-fg">{{ title }}</span>
      <slot name="badge" />
    </header>
    <p v-if="value !== undefined" class="readout">
      {{ value }}<span v-if="unit" class="ml-1 text-base font-normal text-muted">{{ unit }}</span>
    </p>
    <p v-if="sub" class="truncate text-xs text-muted" :title="sub">{{ sub }}</p>
    <slot />
    <Icon v-if="clickable" name="chevron" class="absolute right-3 bottom-3 size-3.5! text-muted transition-colors group-hover:text-fg" />
  </article>
</template>
