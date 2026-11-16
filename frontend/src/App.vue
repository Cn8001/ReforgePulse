<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from './components/Icon.vue'
import HomeView from './components/HomeView.vue'
import ComingSoon from './components/ComingSoon.vue'
import AgentPanel from './components/AgentPanel.vue'
import StatusBar from './components/StatusBar.vue'
import type { IconName } from './lib/icons'

type Page = 'home' | 'cleanup' | 'fix'
const pages: { id: Page; label: string; icon: IconName; hint: string }[] = [
  { id: 'home', label: 'Home', icon: 'home', hint: 'Ctrl+1' },
  { id: 'cleanup', label: 'Cleanup', icon: 'cleanup', hint: 'Ctrl+2' },
  { id: 'fix', label: 'Fix Issues', icon: 'fix', hint: 'Ctrl+3' },
]
const page = ref<Page>('home')
const agentOpen = ref(false)
const elevated = ref<boolean | null>(null) // set by Home once the overview arrives
const activeIndex = computed(() => pages.findIndex((p) => p.id === page.value))

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
  <div class="flex h-full font-sans text-sm text-fg">
    <aside class="flex w-56 shrink-0 flex-col gap-1 p-3">
      <div class="mb-5 flex items-center gap-2.5 px-3 pt-2">
        <span class="grid size-7 place-items-center rounded-lg bg-accent text-white shadow-sm shadow-accent/40">
          <Icon name="fix" />
        </span>
        <span class="text-[15px] font-semibold tracking-tight">ReforgePulse</span>
      </div>

      <nav class="relative flex flex-col gap-1">
        <!-- One highlight slides between items instead of each item lighting up. -->
        <span
          class="absolute inset-x-0 top-0 h-10 rounded-lg bg-hover transition-transform duration-300 ease-[cubic-bezier(.2,.8,.2,1)]"
          :style="{ transform: `translateY(${activeIndex * 44}px)` }"
        >
          <span class="absolute top-1/2 left-0 h-4 w-[3px] -translate-y-1/2 rounded-full bg-accent" />
        </span>
        <button
          v-for="p in pages"
          :key="p.id"
          :title="p.hint"
          class="group relative flex h-10 items-center gap-3 rounded-lg px-3 text-left transition-colors"
          :class="page === p.id ? 'font-medium text-fg' : 'text-muted hover:text-fg'"
          @click="page = p.id"
        >
          <span class="transition-transform duration-200 group-hover:scale-110 group-active:scale-95"
                :class="page === p.id && 'text-accent'">
            <Icon :name="p.icon" />
          </span>
          {{ p.label }}
        </button>
      </nav>

      <button
        class="mt-auto flex h-10 items-center gap-3 rounded-lg px-3 text-muted transition-colors hover:bg-hover hover:text-fg"
        @click="agentOpen = true"
      >
        <Icon name="plug" />
        Connect an agent
      </button>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
    <main class="min-h-0 flex-1 overflow-y-auto p-3 pb-0 pl-0">
      <div class="min-h-full rounded-2xl border border-line bg-surface p-8 backdrop-blur-xl">
        <Transition name="page" mode="out-in">
          <HomeView v-if="page === 'home'" @elevated="elevated = $event" />
          <ComingSoon v-else-if="page === 'cleanup'" key="cleanup" icon="cleanup" title="Cleanup"
                      text="Find what takes up space and remove what is not needed, always after you approve it." />
          <ComingSoon v-else key="fix" icon="fix" title="Fix Issues"
                      text="Crashes, hardware errors and slowdowns, explained in plain language with safe next steps." />
        </Transition>
      </div>
    </main>
    <StatusBar :elevated="elevated" />
    </div>

    <AgentPanel :open="agentOpen" @close="agentOpen = false" />
  </div>
</template>
