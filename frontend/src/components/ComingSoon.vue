<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from '../lib/icons'

defineProps<{ icon: IconName; title: string; text: string }>()
</script>

<template>
  <section class="flex h-full min-h-[28rem] flex-col items-center justify-center gap-3 p-6 text-center">
    <span class="mb-2 grid size-14 place-items-center border border-line bg-panel text-accent"><Icon :name="icon" /></span>
    <p class="caption">module · not in this build</p>
    <h1 class="font-mono text-2xl font-semibold">{{ title }}</h1>
    <p class="max-w-md text-muted">{{ text }}</p>
  </section>
</template>
