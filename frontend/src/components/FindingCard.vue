<script setup lang="ts">
import { ref } from 'vue'
import Icon from './Icon.vue'
import type { Finding } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { severityTone, setResolved } from '../lib/scan'
import { when } from '../lib/format'
import { estimate } from '../lib/scan'

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
        <!-- The whole heading opens the details: a small "Evidence" link was
             too easy to miss. -->
        <button class="group -mx-2 -my-1 block w-[calc(100%+1rem)] rounded px-2 py-1 text-left hover:bg-hover" :aria-expanded="open" @click="open = !open">
          <h3 class="flex items-start gap-1.5 text-[15px] font-semibold">
            <Icon name="chevron" class="mt-1 size-3.5! shrink-0 text-muted transition-transform group-hover:text-fg" :class="open && 'rotate-90'" />
            {{ f.title }}
          </h3>
          <p class="mt-1 pl-5 text-muted">{{ f.summary }}</p>
        </button>
        <p class="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <span class="chip capitalize" :style="{ color: severityTone[f.severity], borderColor: severityTone[f.severity] }">{{ f.severity }} severity</span>
          <span class="chip border-line text-muted">{{ sureness[f.confidence] ?? f.confidence }}</span>
          <span v-if="f.status === 'returned'" class="chip border-warn text-warn"
                :title="`Marked resolved ${when(f.resolvedAt!)}, seen again ${when(f.latest!)}`">Came back</span>
          <span v-else-if="f.status === 'resolved'" class="chip border-line text-muted">Resolved {{ when(f.resolvedAt!) }}</span>
          <span v-if="f.new && f.status !== 'resolved'" class="chip border-fg/40 text-fg">New since last scan</span>
        </p>
        <p v-if="f.status === 'returned'" class="mt-2 text-xs text-warn">
          You marked this resolved on {{ when(f.resolvedAt!) }}, but it happened again on {{ when(f.latest!) }}.
        </p>
        <p v-if="error" class="mt-2 text-xs text-bad">{{ error }}</p>
        <div v-if="open" class="mt-3 flex flex-col gap-4 rounded border border-line bg-raised p-4 text-[14px]">
          <section v-if="f.evidence?.length">
            <h4 class="label mb-1.5">What Windows recorded</h4>
            <ul class="flex flex-col gap-1 font-mono text-xs select-text">
              <li v-for="(e, i) in f.evidence" :key="i">{{ e }}</li>
            </ul>
          </section>
          <section v-if="f.alternatives">
            <h4 class="label mb-1">Other explanations the evidence allows</h4>
            <p>{{ f.alternatives }}</p>
          </section>
          <dl class="grid grid-cols-[9rem_1fr] gap-y-1 text-xs">
            <template v-if="f.latest"><dt class="text-muted">Last seen</dt><dd>{{ when(f.latest) }}</dd></template>
            <template v-if="f.resolvedAt"><dt class="text-muted">Marked resolved</dt><dd>{{ when(f.resolvedAt) }}</dd></template>
            <dt class="text-muted">Key</dt><dd class="font-mono select-text">{{ f.key }}</dd>
          </dl>
          <p class="text-xs text-muted">{{ estimate }}</p>
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
