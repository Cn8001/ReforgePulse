<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

interface Step { title: string; text: string; window: string; ms: number }
const steps: Step[] = [
  { title: 'Install', window: 'Installer', ms: 6000, text: 'Run the installer. Because the app is not yet code-signed, Windows shows a warning the first time. Select More info, then Run anyway.' },
  { title: 'Scan your PC', window: 'ReforgePulse', ms: 5600, text: 'A single scan reviews 30 days of event logs, crash records and drivers. Each scan is saved as a report.' },
  { title: 'Review the issues', window: 'ReforgePulse', ms: 4800, text: 'Every finding shows how serious it is and how certain. Open one to see which causes the evidence points to.' },
  { title: 'Clean up', window: 'ReforgePulse', ms: 5600, text: 'Choose what to remove. ReforgePulse shows the total and asks for confirmation before any file is touched.' },
  { title: 'Undo a change', window: 'ReforgePulse', ms: 4800, text: 'If you change your mind, revert the change from History. The revert is recorded as well.' },
]

const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const cur = ref(0)
const run = ref(0) // remounting the scene restarts its animations
const playing = ref(!reduce)
const inView = ref(false)
const progress = ref(0)
const root = ref<HTMLElement | null>(null)
const paused = computed(() => !playing.value || !inView.value)

function go(i: number) {
  cur.value = (i + steps.length) % steps.length
  progress.value = 0
  run.value++
}

function onKey(e: KeyboardEvent, i: number) {
  const next = e.key === 'ArrowDown' || e.key === 'ArrowRight' ? 1 : e.key === 'ArrowUp' || e.key === 'ArrowLeft' ? -1 : 0
  if (!next) return
  e.preventDefault()
  go(i + next)
  ;(root.value?.querySelectorAll<HTMLButtonElement>('.step')[cur.value])?.focus()
}

let raf = 0
let last = 0
function tick(now: number) {
  const dt = last ? now - last : 0
  last = now
  if (!paused.value && !reduce) {
    progress.value += dt / steps[cur.value].ms
    if (progress.value >= 1) go(cur.value + 1)
  }
  raf = requestAnimationFrame(tick)
}

let io: IntersectionObserver | undefined
onMounted(() => {
  io = new IntersectionObserver(([e]) => {
    const was = inView.value
    inView.value = e.isIntersecting
    if (inView.value && !was) go(cur.value)
  }, { threshold: 0.35 })
  if (root.value) io.observe(root.value)
  raf = requestAnimationFrame(tick)
})
onBeforeUnmount(() => { io?.disconnect(); cancelAnimationFrame(raf) })

const check = 'm5 12 5 5 9-10'
</script>

<template>
  <div ref="root" class="usage">
    <ol class="plain steps" role="tablist" aria-label="Steps">
      <li v-for="(s, i) in steps" :key="s.title">
        <button class="step" role="tab" :aria-selected="i === cur" :tabindex="i === cur ? 0 : -1" aria-controls="mock" @click="go(i)" @keydown="onKey($event, i)">
          <span class="n">{{ i + 1 }}</span>
          <strong>{{ s.title }}</strong>
          <span class="d">{{ s.text }}</span>
          <span class="prog" :style="{ width: i === cur ? `${Math.min(1, progress) * 100}%` : '0' }" />
        </button>
      </li>
    </ol>

    <div class="side">
      <div id="mock" class="mock" :class="{ paused }" role="tabpanel">
        <div class="bar">
          <img :src="'icon.png'" alt="" width="18" height="18">
          <span class="t">{{ steps[cur].window }}</span>
          <span class="wc" aria-hidden="true"><span>&#8211;</span><span>&#9633;</span><span>&#215;</span></span>
        </div>
        <div class="body">
          <div v-if="cur === 0" :key="run" class="scene s-install">
            <div class="dlg" data-a="rise" style="animation-delay:.2s">
              <div class="m-h">Windows protected your PC</div>
              <p class="m-sub">Windows stopped an app it does not recognise from starting.</p>
              <span class="lnk" data-a="press" style="animation-delay:1.4s">More info</span>
              <div class="facts" data-a="rise" style="animation-delay:1.9s">
                <span>App</span><b>reforgepulse-windows-amd64-installer.exe</b>
                <span>Publisher</span><b>Unknown publisher</b>
              </div>
              <div class="acts">
                <span class="m-btn p" data-a="rise" style="animation-delay:2.3s"><span class="ring" data-a="press" style="animation-delay:3.4s" />Run anyway</span>
                <span class="m-btn">Don't run</span>
              </div>
            </div>
            <div class="after" data-a="rise" style="animation-delay:4.2s">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="check" /></svg>
              <span>Installed in <span class="mono">C:\Program Files\ReforgePulse</span>. It asks for administrator rights at each start.</span>
            </div>
          </div>

          <div v-else-if="cur === 1" :key="run" class="scene s-scan">
            <div class="scan-top">
              <div class="gauge">
                <svg viewBox="0 0 64 64" aria-hidden="true"><circle class="bgc" cx="32" cy="32" r="26" fill="none" stroke-width="6" /><circle class="fgc" cx="32" cy="32" r="26" fill="none" stroke-width="6" stroke-linecap="round" /></svg>
                <b data-a="fade" style="animation-delay:4s">87</b>
              </div>
              <div class="txt">
                <div class="m-h" data-a="rise" style="animation-delay:3.8s">4 issues need attention</div>
                <div class="m-sub" data-a="rise" style="animation-delay:4s">Health 87 out of 100. Saved as a report.</div>
              </div>
              <span class="m-btn p" data-a="press" style="animation-delay:.5s">Scan PC</span>
            </div>
            <div class="track"><div class="fill" /></div>
            <ul class="plain sources">
              <li v-for="(s, i) in ['Stop errors, none found', 'Unexpected restarts, 2 found', 'WHEA errors, 6 corrected', 'Display driver resets, 3 found', 'App and service failures, 2 found', 'Devices without drivers, none']" :key="s" data-a="rise" :style="{ animationDelay: `${1 + i * 0.4}s` }">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="check" /></svg>
                <span><b>{{ s.split(', ')[0] }}</b>, {{ s.split(', ')[1] }}</span>
              </li>
            </ul>
          </div>

          <div v-else-if="cur === 2" :key="run" class="scene s-issues">
            <div class="m-panel issue" data-a="rise" style="animation-delay:.2s">
              <div class="issue-top"><span class="dot w" /><span class="ttl">The graphics driver stopped responding 3 times</span><span class="chip w">Medium</span></div>
              <div class="suspects" data-a="fade" style="animation-delay:1.3s">
                <div class="sus"><span>Drivers</span><span class="tr"><i style="width:78%;animation-delay:1.6s" /></span><span class="lv">Likely</span></div>
                <div class="sus"><span>Power</span><span class="tr"><i class="g" style="width:40%;animation-delay:1.8s" /></span><span class="lv">Possible</span></div>
                <div class="sus"><span>Memory</span><span class="tr"><i class="g" style="width:8%;animation-delay:2s" /></span><span class="lv">No evidence</span></div>
                <div class="sus"><span>Heat</span><span class="tr un" /><span class="lv">Unmeasured</span></div>
                <p class="note" data-a="fade" style="animation-delay:2.4s">Unmeasured means Windows cannot measure it on this PC. It does not mean everything is fine.</p>
              </div>
            </div>
            <div class="m-panel issue" data-a="rise" style="animation-delay:.4s">
              <div class="issue-top"><span class="dot" /><span class="ttl">Launcher.exe in UnityPlayer.dll crashed 4 times</span><span class="chip">Low</span></div>
            </div>
            <div class="m-panel issue" data-a="rise" style="animation-delay:.6s">
              <div class="issue-top"><span class="dot" /><span class="ttl">9 apps start with Windows</span><span class="chip">Low</span></div>
            </div>
          </div>

          <div v-else-if="cur === 3" :key="run" class="scene s-clean">
            <div class="m-h" data-a="rise" style="animation-delay:.1s">Cleanup</div>
            <ul class="plain m-panel items">
              <li v-for="(it, i) in [['Temporary files', 'Windows and app temp folders', '1.2 GB'], ['Shader caches', 'Rebuilt by games as needed', '640 MB'], ['Windows Update leftovers', 'Downloaded updates already installed', '410 MB'], ['Old downloads', 'Untouched for 180 days, to the Recycle Bin', '180 MB']]" :key="it[0]">
                <span class="check" data-a="tick" :style="{ animationDelay: `${0.4 + i * 0.3}s` }"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="check" /></svg></span>
                <span class="nm">{{ it[0] }}<small>{{ it[1] }}</small></span>
                <span class="sz">{{ it[2] }}</span>
              </li>
            </ul>
            <div class="foot">
              <span class="tot" data-a="rise" style="animation-delay:1.5s">2.4 GB selected</span>
              <span class="m-btn p" data-a="press" style="animation-delay:2s">Clean up</span>
            </div>
            <div class="toast" data-a="rise" style="animation-delay:4.3s">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="check" /></svg>
              Cleaned up 2.4 GB. The change is in History.
            </div>
            <div class="overlay" aria-hidden="true">
              <div class="dlg">
                <div class="m-h">Clean up 2.4 GB?</div>
                <p class="m-sub">Caches are deleted for good. Old downloads go to the Recycle Bin.</p>
                <div class="acts"><span class="m-btn">Cancel</span><span class="m-btn p" data-a="press" style="animation-delay:3.5s">Clean up</span></div>
              </div>
            </div>
          </div>

          <div v-else :key="run" class="scene s-hist">
            <div class="m-h" data-a="rise" style="animation-delay:.1s">History</div>
            <div class="m-panel log">
              <div class="row grow"><div><div class="entry new">
                <span class="h">c02b77e</span><span class="what">Spotify starts with Windows again<small>Revert of a3f9c21, just now</small></span><span />
              </div></div></div>
              <div class="row" data-a="rise" style="animation-delay:.3s"><div><div class="entry">
                <span class="h">a3f9c21</span><span class="what">Spotify no longer starts with Windows<small>Startup apps, 2 min ago</small></span>
                <span class="m-btn sm" data-a="press" style="animation-delay:1.8s"><span class="b1">Revert</span><span class="b2">Reverted</span></span>
              </div></div></div>
              <div class="row" data-a="rise" style="animation-delay:.5s"><div><div class="entry">
                <span class="h">7be0d14</span><span class="what">Cleaned up 2.4 GB<small>Cleanup, 5 min ago</small></span><span />
              </div></div></div>
              <div class="row" data-a="rise" style="animation-delay:.7s"><div><div class="entry">
                <span class="h">19c4e8a</span><span class="what">Processor top speed set to 4800 MHz on every power plan<small>CPU cap, yesterday</small></span><span class="m-btn sm">Revert</span>
              </div></div></div>
            </div>
          </div>
        </div>
      </div>
      <div class="ctl">
        <span>Step {{ cur + 1 }} of {{ steps.length }}</span>
        <button v-if="!reduce" class="btn btn-sm" type="button" :aria-pressed="!playing" @click="playing = !playing">{{ playing ? 'Pause' : 'Play' }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.usage { display: grid; grid-template-columns: minmax(0, 0.8fr) minmax(0, 1.2fr); gap: 36px; align-items: start; }
@media (max-width: 920px) { .usage { grid-template-columns: minmax(0, 1fr); } }
.steps { display: grid; gap: 4px; }
.step { width: 100%; text-align: left; cursor: pointer; display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 2px 12px; padding: 10px 14px 12px; border-radius: 4px; background: none; border: 1px solid transparent; color: var(--ink); font: inherit; position: relative; overflow: hidden; }
.step:hover { background: color-mix(in srgb, var(--panel) 70%, transparent); }
.step[aria-selected="true"] { background: var(--panel); border-color: var(--line); }
.n { grid-row: span 2; width: 28px; height: 28px; border-radius: 50%; display: grid; place-items: center; font: 600 15px/1 var(--f-display); border: 1px solid var(--line); color: var(--muted); }
.step[aria-selected="true"] .n { background: var(--signal); border-color: var(--signal); color: var(--signal-ink); }
.step strong { font: 600 18px/1.3 var(--f-display); }
.d { font-size: 14px; color: var(--muted); line-height: 1.45; }
.prog { position: absolute; left: 0; bottom: 0; height: 2px; background: var(--signal); }
.side { display: grid; gap: 12px; min-width: 0; }
.ctl { display: flex; justify-content: space-between; align-items: center; gap: 12px; font-size: 14px; color: var(--muted); }

.mock { border-radius: 6px; overflow: hidden; border: 1px solid var(--app-line); background: var(--app-bg); color: var(--app-fg); font-size: 14px; line-height: 1.45; box-shadow: 0 24px 48px -28px rgb(0 0 0 / 0.6); }
.bar { display: flex; align-items: center; gap: 10px; padding: 10px 14px; border-bottom: 1px solid var(--app-line); background: var(--app-panel); }
.bar img { width: 18px; height: 18px; }
.t { font-weight: 600; flex: 1; min-width: 0; }
.wc { display: flex; gap: 18px; color: var(--app-muted); font-size: 13px; }
.body { padding: 22px; height: 410px; position: relative; overflow: hidden; }
@media (max-width: 560px) { .body { height: 500px; } }
.scene { height: 100%; }
.m-h { font: 600 22px/1.2 var(--f-display); }
.m-sub { color: var(--app-muted); }
.m-btn { display: inline-flex; align-items: center; justify-content: center; border-radius: 4px; padding: 8px 14px; font-weight: 600; border: 1px solid var(--app-line); background: var(--app-raised); color: var(--app-fg); white-space: nowrap; position: relative; }
.m-btn.p { background: var(--app-accent); border-color: var(--app-accent); color: #fff; }
.m-btn.sm { padding: 5px 10px; font-size: 13px; display: inline-grid; }
.ring { position: absolute; inset: 0; border-radius: 4px; }
.m-panel { background: var(--app-panel); border: 1px solid var(--app-line); border-radius: 6px; }
.chip { border: 1px solid var(--app-line); border-radius: 999px; padding: 1px 10px; font-size: 12px; color: var(--app-muted); white-space: nowrap; }
.chip.w { border-color: var(--app-warn); color: var(--app-warn); }
.dot { width: 9px; height: 9px; border-radius: 50%; background: var(--app-muted); flex: none; }
.dot.w { background: var(--app-warn); }

.s-install { display: grid; place-items: center; align-content: center; gap: 18px; }
.dlg { width: min(100%, 440px); background: var(--app-panel); border: 1px solid var(--app-line); border-radius: 6px; padding: 20px 22px; display: grid; gap: 10px; }
.lnk { color: var(--app-accent); text-decoration: underline; justify-self: start; position: relative; border-radius: 3px; }
.facts { display: grid; grid-template-columns: auto minmax(0, 1fr); gap: 2px 14px; font-size: 13px; color: var(--app-muted); }
.facts b { font-weight: 500; color: var(--app-fg); overflow-wrap: anywhere; }
.acts { display: flex; justify-content: flex-end; gap: 8px; margin-top: 6px; }
.after { display: flex; align-items: center; gap: 10px; color: var(--app-muted); font-size: 13px; }
.after svg, .sources svg, .toast svg { width: 16px; height: 16px; color: var(--app-ok); flex: none; }

.s-scan { display: grid; gap: 18px; align-content: start; }
.scan-top { display: flex; align-items: center; gap: 16px; }
.gauge { width: 64px; height: 64px; flex: none; position: relative; }
.gauge svg { width: 100%; height: 100%; transform: rotate(-90deg); }
.bgc { stroke: var(--app-line); }
.fgc { stroke: var(--app-ok); stroke-dasharray: 163.4; stroke-dashoffset: 21.2; animation: gaugefill 1s 3.8s cubic-bezier(.2,.7,.2,1) both; }
.gauge b { position: absolute; inset: 0; display: grid; place-items: center; font: 600 22px var(--f-display); }
.txt { flex: 1; min-width: 0; }
.track { height: 4px; border-radius: 2px; background: var(--app-line); overflow: hidden; }
.fill { height: 100%; background: var(--app-accent); transform-origin: left; animation: fill 2.8s .8s linear both; }
.sources { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px 20px; }
@media (max-width: 560px) { .sources { grid-template-columns: minmax(0, 1fr); } }
.sources li { display: flex; gap: 10px; align-items: baseline; color: var(--app-muted); }
.sources li svg { transform: translateY(2px); }
.sources b { color: var(--app-fg); font-weight: 500; }

.s-issues { display: grid; gap: 10px; align-content: start; }
.issue { padding: 12px 14px; display: grid; gap: 8px; }
.issue-top { display: flex; align-items: center; gap: 12px; }
.ttl { flex: 1; min-width: 0; font-weight: 600; }
.suspects { display: grid; gap: 7px; padding-top: 10px; border-top: 1px solid var(--app-line); }
.sus { display: grid; grid-template-columns: 80px minmax(0, 1fr) 96px; gap: 12px; align-items: center; font-size: 13px; }
.lv { color: var(--app-muted); text-align: right; }
.tr { height: 6px; border-radius: 3px; background: var(--app-line); overflow: hidden; }
.tr i { display: block; height: 100%; background: var(--app-warn); transform-origin: left; animation: fill .7s cubic-bezier(.2,.7,.2,1) both; }
.tr i.g { background: var(--app-muted); }
.tr.un { background: repeating-linear-gradient(90deg, var(--app-line) 0 6px, transparent 6px 10px); }
.note { font-size: 12px; color: var(--app-muted); }

.s-clean { display: grid; gap: 14px; align-content: start; }
.items li { display: flex; align-items: center; gap: 12px; padding: 9px 14px; border-top: 1px solid var(--app-line); }
.items li:first-child { border-top: 0; }
.nm { flex: 1; min-width: 0; }
.nm small { display: block; color: var(--app-muted); font-size: 12px; }
.sz { color: var(--app-muted); font-variant-numeric: tabular-nums; }
.check { width: 16px; height: 16px; border-radius: 3px; display: inline-grid; place-items: center; flex: none; background: var(--app-accent); border: 1px solid var(--app-accent); color: #fff; }
.check svg { width: 12px; height: 12px; }
.foot { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.tot { font: 600 20px var(--f-display); }
.toast { display: flex; gap: 10px; align-items: center; padding: 10px 14px; border-radius: 6px; background: var(--app-raised); border: 1px solid var(--app-line); }
.overlay { position: absolute; inset: 0; background: rgb(8 10 12 / 0.6); display: grid; place-items: center; padding: 22px; opacity: 0; visibility: hidden; animation: showhide 1.9s 2.3s linear both; }
.overlay .dlg { width: min(100%, 380px); }

.s-hist { display: grid; gap: 14px; align-content: start; }
.row { display: grid; grid-template-rows: 1fr; }
.row > div { min-height: 0; overflow: hidden; }
.row.grow { animation: grow .5s 2.4s cubic-bezier(.2,.7,.2,1) both; }
.entry { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; gap: 4px 14px; align-items: center; padding: 11px 14px; border-top: 1px solid var(--app-line); }
.row:first-child .entry { border-top: 0; }
.h { font-family: var(--f-mono); font-size: 12px; color: var(--app-accent); }
.entry.new { animation: pulsebg 2s 2.9s both; }
.entry.new .h { color: var(--app-ok); }
.what small { display: block; color: var(--app-muted); font-size: 12px; }
.b1, .b2 { grid-area: 1 / 1; }
.b1 { opacity: 0; animation: out .2s 2.2s both; }
.b2 { animation: fade .2s 2.3s both; }

[data-a] { animation-fill-mode: both; animation-timing-function: cubic-bezier(.2,.7,.2,1); }
[data-a="rise"] { animation-name: rise; animation-duration: .45s; }
[data-a="fade"] { animation-name: fade; animation-duration: .4s; }
[data-a="press"] { animation-name: press; animation-duration: .6s; animation-timing-function: ease-out; }
[data-a="tick"] { animation-name: tick; animation-duration: .2s; }
.paused * { animation-play-state: paused !important; }

@keyframes rise { from { opacity: 0; transform: translateY(8px); } }
@keyframes fade { from { opacity: 0; } }
@keyframes out { from { opacity: 1; } to { opacity: 0; } }
@keyframes fill { from { transform: scaleX(0); } }
@keyframes gaugefill { from { stroke-dashoffset: 163.4; } }
@keyframes press { 0% { box-shadow: 0 0 0 0 rgb(59 142 234 / 0.6); filter: brightness(1); } 30% { filter: brightness(1.3); } 100% { box-shadow: 0 0 0 12px rgb(59 142 234 / 0); filter: brightness(1); } }
@keyframes tick { from { background: transparent; border-color: var(--app-muted); color: transparent; } }
@keyframes showhide { 0% { opacity: 0; visibility: visible; } 12% { opacity: 1; } 88% { opacity: 1; } 100% { opacity: 0; visibility: hidden; } }
@keyframes grow { from { grid-template-rows: 0fr; opacity: 0; } }
@keyframes pulsebg { 0% { background: rgb(59 142 234 / 0.16); } 100% { background: transparent; } }
</style>
