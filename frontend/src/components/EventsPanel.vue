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
const levelName = ['', 'Critical', 'Error', 'Warning']
const key = (g: EventGroup) => `${g.channel}|${g.provider}|${g.id}`
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center gap-2">
      <select v-model.number="days" aria-label="Period" class="field" @change="load">
        <option :value="1">Last day</option><option :value="7">Last 7 days</option><option :value="30">Last 30 days</option>
      </select>
      <button class="btn" :class="warnings && 'border-accent text-accent'" :aria-pressed="warnings" @click="warnings = !warnings; load()">Include warnings</button>
      <button class="btn" :class="hideNoise && 'border-accent text-accent'" :aria-pressed="hideNoise" @click="hideNoise = !hideNoise">Hide harmless</button>
      <label class="flex h-8 min-w-48 flex-1 items-center gap-2 rounded border border-line bg-raised px-2 focus-within:border-accent">
        <Icon name="search" class="size-4! text-muted" />
        <input v-model="query" aria-label="Filter events" placeholder="Source, event ID or text" class="min-w-0 flex-1 bg-transparent outline-none" />
      </label>
      <span class="text-xs text-muted">{{ groups ? `${rows.length} kinds of events` : 'Reading the logs…' }}</span>
    </div>
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <div class="panel">
      <div v-for="g in rows" :key="key(g)" class="border-b border-line last:border-0">
        <button class="flex w-full items-start gap-4 px-4 py-2.5 text-left hover:bg-hover" :aria-expanded="open === key(g)" @click="open = open === key(g) ? null : key(g)">
          <span class="mt-1.5 size-2 shrink-0 rounded-full" :class="g.level === 1 ? 'bg-bad' : g.level === 2 ? 'bg-warn' : 'bg-line'" :title="levelName[g.level]" />
          <span class="w-12 shrink-0 text-right font-semibold tabular-nums">{{ g.count }}×</span>
          <span class="min-w-0 flex-1">
            <span class="block truncate">{{ g.explain || g.message.split('\n')[0] || 'No description' }}</span>
            <span class="block truncate text-xs text-muted">{{ g.provider }}, event {{ g.id }}, {{ g.channel }} log</span>
          </span>
          <span class="shrink-0 text-xs text-muted">{{ since(g.last) }}</span>
        </button>
        <div v-if="open === key(g)" class="border-t border-line bg-raised px-4 py-3 text-xs">
          <p class="text-muted">First {{ new Date(g.first).toLocaleString() }}, last {{ new Date(g.last).toLocaleString() }}</p>
          <pre class="mt-2 font-sans text-[14px] whitespace-pre-wrap select-text">{{ g.message || 'Windows has no text for this event.' }}</pre>
        </div>
      </div>
      <p v-if="groups && !rows.length" class="px-4 py-6 text-muted">No events at this level.</p>
    </div>
  </div>
</template>
