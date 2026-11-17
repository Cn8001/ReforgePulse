<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Application } from '@wailsio/runtime'
import logo from '../assets/logo.svg'
import { Accept, Info } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/aboutservice'
import type { About } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/models'

// Shown instead of the app until the terms and licence are accepted, and
// again whenever either text changes.
const emit = defineEmits<{ accepted: [] }>()
const about = ref<About | null>(null)
const doc = ref<'terms' | 'license'>('terms')
const agreed = ref(false)
const busy = ref(false)
const error = ref('')
onMounted(() => Info().then((a) => (about.value = a)))

async function accept() {
  busy.value = true
  error.value = ''
  try {
    await Accept()
    emit('accepted')
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="flex min-h-0 flex-1 items-center justify-center overflow-y-auto p-6">
    <div class="panel flex max-h-full w-[640px] max-w-full flex-col shadow-2xl">
      <header class="flex items-center gap-4 border-b border-line px-6 py-5">
        <img :src="logo" alt="" class="size-10" />
        <div>
          <h1 class="heading">Before you start</h1>
          <p class="text-muted">Read and accept the terms to use ReforgePulse.</p>
        </div>
      </header>

      <div class="flex min-h-0 flex-col gap-4 px-6 py-5">
        <p class="notice border-warn/40 bg-warn/10">
          ReforgePulse runs as Administrator. It can delete files, uninstall programs and change Windows settings.
          It always shows what it will do and asks first, but you use it at your own risk: it comes without warranty,
          and deleted files may not come back. Back up what matters to you.
        </p>

        <div class="flex min-h-0 flex-col">
          <nav class="tabs" role="tablist">
            <button role="tab" class="tab" :class="doc === 'terms' && 'tab-on'" :aria-selected="doc === 'terms'" @click="doc = 'terms'">Terms of use</button>
            <button role="tab" class="tab" :class="doc === 'license' && 'tab-on'" :aria-selected="doc === 'license'" @click="doc = 'license'">MIT License</button>
          </nav>
          <pre class="h-64 overflow-y-auto border-x border-b border-line bg-raised p-4 font-sans text-xs leading-relaxed whitespace-pre-wrap text-muted select-text">{{ doc === 'terms' ? about?.terms : about?.license }}</pre>
        </div>

        <label class="flex cursor-pointer items-start gap-3">
          <input v-model="agreed" type="checkbox" class="mt-0.5 size-4 accent-[var(--accent)]" />
          <span>I have read the terms and the licence, and I use ReforgePulse at my own risk.</span>
        </label>
        <p v-if="error" class="text-xs text-bad">{{ error }}</p>
      </div>

      <footer class="flex justify-end gap-2 border-t border-line px-6 py-4">
        <button class="btn" @click="Application.Quit()">Quit</button>
        <button class="btn-primary" :disabled="!agreed || busy" @click="accept">Accept and continue</button>
      </footer>
    </div>
  </section>
</template>
