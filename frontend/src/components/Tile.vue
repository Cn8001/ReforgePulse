<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from '../lib/icons'

// tint is a CSS colour; it colours the icon, the value and the top edge.
defineProps<{ title: string; icon: IconName; tint: string; value?: string; unit?: string; sub?: string }>()
</script>

<template>
  <article class="group relative flex min-w-0 flex-col gap-3 overflow-hidden border border-line bg-panel p-4 transition-colors hover:bg-raised">
    <span class="absolute inset-x-0 top-0 h-[2px] opacity-80" :style="{ background: tint }" />
    <header class="flex items-center gap-2">
      <span :style="{ color: tint }"><Icon :name="icon" /></span>
      <span class="caption">{{ title }}</span>
      <slot name="badge" />
    </header>
    <p v-if="value !== undefined" class="font-mono text-3xl leading-none font-semibold tabular-nums" :style="{ color: tint }">
      {{ value }}<span v-if="unit" class="ml-1 text-sm text-muted">{{ unit }}</span>
    </p>
    <p v-if="sub" class="truncate font-mono text-xs text-muted" :title="sub">{{ sub }}</p>
    <slot />
  </article>
</template>
