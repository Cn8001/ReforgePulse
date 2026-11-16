<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { Cancel, List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/taskservice'
import { State, type Status } from '../../bindings/github.com/cn8001/reforgepulse/internal/tasks/models'

defineProps<{ elevated: boolean | null }>()

const running = ref(new Map<string, Status>())
const last = ref<Status | null>(null) // the latest finished task, shown for a moment
let hide: ReturnType<typeof setTimeout> | undefined

function onTask(s: Status) {
  const live = s.state === State.Queued || s.state === State.Running
  const next = new Map(running.value)
  live ? next.set(s.id, s) : next.delete(s.id)
  running.value = next
  if (!live) {
    last.value = s
    clearTimeout(hide)
    hide = setTimeout(() => (last.value = null), 2500)
  }
}

let off: (() => void) | undefined
onMounted(async () => {
  off = Events.On('task', (e) => onTask(e.data))
  for (const s of (await List()) ?? []) onTask(s)
})
onBeforeUnmount(() => off?.())

const current = computed(() => [...running.value.values()].at(-1) ?? null)
const finishedText = computed(() => {
  const s = last.value
  if (!s) return ''
  if (s.state === State.Failed) return `${s.title} failed: ${s.error}`
  if (s.state === State.Cancelled) return `${s.title} cancelled`
  return `${s.title} done`
})
</script>

<template>
  <footer class="flex h-7 shrink-0 items-center gap-4 border-t border-line bg-panel px-3 font-mono text-[11px] text-muted">
    <Transition name="page" mode="out-in">
      <div v-if="current" key="busy" class="flex min-w-0 items-center gap-2">
        <span class="text-accent">[RUN]</span>
        <span class="truncate text-fg">{{ current.title }}<span v-if="current.note" class="text-muted"> :: {{ current.note }}</span></span>
        <span v-if="running.size > 1">(+{{ running.size - 1 }} queued)</span>
        <span class="relative h-1.5 w-28 overflow-hidden bg-hover">
          <span v-if="current.progress >= 0" class="absolute inset-y-0 left-0 bg-accent transition-[width] duration-300"
                :style="{ width: `${current.progress * 100}%` }" />
          <span v-else class="absolute inset-y-0 w-1/3 animate-[slide_1s_linear_infinite] bg-accent" />
        </span>
        <button class="px-1 text-fg transition hover:bg-hover" @click="Cancel(current.id)">^C cancel</button>
      </div>
      <div v-else key="idle" class="flex min-w-0 items-center gap-2">
        <span :class="last?.state === State.Failed ? 'text-bad' : 'text-ok'">{{ last?.state === State.Failed ? '[ERR]' : '[OK]' }}</span>
        <span class="truncate">{{ finishedText || 'ready' }}</span>
      </div>
    </Transition>

    <span class="ml-auto shrink-0" :class="elevated ? 'text-ok' : elevated === false ? 'text-warn' : ''">
      {{ elevated === null ? '' : elevated ? 'admin' : 'limited' }}
    </span>
    <span class="shrink-0">offline · local only</span>
  </footer>
</template>

<style scoped>
@keyframes slide {
  from { left: -33%; }
  to { left: 100%; }
}
</style>
