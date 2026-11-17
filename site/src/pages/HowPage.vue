<script setup lang="ts">
import { computed } from 'vue'
import Walkthrough from '../components/Walkthrough.vue'
import AgentTerminal from '../components/AgentTerminal.vue'
import { href, route } from '../router'

const agent = computed(() => route.value.sub === 'agents')
</script>

<template>
  <section class="page">
    <div class="wrap">
      <div class="page-head">
        <h1>How it works</h1>
        <p>Use it in the window, or let a coding agent read what it finds.</p>
      </div>
      <nav class="tabs" aria-label="How it works">
        <a class="tab" :href="href('how')" :aria-selected="!agent">In the app</a>
        <a class="tab" :href="href('how', 'agents')" :aria-selected="agent">With a coding agent</a>
      </nav>

      <div class="panel">
        <Walkthrough v-if="!agent" />
        <div v-else class="agents">
          <div class="copy">
            <h2>Ask your coding agent</h2>
            <p>Open Agents in the ReforgePulse window and choose Claude Code, Codex or OpenCode, or register the server yourself with a single command. The server runs without administrator rights and can only read.</p>
            <p>You can then ask questions in plain language, such as why the PC restarted, what is using the memory or which programs left folders behind. Any change still has to be approved by you in the window.</p>
          </div>
          <AgentTerminal />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.tabs a { text-decoration: none; }
.panel { padding-top: 28px; }
.agents { display: grid; grid-template-columns: minmax(0, 0.8fr) minmax(0, 1.2fr); gap: 36px; align-items: start; }
@media (max-width: 920px) { .agents { grid-template-columns: minmax(0, 1fr); } }
.copy { display: grid; gap: 14px; }
.copy h2 { font-size: 30px; }
.copy p { color: var(--muted); max-width: 52ch; }
</style>
