<script setup lang="ts">
import { ref } from 'vue'
import { installer, repo, version } from '../content'

const hashCommand = 'certutil -hashfile reforgepulse-windows-amd64-installer.exe SHA256'
const copied = ref('Copy')
const cmd = ref<HTMLElement | null>(null)

async function copy() {
  try {
    await navigator.clipboard.writeText(hashCommand)
    copied.value = 'Copied'
  } catch {
    const r = document.createRange()
    if (cmd.value) r.selectNodeContents(cmd.value)
    const s = getSelection()
    s?.removeAllRanges()
    s?.addRange(r)
    copied.value = 'Selected'
  }
  setTimeout(() => { copied.value = 'Copy' }, 1600)
}
</script>

<template>
  <section class="page">
    <div class="wrap">
      <div class="page-head">
        <h1>Install</h1>
        <p>Download the installer from the latest release and run it. ReforgePulse keeps itself up to date: it asks before downloading an update and verifies every download against the checksums published with the release.</p>
      </div>
      <div class="grid">
        <div class="card box">
          <h2>Download</h2>
          <p>The installer places ReforgePulse in Program Files and adds it to the Start menu.</p>
          <div>
            <a class="btn btn-primary" :href="installer">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3v12m0 0-5-5m5 5 5-5M4 19h16" /></svg>
              Download version {{ version }}
            </a>
          </div>
          <ul>
            <li>Requires 64-bit Windows.</li>
            <li>Asks for administrator rights each time it starts, because it reads hardware details and system logs.</li>
            <li>Stores its data in <code>%LOCALAPPDATA%\ReforgePulse</code>.</li>
            <li>Uses the network only to check GitHub Releases for updates and to download debugger symbols during a crash dump analysis.</li>
          </ul>
        </div>
        <div class="card box">
          <h2>Check the download</h2>
          <p>The app is not yet code-signed. To confirm that the file is exactly the one that was published, compare its hash with the value in the release's <code>SHA256SUMS</code> file.</p>
          <div class="cmd">
            <code ref="cmd">{{ hashCommand }}</code>
            <button type="button" @click="copy">{{ copied }}</button>
          </div>
          <p>When Windows reports that it protected your PC, select More info, then Run anyway.</p>
          <p>Older versions and release notes are on the <a :href="`${repo}/releases`">releases page</a>.</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px; }
@media (max-width: 820px) { .grid { grid-template-columns: minmax(0, 1fr); } }
.box { padding: 24px; display: grid; gap: 14px; align-content: start; min-width: 0; }
.box h2 { font-size: 24px; }
.box p, .box li { color: var(--muted); }
.box ul { margin: 0; padding-left: 20px; display: grid; gap: 6px; }
.box code { color: var(--ink); }
.cmd { display: flex; border: 1px solid var(--line); border-radius: 4px; background: var(--bg); min-width: 0; }
.cmd code { flex: 1; min-width: 0; padding: 11px 14px; overflow-x: auto; white-space: nowrap; }
.cmd button { flex: none; border: 0; border-left: 1px solid var(--line); background: none; color: var(--ink); font: 500 14px var(--f-body); padding: 0 14px; cursor: pointer; }
.cmd button:hover { background: var(--panel); }
</style>
