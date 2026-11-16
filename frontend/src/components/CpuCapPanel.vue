<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Apply, Plans } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/cpucapservice'
import type { PowerPlan } from '../../bindings/github.com/cn8001/reforgepulse/internal/platform/models'

const plans = ref<PowerPlan[]>([])
const error = ref('')
// Starts empty: the cap is the user's number, 0 meaning none.
const pMHz = ref(0)
const eMHz = ref(0)
const pending = ref<{ p: number; e: number } | null>(null)
const busy = ref(false)
const done = ref('')

async function load() {
  try {
    plans.value = (await Plans()) ?? []
  } catch (e) {
    error.value = String(e)
  }
}

async function apply() {
  if (!pending.value) return
  busy.value = true
  try {
    await Apply(pending.value.p, pending.value.e)
    done.value = pending.value.p || pending.value.e ? 'Applied to every power plan.' : 'Caps removed from every power plan.'
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    pending.value = null
  }
}

const show = (v: number) => (v ? `${v} MHz` : 'None')
const capped = computed(() => plans.value.some((p) => p.pCoreAc || p.eCoreAc))
onMounted(load)
</script>

<template>
  <article class="panel">
    <header class="border-b border-line px-5 py-4">
      <h2 class="text-[14px] font-semibold">CPU frequency cap</h2>
      <p class="mt-1 max-w-3xl text-muted">
        Limits the highest clock on every power plan, so switching plans cannot undo it. Use it to test whether lower clocks stop the crashes,
        whatever the cause turns out to be. Enter your own limit in MHz; 0 means no cap.
      </p>
    </header>
    <div class="flex flex-wrap items-end gap-4 px-5 py-4">
      <label class="flex flex-col gap-1 text-xs text-muted">Performance cores (MHz)
        <input v-model.number="pMHz" type="number" min="0" max="7000" step="100" class="field w-36" /></label>
      <label class="flex flex-col gap-1 text-xs text-muted">Efficiency cores (MHz)
        <input v-model.number="eMHz" type="number" min="0" max="7000" step="100" class="field w-36" /></label>
      <button class="btn-primary" @click="pending = { p: pMHz, e: eMHz }">Apply</button>
      <button v-if="capped" class="btn" @click="pending = { p: 0, e: 0 }">Remove caps</button>
      <span v-if="done" class="text-ok">{{ done }}</span>
      <span v-if="error" class="text-bad">{{ error }}</span>
    </div>
    <table class="w-full border-t border-line text-[13px] tabular-nums">
      <tr class="text-muted"><th class="px-5 py-2 text-left font-medium">Power plan</th><th class="text-left font-medium">P-cores, plugged in</th>
        <th class="text-left font-medium">P-cores, battery</th><th class="text-left font-medium">E-cores, plugged in</th><th class="text-left font-medium">E-cores, battery</th></tr>
      <tr v-for="p in plans" :key="p.guid" class="border-t border-line/60">
        <td class="px-5 py-1.5">{{ p.name }}<span v-if="p.active" class="chip ml-2 border-accent text-accent">Active</span></td>
        <td>{{ show(p.pCoreAc) }}</td><td>{{ show(p.pCoreDc) }}</td><td>{{ show(p.eCoreAc) }}</td><td>{{ show(p.eCoreDc) }}</td>
      </tr>
    </table>

    <ConfirmDialog :open="!!pending" :title="pending?.p || pending?.e ? 'Cap the CPU frequency?' : 'Remove the CPU caps?'" confirm="Apply" :busy="busy"
                   @yes="apply" @no="pending = null">
      <template v-if="pending?.p || pending?.e">
        Performance cores at {{ pending?.p ? `${pending.p} MHz` : 'no cap' }} and efficiency cores at {{ pending?.e ? `${pending.e} MHz` : 'no cap' }},
        plugged in and on battery, on all {{ plans.length }} power plans. Peak performance drops a little, and tuning tools such as XTU or MSI Center can override it at boot.
      </template>
      <template v-else>Every power plan goes back to no frequency cap.</template>
      The old values are kept, so you can revert from the change log.
    </ConfirmDialog>
  </article>
</template>
