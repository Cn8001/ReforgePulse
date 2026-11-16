<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from '../lib/icons'

defineProps<{ icon: IconName; title: string; text: string }>()
</script>

<template>
  <section class="grid h-full min-h-96 place-items-center text-center">
    <div class="max-w-sm">
      <span class="mx-auto mb-4 grid size-12 place-items-center rounded-2xl bg-accent/15 text-accent">
        <Icon :name="icon" />
      </span>
      <h1 class="text-xl font-semibold tracking-tight">{{ title }}</h1>
      <p class="mt-2 text-muted">{{ text }}</p>
      <p class="mt-4 inline-block rounded-full border border-line px-3 py-1 text-xs text-muted">Coming in the next version</p>
    </div>
  </section>
</template>
