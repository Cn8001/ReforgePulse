<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import TopBar from './components/TopBar.vue'
import SideNav from './components/SideNav.vue'
import HomeView from './components/HomeView.vue'
import ProcessesView from './components/ProcessesView.vue'
import ComingSoon from './components/ComingSoon.vue'
import CleanupView from './components/CleanupView.vue'
import AgentPanel from './components/AgentPanel.vue'
import SettingsDialog from './components/SettingsDialog.vue'
import StatusBar from './components/StatusBar.vue'
import type { IconName } from './lib/icons'

type Page = 'home' | 'processes' | 'cleanup' | 'fix'
const pages: { id: Page; label: string; icon: IconName }[] = [
  { id: 'home', label: 'Overview', icon: 'home' },
  { id: 'processes', label: 'Processes', icon: 'list' },
  { id: 'cleanup', label: 'Cleanup', icon: 'cleanup' },
  { id: 'fix', label: 'Fix Issues', icon: 'fix' },
]
const page = ref<Page>('home')
const sort = ref<string | undefined>() // how Processes opens when a Home tile sent us there
const agentOpen = ref(false)
const settingsOpen = ref(false)
const elevated = ref<boolean | null>(null) // set by Home once the overview arrives

// Ctrl+1..4 switch pages, Ctrl+, opens settings, Escape closes what is open.
function onKey(e: KeyboardEvent) {
  const n = Number(e.key)
  if (e.ctrlKey && n >= 1 && n <= pages.length) {
    page.value = pages[n - 1].id
    e.preventDefault()
  } else if (e.ctrlKey && e.key === ',') {
    settingsOpen.value = true
    e.preventDefault()
  } else if (e.key === 'Escape') {
    agentOpen.value = settingsOpen.value = false
  }
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

function go(p: Page, s?: string) {
  sort.value = s
  page.value = p
}

function openAgent() {
  settingsOpen.value = false
  agentOpen.value = true
}
</script>

<template>
  <div class="flex h-full flex-col font-sans text-sm text-fg">
    <TopBar :title="pages.find((p) => p.id === page)!.label" @agent="openAgent" @settings="settingsOpen = true" />

    <div class="flex min-h-0 flex-1">
    <SideNav :pages="pages" :page="page" @go="go($event as Page)" />
    <main class="scope min-w-0 flex-1" :class="page === 'processes' ? 'overflow-hidden' : 'overflow-y-auto'">
      <Transition name="page" mode="out-in">
        <HomeView v-if="page === 'home'" @elevated="elevated = $event" @open="(p, s) => go(p as Page, s)" />
        <ProcessesView v-else-if="page === 'processes'" :key="sort" :sort="sort as any" />
        <CleanupView v-else-if="page === 'cleanup'" />
        <ComingSoon v-else key="fix" icon="fix" title="Fix Issues"
                    text="Crashes, WHEA hardware errors and bottlenecks, correlated into incidents with safe next steps." />
      </Transition>
    </main>
    </div>

    <StatusBar :elevated="elevated" />
    <AgentPanel :open="agentOpen" @close="agentOpen = false" />
    <SettingsDialog :open="settingsOpen" @close="settingsOpen = false" @agent="openAgent" />
  </div>
</template>
