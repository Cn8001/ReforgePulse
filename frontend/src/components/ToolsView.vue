<script setup lang="ts">
import { ref } from 'vue'
import HardwarePanel from './HardwarePanel.vue'
import CpuCapPanel from './CpuCapPanel.vue'
import SensorsPanel from './SensorsPanel.vue'
import EventsPanel from './EventsPanel.vue'
import DumpsPanel from './DumpsPanel.vue'
import RecorderPanel from './RecorderPanel.vue'
import ChecksPanel from './ChecksPanel.vue'
import DriversPanel from './DriversPanel.vue'

const props = defineProps<{ tab?: string }>()
const tabs: { id: string; label: string }[] = [
  { id: 'hardware', label: 'Hardware' },
  { id: 'sensors', label: 'Sensors' },
  { id: 'events', label: 'Events' },
  { id: 'dumps', label: 'Minidumps' },
  { id: 'recorder', label: 'Recorder' },
  { id: 'drivers', label: 'Drivers' },
  { id: 'checks', label: 'Checks' },
  { id: 'cpu-cap', label: 'CPU cap' },
]
const tab = ref(props.tab ?? 'hardware')
</script>

<template>
  <section class="mx-auto flex max-w-6xl flex-col gap-5 p-6">
    <header>
      <h1 class="heading">Tools</h1>
      <p class="mt-1 text-muted">Look at each part of the PC up close, test it, and tune it.</p>
    </header>
    <nav class="tabs" aria-label="Tools">
      <button v-for="t in tabs" :key="t.id" class="tab" :class="tab === t.id && 'tab-on'" :aria-current="tab === t.id" @click="tab = t.id">
        {{ t.label }}
      </button>
    </nav>
    <KeepAlive>
      <HardwarePanel v-if="tab === 'hardware'" />
      <SensorsPanel v-else-if="tab === 'sensors'" />
      <EventsPanel v-else-if="tab === 'events'" />
      <DumpsPanel v-else-if="tab === 'dumps'" />
      <RecorderPanel v-else-if="tab === 'recorder'" />
      <DriversPanel v-else-if="tab === 'drivers'" />
      <ChecksPanel v-else-if="tab === 'checks'" />
      <CpuCapPanel v-else />
    </KeepAlive>
  </section>
</template>
