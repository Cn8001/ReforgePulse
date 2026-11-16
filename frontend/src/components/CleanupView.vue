<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import Toggle from './Toggle.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Analyze, Clean } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/cleanupservice'
import { List as ListStartup, SetEnabled } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/startupservice'
import type { CleanupArea, CleanupResult, StartupApp } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const tab = ref<'disk' | 'startup'>('disk')
const error = ref('')

// Disk: analyse, pick, confirm, clean, show what was really freed.
const areas = ref<CleanupArea[] | null>(null)
const picked = ref(new Set<string>())
const analysing = ref(false)
const confirmClean = ref(false)
const cleaning = ref(false)
const result = ref<CleanupResult | null>(null)

async function analyse() {
  analysing.value = true
  error.value = ''
  try {
    areas.value = (await Analyze()) ?? []
    picked.value = new Set(areas.value.filter((a) => a.bytes > 0).map((a) => a.id))
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

// Startup: every change is reversible, disabling asks first.
const apps = ref<StartupApp[]>([])
const busyId = ref('')
const pending = ref<StartupApp | null>(null)

async function loadStartup() {
  try {
    apps.value = (await ListStartup()) ?? []
  } catch (e) {
    error.value = String(e)
  }
}

async function setEnabled(a: StartupApp, on: boolean) {
  busyId.value = a.id
  try {
    await SetEnabled(a.id, a.name, on)
    await loadStartup()
  } catch (e) {
    error.value = String(e)
  } finally {
    busyId.value = ''
    pending.value = null
  }
}

const startupStats = computed(() => ({
  enabled: apps.value.filter((a) => a.enabled).length,
  running: apps.value.filter((a) => a.running).length,
  memory: apps.value.reduce((n, a) => n + a.memory, 0),
}))

onMounted(() => {
  analyse()
  loadStartup()
})
</script>

<template>
  <section class="flex flex-col gap-5 p-6">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="caption">reclaim disk and memory · every change asks first</p>
        <h1 class="mt-1 text-2xl font-semibold">Cleanup</h1>
      </div>
      <div class="flex border border-line font-mono text-xs">
        <button v-for="t in [['disk', 'Disk space'], ['startup', 'Startup apps']] as const" :key="t[0]"
                class="flex items-center gap-2 px-4 py-2 transition"
                :class="tab === t[0] ? 'bg-accent text-white' : 'text-muted hover:bg-hover hover:text-fg'" @click="tab = t[0]">
          <Icon :name="t[0] === 'disk' ? 'disk' : 'power'" class="size-3.5!" />{{ t[1] }}
        </button>
      </div>
    </header>

    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

    <template v-if="tab === 'disk'">
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
    </template>

    <template v-else>
      <div class="grid grid-cols-3 border border-line bg-panel font-mono">
        <div class="border-r border-line px-4 py-3"><p class="caption">start at sign-in</p><p class="mt-1 text-2xl text-accent tabular-nums">{{ startupStats.enabled }}</p></div>
        <div class="border-r border-line px-4 py-3"><p class="caption">running now</p><p class="mt-1 text-2xl tabular-nums">{{ startupStats.running }}</p></div>
        <div class="px-4 py-3"><p class="caption">their memory</p><p class="mt-1 text-2xl text-mem tabular-nums">{{ formatBytes(startupStats.memory) }}</p></div>
      </div>

      <div class="border border-line bg-panel">
        <div v-for="a in apps" :key="a.id" class="flex items-center gap-4 border-b border-line px-4 py-3 last:border-0 hover:bg-hover">
          <Toggle :on="a.enabled" :disabled="busyId === a.id" @toggle="a.enabled ? (pending = a) : setEnabled(a, true)" />
          <div class="min-w-0 flex-1">
            <p class="flex items-center gap-2 font-medium">
              <span class="truncate">{{ a.name }}</span>
              <span v-if="a.running" class="border border-ok/40 px-1.5 font-mono text-[10px] text-ok">running</span>
              <span v-if="!a.enabled" class="border border-line px-1.5 font-mono text-[10px] text-muted">disabled</span>
            </p>
            <p class="truncate font-mono text-[11px] text-muted" :title="a.command">{{ a.source }} · {{ a.command }}</p>
          </div>
          <span class="w-24 text-right font-mono tabular-nums" :class="a.memory ? 'text-mem' : 'text-muted'">{{ a.memory ? formatBytes(a.memory) : '--' }}</span>
        </div>
        <p v-if="!apps.length" class="px-4 py-6 font-mono text-xs text-muted">// no startup apps found</p>
      </div>
      <p class="font-mono text-[11px] text-muted">// disabling works like Task Manager: the app stays installed and can be turned back on here or reverted from the change log.</p>
    </template>

    <ConfirmDialog :open="confirmClean" title="Delete these files?" confirm="delete" danger :busy="cleaning" @yes="clean" @no="confirmClean = false">
      <ul class="mb-3 font-mono text-xs">
        <li v-for="a in selected" :key="a.id" class="flex justify-between py-0.5"><span>{{ a.name }}</span><span class="text-disk">{{ formatBytes(a.bytes) }}</span></li>
      </ul>
      Files are deleted permanently, not moved to the Recycle Bin. Files in use are skipped. The result is written to the change log.
    </ConfirmDialog>

    <ConfirmDialog :open="!!pending" :title="`Disable ${pending?.name} at startup?`" confirm="disable" :busy="!!busyId"
                   @yes="pending && setEnabled(pending, false)" @no="pending = null">
      It will not start at your next sign-in{{ pending?.running ? '; the copy running now keeps running' : '' }}.
      You can turn it back on here at any time.
    </ConfirmDialog>
  </section>
</template>
