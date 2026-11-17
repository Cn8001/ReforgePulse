<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Days, SetDays } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/retentionservice'

// How long history and reports are kept. Choosing a limit deletes what is
// older right away and again at every start, so it asks first.
const days = ref(0)
const custom = ref(60)
const pending = ref<number | null>(null)
const note = ref('')
const error = ref('')
onMounted(async () => {
  days.value = await Days().catch(() => 0)
  if (days.value && days.value !== 10 && days.value !== 30) custom.value = days.value
})

const choices = [
  { label: 'Never', value: 0 },
  { label: '10 days', value: 10 },
  { label: '30 days', value: 30 },
] as const
const isCustom = () => days.value !== 0 && days.value !== 10 && days.value !== 30

function choose(value: number) {
  if (value === days.value) return
  if (value === 0) save(0)
  else pending.value = value
}

async function save(value: number) {
  pending.value = null
  error.value = ''
  try {
    const n = await SetDays(value)
    days.value = value
    note.value = value ? (n ? `Deleted ${n} older ${n === 1 ? 'entry' : 'entries'}.` : 'Nothing was older than that.') : ''
  } catch (e) {
    error.value = String(e)
  }
}
</script>

<template>
  <div class="flex flex-col gap-3 border-b border-line px-5 py-4">
    <div>
      <h3 class="font-medium">Delete old history and reports</h3>
      <p class="text-[14px] text-muted">
        Removes change log entries, saved scans, recordings and dump analyses older than this, now and at every start.
        A deleted change can no longer be reverted from here.
      </p>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex rounded border border-line p-0.5" role="radiogroup" aria-label="Keep history for">
        <button v-for="c in choices" :key="c.value" role="radio" :aria-checked="days === c.value"
                class="rounded-[3px] px-3 py-1 text-[14px] transition"
                :class="days === c.value ? 'bg-accent text-white' : 'text-muted hover:text-fg'" @click="choose(c.value)">{{ c.label }}</button>
        <button role="radio" :aria-checked="isCustom()" class="rounded-[3px] px-3 py-1 text-[14px] transition"
                :class="isCustom() ? 'bg-accent text-white' : 'text-muted hover:text-fg'" @click="choose(custom)">Custom</button>
      </div>
      <label class="flex items-center gap-2 text-muted">
        <input v-model.number="custom" type="number" min="1" max="3650" class="field w-20" aria-label="Custom number of days" />
        days
      </label>
      <button v-if="custom !== days" class="btn" :disabled="!(custom >= 1 && custom <= 3650)" @click="choose(custom)">Use {{ custom }} days</button>
    </div>
    <p v-if="note" class="text-xs text-muted">{{ note }}</p>
    <p v-if="error" class="text-xs text-bad">{{ error }}</p>
  </div>

  <ConfirmDialog :open="pending !== null" :title="`Delete history and reports older than ${pending} days?`" confirm="Delete" danger
                 warning="What is older is deleted now and at every start from here on. This cannot be undone. Are you sure?"
                 @yes="save(pending!)" @no="pending = null">
    Change log entries, saved scans, recordings and dump analyses go. The PC itself is not changed, but a deleted change can no
    longer be reverted from here.
  </ConfirmDialog>
</template>
