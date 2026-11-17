<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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
  { id: 'opencode', name: 'OpenCode', how: 'Added to ~/.config/opencode/opencode.json; by hand, merge this into it' },
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

// The same setup by hand, for when Connect fails or the agent runs where
// Windows cannot see it, such as inside WSL: there the path is /mnt/c/....
const where = ref<'windows' | 'wsl'>('windows')
const path = computed(() =>
  where.value === 'wsl' ? '/mnt/' + exe.value[0].toLowerCase() + exe.value.slice(2).replaceAll('\\', '/') : exe.value,
)
const commands = computed<Record<string, string>>(() => ({
  claude: `claude mcp add --scope user reforgepulse -- "${path.value}" mcp`,
  codex: `codex mcp add reforgepulse -- "${path.value}" mcp`,
  opencode: JSON.stringify({ mcp: { reforgepulse: { type: 'local', command: [path.value, 'mcp'], enabled: true } } }, null, 2),
}))
const copied = ref('')
async function copy(id: string) {
  await Clipboard.SetText(commands.value[id])
  copied.value = id
  setTimeout(() => (copied.value = ''), 1500)
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

      <div class="flex items-center justify-between gap-3">
        <p class="text-xs text-muted">Connect runs the command for you. If it fails, copy it and run it yourself.</p>
        <div class="flex shrink-0 rounded border border-line p-0.5 text-xs" role="radiogroup" aria-label="Where the agent runs">
          <button v-for="w in (['windows', 'wsl'] as const)" :key="w" role="radio" :aria-checked="where === w"
                  class="rounded-sm px-2 py-0.5" :class="where === w ? 'bg-hover font-semibold text-fg' : 'text-muted hover:text-fg'"
                  @click="where = w">{{ w === 'wsl' ? 'WSL' : 'Windows' }}</button>
        </div>
      </div>

      <div class="panel divide-y divide-line overflow-y-auto">
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
            {{ state[a.id]!.error!.includes('not found') ? `${a.name} is not installed on Windows. Run the command below where it is installed.` : state[a.id]!.error }}
          </p>
          <p v-else-if="state[a.id]?.ok" class="truncate font-mono text-[11px] text-muted">{{ state[a.id]!.ok }}</p>
          <div class="relative rounded border border-line bg-raised">
            <pre class="max-h-32 overflow-auto p-2.5 pr-9 font-mono text-[11px] whitespace-pre-wrap break-all select-text">{{ commands[a.id] }}</pre>
            <button class="absolute top-1.5 right-1.5 rounded p-1 text-muted transition hover:bg-hover hover:text-fg"
                    :aria-label="copied === a.id ? 'Copied' : `Copy the ${a.name} command`" @click="copy(a.id)">
              <Icon :name="copied === a.id ? 'check' : 'copy'" class="size-3.5!" />
            </button>
          </div>
        </div>
      </div>

      <p class="mt-auto text-xs text-muted">The connection runs as your user, without administrator rights.</p>
    </aside>
  </Transition>
</template>
