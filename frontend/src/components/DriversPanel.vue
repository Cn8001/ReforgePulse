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
  1: 'Not set up correctly', 3: 'Driver may be corrupt', 10: 'Cannot start', 18: 'Needs its driver reinstalled', 22: 'Turned off',
  24: 'Not present or not working', 28: 'No driver installed', 31: 'Not working properly', 37: 'Driver failed to load',
  39: 'Driver missing or corrupt', 43: 'Stopped after reporting a problem', 48: 'Driver blocked as incompatible', 52: 'Driver not signed',
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
  if (/nvidia/i.test(gpus)) out.push({ name: 'NVIDIA', why: 'Graphics driver', url: 'https://www.nvidia.com/en-us/drivers/' })
  if (/amd|radeon/i.test(gpus) || /amd/i.test(cpu)) out.push({ name: 'AMD', why: 'Graphics and chipset drivers', url: 'https://www.amd.com/en/support/download/drivers.html' })
  if (/intel/i.test(gpus + cpu)) out.push({ name: 'Intel', why: 'Graphics, Wi-Fi and chipset drivers', url: 'https://www.intel.com/content/www/us/en/support/detect.html' })
  const boards: [RegExp, string, string][] = [
    [/micro-star|msi/i, 'MSI', 'https://www.msi.com/support/download'],
    [/asus/i, 'ASUS', 'https://www.asus.com/support/download-center/'],
    [/gigabyte/i, 'Gigabyte', 'https://www.gigabyte.com/Support'],
    [/asrock/i, 'ASRock', 'https://www.asrock.com/support/index.asp'],
  ]
  for (const [re, name, url] of boards) if (re.test(board)) out.push({ name, why: `BIOS, chipset, network and audio for the ${board.replace(/\s*\(.*\)$/, '')}`, url })
  return out
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <p class="text-muted">ReforgePulse does not install drivers. It shows what is missing and opens Windows Update or the maker's site, who know their drivers best.</p>

    <article class="panel">
      <header class="flex items-center justify-between border-b border-line px-4 py-3">
        <h2 class="text-[14px] font-semibold">Devices with problems</h2>
        <button class="btn h-7" @click="Open('device-manager')">Open Device Manager</button>
      </header>
      <p v-if="!problems" class="px-4 py-4 text-muted">Reading devices…</p>
      <p v-else-if="!problems.length" class="px-4 py-4 text-muted">Every device has a working driver.</p>
      <div v-for="d in problems ?? []" :key="d.deviceId" class="border-b border-line px-4 py-2.5 last:border-0">
        <p>{{ d.name || 'Unknown device' }} <span class="text-[14px]" :class="d.errorCode === 28 ? 'text-bad' : 'text-warn'">{{ meaning[d.errorCode] ?? 'Has a problem' }} (code {{ d.errorCode }})</span></p>
        <p class="truncate text-xs text-muted" :title="d.hardwareId">{{ d.class || 'No device class' }}, <span class="font-mono select-text">{{ d.hardwareId }}</span></p>
      </div>
    </article>

    <article class="panel">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <div>
          <h2 class="text-[14px] font-semibold">Drivers Windows Update offers</h2>
          <p class="text-xs text-muted">Microsoft's own signed catalog. Asking takes up to a minute.</p>
        </div>
        <div class="flex gap-2">
          <button class="btn" :disabled="searching" @click="search"><Icon name="refresh" class="size-4!" :class="searching && 'animate-spin'" />{{ searching ? 'Asking…' : offers ? 'Ask again' : 'Ask Windows Update' }}</button>
          <button class="btn-primary" @click="Open('windows-update')">Open Windows Update</button>
        </div>
      </header>
      <p v-if="offers && !offers.length" class="px-4 py-4 text-muted">Windows Update has no drivers this PC is missing.</p>
      <table v-else-if="offers" class="w-full text-[14px]">
        <tr class="text-muted"><th class="px-4 py-2 text-left font-medium">Driver</th><th class="text-left font-medium">Type</th>
          <th class="text-left font-medium">Date</th><th class="pr-4 text-right font-medium">Size</th></tr>
        <tr v-for="u in offers" :key="u.id" class="border-t border-line/60">
          <td class="px-4 py-1.5">{{ u.title }}</td><td class="text-muted">{{ u.class }}</td><td class="text-muted tabular-nums">{{ u.date }}</td>
          <td class="pr-4 text-right tabular-nums">{{ u.sizeMb }} MB</td>
        </tr>
      </table>
      <p v-if="offers?.length" class="border-t border-line px-4 py-2 text-xs text-muted">Install them under Settings, Windows Update, Advanced options, Optional updates. The button opens that page.</p>
    </article>

    <article class="panel">
      <header class="border-b border-line px-4 py-3">
        <h2 class="text-[14px] font-semibold">Makers' driver pages</h2>
        <p class="text-xs text-muted">New graphics and chipset drivers usually appear here before Windows Update has them.</p>
      </header>
      <div v-for="m in makers" :key="m.name" class="flex items-center gap-4 border-b border-line px-4 py-2.5 last:border-0">
        <div class="min-w-0 flex-1"><p class="font-medium">{{ m.name }}</p><p class="truncate text-xs text-muted">{{ m.why }}</p></div>
        <button class="btn h-7" @click="Browser.OpenURL(m.url)">Open site</button>
      </div>
      <p v-if="!makers.length" class="px-4 py-4 text-muted">Reading hardware…</p>
    </article>
  </div>
</template>
