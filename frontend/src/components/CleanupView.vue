<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import DiskCleanup from './DiskCleanup.vue'
import LeftoversPanel from './LeftoversPanel.vue'
import ProgramsPanel from './ProgramsPanel.vue'
import StartupPanel from './StartupPanel.vue'
import FeaturesPanel from './FeaturesPanel.vue'
import type { IconName } from '../lib/icons'

const tabs: { id: string; label: string; icon: IconName }[] = [
  { id: 'disk', label: 'Disk space', icon: 'disk' },
  { id: 'leftovers', label: 'Leftovers', icon: 'cleanup' },
  { id: 'downloads', label: 'Old downloads', icon: 'history' },
  { id: 'programs', label: 'Programs', icon: 'list' },
  { id: 'startup', label: 'Startup apps', icon: 'power' },
  { id: 'features', label: 'Features', icon: 'settings' },
]
const props = defineProps<{ tab?: string }>()
const tab = ref(props.tab ?? 'disk')
</script>

<template>
  <section class="mx-auto flex max-w-6xl flex-col gap-5 p-6">
    <header>
      <h1 class="heading">Cleanup</h1>
      <p class="mt-1 text-muted">Free disk space and memory. Each change shows what it will do and waits for your confirmation.</p>
    </header>
    <nav class="tabs" aria-label="Cleanup sections">
      <button v-for="t in tabs" :key="t.id" class="tab" :class="tab === t.id && 'tab-on'" :aria-current="tab === t.id" @click="tab = t.id">
        <Icon :name="t.icon" class="size-4!" />{{ t.label }}
      </button>
    </nav>
    <KeepAlive>
      <DiskCleanup v-if="tab === 'disk'" />
      <LeftoversPanel v-else-if="tab === 'leftovers'" key="leftovers" kind="leftovers" />
      <LeftoversPanel v-else-if="tab === 'downloads'" key="downloads" kind="downloads" />
      <ProgramsPanel v-else-if="tab === 'programs'" />
      <StartupPanel v-else-if="tab === 'startup'" />
      <FeaturesPanel v-else />
    </KeepAlive>
  </section>
</template>
