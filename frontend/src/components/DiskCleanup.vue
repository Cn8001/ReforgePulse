<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Analyze, Clean } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/cleanupservice'
import type { CleanupArea, CleanupResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const error = ref('')
const areas = ref<CleanupArea[] | null>(null)
const picked = ref(new Set<string>())
const analysing = ref(false)
const confirmClean = ref(false)
const cleaning = ref(false)
const result = ref<CleanupResult | null>(null)

// The Recycle Bin is the one area people may still want, so it starts unticked.
async function analyse() {
  analysing.value = true
  error.value = ''
  try {
    areas.value = (await Analyze()) ?? []
    picked.value = new Set(areas.value.filter((a) => a.bytes > 0 && a.id !== 'recycle-bin').map((a) => a.id))
  } catch (e) {
    error.value = String(e)
  } finally {
    analysing.value = false
  }
}

function pick(id: string) {
  const next = new Set(picked.value)
  next.has(id) ? next.delete(id) : next.add(id)
  picked.value = next
}

const selected = computed(() => (areas.value ?? []).filter((a) => picked.value.has(a.id)))
const selectedBytes = computed(() => selected.value.reduce((n, a) => n + a.bytes, 0))
const totalBytes = computed(() => (areas.value ?? []).reduce((n, a) => n + a.bytes, 0))

async function clean() {
  cleaning.value = true
  try {
    result.value = await Clean([...picked.value])
  } catch (e) {
    error.value = String(e)
  } finally {
    cleaning.value = false
    confirmClean.value = false
    await analyse()
  }
}

onMounted(analyse)
</script>

<template>
  <div class="flex flex-col gap-5">
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

      <div v-if="result" class="flex items-center gap-3 border border-ok/40 bg-ok/10 px-4 py-3 font-mono text-xs text-ok">
        <Icon name="check" />
        freed {{ formatBytes(result.freed) }} · {{ result.deleted }} files deleted · {{ result.skipped }} skipped (in use) · logged as {{ result.changeId }}
      </div>

      <div class="grid grid-cols-3 border border-line bg-panel font-mono">
        <div class="border-r border-line px-4 py-3"><p class="caption">reclaimable</p><p class="mt-1 text-2xl text-disk tabular-nums">{{ areas ? formatBytes(totalBytes) : '--' }}</p></div>
        <div class="border-r border-line px-4 py-3"><p class="caption">selected</p><p class="mt-1 text-2xl tabular-nums">{{ areas ? formatBytes(selectedBytes) : '--' }}</p></div>
        <div class="flex items-center justify-end gap-2 px-4 py-3">
          <button class="flex h-9 items-center gap-2 border border-line px-3 font-mono text-xs transition hover:bg-hover disabled:opacity-50"
                  :disabled="analysing" @click="analyse">
            <Icon name="refresh" class="size-3.5!" :class="analysing && 'animate-spin'" />analyse
          </button>
          <button class="h-9 bg-accent px-4 font-mono text-xs font-bold text-white transition hover:brightness-110 disabled:opacity-50"
                  :disabled="!selectedBytes || cleaning" @click="confirmClean = true">clean selected</button>
        </div>
      </div>

      <div class="border border-line bg-panel">
        <p v-if="!areas" class="px-4 py-6 font-mono text-xs text-muted">// analysing…</p>
        <label v-for="a in areas ?? []" :key="a.id"
               class="flex cursor-pointer items-center gap-4 border-b border-line px-4 py-3 transition last:border-0 hover:bg-hover">
          <input type="checkbox" class="size-4 accent-[var(--accent)]" :checked="picked.has(a.id)" :disabled="!a.bytes" @change="pick(a.id)" />
          <div class="min-w-0 flex-1">
            <p class="font-medium">{{ a.name }}</p>
            <p class="truncate text-xs text-muted">{{ a.note }}</p>
          </div>
          <span class="w-24 text-right font-mono text-xs text-muted tabular-nums">{{ a.files }} files</span>
          <span class="w-24 text-right font-mono tabular-nums" :class="a.bytes ? 'text-disk' : 'text-muted'">{{ formatBytes(a.bytes) }}</span>
        </label>
      </div>
        <ConfirmDialog :open="confirmClean" title="Delete these files?" confirm="delete" danger :busy="cleaning" @yes="clean" @no="confirmClean = false">
      <ul class="mb-3 font-mono text-xs">
        <li v-for="a in selected" :key="a.id" class="flex justify-between py-0.5"><span>{{ a.name }}</span><span class="text-disk">{{ formatBytes(a.bytes) }}</span></li>
      </ul>
      Files are deleted permanently, not moved to the Recycle Bin. Files in use are skipped. The result is written to the change log.
    </ConfirmDialog>

  </div>
</template>
