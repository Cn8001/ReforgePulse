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
  <div class="flex flex-col gap-4">
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <div class="panel">
      <header class="border-b border-line px-4 py-3">
        <p>
          <b>{{ startupStats.enabled }}</b> apps start when you sign in. {{ startupStats.running }} of them are running now and use
          <b>{{ formatBytes(startupStats.memory) }}</b> of memory.
        </p>
        <p class="mt-0.5 text-xs text-muted">Turning one off works like Task Manager: the app stays installed and can be turned back on here or from the change log.</p>
      </header>
      <div v-for="a in apps" :key="a.id" class="flex items-center gap-4 border-b border-line px-4 py-3 last:border-0 hover:bg-hover">
        <Toggle :on="a.enabled" :label="`Start ${a.name} at sign-in`" :disabled="busyId === a.id" @toggle="a.enabled ? (pending = a) : setEnabled(a, true)" />
        <div class="min-w-0 flex-1">
          <p class="flex items-center gap-2">
            <span class="truncate font-medium" :class="!a.enabled && 'text-muted'">{{ a.name }}</span>
            <span v-if="a.running" class="chip border-line text-muted">Running</span>
          </p>
          <p class="truncate text-xs text-muted" :title="a.command">{{ a.source }}: <span class="font-mono">{{ a.command }}</span></p>
        </div>
        <span class="w-24 text-right tabular-nums" :class="!a.memory && 'text-muted'">{{ a.memory ? formatBytes(a.memory) : '–' }}</span>
      </div>
      <p v-if="!apps.length" class="px-4 py-6 text-muted">No apps start with Windows.</p>
    </div>

    <ConfirmDialog :open="!!pending" :title="`Stop ${pending?.name} from starting with Windows?`" confirm="Turn off" :busy="!!busyId"
                   @yes="pending && setEnabled(pending, false)" @no="pending = null">
      It will not start at your next sign-in{{ pending?.running ? '. The copy running now keeps running' : '' }}. You can turn it back on here at any time.
    </ConfirmDialog>
  </div>
</template>
