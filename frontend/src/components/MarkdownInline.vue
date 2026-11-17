<script setup lang="ts">
import type { Inline } from '../lib/markdown'

defineProps<{ parts: Inline[] }>()
const emit = defineEmits<{ open: [url: string] }>()

// Only web links open, in the system browser; anything else stays text.
const web = (href: string) => /^https?:\/\//i.test(href)
</script>

<template>
  <template v-for="(p, i) in parts" :key="i">
    <strong v-if="p.t === 'b'" class="font-semibold text-fg">{{ p.text }}</strong>
    <em v-else-if="p.t === 'i'">{{ p.text }}</em>
    <code v-else-if="p.t === 'code'" class="rounded bg-panel px-1 font-mono text-[0.95em]">{{ p.text }}</code>
    <button v-else-if="p.t === 'link' && web(p.href)" class="text-accent hover:underline" :title="p.href" @click="emit('open', p.href)">{{ p.text }}</button>
    <template v-else>{{ p.text }}</template>
  </template>
</template>
