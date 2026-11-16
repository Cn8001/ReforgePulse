<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { List, SetEnabled } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/featureservice'
import type { WindowsFeature } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

const error = ref('')
const feats = ref<WindowsFeature[] | null>(null)
const pending = ref<WindowsFeature | null>(null)
const busy = ref(false)
const restart = ref(false)
const showAll = ref(false)

async function load() {
  try {
    feats.value = (await List()) ?? []
  } catch (e) {
    error.value = String(e)
  }
}

async function turnOff() {
  if (!pending.value) return
  busy.value = true
  try {
    if (await SetEnabled(pending.value.name, false)) restart.value = true
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    pending.value = null
  }
}

const rows = computed(() => (feats.value ?? []).filter((f) => showAll.value || f.suggest))
onMounted(load)
</script>

<template>
  <div class="flex flex-col gap-5">
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>
    <p v-if="restart" class="border border-warn/40 bg-warn/10 px-4 py-3 font-mono text-xs text-warn">restart Windows to finish turning features off</p>

    <div class="flex items-center justify-between border border-line bg-panel px-4 py-3 font-mono text-xs">
      <span class="text-muted">{{ feats ? `${feats.length} features on · ${feats.filter((f) => f.suggest).length} worth turning off` : 'reading features (takes a few seconds)…' }}</span>
      <button class="h-8 border px-3 transition" :class="showAll ? 'border-accent bg-accent/15 text-accent' : 'border-line text-muted hover:text-fg'"
              @click="showAll = !showAll">show all</button>
    </div>

    <div class="border border-line bg-panel">
      <p v-if="feats && !rows.length" class="px-4 py-6 font-mono text-xs text-ok">// nothing worth turning off</p>
      <div v-for="f in rows" :key="f.name" class="flex items-center gap-4 border-b border-line px-4 py-3 last:border-0 hover:bg-hover">
        <div class="min-w-0 flex-1">
          <p class="flex items-center gap-2 font-medium">
            {{ f.title }}
            <span v-if="f.suggest" class="border border-warn/50 px-1.5 font-mono text-[10px] text-warn">suggested off</span>
            <span v-if="f.protected" class="border border-ok/50 px-1.5 font-mono text-[10px] text-ok">needed</span>
          </p>
          <p class="truncate font-mono text-[11px] text-muted">{{ f.name }}{{ f.note ? ' · ' + f.note : '' }}</p>
        </div>
        <button class="h-7 border border-line px-2.5 font-mono text-[11px] transition hover:border-warn hover:text-warn disabled:opacity-40"
                :disabled="f.protected" @click="pending = f">turn off</button>
      </div>
    </div>

    <ConfirmDialog :open="!!pending" :title="`Turn off ${pending?.title}?`" confirm="turn off" :busy="busy" @yes="turnOff" @no="pending = null">
      {{ pending?.note }} Windows may need a restart. It can be turned back on from the change log.
    </ConfirmDialog>
  </div>
</template>
