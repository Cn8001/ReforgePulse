<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import { List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/eventservice'
import type { EventGroup } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { since } from '../lib/format'

const groups = ref<EventGroup[] | null>(null)
const days = ref(7)
const warnings = ref(false)
const hideNoise = ref(true)
const query = ref('')
const open = ref<string | null>(null)
const error = ref('')

async function load() {
  groups.value = null
  try {
    groups.value = (await List(days.value, warnings.value)) ?? []
  } catch (e) {
    error.value = String(e)
    groups.value = []
  }
}
onMounted(load)

const rows = computed(() => {
  const q = query.value.trim().toLowerCase()
  return (groups.value ?? []).filter(
    (g) => (!hideNoise.value || !g.noise) &&
      (!q || g.provider.toLowerCase().includes(q) || String(g.id) === q || g.message.toLowerCase().includes(q)),
  )
})
const levelName = ['', 'critical', 'error', 'warning']
const levelTone = ['', 'text-bad', 'text-warn', 'text-muted']
const key = (g: EventGroup) => `${g.channel}|${g.provider}|${g.id}`
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center gap-2 border border-line bg-panel px-4 py-3 font-mono text-xs">
      <select v-model.number="days" class="h-8 border border-line bg-raised px-2" @change="load">
        <option :value="1">last day</option><option :value="7">last 7 days</option><option :value="30">last 30 days</option>
      </select>
      <button class="h-8 border px-3" :class="warnings ? 'border-accent text-accent' : 'border-line text-muted'" @click="warnings = !warnings; load()">warnings</button>
      <button class="h-8 border px-3" :class="hideNoise ? 'border-accent text-accent' : 'border-line text-muted'" @click="hideNoise = !hideNoise">hide harmless</button>
      <label class="flex h-8 flex-1 items-center gap-2 border border-line px-2 focus-within:border-accent">
        <Icon name="search" class="size-3.5! text-muted" />
        <input v-model="query" placeholder="source, ID or text" class="min-w-0 flex-1 bg-transparent outline-none" />
      </label>
      <span class="text-muted">{{ groups ? `${rows.length} kinds` : 'reading…' }}</span>
    </div>
    <p v-if="error" class="font-mono text-xs text-bad">{{ error }}</p>

    <div class="border border-line bg-panel">
      <div v-for="g in rows" :key="key(g)" class="border-b border-line last:border-0">
        <button class="flex w-full items-start gap-3 px-4 py-2.5 text-left hover:bg-hover" @click="open = open === key(g) ? null : key(g)">
          <span class="w-14 shrink-0 font-mono text-[10px] uppercase" :class="levelTone[g.level]">{{ levelName[g.level] }}</span>
          <span class="w-12 shrink-0 text-right font-mono text-xs font-bold tabular-nums">{{ g.count }}×</span>
          <span class="min-w-0 flex-1">
            <span class="block truncate font-mono text-xs">{{ g.provider }} <span class="text-muted">{{ g.id }} · {{ g.channel }}</span></span>
            <span class="block truncate text-xs" :class="g.explain ? 'text-fg' : 'text-muted'">{{ g.explain || g.message.split('\n')[0] }}</span>
          </span>
          <span class="shrink-0 font-mono text-[11px] text-muted">{{ since(g.last) }}</span>
        </button>
        <div v-if="open === key(g)" class="border-t border-line bg-raised px-4 py-3 font-mono text-[11px]">
          <p class="text-muted">first {{ new Date(g.first).toLocaleString() }} · last {{ new Date(g.last).toLocaleString() }}</p>
          <pre class="mt-2 whitespace-pre-wrap select-text">{{ g.message || '(no message text)' }}</pre>
        </div>
      </div>
      <p v-if="groups && !rows.length" class="px-4 py-6 font-mono text-xs text-ok">// nothing at this level</p>
    </div>
  </div>
</template>
