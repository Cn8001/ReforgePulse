<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import Icon from './Icon.vue'
import { List, SetEnabled } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/featureservice'
import type { WindowsFeature } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

const error = ref('')
const feats = ref<WindowsFeature[] | null>(null)
const pending = ref<WindowsFeature | null>(null)
// Features being switched now. DISM can take minutes, so the dialog closes
// at once and the row and status bar show the work instead.
const working = ref(new Set<string>())
const showAll = ref(false)

async function load() {
  try {
    feats.value = (await List()) ?? []
  } catch (e) {
    error.value = String(e)
  }
}

async function turnOff() {
  const f = pending.value
  if (!f) return
  pending.value = null
  working.value = new Set(working.value).add(f.name)
  try {
    await SetEnabled(f.name, false)
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    const next = new Set(working.value)
    next.delete(f.name)
    working.value = next
  }
}

const rows = computed(() => (feats.value ?? []).filter((f) => showAll.value || f.suggest || f.pending))
// Windows' own word on what waits for a restart, so the notice goes once
// the restart has happened, not when the window forgets.
const waiting = computed(() => (feats.value ?? []).filter((f) => f.pending))
const dismissed = ref(false)
onMounted(load)
</script>

<template>
  <div class="flex flex-col gap-4">
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <div v-if="waiting.length && !dismissed" class="notice flex items-start gap-3 border-warn/40 bg-warn/10" role="status">
      <p class="flex-1">Restart Windows to finish switching {{ waiting.map((f) => f.title).join(', ') }}.</p>
      <button class="rounded p-1 text-muted hover:bg-hover hover:text-fg" aria-label="Hide this notice" @click="dismissed = true">
        <Icon name="close" class="size-3.5!" />
      </button>
    </div>

    <div class="panel">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <p v-if="feats"><b>{{ feats.filter((f) => f.suggest).length }}</b> of {{ feats.length }} Windows features that are on are rarely needed or a known risk.</p>
        <p v-else>Reading Windows features, which takes a few seconds…</p>
        <button class="btn" :class="showAll && 'border-accent text-accent'" :aria-pressed="showAll" @click="showAll = !showAll">Show all</button>
      </header>
      <p v-if="feats && !rows.length" class="px-4 py-6 text-muted">Nothing here is worth turning off.</p>
      <div v-for="f in rows" :key="f.name" class="flex items-center gap-4 border-b border-line px-4 py-3 last:border-0 hover:bg-hover">
        <div class="min-w-0 flex-1">
          <p class="flex items-center gap-2">
            <span class="font-medium">{{ f.title }}</span>
            <span v-if="f.suggest" class="chip border-warn/50 text-warn">Can go</span>
            <span v-if="f.protected" class="chip border-line text-muted">Keep</span>
          </p>
          <p class="truncate text-xs text-muted">{{ f.note || 'No description.' }} <span class="font-mono">{{ f.name }}</span></p>
        </div>
        <span v-if="f.pending" class="shrink-0 text-xs text-muted">{{ f.pending === 'off' ? 'Off' : 'On' }} after a restart</span>
        <button v-else class="btn h-7" :disabled="f.protected || working.has(f.name)" @click="pending = f">
          {{ working.has(f.name) ? 'Turning off…' : 'Turn off' }}
        </button>
      </div>
    </div>

    <ConfirmDialog :open="!!pending" :title="`Turn off ${pending?.title}?`" confirm="Turn off" @yes="turnOff" @no="pending = null">
      {{ pending?.note }} Windows may need a restart. It can be turned back on from the change log.
    </ConfirmDialog>
  </div>
</template>
