<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import * as Leftovers from '../../bindings/github.com/cn8001/reforgepulse/internal/app/leftoverservice'
import * as OldFiles from '../../bindings/github.com/cn8001/reforgepulse/internal/app/oldfilesservice'
import type { Leftover } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes, since } from '../lib/format'

// One panel for both: folders uninstalled programs left, and old downloads.
const props = defineProps<{ kind: 'leftovers' | 'downloads' }>()
const { List, Recycle } = props.kind === 'downloads' ? OldFiles : Leftovers
const texts = props.kind === 'downloads'
  ? { found: 'in old downloads', scanning: 'Looking through Downloads…', none: 'Nothing in Downloads has sat untouched for 180 days.',
      note: 'Files and folders of 1 MB or more whose newest file is older than 180 days: usually installers run once and archives already extracted.' }
  : { found: 'in leftover folders', scanning: 'Looking for folders no program claims…', none: 'No leftovers found.',
      note: 'Folders that no installed program, process, service or startup entry points to, untouched for 90 days. They are matched by name, so check each one.' }

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
    done.value = `${selected.value.length} items, ${formatBytes(selectedBytes.value)}, moved to the Recycle Bin`
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
  <div class="flex flex-col gap-4">
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <p v-if="done" class="notice border-ok/40 bg-ok/10">{{ done }}</p>

    <div class="panel">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <div>
          <p>
            <template v-if="items"><b class="readout text-[22px]">{{ formatBytes(totalBytes) }}</b> {{ texts.found }}, <b>{{ formatBytes(selectedBytes) }}</b> selected.</template>
            <template v-else>{{ texts.scanning }}</template>
          </p>
          <p class="mt-0.5 text-xs text-muted">{{ texts.note }}</p>
        </div>
        <div class="flex gap-2">
          <button class="btn" :disabled="scanning" @click="scan"><Icon name="refresh" class="size-4!" :class="scanning && 'animate-spin'" />Look again</button>
          <button class="btn-primary" :disabled="!picked.size || busy" @click="confirm = true">Move to Recycle Bin</button>
        </div>
      </header>
      <p v-if="items && !items.length" class="px-4 py-6 text-muted">{{ texts.none }}</p>
      <label v-for="l in items ?? []" :key="l.path" class="flex cursor-pointer items-center gap-4 border-b border-line px-4 py-2.5 transition last:border-0 hover:bg-hover">
        <input type="checkbox" class="size-4 accent-[var(--accent)]" :checked="picked.has(l.path)" @change="pick(l.path)" />
        <span class="min-w-0 flex-1">
          <span class="block truncate font-mono text-xs select-text" :title="l.path">{{ l.path }}</span>
          <span class="block text-xs text-muted">In {{ l.root }}, {{ l.files.toLocaleString() }} files, last changed {{ since(l.modified) }}</span>
        </span>
        <span class="w-24 text-right tabular-nums">{{ formatBytes(l.size) }}</span>
      </label>
    </div>

    <ConfirmDialog :open="confirm" :title="`Move ${selected.length} items to the Recycle Bin?`" confirm="Move to Recycle Bin" :busy="busy"
                   @yes="recycle" @no="confirm = false">
      <ul class="mb-3 max-h-40 overflow-y-auto font-mono text-xs text-fg">
        <li v-for="l in selected" :key="l.path" class="truncate py-0.5">{{ l.path }}</li>
      </ul>
      They stay in the Recycle Bin until it is emptied, so they can be restored.
    </ConfirmDialog>
  </div>
</template>
