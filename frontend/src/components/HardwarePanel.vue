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
  <div class="flex flex-col gap-5">
    <p v-if="error" class="border border-bad/40 bg-bad/10 px-4 py-3 font-mono text-xs text-bad">ERROR: {{ error }}</p>

    <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <article class="border border-line bg-panel p-4">
        <p class="caption">processor</p>
        <p class="mt-1 font-semibold">{{ facts?.CPU.Name ?? '…' }}</p>
        <dl class="mt-3 grid grid-cols-[6rem_1fr] gap-y-1 font-mono text-xs">
          <dt class="text-muted">cores</dt><dd>{{ facts ? `${facts.CPU.Cores}` : '--' }}<span v-if="pCount" class="text-muted"> · {{ pCount }} P + {{ eCount }} E</span></dd>
          <dt class="text-muted">threads</dt><dd>{{ facts?.CPU.Threads ?? '--' }}</dd>
          <dt class="text-muted">base clock</dt><dd>{{ facts ? `${facts.CPU.MaxMHz} MHz` : '--' }}</dd>
          <dt class="text-muted">peak now</dt><dd class="text-accent">{{ peak ? `${Math.round(peak)} MHz` : '--' }}</dd>
          <dt class="text-muted">cache</dt><dd>{{ facts ? `L2 ${facts.CPU.L2KB / 1024} MB · L3 ${facts.CPU.L3KB / 1024} MB` : '--' }}</dd>
        </dl>
      </article>
      <article class="border border-line bg-panel p-4">
        <p class="caption">motherboard</p>
        <p class="mt-1 font-semibold">{{ facts?.Board ?? '…' }}</p>
        <dl class="mt-3 grid grid-cols-[6rem_1fr] gap-y-1 font-mono text-xs">
          <dt class="text-muted">bios</dt><dd>{{ facts ? `${facts.BIOS} · ${facts.BIOSDate}` : '--' }}</dd>
          <dt class="text-muted">hypervisor</dt><dd>{{ facts ? (facts.Hypervisor ? 'on (Hyper-V, WSL 2 or VBS)' : 'off') : '--' }}</dd>
        </dl>
        <p class="caption mt-4">graphics</p>
        <p v-for="g in facts?.GPUs ?? []" :key="g.Name" class="mt-1 font-mono text-xs">{{ g.Name }} <span class="text-muted">· {{ g.DriverVersion }} · {{ g.DriverDate }}</span></p>
      </article>
      <article class="border border-line bg-panel p-4">
        <p class="caption">drives</p>
        <div v-for="d in facts?.Disks ?? []" :key="d.Name" class="mt-2 font-mono text-xs">
          <p class="text-fg">{{ d.Name }}</p>
          <p class="text-muted">{{ d.Media }} · <span :class="d.Health === 'Healthy' ? 'text-ok' : 'text-bad'">{{ d.Health }}</span>
            <template v-if="d.TempC >= 0"> · {{ d.TempC }}°C</template><template v-if="d.Wear >= 0"> · {{ d.Wear }}% worn</template></p>
        </div>
      </article>
    </div>

    <article class="border border-line bg-panel">
      <header class="border-b border-line px-4 py-3"><p class="caption">memory · {{ facts?.Memory?.length ?? 0 }} sticks · {{ formatBytes((facts?.Memory ?? []).reduce((n, m) => n + m.CapacityBytes, 0)) }}</p></header>
      <table class="w-full font-mono text-xs">
        <tr class="text-muted"><td class="px-4 py-2">slot</td><td>maker</td><td>part</td><td>size</td><td>rated</td><td>running at</td></tr>
        <tr v-for="m in facts?.Memory ?? []" :key="m.Slot" class="border-t border-line/60">
          <td class="px-4 py-1.5">{{ m.Slot }}</td><td>{{ m.Maker }}</td><td>{{ m.Part }}</td><td>{{ formatBytes(m.CapacityBytes) }}</td>
          <td>{{ m.RatedMHz }} MT/s</td><td :class="m.ConfiguredMHz > m.RatedMHz ? 'text-warn' : ''">{{ m.ConfiguredMHz }} MT/s</td>
        </tr>
      </table>
    </article>

    <article class="border border-line bg-panel">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <p class="caption">every core · live · click to pick for the test</p>
        <div class="flex flex-wrap gap-1 font-mono text-[11px]">
          <button class="border border-line px-2 py-1 hover:bg-hover" @click="pickWhere((c) => c.kind === 'P')">P-cores</button>
          <button class="border border-line px-2 py-1 hover:bg-hover" @click="pickWhere((c) => c.kind === 'E')">E-cores</button>
          <button class="border border-line px-2 py-1 hover:bg-hover" @click="pickWhere((c) => c.whea > 0)">with errors</button>
          <button class="border border-line px-2 py-1 hover:bg-hover" @click="pickWhere(() => true)">all</button>
          <button class="border border-line px-2 py-1 hover:bg-hover" @click="picked = new Set()">none</button>
        </div>
      </header>
      <div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-1.5 p-3">
        <button v-for="c in cores" :key="c.index"
                class="relative flex flex-col gap-1 border p-2 text-left font-mono text-[11px] transition"
                :class="picked.has(c.index) ? 'border-accent bg-accent/10' : 'border-line hover:bg-hover'" @click="pick(c.index)">
          <span class="flex items-center justify-between">
            <span><b class="text-fg">LP{{ c.index }}</b> <span class="text-muted">{{ c.kind }}{{ c.core }}</span></span>
            <span v-if="c.whea" class="bg-bad px-1 text-[10px] font-bold text-white" :title="`${c.whea} WHEA errors named this processor in 30 days`">{{ c.whea }}</span>
          </span>
          <span class="h-1.5 bg-hover"><span class="block h-full transition-[width] duration-500"
                :style="{ width: `${loads[c.index]?.load ?? 0}%`, background: loadTone(loads[c.index]) }" /></span>
          <span class="flex justify-between text-muted"><span>{{ Math.round(loads[c.index]?.load ?? 0) }}%</span>
            <span class="text-fg">{{ loads[c.index] ? Math.round(loads[c.index].mhz) : '--' }}</span></span>
          <span class="text-[10px] text-muted">apic {{ c.apic }}{{ c.sibling >= 0 ? ` · ↔ LP${c.sibling}` : '' }}</span>
        </button>
      </div>
      <footer class="flex flex-wrap items-center gap-3 border-t border-line px-4 py-3 font-mono text-xs">
        <span class="text-muted">core test: {{ picked.size }} picked ·</span>
        <label class="flex items-center gap-2">seconds each
          <input v-model.number="seconds" type="number" min="5" max="600" class="h-8 w-20 border border-line bg-raised px-2 outline-none focus:border-accent" /></label>
        <button class="h-8 bg-accent px-4 font-bold text-white transition hover:brightness-110 disabled:opacity-50"
                :disabled="!picked.size || testing" @click="confirm = true">{{ testing ? 'testing… (status bar)' : 'run test' }}</button>
        <span class="text-muted">≈ {{ Math.ceil((picked.size * seconds) / 60) }} min</span>
      </footer>
      <div v-if="result" class="border-t border-line px-4 py-3">
        <p class="font-mono text-xs" :class="result.newWhea || (result.results ?? []).some((r) => r.errors) ? 'text-bad' : 'text-ok'">
          <Icon :name="result.newWhea || (result.results ?? []).some((r) => r.errors) ? 'close' : 'check'" class="inline size-3.5!" />
          {{ (result.results ?? []).filter((r) => r.errors).length }} processors made calculation errors · {{ result.newWhea }} new WHEA errors during the test
        </p>
        <table class="mt-2 w-full font-mono text-xs">
          <tr class="text-muted"><td class="py-1">processor</td><td>rounds</td><td>errors</td><td>avg clock</td></tr>
          <tr v-for="r in result.results ?? []" :key="r.index" class="border-t border-line/60">
            <td class="py-1">LP{{ r.index }}</td><td>{{ r.rounds }}</td>
            <td :class="r.errors ? 'font-bold text-bad' : 'text-ok'">{{ r.skipped || r.errors }}</td>
            <td>{{ r.mhz ? `${Math.round(r.mhz)} MHz` : '--' }}</td>
          </tr>
        </table>
      </div>
    </article>

    <ConfirmDialog :open="confirm" :title="`Stress ${picked.size} processors, ${seconds} s each?`" confirm="run test" danger @yes="test" @no="confirm = false">
      Each picked processor runs at full load alone, so it boosts as high as it can. That is where a degraded core makes mistakes,
      and on an unstable PC it can also freeze or blue-screen it. Save your work first. Cancel any time from the status bar.
    </ConfirmDialog>
  </div>
</template>
