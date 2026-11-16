<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from '../lib/icons'

// A readout panel: the label, one big number, the detail under it.
defineProps<{ title: string; icon: IconName; value?: string; unit?: string; sub?: string }>()
</script>

<template>
  <article class="panel flex min-w-0 flex-col gap-3 p-4 transition-colors hover:bg-raised">
    <header class="flex items-center gap-2 text-muted">
      <Icon :name="icon" class="size-4!" />
      <span class="text-[13px] font-medium text-fg">{{ title }}</span>
      <slot name="badge" />
    </header>
    <p v-if="value !== undefined" class="readout">
      {{ value }}<span v-if="unit" class="ml-1 text-base font-normal text-muted">{{ unit }}</span>
    </p>
    <p v-if="sub" class="truncate text-xs text-muted" :title="sub">{{ sub }}</p>
    <slot />
  </article>
</template>
