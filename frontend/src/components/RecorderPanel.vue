<script setup lang="ts">
import { ref } from 'vue'
import RecordingView from './RecordingView.vue'
import { Record } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/recorderservice'
import type { Recording } from '../../bindings/github.com/cn8001/reforgepulse/internal/recorder/models'

const minutes = ref(10)
const rec = ref<Recording | null>(null)
const running = ref(false)
const error = ref('')

async function start() {
  running.value = true
  rec.value = null
  error.value = ''
  try {
    rec.value = await Record(minutes.value)
  } catch (e) {
    error.value = String(e)
  } finally {
    running.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel flex flex-wrap items-center gap-3 px-4 py-3">
      <p class="min-w-64 flex-1 text-muted">Record processor, graphics and memory load and every process's memory while you play or work, then see what happened.</p>
      <label class="flex items-center gap-2 text-muted">Minutes <input v-model.number="minutes" type="number" min="1" max="1440" class="field w-20" /></label>
      <button class="btn-primary" :disabled="running" @click="start">{{ running ? 'Recording…' : 'Start recording' }}</button>
    </div>
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <p v-if="running" class="text-xs text-muted">Stop early from the status bar; what was recorded so far is kept.</p>

    <p v-if="rec?.id" class="text-xs text-muted">Saved under Reports, with your earlier recordings.</p>
    <RecordingView v-if="rec" :rec="rec" />
  </div>
</template>
