<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Clipboard } from '@wailsio/runtime'
import Icon from './Icon.vue'
import { Connect, Executable, Status } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/agentservice'
import type { AgentStatus, McpEvent } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { when } from '../lib/format'

const props = defineProps<{ open: boolean }>()
defineEmits<{ close: [] }>()

const exe = ref('reforgepulse.exe')
onMounted(() => Executable().then((p) => (exe.value = p)))

const agents = [
  { id: 'claude', name: 'Claude Code', how: 'Added with claude mcp add, for your user' },
  { id: 'codex', name: 'Codex', how: 'Added with codex mcp add' },
  { id: 'opencode', name: 'OpenCode', how: 'Added to ~/.config/opencode/opencode.json' },
] as const

// What each agent's own config says, and whether its server runs now. The
// agents' CLIs answer slowly, so this is read when the panel opens.
const status = ref<Record<string, AgentStatus>>({})
const checking = ref(false)
async function check() {
  checking.value = true
  try {
    status.value = Object.fromEntries(((await Status()) ?? []).map((s) => [s.agent, s]))
  } catch {
    status.value = {}
  } finally {
    checking.value = false
  }
}
watch(() => props.open, (open) => open && check(), { immediate: true })

type Tone = { dot: string; text: string; label: string }
function tone(id: string): Tone {
  const s = status.value[id]
  if (!s) return { dot: 'bg-line', text: 'text-muted', label: checking.value ? 'Checking…' : 'Unknown' }
  if (s.live) return { dot: 'bg-ok', text: 'text-ok', label: 'Connected' }
  if (s.registered) return { dot: 'bg-muted', text: 'text-muted', label: 'Set up, not running' }
  if (!s.installed) return { dot: 'bg-line', text: 'text-muted', label: 'Not installed' }
  return { dot: 'bg-bad', text: 'text-bad', label: 'Not set up' }
}

type State = { busy?: boolean; error?: string }
const state = ref<Record<string, State>>({})
async function connect(id: string) {
  state.value[id] = { busy: true }
  try {
    await Connect(id)
    state.value[id] = {}
    await check()
  } catch (e) {
    state.value[id] = { error: String(e) }
  }
}

// What the server did, one line per event.
function line(e: McpEvent): { text: string; bad: boolean } {
  if (e.event === 'start') return { text: `Started, process ${e.pid}`, bad: false }
  if (e.event === 'end') return e.error ? { text: `Ended with an error: ${e.error}`, bad: true } : { text: 'Ended', bad: false }
  const took = e.millis ? `, ${e.millis < 1000 ? `${e.millis} ms` : `${(e.millis / 1000).toFixed(1)} s`}` : ''
  return e.error ? { text: `${e.tool} failed: ${e.error}`, bad: true } : { text: `Called ${e.tool}${took}`, bad: false }
}

// Activity and the command stay folded until asked for.
const showLog = ref(new Set<string>())
const showCmd = ref(new Set<string>())
function toggled(set: Set<string>, id: string) {
  const next = new Set(set)
  next.has(id) ? next.delete(id) : next.add(id)
  return next
}
const toggleLog = (id: string) => (showLog.value = toggled(showLog.value, id))
const toggleCmd = (id: string) => (showCmd.value = toggled(showCmd.value, id))

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
           class="fixed inset-y-0 right-0 z-50 flex w-[460px] max-w-full flex-col gap-5 border-l border-line bg-panel p-6 shadow-2xl">
      <header class="flex items-start justify-between gap-4">
        <div>
          <h2 id="agents-title" class="text-[18px] font-semibold">Connect an AI agent</h2>
          <p class="mt-1 text-muted">The agent can run the scan and read what ReforgePulse sees. It cannot change anything; changes are made here, by you.</p>
        </div>
        <button class="grid size-7 place-items-center rounded text-muted transition hover:bg-hover hover:text-fg" aria-label="Close" @click="$emit('close')"><Icon name="close" /></button>
      </header>

      <div class="flex items-center justify-between gap-3">
        <p class="text-xs text-muted">Green means the agent's ReforgePulse server is running now. Agents start it themselves when they need it.</p>
        <button class="btn h-7 shrink-0 text-xs" :disabled="checking" @click="check">
          <Icon name="refresh" class="size-3.5!" :class="checking && 'animate-spin'" />Check again
        </button>
      </div>

      <div class="panel min-h-0 divide-y divide-line overflow-y-auto [scrollbar-gutter:stable]">
        <div v-for="a in agents" :key="a.id" class="flex flex-col gap-2 px-4 py-3">
          <div class="flex items-center gap-3">
            <div class="min-w-0 flex-1">
              <p class="font-medium">{{ a.name }}</p>
              <p class="flex items-center gap-1.5 text-xs" :class="tone(a.id).text" role="status">
                <span class="size-2 shrink-0 rounded-full" :class="tone(a.id).dot" />{{ tone(a.id).label }}
              </p>
            </div>
            <button :class="status[a.id]?.registered ? 'btn' : 'btn-primary'" :disabled="state[a.id]?.busy" @click="connect(a.id)">
              {{ state[a.id]?.busy ? 'Connecting…' : status[a.id]?.registered ? 'Connect again' : 'Connect' }}
            </button>
          </div>
          <p v-if="status[a.id]?.note && !status[a.id]?.live" class="text-xs text-muted">{{ status[a.id]!.note }}</p>
          <p v-if="state[a.id]?.error" class="text-xs text-bad select-text">
            {{ state[a.id]!.error!.includes('not found') ? `${a.name} is not installed on Windows. Show the command and run it where it is installed.` : state[a.id]!.error }}
          </p>

          <div class="flex flex-wrap gap-x-4 text-xs">
            <button class="flex items-center gap-1 text-muted hover:text-fg" :aria-expanded="showLog.has(a.id)" @click="toggleLog(a.id)">
              <Icon name="chevron" class="size-3! transition-transform" :class="showLog.has(a.id) && 'rotate-90'" />
              Activity<template v-if="status[a.id]?.events?.length"> ({{ status[a.id]!.events!.length }})</template>
            </button>
            <button class="flex items-center gap-1 text-muted hover:text-fg" :aria-expanded="showCmd.has(a.id)" @click="toggleCmd(a.id)">
              <Icon name="chevron" class="size-3! transition-transform" :class="showCmd.has(a.id) && 'rotate-90'" />Show the command
            </button>
          </div>

          <ul v-if="showLog.has(a.id)" class="max-h-48 overflow-y-auto rounded border border-line bg-raised py-1 text-xs">
            <li v-if="!status[a.id]?.events?.length" class="px-2.5 py-1 text-muted">Nothing yet. It shows here once {{ a.name }} uses ReforgePulse.</li>
            <li v-for="(e, i) in status[a.id]?.events ?? []" :key="i" class="flex gap-3 px-2.5 py-1">
              <span class="w-28 shrink-0 text-muted tabular-nums">{{ when(e.at) }}</span>
              <span class="min-w-0 break-words" :class="line(e).bad && 'text-bad'">{{ line(e).text }}</span>
            </li>
          </ul>

          <div v-if="showCmd.has(a.id)" class="flex flex-col gap-2">
            <div class="flex items-center justify-between gap-3">
              <p class="text-xs text-muted">{{ a.how }}. Run it yourself if Connect fails.</p>
              <div class="flex shrink-0 rounded border border-line p-0.5 text-xs" role="radiogroup" aria-label="Where the agent runs">
                <button v-for="w in (['windows', 'wsl'] as const)" :key="w" role="radio" :aria-checked="where === w"
                        class="rounded-sm px-2 py-0.5" :class="where === w ? 'bg-hover text-fg' : 'text-muted hover:text-fg'"
                        @click="where = w">{{ w === 'wsl' ? 'WSL' : 'Windows' }}</button>
              </div>
            </div>
            <div class="relative rounded border border-line bg-raised">
              <pre class="max-h-32 overflow-auto p-2.5 pr-9 font-mono text-[12px] whitespace-pre-wrap break-all select-text">{{ commands[a.id] }}</pre>
              <button class="absolute top-1.5 right-1.5 rounded p-1 text-muted transition hover:bg-hover hover:text-fg"
                      :aria-label="copied === a.id ? 'Copied' : `Copy the ${a.name} command`" @click="copy(a.id)">
                <Icon :name="copied === a.id ? 'check' : 'copy'" class="size-3.5!" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <p class="mt-auto text-xs text-muted">The connection runs as your user, without administrator rights.</p>
    </aside>
  </Transition>
</template>
