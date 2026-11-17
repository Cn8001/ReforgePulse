<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const command = 'claude mcp add --scope user reforgepulse -- "C:\\Program Files\\ReforgePulse\\reforgepulse.exe" mcp'
const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const typed = ref(command)
const shown = ref(4)
const typing = ref(false)
let run = 0
const timers: number[] = []
const later = (ms: number, f: () => void) => timers.push(window.setTimeout(f, ms))

function play() {
  if (reduce) return
  const id = ++run
  timers.splice(0).forEach(clearTimeout)
  typed.value = ''
  shown.value = 0
  typing.value = true
  let i = 0
  const type = () => {
    if (id !== run) return
    typed.value = command.slice(0, ++i)
    if (i < command.length) return later(18 + Math.random() * 30, type)
    typing.value = false
    for (let j = 1; j <= 4; j++) later(400 + j * 800, () => { if (id === run) shown.value = j })
  }
  later(300, type)
}

onMounted(play)
onBeforeUnmount(() => { run++; timers.splice(0).forEach(clearTimeout) })
</script>

<template>
  <div class="term">
    <div class="head"><span>Terminal</span><button v-if="!reduce" type="button" @click="play">Replay</button></div>
    <div class="body">
      <p><span class="pr">&gt; </span>{{ typed }}<span v-if="typing" class="caret" /></p>
      <p class="dim" :class="{ hide: shown < 1 }">Added stdio MCP server reforgepulse to user config</p>
      <p class="you" :class="{ hide: shown < 2 }">Why did my PC restart last night?</p>
      <p class="tool" :class="{ hide: shown < 3 }">reforgepulse: get_crash_evidence</p>
      <p class="ai" :class="{ hide: shown < 4 }">There was one unexpected restart at 02:14 (Kernel-Power 41). No crash was recorded before it and no dump was written. A restart like this usually points to power or heat rather than a driver. The CPU temperature is not measured on this PC. If you run HWiNFO with shared memory enabled and scan again, it will be included.</p>
    </div>
  </div>
</template>

<style scoped>
.term { background: var(--screen); border: 1px solid var(--screen-edge); border-radius: 6px; color: var(--screen-fg); min-width: 0; }
.head { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 10px 14px; border-bottom: 1px solid var(--grat); font-size: 13px; color: var(--screen-muted); }
.head button { font: 500 13px var(--f-body); color: var(--screen-fg); background: none; border: 1px solid var(--grat); border-radius: 4px; padding: 4px 10px; cursor: pointer; }
.body { padding: 16px; font-family: var(--f-mono); font-size: 13px; line-height: 1.7; min-height: 320px; display: grid; align-content: start; gap: 2px; }
p { white-space: pre-wrap; overflow-wrap: anywhere; transition: opacity .3s; }
.pr, .tool { color: var(--trace); }
.dim { color: var(--screen-muted); }
.you, .ai { font-family: var(--f-body); font-size: 15px; }
.you { margin-top: 14px; }
.hide { opacity: 0; }
.caret { display: inline-block; width: 8px; height: 1.1em; background: var(--trace); vertical-align: text-bottom; animation: blink 1s steps(1) infinite; }
@keyframes blink { 50% { opacity: 0; } }
</style>
