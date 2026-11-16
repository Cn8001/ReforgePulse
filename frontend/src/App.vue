<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from './components/Icon.vue'
import HomeView from './components/HomeView.vue'
import ComingSoon from './components/ComingSoon.vue'
import AgentPanel from './components/AgentPanel.vue'
import StatusBar from './components/StatusBar.vue'
import logo from './assets/logo.svg'
import { theme } from './lib/theme'
import type { IconName } from './lib/icons'

type Page = 'home' | 'cleanup' | 'fix'
const pages: { id: Page; label: string; icon: IconName }[] = [
  { id: 'home', label: 'Overview', icon: 'home' },
  { id: 'cleanup', label: 'Cleanup', icon: 'cleanup' },
  { id: 'fix', label: 'Fix Issues', icon: 'fix' },
]
const page = ref<Page>('home')
const agentOpen = ref(false)
const elevated = ref<boolean | null>(null) // set by Home once the overview arrives

// Ctrl+1..3 switch pages, Escape closes the agent panel.
function onKey(e: KeyboardEvent) {
  const n = Number(e.key)
  if (e.ctrlKey && n >= 1 && n <= pages.length) {
    page.value = pages[n - 1].id
    e.preventDefault()
  } else if (e.key === 'Escape') {
    agentOpen.value = false
  }
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="flex h-full flex-col font-sans text-sm text-fg">
    <div class="flex min-h-0 flex-1">
      <aside class="flex w-56 shrink-0 flex-col border-r border-line bg-panel/60">
        <div class="flex items-center gap-2.5 border-b border-line px-4 py-4">
          <img :src="logo" alt="" class="size-8" />
          <div class="leading-tight">
            <p class="font-mono text-[13px] font-bold tracking-wide">REFORGEPULSE</p>
            <p class="caption text-[10px]! tracking-normal!">diagnose · fix · verify</p>
          </div>
        </div>

        <nav class="flex flex-col py-2">
          <button
            v-for="(p, i) in pages"
            :key="p.id"
            class="group relative flex h-10 items-center gap-3 px-4 text-left transition-colors duration-150"
            :class="page === p.id ? 'bg-hover text-fg' : 'text-muted hover:bg-hover/60 hover:text-fg'"
            @click="page = p.id"
          >
            <span class="absolute inset-y-0 left-0 w-[3px] bg-accent transition-transform duration-200 ease-out"
                  :class="page === p.id ? 'scale-y-100' : 'scale-y-0'" />
            <span :class="page === p.id && 'text-accent'"><Icon :name="p.icon" /></span>
            <span class="flex-1" :class="page === p.id && 'font-semibold'">{{ p.label }}</span>
            <kbd class="font-mono text-[10px] text-muted opacity-0 transition-opacity group-hover:opacity-100">^{{ i + 1 }}</kbd>
          </button>
        </nav>

        <div class="mt-auto border-t border-line">
          <button
            class="flex h-10 w-full items-center gap-3 px-4 text-muted transition-colors hover:bg-hover hover:text-fg"
            @click="agentOpen = true"
          >
            <Icon name="plug" />
            <span class="flex-1 text-left">Connect an agent</span>
            <span class="font-mono text-[10px]">MCP</span>
          </button>
          <button
            class="flex h-10 w-full items-center gap-3 px-4 text-muted transition-colors hover:bg-hover hover:text-fg"
            @click="theme = theme === 'dark' ? 'light' : 'dark'"
          >
            <Transition name="page" mode="out-in">
              <Icon :key="theme" :name="theme === 'dark' ? 'moon' : 'sun'" />
            </Transition>
            <span class="flex-1 text-left">{{ theme === 'dark' ? 'Dark' : 'Light' }} theme</span>
          </button>
        </div>
      </aside>

      <main class="scope min-w-0 flex-1 overflow-y-auto">
        <Transition name="page" mode="out-in">
          <HomeView v-if="page === 'home'" @elevated="elevated = $event" />
          <ComingSoon v-else-if="page === 'cleanup'" key="cleanup" icon="cleanup" title="Cleanup"
                      text="Find what takes up space and remove what is not needed, always after you approve it." />
          <ComingSoon v-else key="fix" icon="fix" title="Fix Issues"
                      text="Crashes, WHEA hardware errors and bottlenecks, correlated into incidents with safe next steps." />
        </Transition>
      </main>
    </div>

    <StatusBar :elevated="elevated" />
    <AgentPanel :open="agentOpen" @close="agentOpen = false" />
  </div>
</template>
