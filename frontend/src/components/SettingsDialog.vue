<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Clipboard } from '@wailsio/runtime'
import Icon from './Icon.vue'
import { theme } from '../lib/theme'
import { Info } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/aboutservice'
import type { About } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/models'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; agent: [] }>()

const about = ref<About | null>(null)
onMounted(() => Info().then((a) => (about.value = a)))
const showTerms = ref(false)
</script>

<template>
  <Transition enter-from-class="opacity-0" leave-to-class="opacity-0"
              enter-active-class="transition-opacity duration-150" leave-active-class="transition-opacity duration-150">
    <div v-if="open" class="fixed inset-0 z-40 grid place-items-center bg-black/50 p-6" @click.self="emit('close')">
      <section role="dialog" aria-modal="true" aria-labelledby="settings-title" class="panel flex max-h-full w-[560px] max-w-full flex-col shadow-2xl">
        <header class="flex items-center justify-between border-b border-line px-5 py-3">
          <h2 id="settings-title" class="text-[15px] font-semibold">Settings</h2>
          <button class="grid size-7 place-items-center rounded text-muted transition hover:bg-hover hover:text-fg" aria-label="Close" @click="emit('close')"><Icon name="close" /></button>
        </header>

        <div class="flex flex-col overflow-y-auto">
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

          <div class="flex flex-col gap-2 px-5 py-4">
            <h3 class="font-medium">About</h3>
            <dl class="grid grid-cols-[7rem_1fr] gap-y-1.5 text-[13px]">
              <dt class="text-muted">Version</dt><dd class="tabular-nums">{{ about?.version ?? '–' }}</dd>
              <dt class="text-muted">Data folder</dt>
              <dd class="flex min-w-0 items-center gap-2">
                <span class="truncate font-mono text-xs select-text" :title="about?.dataDir">{{ about?.dataDir ?? '–' }}</span>
                <button v-if="about" class="text-muted transition hover:text-fg" aria-label="Copy the folder path" @click="Clipboard.SetText(about.dataDir)">
                  <Icon name="copy" class="size-3.5!" />
                </button>
              </dd>
              <dt class="text-muted">Licence</dt>
              <dd>Free to use, provided as is, without warranty.
                <button class="text-accent hover:underline" @click="showTerms = !showTerms">{{ showTerms ? 'Hide' : 'Read' }} the terms</button>
              </dd>
            </dl>
            <pre v-if="showTerms" class="mt-2 max-h-64 overflow-y-auto rounded border border-line bg-raised p-3 font-sans text-xs leading-relaxed whitespace-pre-wrap text-muted select-text">{{ about?.terms }}</pre>
          </div>
        </div>
      </section>
    </div>
  </Transition>
</template>
