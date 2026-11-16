<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Clipboard } from '@wailsio/runtime'
import Icon from './Icon.vue'
import { Connect, Executable } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/agentservice'

defineProps<{ open: boolean }>()
defineEmits<{ close: [] }>()

const exe = ref('reforgepulse.exe')
onMounted(() => Executable().then((p) => (exe.value = p)))

const agents = [
  { id: 'claude', name: 'Claude Code', how: 'Added with claude mcp add, for your user' },
  { id: 'codex', name: 'Codex', how: 'Added with codex mcp add' },
  { id: 'opencode', name: 'OpenCode', how: 'Added to ~/.config/opencode/opencode.json' },
] as const

type State = { busy?: boolean; ok?: string; error?: string }
const state = ref<Record<string, State>>({})

async function connect(id: string) {
  state.value[id] = { busy: true }
  try {
    const out = await Connect(id)
    state.value[id] = { ok: out.trim().split('\n').pop() || 'Connected' }
  } catch (e) {
    state.value[id] = { error: String(e) }
  }
}

// For an agent Windows cannot see, such as one running inside WSL.
const wslPath = () => '/mnt/' + exe.value[0].toLowerCase() + exe.value.slice(2).replaceAll('\\', '/')
const copied = ref(false)
async function copyManual() {
  await Clipboard.SetText(`claude mcp add reforgepulse -- "${wslPath()}" mcp`)
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
    <aside v-if="open" role="dialog" aria-modal="true" aria-labelledby="agents-title"
           class="fixed inset-y-0 right-0 z-50 flex w-[440px] max-w-full flex-col gap-5 border-l border-line bg-panel p-6 shadow-2xl">
      <header class="flex items-start justify-between gap-4">
        <div>
          <h2 id="agents-title" class="text-[17px] font-semibold">Connect an AI agent</h2>
          <p class="mt-1 text-muted">The agent can run the scan and read what ReforgePulse sees. It cannot change anything; changes are made here, by you.</p>
        </div>
        <button class="grid size-7 place-items-center rounded text-muted transition hover:bg-hover hover:text-fg" aria-label="Close" @click="$emit('close')"><Icon name="close" /></button>
      </header>

      <div class="panel divide-y divide-line">
        <div v-for="a in agents" :key="a.id" class="flex flex-col gap-2 px-4 py-3">
          <div class="flex items-center gap-3">
            <div class="min-w-0 flex-1">
              <p class="font-medium">{{ a.name }}</p>
              <p class="text-xs text-muted">{{ a.how }}</p>
            </div>
            <button :class="state[a.id]?.ok ? 'btn' : 'btn-primary'" :disabled="state[a.id]?.busy" @click="connect(a.id)">
              {{ state[a.id]?.busy ? 'Connecting…' : state[a.id]?.ok ? 'Connected' : 'Connect' }}
            </button>
          </div>
          <p v-if="state[a.id]?.error" class="text-xs text-bad">
            {{ state[a.id]!.error!.includes('not found') ? `${a.name} is not installed on Windows. If it runs in WSL, use the command below.` : state[a.id]!.error }}
          </p>
          <p v-else-if="state[a.id]?.ok" class="truncate font-mono text-[11px] text-muted">{{ state[a.id]!.ok }}</p>
        </div>
      </div>

      <div class="mt-auto">
        <h3 class="label mb-2">For an agent running inside WSL</h3>
        <div class="relative rounded border border-line bg-raised">
          <pre class="overflow-x-auto p-3 pr-10 font-mono text-[11px] whitespace-pre-wrap break-all select-text">claude mcp add reforgepulse -- "{{ wslPath() }}" mcp</pre>
          <button class="absolute top-2 right-2 rounded p-1 text-muted transition hover:text-fg" :aria-label="copied ? 'Copied' : 'Copy the command'" @click="copyManual">
            <Icon :name="copied ? 'check' : 'copy'" class="size-3.5!" />
          </button>
        </div>
        <p class="mt-2 text-xs text-muted">The connection runs as your user, without administrator rights.</p>
      </div>
    </aside>
  </Transition>
</template>
