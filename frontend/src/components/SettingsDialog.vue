<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Browser, Clipboard } from '@wailsio/runtime'
import logo from '../assets/logo.svg'
import Icon from './Icon.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import { theme } from '../lib/theme'
import { Info } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/aboutservice'
import type { About } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/models'
import { Check, Download, Restart } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/updateservice'
import type { Release } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { formatBytes } from '../lib/format'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; agent: [] }>()

const about = ref<About | null>(null)
onMounted(() => Info().then((a) => (about.value = a)))
const tab = ref<'general' | 'updates' | 'about'>('general')
const tabs = [
  { id: 'general', label: 'General' },
  { id: 'updates', label: 'Updates' },
  { id: 'about', label: 'About' },
] as const
// One long text open at a time: the licence or the terms.
const reading = ref<'license' | 'terms' | null>(null)
const read = (what: 'license' | 'terms') => (reading.value = reading.value === what ? null : what)

// Nothing is fetched until the user presses the button.
type Update = 'idle' | 'checking' | 'latest' | 'available' | 'downloading' | 'ready'
const update = ref<Update>('idle')
const release = ref<Release | null>(null)
const updateError = ref('')
const showNotes = ref(false)
const confirmRestart = ref(false)

async function check() {
  update.value = 'checking'
  updateError.value = ''
  try {
    release.value = await Check()
    update.value = release.value ? 'available' : 'latest'
  } catch (e) {
    updateError.value = String(e)
    update.value = 'idle'
  }
}
async function download() {
  update.value = 'downloading'
  updateError.value = ''
  try {
    await Download(release.value!.version)
    update.value = 'ready'
  } catch (e) {
    updateError.value = String(e)
    update.value = 'available'
  }
}
async function restart() {
  try {
    await Restart()
  } catch (e) {
    updateError.value = String(e)
    confirmRestart.value = false
  }
}
</script>

<template>
  <Transition enter-from-class="opacity-0" leave-to-class="opacity-0"
              enter-active-class="transition-opacity duration-150" leave-active-class="transition-opacity duration-150">
    <div v-if="open" class="fixed inset-0 z-40 grid place-items-center bg-black/50 p-6" @click.self="emit('close')">
      <section role="dialog" aria-modal="true" aria-labelledby="settings-title" class="panel flex max-h-full min-h-[380px] w-[560px] max-w-full flex-col shadow-2xl">
        <header class="flex items-center justify-between border-b border-line px-5 py-3">
          <h2 id="settings-title" class="text-[15px] font-semibold">Settings</h2>
          <button class="grid size-7 place-items-center rounded text-muted transition hover:bg-hover hover:text-fg" aria-label="Close" @click="emit('close')"><Icon name="close" /></button>
        </header>
        <nav class="tabs px-3" role="tablist">
          <button v-for="t in tabs" :key="t.id" role="tab" class="tab" :class="tab === t.id && 'tab-on'" :aria-selected="tab === t.id" @click="tab = t.id">
            {{ t.label }}
          </button>
        </nav>

        <div v-if="tab === 'general'" class="flex flex-col overflow-y-auto">
          <div class="flex items-center justify-between gap-4 border-b border-line px-5 py-4">
            <div>
              <h3 class="font-medium">Theme</h3>
              <p class="text-[13px] text-muted">Dark by default. The accent colour follows Windows.</p>
            </div>
            <div class="flex rounded border border-line p-0.5" role="radiogroup" aria-label="Theme">
              <button v-for="t in ['dark', 'light'] as const" :key="t" role="radio" :aria-checked="theme === t"
                      class="flex items-center gap-1.5 rounded-[3px] px-3 py-1 text-[13px] capitalize transition"
                      :class="theme === t ? 'bg-accent text-white' : 'text-muted hover:text-fg'" @click="theme = t">
                <Icon :name="t === 'dark' ? 'moon' : 'sun'" class="size-3.5!" />{{ t }}
              </button>
            </div>
          </div>

          <div class="flex items-center justify-between gap-4 border-b border-line px-5 py-4">
            <div>
              <h3 class="font-medium">AI agents</h3>
              <p class="text-[13px] text-muted">Let Claude Code, Codex or OpenCode read what ReforgePulse sees.</p>
            </div>
            <button class="btn shrink-0" @click="emit('agent')"><Icon name="plug" class="size-4!" />Connect</button>
          </div>
        </div>

        <div v-else-if="tab === 'updates'" class="flex flex-col overflow-y-auto">
          <div class="flex flex-col gap-3 px-5 py-4">
            <div class="flex items-center justify-between gap-4">
              <div>
                <h3 class="font-medium">Updates</h3>
                <p class="text-[13px] text-muted">
                  <template v-if="update === 'latest'">Version {{ about?.version }} is the latest.</template>
                  <template v-else-if="update === 'available' || update === 'downloading'">Version {{ release?.version }} is available. You have {{ about?.version }}.</template>
                  <template v-else-if="update === 'ready'">Version {{ release?.version }} is downloaded and verified. Restart to finish.</template>
                  <template v-else>Looks at GitHub Releases only when you press the button.</template>
                </p>
              </div>
              <button v-if="update === 'ready'" class="btn-primary shrink-0" @click="confirmRestart = true">Restart now</button>
              <button v-else-if="update === 'available' || update === 'downloading'" class="btn-primary shrink-0"
                      :disabled="update === 'downloading'" @click="download">
                {{ update === 'downloading' ? 'Downloading…' : `Download${release?.size ? ` (${formatBytes(release.size)})` : ''}` }}
              </button>
              <button v-else class="btn shrink-0" :disabled="update === 'checking'" @click="check">
                <Icon name="refresh" class="size-3.5!" />{{ update === 'checking' ? 'Checking…' : 'Check for updates' }}
              </button>
            </div>
            <p v-if="updateError" class="text-xs text-bad select-text">{{ updateError }}</p>
            <template v-if="release && update !== 'latest'">
              <p class="text-xs text-muted">
                The file is checked against the release's SHA256SUMS before it replaces this one. The update is recorded in the change log.
                <button class="text-accent hover:underline" @click="Browser.OpenURL(release.url)">See it on GitHub</button>
              </p>
              <button v-if="release.notes" class="flex items-center gap-1 self-start text-xs text-muted hover:text-fg" :aria-expanded="showNotes" @click="showNotes = !showNotes">
                <Icon name="chevron" class="size-3! transition-transform" :class="showNotes && 'rotate-90'" />What is new
              </button>
              <pre v-if="showNotes" class="max-h-48 overflow-y-auto rounded border border-line bg-raised p-3 font-sans text-xs leading-relaxed whitespace-pre-wrap text-muted select-text">{{ release.notes }}</pre>
            </template>
          </div>

        </div>

        <div v-else class="flex flex-col overflow-y-auto">
          <div class="flex items-center gap-4 border-b border-line px-5 py-5">
            <img :src="logo" alt="" class="size-12" />
            <div class="min-w-0 flex-1">
              <h3 class="text-[17px] font-semibold">ReforgePulse <span class="text-[13px] font-normal text-muted tabular-nums">{{ about?.version }}</span></h3>
              <p class="text-[13px] text-muted">Finds why a Windows PC crashes or slows down, and cleans up what it no longer needs.</p>
            </div>
          </div>

          <dl class="grid grid-cols-[7rem_1fr] gap-y-2 border-b border-line px-5 py-4 text-[13px]">
            <dt class="text-muted">Licence</dt>
            <dd>MIT License. Free and open source, provided as is, without warranty.</dd>
            <dt class="text-muted">Source</dt>
            <dd class="flex flex-wrap gap-x-3">
              <button v-if="about" class="text-accent hover:underline" @click="Browser.OpenURL(about.source)">GitHub</button>
              <button v-if="about" class="text-accent hover:underline" @click="Browser.OpenURL(`${about.source}/issues`)">Report a problem</button>
            </dd>
            <dt class="text-muted">Data folder</dt>
            <dd class="flex min-w-0 items-center gap-2">
              <span class="truncate font-mono text-xs select-text" :title="about?.dataDir">{{ about?.dataDir ?? '–' }}</span>
              <button v-if="about" class="text-muted transition hover:text-fg" aria-label="Copy the folder path" @click="Clipboard.SetText(about.dataDir)">
                <Icon name="copy" class="size-3.5!" />
              </button>
            </dd>
            <dt class="text-muted">Built with</dt>
            <dd>Wails, Vue and Tailwind CSS</dd>
          </dl>

          <div class="flex flex-col gap-3 px-5 py-4">
            <div class="flex gap-2">
              <button class="btn" :aria-expanded="reading === 'license'" @click="read('license')">{{ reading === 'license' ? 'Hide' : 'Read' }} the licence</button>
              <button class="btn" :aria-expanded="reading === 'terms'" @click="read('terms')">{{ reading === 'terms' ? 'Hide' : 'Read' }} the terms</button>
            </div>
            <pre v-if="reading" class="max-h-64 overflow-y-auto rounded border border-line bg-raised p-3 font-sans text-xs leading-relaxed whitespace-pre-wrap text-muted select-text">{{ reading === 'license' ? about?.license : about?.terms }}</pre>
          </div>
        </div>
      </section>
    </div>
  </Transition>
  <ConfirmDialog :open="confirmRestart" title="Restart ReforgePulse now?" confirm="Restart"
                 warning="Anything still running, such as a scan or a core test, stops. Are you sure?"
                 @yes="restart" @no="confirmRestart = false">
    ReforgePulse closes, puts version {{ release?.version }} in place and opens again.
  </ConfirmDialog>
</template>
