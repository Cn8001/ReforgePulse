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
      <section class="flex max-h-full w-[560px] max-w-full flex-col border border-line bg-panel shadow-2xl">
        <header class="flex items-center justify-between border-b border-line px-5 py-3">
          <div class="flex items-center gap-2"><Icon name="settings" /><span class="font-semibold">Settings</span></div>
          <button class="p-1.5 text-muted transition hover:bg-hover hover:text-fg" @click="emit('close')"><Icon name="close" /></button>
        </header>

        <div class="flex flex-col overflow-y-auto">
          <div class="flex items-center justify-between gap-4 border-b border-line px-5 py-4">
            <div>
              <p class="caption">appearance</p>
              <p class="mt-1 text-muted">Dark is the default. The accent follows Windows.</p>
            </div>
            <div class="flex border border-line font-mono text-xs">
              <button v-for="t in ['dark', 'light'] as const" :key="t" class="flex items-center gap-1.5 px-3 py-1.5 transition"
                      :class="theme === t ? 'bg-accent text-white' : 'text-muted hover:bg-hover hover:text-fg'" @click="theme = t">
                <Icon :name="t === 'dark' ? 'moon' : 'sun'" class="size-3.5!" />{{ t }}
              </button>
            </div>
          </div>

          <div class="flex items-center justify-between gap-4 border-b border-line px-5 py-4">
            <div>
              <p class="caption">ai agents</p>
              <p class="mt-1 text-muted">Claude Code, Codex or OpenCode can read what ReforgePulse sees.</p>
            </div>
            <button class="flex shrink-0 items-center gap-2 border border-line px-3 py-1.5 font-mono text-xs transition hover:bg-hover"
                    @click="emit('agent')"><Icon name="plug" class="size-3.5!" />connect</button>
          </div>

          <div class="flex flex-col gap-2 px-5 py-4">
            <p class="caption">about</p>
            <dl class="grid grid-cols-[6rem_1fr] gap-y-1.5 font-mono text-xs">
              <dt class="text-muted">version</dt><dd>{{ about?.version ?? '--' }}</dd>
              <dt class="text-muted">data</dt>
              <dd class="flex min-w-0 items-center gap-2">
                <span class="truncate select-text" :title="about?.dataDir">{{ about?.dataDir ?? '--' }}</span>
                <button v-if="about" class="text-muted transition hover:text-fg" title="Copy" @click="Clipboard.SetText(about.dataDir)">
                  <Icon name="copy" class="size-3.5!" />
                </button>
              </dd>
              <dt class="text-muted">licence</dt>
              <dd>free to use, provided as is, without warranty ·
                <button class="text-accent hover:underline" @click="showTerms = !showTerms">{{ showTerms ? 'hide' : 'read' }} terms</button>
              </dd>
            </dl>
            <pre v-if="showTerms" class="mt-2 max-h-64 overflow-y-auto border border-line bg-raised p-3 font-mono text-[11px] leading-relaxed whitespace-pre-wrap text-muted select-text">{{ about?.terms }}</pre>
          </div>
        </div>
      </section>
    </div>
  </Transition>
</template>
