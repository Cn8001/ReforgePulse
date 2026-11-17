<script setup lang="ts">
import PageLayout from './PageLayout.vue'
import { ref } from 'vue'
import DiskCleanup from './DiskCleanup.vue'
import LeftoversPanel from './LeftoversPanel.vue'
import ProgramsPanel from './ProgramsPanel.vue'
import StartupPanel from './StartupPanel.vue'
import FeaturesPanel from './FeaturesPanel.vue'

const tabs: { id: string; label: string }[] = [
  { id: 'disk', label: 'Disk space' },
  { id: 'leftovers', label: 'Leftovers' },
  { id: 'downloads', label: 'Old downloads' },
  { id: 'programs', label: 'Programs' },
  { id: 'startup', label: 'Startup apps' },
  { id: 'features', label: 'Features' },
]
const props = defineProps<{ tab?: string }>()
const tab = ref(props.tab ?? 'disk')
</script>

<template>
  <PageLayout>
    <template #head>
      <header>
        <h1 class="heading">Cleanup</h1>
        <p class="mt-1 text-muted">Free disk space and memory. Each change shows what it will do and waits for your confirmation.</p>
      </header>
      <nav class="tabs" aria-label="Cleanup sections">
        <button v-for="t in tabs" :key="t.id" class="tab" :class="tab === t.id && 'tab-on'" :aria-current="tab === t.id" @click="tab = t.id">
          {{ t.label }}
        </button>
      </nav>
    </template>
    <KeepAlive>
      <DiskCleanup v-if="tab === 'disk'" />
      <LeftoversPanel v-else-if="tab === 'leftovers'" key="leftovers" kind="leftovers" />
      <LeftoversPanel v-else-if="tab === 'downloads'" key="downloads" kind="downloads" />
      <ProgramsPanel v-else-if="tab === 'programs'" />
      <StartupPanel v-else-if="tab === 'startup'" />
      <FeaturesPanel v-else />
    </KeepAlive>
  </PageLayout>
</template>
