<script setup lang="ts">
import { Window } from '@wailsio/runtime'
import Icon from './Icon.vue'
import WindowControls from './WindowControls.vue'
import logo from '../assets/logo.svg'

defineProps<{ title: string }>()
const emit = defineEmits<{ agent: []; settings: [] }>()
</script>

<template>
  <header class="drag flex h-10 shrink-0 items-stretch border-b border-line bg-panel" @dblclick.self="Window.ToggleMaximise()">
    <div class="flex w-56 shrink-0 items-center gap-2.5 border-r border-line px-4" @dblclick="Window.ToggleMaximise()">
      <img :src="logo" alt="" class="size-5" />
      <span class="font-mono text-[12px] font-bold tracking-wide">REFORGEPULSE</span>
    </div>
    <div class="flex flex-1 items-center px-4 font-mono text-xs text-muted" @dblclick="Window.ToggleMaximise()">
      <span class="text-accent">~/</span>{{ title.toLowerCase().replace(' ', '-') }}
    </div>
    <div class="no-drag flex items-center gap-1 pr-2">
      <button class="flex h-7 items-center gap-2 px-2.5 text-muted transition hover:bg-hover hover:text-fg" title="Connect an AI agent over MCP"
              @click="emit('agent')">
        <Icon name="plug" class="size-4!" /><span class="font-mono text-[11px]">MCP</span>
      </button>
      <button class="group grid size-7 place-items-center text-muted transition hover:bg-hover hover:text-fg" title="Settings (Ctrl+,)"
              @click="emit('settings')">
        <Icon name="settings" class="size-4! transition-transform duration-300 group-hover:rotate-90" />
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
