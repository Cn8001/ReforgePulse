<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import type { Suspect } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

// No verdict is shown: the evidence is listed and the reader weighs it.
// Areas with the most pointing at them come first.
const props = defineProps<{ suspects: Suspect[] }>()
const open = ref<string | null>(null)
const sorted = computed(() => [...props.suspects].sort((a, b) => (b.for?.length ?? 0) - (a.for?.length ?? 0)))
const count = (n: number, word: string) => (n ? `${n} ${word}` : '')
</script>

<template>
  <article class="panel">
    <header class="border-b border-line px-5 py-3">
      <h2 class="text-[15px] font-semibold">Where the instability points</h2>
      <p class="mt-0.5 text-xs text-muted">Every possible cause, weighed against what Windows recorded. More than one can be true.</p>
    </header>
    <ul class="divide-y divide-line">
      <li v-for="s in sorted" :key="s.area">
        <button class="flex w-full items-center gap-3 px-5 py-2.5 text-left transition hover:bg-hover" :aria-expanded="open === s.area"
                @click="open = open === s.area ? null : s.area">
          <span class="w-44 shrink-0 font-medium">{{ s.title }}</span>
          <span class="flex-1 text-[14px]">{{ count(s.for?.length ?? 0, 'pointing here') }}</span>
          <span class="text-xs text-muted">{{ count(s.against?.length ?? 0, 'pointing away') }}</span>
          <Icon name="chevron" class="size-3.5! text-muted transition-transform" :class="open === s.area && 'rotate-90'" />
        </button>
        <div v-if="open === s.area" class="grid gap-4 bg-raised px-5 py-4 text-[14px] md:grid-cols-3">
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
