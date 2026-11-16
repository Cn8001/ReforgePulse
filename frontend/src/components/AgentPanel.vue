<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Clipboard } from '@wailsio/runtime'
import Icon from './Icon.vue'
import { Executable } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/agentservice'

defineProps<{ open: boolean }>()
defineEmits<{ close: [] }>()

const exe = ref('reforgepulse.exe')
onMounted(() => Executable().then((p) => (exe.value = p)))

const agents = computed(() => [
  { name: 'Claude Code', how: 'Run in a terminal:', text: `claude mcp add reforgepulse -- "${exe.value}" mcp` },
  { name: 'Codex', how: 'Run in a terminal:', text: `codex mcp add reforgepulse -- "${exe.value}" mcp` },
  {
    name: 'OpenCode',
    how: 'Add to opencode.json:',
    text: JSON.stringify({ mcp: { reforgepulse: { type: 'local', command: [exe.value, 'mcp'] } } }, null, 2),
  },
])
const tab = ref(0)
const copied = ref(false)

async function copy() {
  await Clipboard.SetText(agents.value[tab.value].text)
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <Transition enter-from-class="opacity-0" leave-to-class="opacity-0"
              enter-active-class="transition-opacity duration-200" leave-active-class="transition-opacity duration-200">
    <div v-if="open" class="fixed inset-0 z-40 bg-black/40" @click="$emit('close')" />
  </Transition>
  <Transition enter-from-class="translate-x-full" leave-to-class="translate-x-full"
              enter-active-class="transition-transform duration-200 ease-out" leave-active-class="transition-transform duration-150 ease-in">
    <aside v-if="open"
           class="fixed inset-y-0 right-0 z-50 flex w-[440px] max-w-full flex-col gap-5 border-l border-line bg-panel p-6 shadow-2xl">
      <header class="flex items-start justify-between gap-4">
        <div>
          <p class="caption">mcp · stdio · read-only</p>
          <h2 class="mt-1 text-lg font-semibold">Connect an agent</h2>
          <p class="mt-1 text-muted">Let an AI agent read what ReforgePulse sees. It can only look; every change still needs your approval here.</p>
        </div>
        <button class="p-1.5 text-muted transition hover:bg-hover hover:text-fg" @click="$emit('close')">
          <Icon name="close" />
        </button>
      </header>

      <div class="flex border border-line">
        <button v-for="(a, i) in agents" :key="a.name"
                class="flex-1 border-r border-line px-2 py-2 font-mono text-xs transition last:border-r-0"
                :class="tab === i ? 'bg-accent text-white' : 'text-muted hover:bg-hover hover:text-fg'"
                @click="tab = i">
          {{ a.name }}
        </button>
      </div>

      <div>
        <p class="mb-2 text-muted">{{ agents[tab].how }}</p>
        <div class="relative border border-line bg-raised">
          <pre class="overflow-x-auto p-4 pr-12 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all select-text">{{ agents[tab].text }}</pre>
          <button class="absolute top-2 right-2 p-1.5 text-muted transition hover:bg-hover hover:text-fg"
                  :title="copied ? 'Copied' : 'Copy'" @click="copy">
            <Icon :name="copied ? 'check' : 'copy'" />
          </button>
        </div>
      </div>

      <p class="mt-auto font-mono text-[11px] text-muted">
        The agent starts <code class="font-mono">reforgepulse.exe mcp</code> as your user, without administrator rights.
      </p>
    </aside>
  </Transition>
</template>
