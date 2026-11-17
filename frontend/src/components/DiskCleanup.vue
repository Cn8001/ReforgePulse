<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import TaskProgress from './TaskProgress.vue'
import { cancelled } from '../lib/tasks'
import { Analyze, Clean } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/cleanupservice'
import type { CleanupArea, CleanupResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const error = ref('')
const areas = ref<CleanupArea[] | null>(null)
const picked = ref(new Set<string>())
const analysing = ref(false)
const confirmClean = ref(false)
const cleaning = ref(false)
const stopped = ref(false)
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

// The question closes as soon as it is answered; the work shows below with
// its own bar and Cancel, and the window stays usable.
async function clean() {
  confirmClean.value = false
  cleaning.value = true
  stopped.value = false
  try {
    result.value = await Clean([...picked.value])
  } catch (e) {
    if (cancelled(e)) stopped.value = true
    else error.value = String(e)
  } finally {
    cleaning.value = false
    await analyse()
  }
}

onMounted(analyse)
</script>

<template>
  <div class="flex flex-col gap-4">
    <TaskProgress v-if="cleaning" title="Cleaning up" />
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <p v-if="stopped" class="notice border-line bg-raised">Stopped. What was already deleted stays deleted; the change log says how far it got.</p>
    <p v-if="result" class="notice border-ok/40 bg-ok/10">
      Freed <b>{{ formatBytes(result.freed) }}</b> by deleting {{ result.deleted }} files.
      <template v-if="result.skipped">{{ result.skipped }} files were in use and left alone.</template>
      Recorded in the change log as <span class="font-mono text-xs">{{ result.changeId }}</span>.
    </p>

    <div class="panel">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <p>
          <template v-if="areas"><b class="readout text-[22px]">{{ formatBytes(totalBytes) }}</b> can be freed, <b>{{ formatBytes(selectedBytes) }}</b> selected.</template>
          <template v-else>Measuring what can be freed…</template>
        </p>
        <div class="flex gap-2">
          <button class="btn" :disabled="analysing" @click="analyse">
            <Icon name="refresh" class="size-4!" :class="analysing && 'animate-spin'" />Measure again
          </button>
          <button class="btn-primary" :disabled="!selectedBytes || cleaning" @click="confirmClean = true">Clean selected</button>
        </div>
      </header>
      <label v-for="a in areas ?? []" :key="a.id" class="flex cursor-pointer items-center gap-4 border-b border-line px-4 py-3 transition last:border-0 hover:bg-hover">
        <input type="checkbox" class="size-4 accent-[var(--accent)]" :checked="picked.has(a.id)" :disabled="!a.bytes" @change="pick(a.id)" />
        <span class="min-w-0 flex-1">
          <span class="block font-medium">{{ a.name }}</span>
          <span class="block truncate text-xs text-muted">{{ a.note }}</span>
        </span>
        <span class="w-24 text-right text-xs text-muted tabular-nums">{{ a.files.toLocaleString() }} files</span>
        <span class="w-24 text-right tabular-nums" :class="!a.bytes && 'text-muted'">{{ formatBytes(a.bytes) }}</span>
      </label>
    </div>

    <ConfirmDialog :open="confirmClean" title="Delete these files?" confirm="Delete files" danger
                   warning="Deleted files cannot be brought back. Are you sure?" @yes="clean" @no="confirmClean = false">
      <ul class="mb-3 text-[14px] text-fg">
        <li v-for="a in selected" :key="a.id" class="flex justify-between py-0.5"><span>{{ a.name }}</span><span class="tabular-nums">{{ formatBytes(a.bytes) }}</span></li>
      </ul>
      They are not moved to the Recycle Bin. Files in use are skipped, and the result is recorded in the change log.
    </ConfirmDialog>
  </div>
</template>
