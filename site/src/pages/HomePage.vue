<script setup lang="ts">
import LiveDemo from '../components/LiveDemo.vue'
import { installer, repo, version } from '../content'
import { href } from '../router'
</script>

<template>
  <section class="page">
    <div class="wrap hero">
      <div class="intro">
        <h1>Find out why your PC crashes, and clean up what it no longer needs.</h1>
        <p class="lede">ReforgePulse reads the records Windows already keeps, such as event logs, hardware error reports, crash dumps and sensor readings, and explains what it finds in plain language. Nothing on your PC is changed until you have seen exactly what will happen and agreed to it.</p>
        <div class="actions">
          <a class="btn btn-primary" :href="installer">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3v12m0 0-5-5m5 5 5-5M4 19h16" /></svg>
            Download for Windows
          </a>
          <a class="btn" :href="repo">
            <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C6.5 2 2 6.6 2 12.2c0 4.5 2.9 8.3 6.8 9.7.5.1.7-.2.7-.5v-1.7c-2.8.6-3.4-1.4-3.4-1.4-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.6 2.4 1.1 2.9.9.1-.7.4-1.1.6-1.4-2.2-.3-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.8 1a9.4 9.4 0 0 1 5 0c1.9-1.3 2.8-1 2.8-1 .5 1.4.2 2.4.1 2.7.6.7 1 1.6 1 2.7 0 3.9-2.3 4.7-4.6 5 .4.3.7.9.7 1.9v2.8c0 .3.2.6.7.5A10.2 10.2 0 0 0 22 12.2C22 6.6 17.5 2 12 2Z" /></svg>
            Source on GitHub
          </a>
        </div>
        <p class="meta">Version {{ version }}, for 64-bit Windows. Free and open source under the MIT licence.</p>
        <p class="more"><a :href="href('features')">See what it does</a> or <a :href="href('how')">how it works</a>.</p>
      </div>
      <LiveDemo />
    </div>
  </section>
</template>

<style scoped>
.hero { display: grid; grid-template-columns: minmax(0, 360px) minmax(0, 1fr); gap: 48px; align-items: center; }
@media (max-width: 1060px) { .hero { grid-template-columns: minmax(0, 1fr); gap: 36px; } }
.intro { display: grid; gap: 16px; }
h1 { font-size: clamp(34px, 3.6vw, 44px); }
.lede { color: var(--muted); font-size: 17px; }
.actions { display: grid; grid-template-columns: minmax(0, 1fr); gap: 10px; margin-top: 8px; max-width: 360px; }
@media (max-width: 1060px) and (min-width: 520px) { .actions { grid-template-columns: repeat(2, minmax(0, 1fr)); max-width: 520px; } }
.meta, .more { font-size: 14px; color: var(--muted); }
.more a { color: var(--ink); }
</style>
