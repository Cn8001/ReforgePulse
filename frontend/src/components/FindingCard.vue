<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import type { Finding } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { severityTone } from '../lib/scan'

defineProps<{ f: Finding }>()
const emit = defineEmits<{ act: [kind: string, arg?: string] }>()
const open = ref(false)
</script>

<template>
  <article class="relative border border-line bg-panel">
    <span class="absolute inset-y-0 left-0 w-[3px]" :style="{ background: severityTone[f.severity] }" />
    <div class="flex items-start gap-4 px-5 py-4">
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <span class="border px-1.5 font-mono text-[10px] uppercase" :style="{ color: severityTone[f.severity], borderColor: severityTone[f.severity] }">{{ f.severity }}</span>
          <span class="border border-line px-1.5 font-mono text-[10px] text-muted uppercase">{{ f.confidence }}</span>
          <h3 class="font-semibold">{{ f.title }}</h3>
        </div>
        <p class="mt-1.5 text-muted">{{ f.summary }}</p>
        <button v-if="f.evidence?.length || f.alternatives" class="mt-2 flex items-center gap-1 font-mono text-[11px] text-muted hover:text-fg" @click="open = !open">
          <Icon name="chevron" class="size-3! transition-transform" :class="open && 'rotate-90'" />evidence
        </button>
        <div v-if="open" class="mt-2 flex flex-col gap-2 border-l border-line pl-3 font-mono text-[11px]">
          <p v-for="(e, i) in f.evidence" :key="i" class="text-muted select-text">{{ e }}</p>
          <p v-if="f.alternatives" class="text-fg"><span class="text-muted">also possible:</span> {{ f.alternatives }}</p>
        </div>
      </div>
      <button v-if="f.action" class="shrink-0 bg-accent px-3 py-1.5 font-mono text-xs font-bold text-white transition hover:brightness-110"
              @click="emit('act', f.action.kind, f.action.arg)">{{ f.action.label }}</button>
    </div>
  </article>
</template>
