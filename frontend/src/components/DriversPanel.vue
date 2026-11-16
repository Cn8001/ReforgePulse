<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Browser } from '@wailsio/runtime'
import Icon from './Icon.vue'
import { Offers, Open, Problems } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/driverservice'
import type { DriverUpdate, ProblemDevice } from '../../bindings/github.com/cn8001/reforgepulse/internal/platform/models'
import { facts, loadFacts } from '../lib/hardware'

const problems = ref<ProblemDevice[] | null>(null)
const offers = ref<DriverUpdate[] | null>(null)
const searching = ref(false)
const error = ref('')

const meaning: Record<number, string> = {
  1: 'not set up correctly', 3: 'driver may be corrupt', 10: 'cannot start', 18: 'reinstall its driver', 22: 'disabled',
  24: 'not present or not working', 28: 'no driver installed', 31: 'not working properly', 37: 'driver failed to load',
  39: 'driver missing or corrupt', 43: 'stopped after reporting a problem', 48: 'driver blocked as incompatible', 52: 'driver not signed',
}

onMounted(async () => {
  loadFacts()
  try {
    problems.value = (await Problems()) ?? []
  } catch (e) {
    error.value = String(e)
  }
})

async function search() {
  searching.value = true
  try {
    offers.value = (await Offers()) ?? []
  } catch (e) {
    error.value = String(e)
  } finally {
    searching.value = false
  }
}

// The makers' own driver pages, picked from what is actually in this PC.
const makers = computed(() => {
  const f = facts.value
  const out: { name: string; why: string; url: string }[] = []
  const gpus = (f?.GPUs ?? []).map((g) => g.Name).join(' ')
  const cpu = f?.CPU.Name ?? ''
  const board = f?.Board ?? ''
  if (/nvidia/i.test(gpus)) out.push({ name: 'NVIDIA', why: 'graphics driver', url: 'https://www.nvidia.com/en-us/drivers/' })
  if (/amd|radeon/i.test(gpus) || /amd/i.test(cpu)) out.push({ name: 'AMD', why: 'graphics and chipset', url: 'https://www.amd.com/en/support/download/drivers.html' })
  if (/intel/i.test(gpus + cpu)) out.push({ name: 'Intel', why: 'graphics, Wi-Fi, chipset', url: 'https://www.intel.com/content/www/us/en/support/detect.html' })
  const boards: [RegExp, string, string][] = [
    [/micro-star|msi/i, 'MSI', 'https://www.msi.com/support/download'],
    [/asus/i, 'ASUS', 'https://www.asus.com/support/download-center/'],
    [/gigabyte/i, 'Gigabyte', 'https://www.gigabyte.com/Support'],
    [/asrock/i, 'ASRock', 'https://www.asrock.com/support/index.asp'],
  ]
  for (const [re, name, url] of boards) if (re.test(board)) out.push({ name, why: `${board.replace(/\s*\(.*\)$/, '')}: BIOS, chipset, LAN, audio`, url })
  return out
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">{{ error }}</p>

    <article class="border border-line bg-panel">
      <header class="flex items-center justify-between border-b border-line px-4 py-3">
        <p class="caption">devices with problems</p>
        <button class="font-mono text-[11px] text-accent hover:underline" @click="Open('device-manager')">open Device Manager</button>
      </header>
      <p v-if="!problems" class="px-4 py-4 font-mono text-xs text-muted">// reading devices…</p>
      <p v-else-if="!problems.length" class="px-4 py-4 font-mono text-xs text-ok">// every device has a working driver</p>
      <div v-for="d in problems ?? []" :key="d.deviceId" class="border-b border-line px-4 py-2.5 last:border-0">
        <p class="text-sm">{{ d.name || 'Unknown device' }} <span class="font-mono text-[11px]" :class="d.errorCode === 28 ? 'text-bad' : 'text-warn'">· {{ meaning[d.errorCode] ?? 'problem' }} (code {{ d.errorCode }})</span></p>
        <p class="truncate font-mono text-[11px] text-muted select-text" :title="d.hardwareId">{{ d.class || 'no class' }} · {{ d.hardwareId }}</p>
      </div>
    </article>

    <article class="border border-line bg-panel">
      <header class="flex items-center justify-between border-b border-line px-4 py-3">
        <p class="caption">windows update · drivers offered for this pc</p>
        <div class="flex items-center gap-3">
          <button class="h-8 border border-line px-3 font-mono text-[11px] transition hover:bg-hover disabled:opacity-50" :disabled="searching" @click="search">
            <Icon name="refresh" class="inline size-3!" :class="searching && 'animate-spin'" /> {{ searching ? 'asking…' : offers ? 'ask again' : 'ask windows update' }}</button>
          <button class="h-8 bg-accent px-3 font-mono text-[11px] font-bold text-white transition hover:brightness-110" @click="Open('windows-update')">install in Windows Update</button>
        </div>
      </header>
      <p v-if="!offers" class="px-4 py-4 font-mono text-xs text-muted">// Windows Update is Microsoft's own signed driver catalog. Asking takes up to a minute.</p>
      <p v-else-if="!offers.length" class="px-4 py-4 font-mono text-xs text-ok">// Windows Update offers no drivers this PC is missing</p>
      <table v-else class="w-full font-mono text-xs">
        <tr class="text-muted"><td class="px-4 py-2">driver</td><td>class</td><td>date</td><td class="pr-4 text-right">size</td></tr>
        <tr v-for="u in offers" :key="u.id" class="border-t border-line/60">
          <td class="px-4 py-1.5">{{ u.title }}</td><td class="text-muted">{{ u.class }}</td><td class="text-muted">{{ u.date }}</td>
          <td class="pr-4 text-right">{{ u.sizeMb }} MB</td>
        </tr>
      </table>
      <p v-if="offers?.length" class="border-t border-line px-4 py-2 font-mono text-[11px] text-muted">// these are under Settings › Windows Update › Advanced options › Optional updates; the button opens that page.</p>
    </article>

    <article class="border border-line bg-panel">
      <header class="border-b border-line px-4 py-3"><p class="caption">makers' official driver pages · newest drivers usually land here first</p></header>
      <div v-for="m in makers" :key="m.name" class="flex items-center gap-4 border-b border-line px-4 py-2.5 last:border-0">
        <div class="min-w-0 flex-1"><p class="font-medium">{{ m.name }}</p><p class="truncate font-mono text-[11px] text-muted">{{ m.why }}</p></div>
        <button class="h-8 border border-line px-3 font-mono text-[11px] transition hover:border-accent hover:text-accent" @click="Browser.OpenURL(m.url)">open site</button>
      </div>
      <p v-if="!makers.length" class="px-4 py-4 font-mono text-xs text-muted">// reading hardware…</p>
    </article>
  </div>
</template>
