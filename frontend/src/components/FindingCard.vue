<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import type { Finding } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { severityTone } from '../lib/scan'

defineProps<{ f: Finding }>()
const emit = defineEmits<{ act: [kind: string, arg?: string] }>()
const open = ref(false)

const sureness: Record<string, string> = { confirmed: 'Confirmed', likely: 'Likely', possible: 'Possible' }
</script>

<template>
  <article class="panel">
    <div class="flex items-start gap-4 px-5 py-4">
      <span class="mt-1.5 size-2.5 shrink-0 rounded-full" :style="{ background: severityTone[f.severity] }" />
      <div class="min-w-0 flex-1">
        <h3 class="text-[14px] font-semibold">{{ f.title }}</h3>
        <p class="mt-1 text-muted">{{ f.summary }}</p>
        <p class="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <span class="chip capitalize" :style="{ color: severityTone[f.severity], borderColor: severityTone[f.severity] }">{{ f.severity }} severity</span>
          <span class="chip border-line text-muted">{{ sureness[f.confidence] ?? f.confidence }}</span>
          <button v-if="f.evidence?.length || f.alternatives" class="flex items-center gap-1 text-muted hover:text-fg" :aria-expanded="open" @click="open = !open">
            <Icon name="chevron" class="size-3! transition-transform" :class="open && 'rotate-90'" />Evidence
          </button>
        </p>
        <div v-if="open" class="mt-3 flex flex-col gap-1.5 border-l-2 border-line pl-3 text-xs">
          <p v-for="(e, i) in f.evidence" :key="i" class="text-muted select-text">{{ e }}</p>
          <p v-if="f.alternatives" class="mt-1"><span class="text-muted">Other explanations:</span> {{ f.alternatives }}</p>
        </div>
      </div>
      <button v-if="f.action" class="btn shrink-0" @click="emit('act', f.action.kind, f.action.arg)">{{ f.action.label }}</button>
    </div>
  </article>
</template>
