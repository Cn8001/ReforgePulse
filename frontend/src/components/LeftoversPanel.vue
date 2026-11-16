<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { List, Recycle } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/leftoverservice'
import type { Leftover } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes, since } from '../lib/format'

const error = ref('')
const items = ref<Leftover[] | null>(null)
const picked = ref(new Set<string>())
const scanning = ref(false)
const confirm = ref(false)
const busy = ref(false)
const done = ref('')

// Nothing starts ticked: matching is by name, so each folder is the user's call.
async function scan() {
  scanning.value = true
  error.value = ''
  try {
    items.value = (await List()) ?? []
    picked.value = new Set()
  } catch (e) {
    error.value = String(e)
  } finally {
    scanning.value = false
  }
}

function pick(p: string) {
  const next = new Set(picked.value)
  next.has(p) ? next.delete(p) : next.add(p)
  picked.value = next
}

const selected = computed(() => (items.value ?? []).filter((l) => picked.value.has(l.path)))
const selectedBytes = computed(() => selected.value.reduce((n, l) => n + l.size, 0))
const totalBytes = computed(() => (items.value ?? []).reduce((n, l) => n + l.size, 0))

async function recycle() {
  busy.value = true
  try {
    await Recycle([...picked.value])
    done.value = `${selected.value.length} folders, ${formatBytes(selectedBytes.value)}, moved to the Recycle Bin`
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    confirm.value = false
    await scan()
  }
}

onMounted(scan)
</script>

<template>
  <div class="flex flex-col gap-5">
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>
    <div v-if="done" class="flex items-center gap-3 border border-ok/40 bg-ok/10 px-4 py-3 font-mono text-xs text-ok"><Icon name="check" />{{ done }}</div>

    <div class="grid grid-cols-3 border border-line bg-panel font-mono">
      <div class="border-r border-line px-4 py-3"><p class="caption">leftovers found</p><p class="mt-1 text-2xl text-disk tabular-nums">{{ items ? formatBytes(totalBytes) : '--' }}</p></div>
      <div class="border-r border-line px-4 py-3"><p class="caption">selected</p><p class="mt-1 text-2xl tabular-nums">{{ formatBytes(selectedBytes) }}</p></div>
      <div class="flex items-center justify-end gap-2 px-4 py-3">
        <button class="flex h-9 items-center gap-2 border border-line px-3 font-mono text-xs transition hover:bg-hover disabled:opacity-50"
                :disabled="scanning" @click="scan"><Icon name="refresh" class="size-3.5!" :class="scanning && 'animate-spin'" />scan</button>
        <button class="h-9 bg-accent px-4 font-mono text-xs font-bold text-white transition hover:brightness-110 disabled:opacity-50"
                :disabled="!picked.size || busy" @click="confirm = true">recycle selected</button>
      </div>
    </div>

    <div class="border border-line bg-panel">
      <p v-if="!items" class="px-4 py-6 font-mono text-xs text-muted">// looking for folders no program claims…</p>
      <p v-else-if="!items.length" class="px-4 py-6 font-mono text-xs text-ok">// no leftovers found</p>
      <label v-for="l in items ?? []" :key="l.path" class="flex cursor-pointer items-center gap-4 border-b border-line px-4 py-2.5 transition last:border-0 hover:bg-hover">
        <input type="checkbox" class="size-4 accent-[var(--accent)]" :checked="picked.has(l.path)" @change="pick(l.path)" />
        <div class="min-w-0 flex-1">
          <p class="truncate font-mono text-xs select-text" :title="l.path">{{ l.path }}</p>
          <p class="font-mono text-[11px] text-muted">{{ l.root }} · {{ l.files }} files · last changed {{ since(l.modified) }}</p>
        </div>
        <span class="w-24 text-right font-mono tabular-nums text-disk">{{ formatBytes(l.size) }}</span>
      </label>
    </div>
    <p class="font-mono text-[11px] text-muted">// folders nothing on this PC points to (no installed program, process, service or startup entry, no matching name) and untouched for 90 days. matched by name, so check each one.</p>

    <ConfirmDialog :open="confirm" :title="`Move ${selected.length} folders to the Recycle Bin?`" confirm="move to recycle bin" :busy="busy"
                   @yes="recycle" @no="confirm = false">
      <ul class="mb-3 max-h-40 overflow-y-auto font-mono text-[11px]">
        <li v-for="l in selected" :key="l.path" class="truncate py-0.5">{{ l.path }}</li>
      </ul>
      They can be restored from the Recycle Bin until it is emptied.
    </ConfirmDialog>
  </div>
</template>
