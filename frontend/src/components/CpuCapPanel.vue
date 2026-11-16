<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Apply, Plans } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/cpucapservice'
import type { PowerPlan } from '../../bindings/github.com/cn8001/reforgepulse/internal/platform/models'

const plans = ref<PowerPlan[]>([])
const error = ref('')
const pMHz = ref(5500)
const eMHz = ref(0)
const pending = ref<{ p: number; e: number } | null>(null)
const busy = ref(false)
const done = ref('')

async function load() {
  try {
    plans.value = (await Plans()) ?? []
    const active = plans.value.find((p) => p.active)
    if (active?.pCoreAc) pMHz.value = active.pCoreAc
    if (active) eMHz.value = active.eCoreAc
  } catch (e) {
    error.value = String(e)
  }
}

async function apply() {
  if (!pending.value) return
  busy.value = true
  try {
    await Apply(pending.value.p, pending.value.e)
    done.value = pending.value.p || pending.value.e ? 'cap applied to every plan, logged and revertible' : 'caps removed from every plan'
    await load()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
    pending.value = null
  }
}

const show = (v: number) => (v ? `${v}` : '—')
const capped = computed(() => plans.value.some((p) => p.pCoreAc || p.eCoreAc))
onMounted(load)
</script>

<template>
  <article class="border border-line bg-panel">
    <header class="border-b border-line px-5 py-3">
      <p class="caption">tool · reversible</p>
      <h3 class="mt-1 font-semibold">CPU frequency cap</h3>
      <p class="mt-1 text-muted">Limits the top clock on every power plan, so switching plans cannot undo it. A common way to keep an unstable 13th or 14th gen Intel CPU stable while waiting for a fix. 0 means no cap.</p>
    </header>
    <div class="flex flex-wrap items-end gap-4 px-5 py-4 font-mono text-xs">
      <label class="flex flex-col gap-1"><span class="caption">p-cores mhz</span>
        <input v-model.number="pMHz" type="number" min="0" max="7000" step="100" class="h-9 w-32 border border-line bg-raised px-2 outline-none focus:border-accent" /></label>
      <label class="flex flex-col gap-1"><span class="caption">e-cores mhz</span>
        <input v-model.number="eMHz" type="number" min="0" max="7000" step="100" class="h-9 w-32 border border-line bg-raised px-2 outline-none focus:border-accent" /></label>
      <button class="h-9 bg-accent px-4 font-bold text-white transition hover:brightness-110" @click="pending = { p: pMHz, e: eMHz }">apply</button>
      <button v-if="capped" class="h-9 border border-line px-4 transition hover:bg-hover" @click="pending = { p: 0, e: 0 }">remove caps</button>
      <span v-if="done" class="text-ok">{{ done }}</span>
      <span v-if="error" class="text-bad">{{ error }}</span>
    </div>
    <table class="w-full border-t border-line font-mono text-xs">
      <tr class="text-muted"><td class="px-5 py-2">plan</td><td>p-core ac</td><td>p-core dc</td><td>e-core ac</td><td>e-core dc</td></tr>
      <tr v-for="p in plans" :key="p.guid" class="border-t border-line/60">
        <td class="px-5 py-1.5" :class="p.active && 'text-accent'">{{ p.name }}{{ p.active ? ' · active' : '' }}</td>
        <td>{{ show(p.pCoreAc) }}</td><td>{{ show(p.pCoreDc) }}</td><td>{{ show(p.eCoreAc) }}</td><td>{{ show(p.eCoreDc) }}</td>
      </tr>
    </table>

    <ConfirmDialog :open="!!pending" :title="pending?.p || pending?.e ? 'Cap CPU frequency?' : 'Remove the CPU caps?'" confirm="apply" :busy="busy"
                   @yes="apply" @no="pending = null">
      <template v-if="pending?.p || pending?.e">
        P-cores at {{ pending?.p || 'no cap' }} MHz and E-cores at {{ pending?.e || 'no cap' }} MHz, plugged in and on battery, on all {{ plans.length }} power plans.
        Peak performance drops a little. Tuning tools such as XTU or MSI Center can override this at boot.
      </template>
      <template v-else>Every plan goes back to no frequency cap.</template>
      The old values are saved; revert from the change log at any time.
    </ConfirmDialog>
  </article>
</template>
