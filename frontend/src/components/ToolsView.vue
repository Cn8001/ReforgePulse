<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import HardwarePanel from './HardwarePanel.vue'
import CpuCapPanel from './CpuCapPanel.vue'
import type { IconName } from '../lib/icons'

const props = defineProps<{ tab?: string }>()
const tabs: { id: string; label: string; icon: IconName }[] = [
  { id: 'hardware', label: 'Hardware & cores', icon: 'cpu' },
  { id: 'cpu-cap', label: 'CPU cap', icon: 'fix' },
]
const tab = ref(props.tab ?? 'hardware')
</script>

<template>
  <section class="flex flex-col gap-5 p-6">
    <header class="flex flex-col gap-4">
      <div>
        <p class="caption">look closer · test · tune</p>
        <h1 class="mt-1 text-2xl font-semibold">Tools</h1>
      </div>
      <div class="flex flex-wrap border border-line font-mono text-xs self-start">
        <button v-for="t in tabs" :key="t.id" class="flex items-center gap-2 border-r border-line px-4 py-2 transition last:border-r-0"
                :class="tab === t.id ? 'bg-accent text-white' : 'text-muted hover:bg-hover hover:text-fg'" @click="tab = t.id">
          <Icon :name="t.icon" class="size-3.5!" />{{ t.label }}
        </button>
      </div>
    </header>
    <KeepAlive>
      <HardwarePanel v-if="tab === 'hardware'" />
      <CpuCapPanel v-else />
    </KeepAlive>
  </section>
</template>
