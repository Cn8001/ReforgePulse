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
  <div class="flex flex-col gap-4">
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <div class="panel">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <div>
          <p v-if="progs"><b>{{ forgotten.length }}</b> of {{ progs.length }} programs have not been started in 90 days, using <b>{{ formatBytes(forgotten.reduce((n, p) => n + p.size, 0)) }}</b>.</p>
          <p v-else>Reading installed programs…</p>
          <p class="mt-0.5 text-xs text-muted">Last starts come from Windows' own launch records. Shared runtimes and drivers are never marked.</p>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn" :class="onlyForgotten && 'border-accent text-accent'" :aria-pressed="onlyForgotten" @click="onlyForgotten = !onlyForgotten">Unused only</button>
          <label class="flex h-8 w-44 items-center gap-2 rounded border border-line bg-raised px-2 focus-within:border-accent">
            <Icon name="search" class="size-4! text-muted" />
            <input v-model="query" aria-label="Filter programs" placeholder="Filter" spellcheck="false" class="min-w-0 flex-1 bg-transparent text-[14px] outline-none" />
          </label>
        </div>
      </header>
      <p v-if="progs && !rows.length" class="px-4 py-6 text-muted">{{ onlyForgotten ? 'Every program has been started in the last 90 days.' : 'No programs match.' }}</p>
      <div v-for="p in rows" :key="p.id" class="flex items-center gap-4 border-b border-line px-4 py-2.5 last:border-0 hover:bg-hover">
        <div class="min-w-0 flex-1">
          <p class="flex items-center gap-2">
            <span class="truncate font-medium">{{ p.name }}</span>
            <span v-if="p.forgotten" class="chip border-warn/50 text-warn">Unused</span>
          </p>
          <p class="truncate text-xs text-muted">{{ p.publisher || 'Unknown publisher' }}<template v-if="p.version">, version {{ p.version }}</template>. {{ p.basis.charAt(0).toUpperCase() + p.basis.slice(1) }}.</p>
        </div>
        <span class="w-20 text-right tabular-nums" :class="!p.size && 'text-muted'">{{ p.size ? formatBytes(p.size) : '–' }}</span>
        <button class="btn h-7" :disabled="!p.uninstaller" @click="pending = p">Uninstall</button>
      </div>
    </div>

    <ConfirmDialog :open="!!pending" :title="`Uninstall ${pending?.name}?`" confirm="Run uninstaller" danger
                   warning="This cannot be undone: to get it back you would have to install it again. Are you sure?" :busy="busy" @yes="uninstall" @no="pending = null">
      ReforgePulse starts the program's own uninstaller, which may ask its own questions. Anything it leaves behind shows up under Leftovers later.
    </ConfirmDialog>
  </div>
</template>
