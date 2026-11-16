<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import type { Suspect } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

defineProps<{ suspects: Suspect[] }>()
const open = ref<string | null>(null)

// Unmeasured is its own state, never shown as clean: Windows cannot see it.
const verdict: Record<string, { text: string; tone: string; dot: string }> = {
  strong: { text: 'Strong evidence', tone: 'text-bad', dot: 'var(--bad)' },
  some: { text: 'Some evidence', tone: 'text-warn', dot: 'var(--warn)' },
  none: { text: 'No evidence', tone: 'text-muted', dot: 'var(--line)' },
  unmeasured: { text: 'Not measured', tone: 'text-muted', dot: 'transparent' },
}
</script>

<template>
  <article class="panel">
    <header class="border-b border-line px-5 py-3">
      <h2 class="text-[14px] font-semibold">Where the instability points</h2>
      <p class="mt-0.5 text-xs text-muted">Every possible cause, weighed against what Windows recorded. More than one can be true.</p>
    </header>
    <ul class="divide-y divide-line">
      <li v-for="s in suspects" :key="s.area">
        <button class="flex w-full items-center gap-3 px-5 py-2.5 text-left transition hover:bg-hover" :aria-expanded="open === s.area"
                @click="open = open === s.area ? null : s.area">
          <span class="size-2.5 shrink-0 rounded-full border border-line" :style="{ background: verdict[s.verdict]?.dot }" />
          <span class="w-44 shrink-0 font-medium">{{ s.title }}</span>
          <span class="flex-1 text-[13px]" :class="verdict[s.verdict]?.tone">{{ verdict[s.verdict]?.text ?? s.verdict }}</span>
          <span class="text-xs text-muted">{{ s.for?.length ?? 0 }} for, {{ s.against?.length ?? 0 }} against</span>
          <Icon name="chevron" class="size-3.5! text-muted transition-transform" :class="open === s.area && 'rotate-90'" />
        </button>
        <div v-if="open === s.area" class="grid gap-4 bg-raised px-5 py-4 text-[13px] md:grid-cols-3">
          <div>
            <h3 class="label mb-1.5">Points here</h3>
            <p v-for="(x, i) in s.for ?? []" :key="i" class="mb-1 select-text">{{ x }}</p>
            <p v-if="!s.for?.length" class="text-muted">Nothing.</p>
          </div>
          <div>
            <h3 class="label mb-1.5">Points away</h3>
            <p v-for="(x, i) in s.against ?? []" :key="i" class="mb-1 select-text">{{ x }}</p>
            <p v-if="!s.against?.length" class="text-muted">Nothing.</p>
          </div>
          <div>
            <h3 class="label mb-1.5">Check next</h3>
            <p v-for="(x, i) in s.checks ?? []" :key="i" class="mb-1">{{ x }}</p>
          </div>
        </div>
      </li>
    </ul>
  </article>
</template>
