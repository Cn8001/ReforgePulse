<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import type { Finding } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { severityTone, setResolved } from '../lib/scan'
import { when } from '../lib/format'

const props = defineProps<{ f: Finding }>()
const emit = defineEmits<{ act: [kind: string, arg?: string] }>()
const open = ref(false)
const busy = ref(false)
const error = ref('')

async function mark(resolved: boolean) {
  busy.value = true
  error.value = ''
  try {
    await setResolved(props.f, resolved)
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
  }
}

const sureness: Record<string, string> = { confirmed: 'Confirmed', likely: 'Likely', possible: 'Possible' }
</script>

<template>
  <article class="panel" :class="f.status === 'resolved' && 'opacity-70'">
    <div class="flex items-start gap-4 px-5 py-4">
      <span class="mt-1.5 size-2.5 shrink-0 rounded-full"
            :style="{ background: f.status === 'resolved' ? 'var(--muted)' : severityTone[f.severity] }" />
      <div class="min-w-0 flex-1">
        <h3 class="text-[15px] font-semibold">{{ f.title }}</h3>
        <p class="mt-1 text-muted">{{ f.summary }}</p>
        <p class="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <span class="chip capitalize" :style="{ color: severityTone[f.severity], borderColor: severityTone[f.severity] }">{{ f.severity }} severity</span>
          <span class="chip border-line text-muted">{{ sureness[f.confidence] ?? f.confidence }}</span>
          <span v-if="f.status === 'returned'" class="chip border-warn text-warn"
                :title="`Marked resolved ${when(f.resolvedAt!)}, seen again ${when(f.latest!)}`">Came back</span>
          <span v-else-if="f.status === 'resolved'" class="chip border-line text-muted">Resolved {{ when(f.resolvedAt!) }}</span>
          <span v-if="f.new && f.status !== 'resolved'" class="chip border-fg/40 text-fg">New since last scan</span>
          <button v-if="f.evidence?.length || f.alternatives" class="flex items-center gap-1 text-muted hover:text-fg" :aria-expanded="open" @click="open = !open">
            <Icon name="chevron" class="size-3! transition-transform" :class="open && 'rotate-90'" />Evidence
          </button>
        </p>
        <p v-if="f.status === 'returned'" class="mt-2 text-xs text-warn">
          You marked this resolved on {{ when(f.resolvedAt!) }}, but it happened again on {{ when(f.latest!) }}.
        </p>
        <p v-if="error" class="mt-2 text-xs text-bad">{{ error }}</p>
        <div v-if="open" class="mt-3 flex flex-col gap-1.5 border-l-2 border-line pl-3 text-xs">
          <p v-for="(e, i) in f.evidence" :key="i" class="text-muted select-text">{{ e }}</p>
          <p v-if="f.alternatives" class="mt-1"><span class="text-muted">Other explanations:</span> {{ f.alternatives }}</p>
        </div>
      </div>
      <div class="flex shrink-0 flex-col items-end gap-2">
        <button v-if="f.action && f.status !== 'resolved'" class="btn" @click="emit('act', f.action.kind, f.action.arg)">{{ f.action.label }}</button>
        <button v-if="f.status === 'resolved'" class="btn" :disabled="busy" @click="mark(false)">Reopen</button>
        <button v-else class="btn" :disabled="busy"
                title="Hide it until new evidence turns up. Nothing on the PC changes." @click="mark(true)">
          <Icon name="check" class="size-3.5!" />Mark resolved
        </button>
      </div>
    </div>
  </article>
</template>
