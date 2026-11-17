<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import TopBar from './components/TopBar.vue'
import SideNav from './components/SideNav.vue'
import HomeView from './components/HomeView.vue'
import ProcessesView from './components/ProcessesView.vue'
import FixView from './components/FixView.vue'
import ToolsView from './components/ToolsView.vue'
import HistoryView from './components/HistoryView.vue'
import CleanupView from './components/CleanupView.vue'
import AgentPanel from './components/AgentPanel.vue'
import SettingsDialog from './components/SettingsDialog.vue'
import StatusBar from './components/StatusBar.vue'
import TermsGate from './components/TermsGate.vue'
import { Accepted } from '../bindings/github.com/cn8001/reforgepulse/internal/app/aboutservice'
import type { IconName } from './lib/icons'

type Page = 'home' | 'processes' | 'cleanup' | 'fix' | 'history' | 'tools'
const pages: { id: Page; label: string; icon: IconName }[] = [
  { id: 'home', label: 'Overview', icon: 'home' },
  { id: 'processes', label: 'Processes', icon: 'list' },
  { id: 'cleanup', label: 'Cleanup', icon: 'cleanup' },
  { id: 'fix', label: 'Issues', icon: 'fix' },
  { id: 'history', label: 'History', icon: 'history' },
  { id: 'tools', label: 'Tools', icon: 'settings' },
]
const page = ref<Page>('home')
const sort = ref<string | undefined>() // how the page opens: the Processes sort or the Cleanup tab
const agentOpen = ref(false)
const settingsOpen = ref(false)
const elevated = ref<boolean | null>(null) // set by Home once the overview arrives
// null while asking; nothing of the app shows until the terms are accepted.
// If the answer cannot be read, the terms are shown again rather than skipped.
const accepted = ref<boolean | null>(null)
Accepted().then((ok) => (accepted.value = ok)).catch(() => (accepted.value = false))

// Ctrl+1..6 switch pages, Ctrl+, opens settings, Escape closes what is open.
function onKey(e: KeyboardEvent) {
  if (!accepted.value) return
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
    <TopBar :title="accepted ? pages.find((p) => p.id === page)!.label : ''" :locked="!accepted"
            @agent="openAgent" @settings="settingsOpen = true" />

    <TermsGate v-if="accepted === false" @accepted="accepted = true" />
    <template v-else-if="accepted">
    <div class="flex min-h-0 flex-1">
    <SideNav :pages="pages" :page="page" @go="go($event as Page)" />
    <main class="min-w-0 flex-1" :class="page === 'processes' ? 'overflow-hidden' : 'overflow-y-auto'">
      <Transition name="page" mode="out-in">
        <HomeView v-if="page === 'home'" @elevated="elevated = $event" @open="(p, s) => go(p as Page, s)" />
        <ProcessesView v-else-if="page === 'processes'" :key="sort" :sort="sort as any" />
        <CleanupView v-else-if="page === 'cleanup'" :key="sort" :tab="sort" />
        <FixView v-else-if="page === 'fix'" @open="(p, s) => go(p as Page, s)" />
        <HistoryView v-else-if="page === 'history'" :key="sort" :tab="sort" @open="(p, s) => go(p as Page, s)" />
        <ToolsView v-else :key="sort" :tab="sort" />
      </Transition>
    </main>
    </div>

    <StatusBar :elevated="elevated" />
    <AgentPanel :open="agentOpen" @close="agentOpen = false" />
    <SettingsDialog :open="settingsOpen" @close="settingsOpen = false" @agent="openAgent" />
    </template>
  </div>
</template>
