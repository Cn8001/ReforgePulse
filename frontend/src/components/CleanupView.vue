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
  <section class="flex flex-col gap-5 p-6">
    <header class="flex flex-col gap-4">
      <div>
        <p class="caption">reclaim disk and memory · every change asks first and is logged</p>
        <h1 class="mt-1 text-2xl font-semibold">Cleanup</h1>
      </div>
      <div class="flex border border-line font-mono text-xs">
        <button v-for="t in tabs" :key="t.id" class="flex items-center gap-2 border-r border-line px-4 py-2 transition last:border-r-0"
                :class="tab === t.id ? 'bg-accent text-white' : 'text-muted hover:bg-hover hover:text-fg'" @click="tab = t.id">
          <Icon :name="t.icon" class="size-3.5!" />{{ t.label }}
        </button>
      </div>
    </header>
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
