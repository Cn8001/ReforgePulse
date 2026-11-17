<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'
import Markdown from './Markdown.vue'
import { Info } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/aboutservice'
import { formatBytes } from '../lib/format'
import {
  askRestart, cancelRestart, closePrompt, download, prompting, release, restart, restarting, update, updateError,
} from '../lib/update'

// Asks once a start-up check finds a newer version. Nothing is downloaded or
// restarted without the user choosing it here or in Settings.
const current = ref('')
onMounted(() => Info().then((a) => (current.value = a.version)))
</script>

<template>
  <Transition enter-from-class="opacity-0" leave-to-class="opacity-0"
              enter-active-class="transition-opacity duration-150" leave-active-class="transition-opacity duration-150">
    <div v-if="prompting && release" class="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6">
      <section role="dialog" aria-modal="true" aria-labelledby="update-title" class="panel flex max-h-full w-[520px] max-w-full flex-col shadow-2xl">
        <header class="border-b border-line px-5 py-4">
          <h2 id="update-title" class="text-[16px] font-semibold">ReforgePulse {{ release.version }} is available</h2>
          <p class="mt-1 text-muted">You have {{ current }}. Download it now? It installs when you restart.</p>
        </header>
        <div class="flex min-h-0 flex-col gap-3 overflow-y-auto px-5 py-4">
          <div v-if="release.notes" class="max-h-56 overflow-y-auto rounded border border-line bg-raised p-3 text-xs text-muted">
            <Markdown :source="release.notes" />
          </div>
          <p class="text-xs text-muted">
            The file is checked against the release's SHA256SUMS before it replaces this one, and the update is recorded in the change log.
          </p>
          <p v-if="updateError" class="text-xs text-bad select-text">{{ updateError }}</p>
        </div>
        <footer class="flex justify-end gap-2 border-t border-line px-5 py-3">
          <button class="btn" :disabled="update === 'downloading'" @click="closePrompt">{{ update === 'ready' ? 'Later' : 'Not now' }}</button>
          <button v-if="update === 'ready'" class="btn-primary" @click="askRestart">Restart now</button>
          <button v-else class="btn-primary" :disabled="update === 'downloading'" @click="download">
            {{ update === 'downloading' ? 'Downloading…' : `Download${release.size ? ` (${formatBytes(release.size)})` : ''}` }}
          </button>
        </footer>
      </section>
    </div>
  </Transition>
  <ConfirmDialog :open="restarting" title="Restart ReforgePulse now?" confirm="Restart"
                 warning="Anything still running, such as a scan or a core test, stops. Are you sure?"
                 @yes="restart" @no="cancelRestart">
    ReforgePulse closes, puts version {{ release?.version }} in place and opens again.
  </ConfirmDialog>
</template>
