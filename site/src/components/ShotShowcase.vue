<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { href } from '../router'

const shots = [
  { src: 'overview.png', label: 'Overview', feature: 'overview', alt: 'Overview: a health score of 87, three issues with their severity, and live processor, graphics and memory load' },
  { src: 'issues.png', label: 'Issues', feature: 'issues', alt: 'Issues: each finding with its severity and confidence' },
  { src: 'cleanup.png', label: 'Cleanup', feature: 'cleanup', alt: 'Cleanup: 2.8 GB can be freed, with 2.4 GB selected' },
  { src: 'processes.png', label: 'Processes', feature: 'processes', alt: 'Processes: a tree of running programs with ports and child processes' },
  { src: 'minidumps.png', label: 'Minidumps', feature: 'minidumps', alt: 'Minidumps: crash dumps with their stop codes, one analysed' },
  { src: 'history.png', label: 'History', feature: 'history', alt: 'History: changes with Revert buttons' },
]

const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const cur = ref(0)
const playing = ref(!reduce)
let timer = 0

function show(i: number) {
  cur.value = (i + shots.length) % shots.length
  restart()
}
function restart() {
  clearInterval(timer)
  if (playing.value) timer = window.setInterval(() => { cur.value = (cur.value + 1) % shots.length }, 4500)
}
function toggle() {
  playing.value = !playing.value
  restart()
}

onMounted(restart)
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <figure class="show">
    <a class="frame" :href="href('features', shots[cur].feature)" :aria-label="`${shots[cur].label}: see this feature`">
      <img v-for="(s, i) in shots" :key="s.src" :src="s.src" :alt="i === cur ? s.alt : ''" :class="{ on: i === cur }" width="2200" height="1400" :loading="i === 0 ? 'eager' : 'lazy'">
    </a>
    <figcaption>
      <div class="tabs" role="tablist" aria-label="Screens">
        <button v-for="(s, i) in shots" :key="s.src" class="tab" role="tab" :aria-selected="i === cur" @click="show(i)">{{ s.label }}</button>
      </div>
      <button v-if="!reduce" class="btn btn-sm" type="button" :aria-pressed="!playing" @click="toggle">{{ playing ? 'Pause' : 'Play' }}</button>
    </figcaption>
    <p class="note">Screenshots show a made-up PC.</p>
  </figure>
</template>

<style scoped>
.show { margin: 0; display: grid; gap: 10px; min-width: 0; }
.frame { display: grid; border-radius: 6px; overflow: hidden; border: 1px solid var(--line); background: var(--app-bg); box-shadow: 0 24px 50px -30px rgb(0 0 0 / 0.6); }
.frame img { grid-area: 1 / 1; width: 100%; opacity: 0; transition: opacity .6s ease; }
.frame img.on { opacity: 1; }
figcaption { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-width: 0; }
figcaption .tabs { flex: 1; min-width: 0; border-bottom: 0; }
.tab { font-size: 14px; padding: 6px 10px 8px; }
.note { font-size: 13px; color: var(--muted); }
</style>
