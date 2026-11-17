<script setup lang="ts">
import { Window } from '@wailsio/runtime'
import Icon from './Icon.vue'
import WindowControls from './WindowControls.vue'
import logo from '../assets/logo.svg'

// locked hides what opens other screens, while the terms are unanswered.
defineProps<{ title: string; locked?: boolean }>()
const emit = defineEmits<{ agent: []; settings: [] }>()
</script>

<template>
  <header class="drag flex h-10 shrink-0 items-stretch border-b border-line bg-panel" @dblclick.self="Window.ToggleMaximise()">
    <div class="flex w-56 shrink-0 items-center gap-2.5 border-r border-line px-4" @dblclick="Window.ToggleMaximise()">
      <img :src="logo" alt="" class="size-5" />
      <span class="font-display text-[14px] font-semibold">ReforgePulse</span>
    </div>
    <div class="flex flex-1 items-center px-4 text-[13px] text-muted" @dblclick="Window.ToggleMaximise()">{{ title }}</div>
    <div v-if="!locked" class="no-drag flex items-center gap-1 pr-2">
      <button class="flex h-7 items-center gap-1.5 rounded px-2.5 text-[12px] text-muted transition hover:bg-hover hover:text-fg"
              title="Connect an AI agent" @click="emit('agent')">
        <Icon name="plug" class="size-4!" />Agents
      </button>
      <button class="grid size-7 place-items-center rounded text-muted transition hover:bg-hover hover:text-fg" title="Settings (Ctrl+,)"
              aria-label="Settings" @click="emit('settings')">
        <Icon name="settings" class="size-4!" />
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
