<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { List, Uninstall } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/programservice'
import type { InstalledProgram } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const error = ref('')
const progs = ref<InstalledProgram[] | null>(null)
const onlyForgotten = ref(true)
const query = ref('')
const pending = ref<InstalledProgram | null>(null)
const busy = ref(false)

async function load() {
  try {
    progs.value = (await List()) ?? []
  } catch (e) {
    error.value = String(e)
  }
}

const rows = computed(() => {
  const q = query.value.trim().toLowerCase()
  return (progs.value ?? []).filter(
    (p) => (!onlyForgotten.value || p.forgotten) && (!q || p.name.toLowerCase().includes(q) || p.publisher.toLowerCase().includes(q)),
  )
})
const forgotten = computed(() => (progs.value ?? []).filter((p) => p.forgotten))

async function uninstall() {
  if (!pending.value) return
  busy.value = true
  try {
    await Uninstall(pending.value.id)
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    pending.value = null
  }
}

onMounted(load)
</script>

<template>
  <div class="flex flex-col gap-5">
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

    <div class="grid grid-cols-3 border border-line bg-panel font-mono">
      <div class="border-r border-line px-4 py-3"><p class="caption">installed</p><p class="mt-1 text-2xl tabular-nums">{{ progs?.length ?? '--' }}</p></div>
      <div class="border-r border-line px-4 py-3"><p class="caption">forgotten · 90+ days</p>
        <p class="mt-1 text-2xl text-warn tabular-nums">{{ forgotten.length }} <span class="text-sm text-muted">{{ formatBytes(forgotten.reduce((n, p) => n + p.size, 0)) }}</span></p></div>
      <div class="flex items-center justify-end gap-2 px-4 py-3">
        <button class="h-9 border px-3 font-mono text-xs transition"
                :class="onlyForgotten ? 'border-accent bg-accent/15 text-accent' : 'border-line text-muted hover:text-fg'"
                @click="onlyForgotten = !onlyForgotten">forgotten only</button>
        <label class="flex h-9 w-44 items-center gap-2 border border-line px-2 focus-within:border-accent">
          <Icon name="search" class="size-3.5! text-muted" />
          <input v-model="query" placeholder="filter" spellcheck="false" class="min-w-0 flex-1 bg-transparent font-mono text-xs outline-none" />
        </label>
      </div>
    </div>

    <div class="border border-line bg-panel">
      <p v-if="!progs" class="px-4 py-6 font-mono text-xs text-muted">// reading installed programs…</p>
      <p v-else-if="!rows.length" class="px-4 py-6 font-mono text-xs text-ok">// nothing here{{ onlyForgotten ? ': every program was started in the last 90 days' : '' }}</p>
      <div v-for="p in rows" :key="p.id" class="flex items-center gap-4 border-b border-line px-4 py-2.5 last:border-0 hover:bg-hover">
        <div class="min-w-0 flex-1">
          <p class="flex items-center gap-2 font-medium">
            <span class="truncate">{{ p.name }}</span>
            <span v-if="p.forgotten" class="shrink-0 border border-warn/50 px-1.5 font-mono text-[10px] text-warn">forgotten</span>
          </p>
          <p class="truncate font-mono text-[11px] text-muted">{{ p.publisher || 'unknown publisher' }} · {{ p.version || '?' }} · {{ p.basis }}</p>
        </div>
        <span class="w-20 text-right font-mono text-xs tabular-nums" :class="p.size ? '' : 'text-muted'">{{ p.size ? formatBytes(p.size) : '--' }}</span>
        <button class="h-7 border border-line px-2.5 font-mono text-[11px] transition hover:border-bad hover:text-bad disabled:opacity-40"
                :disabled="!p.uninstaller" @click="pending = p">uninstall</button>
      </div>
    </div>

    <ConfirmDialog :open="!!pending" :title="`Uninstall ${pending?.name}?`" confirm="run uninstaller" danger :busy="busy"
                   @yes="uninstall" @no="pending = null">
      ReforgePulse starts the program's own uninstaller, which may ask its own questions. Settings and saved data it leaves behind show up under Leftovers later.
    </ConfirmDialog>
  </div>
</template>
