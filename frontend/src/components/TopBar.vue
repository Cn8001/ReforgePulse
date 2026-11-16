<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Window } from '@wailsio/runtime'
import Icon from './Icon.vue'
import WindowControls from './WindowControls.vue'
import logo from '../assets/logo.svg'
import type { IconName } from '../lib/icons'

const props = defineProps<{ pages: { id: string; label: string; icon: IconName }[]; page: string }>()
const emit = defineEmits<{ go: [id: string]; agent: []; settings: [] }>()

// One underline slides to the active tab instead of each tab lighting up.
const tabs = ref<HTMLElement[]>([])
const underline = ref({ left: 0, width: 0 })
function place() {
  const el = tabs.value[props.pages.findIndex((p) => p.id === props.page)]
  if (el) underline.value = { left: el.offsetLeft, width: el.offsetWidth }
}
watch(() => props.page, () => nextTick(place))
onMounted(() => {
  nextTick(place)
  window.addEventListener('resize', place)
})
onBeforeUnmount(() => window.removeEventListener('resize', place))
</script>

<template>
  <header class="drag flex h-11 shrink-0 items-stretch border-b border-line bg-panel" @dblclick.self="Window.ToggleMaximise()">
    <div class="flex items-center gap-2.5 pr-5 pl-4">
      <img :src="logo" alt="" class="size-6" />
      <span class="font-mono text-[13px] font-bold tracking-wide">REFORGEPULSE</span>
    </div>

    <nav class="no-drag relative flex items-stretch">
      <button v-for="(p, i) in pages" :key="p.id" ref="tabs" :title="`Ctrl+${i + 1}`"
              class="flex items-center gap-2 px-4 transition-colors"
              :class="page === p.id ? 'text-fg' : 'text-muted hover:bg-hover hover:text-fg'"
              @click="emit('go', p.id)">
        <span :class="page === p.id && 'text-accent'"><Icon :name="p.icon" class="size-4!" /></span>
        <span :class="page === p.id && 'font-semibold'">{{ p.label }}</span>
      </button>
      <span class="absolute bottom-0 h-[2px] bg-accent transition-all duration-300 ease-[cubic-bezier(.2,.8,.2,1)]"
            :style="{ left: `${underline.left}px`, width: `${underline.width}px` }" />
    </nav>

    <div class="flex-1" @dblclick="Window.ToggleMaximise()" />

    <div class="no-drag flex items-center gap-1 pr-2">
      <button class="flex h-8 items-center gap-2 px-3 text-muted transition hover:bg-hover hover:text-fg" title="Connect an AI agent over MCP"
              @click="emit('agent')">
        <Icon name="plug" class="size-4!" /><span class="font-mono text-[11px]">MCP</span>
      </button>
      <button class="grid size-8 place-items-center text-muted transition hover:bg-hover hover:text-fg hover:[&>svg]:rotate-90" title="Settings"
              @click="emit('settings')">
        <Icon name="settings" class="size-4! transition-transform duration-300" />
      </button>
    </div>
    <WindowControls />
  </header>
</template>

<style scoped>
.drag {
  --wails-draggable: drag;
}
.no-drag {
  --wails-draggable: no-drag;
}
</style>
