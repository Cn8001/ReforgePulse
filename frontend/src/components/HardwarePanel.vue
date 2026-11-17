<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { Cores, Loads, TestCores } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/hardwareservice'
import { facts, loadFacts } from '../lib/hardware'
import type { CoreInfo, CoreTest } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import type { CoreLoad } from '../../bindings/github.com/cn8001/reforgepulse/internal/platform/models'
import { formatBytes } from '../lib/format'

const cores = ref<CoreInfo[]>([])
const loads = ref<Record<number, CoreLoad>>({})
const error = ref('')

// The live grid samples every second while the window is visible.
let timer: ReturnType<typeof setInterval> | undefined
async function sample() {
  if (document.hidden) return
  try {
    const ls = (await Loads()) ?? []
    loads.value = Object.fromEntries(ls.map((l) => [l.index, l]))
  } catch {} // a missed sample leaves the last values
}

onMounted(async () => {
  sample()
  timer = setInterval(sample, 1000)
  try {
    ;[cores.value] = await Promise.all([Cores().then((c) => c ?? []), loadFacts()])
  } catch (e) {
    error.value = String(e)
  }
})
onBeforeUnmount(() => clearInterval(timer))

const pCount = computed(() => new Set(cores.value.filter((c) => c.kind === 'P').map((c) => c.core)).size)
const eCount = computed(() => new Set(cores.value.filter((c) => c.kind === 'E').map((c) => c.core)).size)
const peak = computed(() => Math.max(0, ...Object.values(loads.value).map((l) => l.mhz)))

// Core test: pick processors, confirm, run, read the result.
const picked = ref(new Set<number>())
const seconds = ref(30)
const confirm = ref(false)
const testing = ref(false)
const result = ref<CoreTest | null>(null)

function pick(i: number) {
  const next = new Set(picked.value)
  next.has(i) ? next.delete(i) : next.add(i)
  picked.value = next
}
function pickWhere(f: (c: CoreInfo) => boolean) {
  picked.value = new Set(cores.value.filter(f).map((c) => c.index))
}

async function test() {
  confirm.value = false
  testing.value = true
  result.value = null
  try {
    result.value = await TestCores([...picked.value].sort((a, b) => a - b), seconds.value)
  } catch (e) {
    error.value = String(e)
  } finally {
    testing.value = false
  }
}

const loadTone = (l?: CoreLoad) => (!l ? 'var(--hover)' : l.load > 85 ? 'var(--bad)' : l.load > 50 ? 'var(--warn)' : 'var(--accent)')
</script>

<template>
  <div class="flex flex-col gap-4">
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>

    <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <article class="panel p-4">
        <h2 class="label">Processor</h2>
        <p class="mt-1 font-semibold">{{ facts?.CPU.Name ?? 'Reading…' }}</p>
        <dl class="mt-3 grid grid-cols-[7rem_minmax(0,1fr)] gap-y-1.5 text-[14px] tabular-nums">
          <dt class="text-muted">Cores</dt><dd>{{ facts?.CPU.Cores ?? '–' }}<span v-if="pCount" class="text-muted">, {{ pCount }} performance and {{ eCount }} efficiency</span></dd>
          <dt class="text-muted">Threads</dt><dd>{{ facts?.CPU.Threads ?? '–' }}</dd>
          <dt class="text-muted">Base clock</dt><dd>{{ facts ? `${facts.CPU.MaxMHz} MHz` : '–' }}</dd>
          <dt class="text-muted">Fastest core now</dt><dd>{{ peak ? `${Math.round(peak)} MHz` : '–' }}</dd>
          <dt class="text-muted">Cache</dt><dd>{{ facts ? `${facts.CPU.L2KB / 1024} MB L2, ${facts.CPU.L3KB / 1024} MB L3` : '–' }}</dd>
        </dl>
      </article>
      <article class="panel p-4">
        <h2 class="label">Motherboard</h2>
        <p class="mt-1 font-semibold">{{ facts?.Board ?? 'Reading…' }}</p>
        <dl class="mt-3 grid grid-cols-[7rem_minmax(0,1fr)] gap-y-1.5 text-[14px]">
          <dt class="text-muted">BIOS</dt><dd>{{ facts ? `${facts.BIOS}, ${facts.BIOSDate}` : '–' }}</dd>
          <dt class="text-muted">Hypervisor</dt><dd>{{ facts ? (facts.Hypervisor ? 'On (Hyper-V, WSL 2 or VBS)' : 'Off') : '–' }}</dd>
        </dl>
        <h2 class="label mt-4">Graphics</h2>
        <p v-for="g in facts?.GPUs ?? []" :key="g.Name" class="mt-1 text-[14px]">{{ g.Name }}<br /><span class="text-xs text-muted">Driver {{ g.DriverVersion }}, {{ g.DriverDate }}</span></p>
      </article>
      <article class="panel p-4">
        <h2 class="label">Drives</h2>
        <div v-for="d in facts?.Disks ?? []" :key="d.Name" class="mt-2 text-[14px]">
          <p>{{ d.Name }}</p>
          <p class="text-xs text-muted">{{ d.Media }}, <span :class="d.Health !== 'Healthy' && 'text-bad'">{{ d.Health.toLowerCase() }}</span>
            <template v-if="d.TempC >= 0">, {{ d.TempC }} °C</template><template v-if="d.Wear >= 0">, {{ d.Wear }}% worn</template></p>
        </div>
      </article>
    </div>

    <article class="panel">
      <header class="border-b border-line px-4 py-3">
        <h2 class="text-[14px] font-semibold">Memory</h2>
        <p class="text-xs text-muted">{{ facts?.Memory?.length ?? 0 }} modules, {{ formatBytes((facts?.Memory ?? []).reduce((n, m) => n + m.CapacityBytes, 0)) }} in total</p>
      </header>
      <table class="w-full text-[14px] tabular-nums">
        <tr class="text-muted"><th class="px-4 py-2 text-left font-medium">Slot</th><th class="text-left font-medium">Maker</th><th class="text-left font-medium">Part number</th>
          <th class="text-left font-medium">Size</th><th class="text-left font-medium">Rated</th><th class="text-left font-medium">Running at</th></tr>
        <tr v-for="m in facts?.Memory ?? []" :key="m.Slot" class="border-t border-line/60">
          <td class="px-4 py-1.5">{{ m.Slot }}</td><td>{{ m.Maker }}</td><td class="font-mono text-xs">{{ m.Part }}</td><td>{{ formatBytes(m.CapacityBytes) }}</td>
          <td>{{ m.RatedMHz }} MT/s</td><td :class="m.ConfiguredMHz > m.RatedMHz && 'text-warn'">{{ m.ConfiguredMHz }} MT/s</td>
        </tr>
      </table>
    </article>

    <article class="panel">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <div>
          <h2 class="text-[14px] font-semibold">Cores</h2>
          <p class="text-xs text-muted">Live load and clock of each logical processor. A red count is the hardware errors Windows logged for it in 30 days. Select cores to test them.</p>
        </div>
        <div class="flex flex-wrap gap-1">
          <button class="btn h-7 text-xs" @click="pickWhere((c) => c.kind === 'P')">Performance</button>
          <button class="btn h-7 text-xs" @click="pickWhere((c) => c.kind === 'E')">Efficiency</button>
          <button class="btn h-7 text-xs" @click="pickWhere((c) => c.whea > 0)">With errors</button>
          <button class="btn h-7 text-xs" @click="pickWhere(() => true)">All</button>
          <button class="btn h-7 text-xs" @click="picked = new Set()">None</button>
        </div>
      </header>
      <div class="grid grid-cols-[repeat(auto-fill,minmax(8rem,1fr))] gap-1.5 p-3">
        <button v-for="c in cores" :key="c.index" :aria-pressed="picked.has(c.index)"
                class="flex flex-col gap-1.5 rounded border p-2 text-left text-xs tabular-nums transition"
                :class="picked.has(c.index) ? 'border-accent bg-accent/10' : 'border-line hover:bg-hover'" @click="pick(c.index)">
          <span class="flex items-center justify-between">
            <span><b class="text-[14px]">{{ c.index }}</b> <span class="text-muted">{{ c.kind === 'P' ? 'P-core' : c.kind === 'E' ? 'E-core' : 'Core' }} {{ c.core }}</span></span>
            <span v-if="c.whea" class="rounded-full bg-bad px-1.5 text-[12px] font-semibold text-white" :title="`${c.whea} hardware errors named this processor in 30 days`">{{ c.whea }}</span>
          </span>
          <span class="h-1 overflow-hidden rounded-full bg-hover"><span class="block h-full rounded-full transition-[width] duration-500"
                :style="{ width: `${loads[c.index]?.load ?? 0}%`, background: loadTone(loads[c.index]) }" /></span>
          <span class="flex justify-between"><span class="text-muted">{{ Math.round(loads[c.index]?.load ?? 0) }}%</span>
            <span>{{ loads[c.index] ? `${Math.round(loads[c.index].mhz)} MHz` : '–' }}</span></span>
          <span class="font-mono text-[12px] text-muted">APIC {{ c.apic }}<template v-if="c.sibling >= 0">, pairs with {{ c.sibling }}</template></span>
        </button>
      </div>
      <footer class="flex flex-wrap items-center gap-3 border-t border-line px-4 py-3">
        <span>Core test on <b>{{ picked.size }}</b> selected</span>
        <label class="flex items-center gap-2 text-muted">Seconds each
          <input v-model.number="seconds" type="number" min="5" max="600" class="field w-20" /></label>
        <span class="text-xs text-muted">About {{ Math.ceil((picked.size * seconds) / 60) }} min</span>
        <button class="btn-primary ml-auto" :disabled="!picked.size || testing" @click="confirm = true">{{ testing ? 'Testing…' : 'Run core test' }}</button>
      </footer>
      <div v-if="result" class="border-t border-line px-4 py-3">
        <p :class="result.newWhea || (result.results ?? []).some((r) => r.errors) ? 'text-bad' : ''">
          {{ (result.results ?? []).filter((r) => r.errors).length ? `${(result.results ?? []).filter((r) => r.errors).length} cores returned wrong results.` : 'Every core returned correct results.' }}
          {{ result.newWhea ? `Windows logged ${result.newWhea} hardware errors during the test.` : 'No hardware errors were logged during the test.' }}
        </p>
        <table class="mt-2 w-full text-[14px] tabular-nums">
          <tr class="text-muted"><th class="py-1 text-left font-medium">Processor</th><th class="text-left font-medium">Rounds</th>
            <th class="text-left font-medium">Wrong results</th><th class="text-left font-medium">Average clock</th></tr>
          <tr v-for="r in result.results ?? []" :key="r.index" class="border-t border-line/60">
            <td class="py-1">{{ r.index }}</td><td>{{ r.rounds }}</td>
            <td :class="r.errors ? 'font-semibold text-bad' : ''">{{ r.skipped || r.errors }}</td>
            <td>{{ r.mhz ? `${Math.round(r.mhz)} MHz` : '–' }}</td>
          </tr>
        </table>
      </div>
    </article>

    <ConfirmDialog :open="confirm" :title="`Test ${picked.size} cores for ${seconds} seconds each?`" confirm="Run core test" danger
                   warning="If a core is unstable the PC may freeze or restart. Save your work first." @yes="test" @no="confirm = false">
      Each selected core runs at full load on its own, so it boosts as high as it can, which is where a degraded core makes mistakes.
      On an unstable PC this can freeze it or cause a blue screen, so save your work first. You can cancel from the status bar.
    </ConfirmDialog>
  </div>
</template>
