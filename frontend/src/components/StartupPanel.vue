<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Toggle from './Toggle.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { List as ListStartup, SetEnabled } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/startupservice'
import type { StartupApp } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

const error = ref('')
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

onMounted(loadStartup)
</script>

<template>
  <div class="flex flex-col gap-5">
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

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
        <ConfirmDialog :open="!!pending" :title="`Disable ${pending?.name} at startup?`" confirm="disable" :busy="!!busyId"
                   @yes="pending && setEnabled(pending, false)" @no="pending = null">
      It will not start at your next sign-in{{ pending?.running ? '; the copy running now keeps running' : '' }}.
      You can turn it back on here at any time.
    </ConfirmDialog>
  </div>
</template>
