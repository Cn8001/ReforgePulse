<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Window } from '@wailsio/runtime'
import Icon from './Icon.vue'

// Minimise, maximise and close for the frameless window, as in bmeditor.
const maximised = ref(false)
const sync = () => void Window.IsMaximised().then((m) => (maximised.value = m)).catch(() => {})
onMounted(() => {
  sync()
  window.addEventListener('resize', sync)
})
onBeforeUnmount(() => window.removeEventListener('resize', sync))
</script>

<template>
  <div class="no-drag flex self-stretch">
    <button class="grid w-12 place-items-center text-muted transition hover:bg-hover hover:text-fg" title="Minimise" @click="Window.Minimise()">
      <Icon name="minimize" class="size-4!" />
    </button>
    <button class="grid w-12 place-items-center text-muted transition hover:bg-hover hover:text-fg" :title="maximised ? 'Restore' : 'Maximise'"
            @click="Window.ToggleMaximise()">
      <Icon :name="maximised ? 'restore' : 'maximize'" class="size-3.5!" />
    </button>
    <button class="grid w-12 place-items-center text-muted transition hover:bg-[#c42b1c] hover:text-white" title="Close" @click="Window.Close()">
      <Icon name="close" class="size-4!" />
    </button>
  </div>
</template>
