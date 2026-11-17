<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { Cancel, List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/taskservice'
import { State, type Status } from '../../bindings/github.com/cn8001/reforgepulse/internal/tasks/models'
import Icon from './Icon.vue'

defineProps<{ elevated: boolean | null }>()

const running = ref(new Map<string, Status>())
// The first List can answer after a task's final event; those IDs must not
// come back as running.
const ended = new Set<string>()
const last = ref<Status | null>(null) // the latest finished task, shown for a moment
let hide: ReturnType<typeof setTimeout> | undefined
// Finished tasks of this session for the details panel, newest first; a
// handful is enough to see what just failed.
const done = ref<{ s: Status; at: number }[]>([])
const open = ref(false)
const now = ref(Date.now())
let tick: ReturnType<typeof setInterval> | undefined

function onTask(s: Status) {
  const live = s.state === State.Queued || s.state === State.Running
  if (live && ended.has(s.id)) return
  if (!live) ended.add(s.id)
  const next = new Map(running.value)
  live ? next.set(s.id, s) : next.delete(s.id)
  running.value = next
  if (!live) {
    done.value = [{ s, at: Date.now() }, ...done.value].slice(0, 8)
    last.value = s
    clearTimeout(hide)
    hide = setTimeout(() => (last.value = null), 2500)
  }
}

let off: (() => void) | undefined
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') open.value = false
}
onMounted(async () => {
  off = Events.On('task', (e) => onTask(e.data))
  tick = setInterval(() => (now.value = Date.now()), 1000)
  window.addEventListener('keydown', onKey)
  for (const s of (await List()) ?? []) onTask(s)
})
onBeforeUnmount(() => {
  off?.()
  clearInterval(tick)
  window.removeEventListener('keydown', onKey)
})

function elapsed(from: string, to = now.value): string {
  const s = Math.max(0, Math.round((to - new Date(from).getTime()) / 1000))
  return s < 60 ? `${s} s` : `${Math.floor(s / 60)} min ${s % 60} s`
}

const outcome: Record<string, string> = {
  [State.Done]: 'Done', [State.Failed]: 'Failed', [State.Cancelled]: 'Cancelled',
}

const current = computed(() => [...running.value.values()].at(-1) ?? null)
const finishedText = computed(() => {
  const s = last.value
  if (!s) return ''
  if (s.state === State.Failed) return `${s.title} failed: ${s.error}`
  if (s.state === State.Cancelled) return `${s.title} cancelled`
  return `${s.title}: done`
})
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-30" @click="open = false" />
  <Transition name="page">
    <div v-if="open" role="dialog" aria-label="Tasks"
         class="panel fixed bottom-8 left-2 z-40 flex max-h-[60vh] w-[440px] max-w-[calc(100vw-1rem)] flex-col overflow-hidden shadow-2xl">
      <div class="flex items-center justify-between border-b border-line px-4 py-2.5">
        <h2 class="text-[13px] font-semibold">Tasks</h2>
        <button class="rounded p-1 text-muted hover:bg-hover hover:text-fg" aria-label="Close" @click="open = false">
          <Icon name="close" class="size-3.5!" />
        </button>
      </div>
      <div class="overflow-y-auto">
        <p v-if="!running.size && !done.length" class="px-4 py-6 text-center text-muted">Nothing has run yet.</p>
        <div v-for="t in running.values()" :key="t.id" class="flex flex-col gap-1.5 border-b border-line px-4 py-3">
          <div class="flex items-center gap-2">
            <span class="size-2 shrink-0 animate-pulse rounded-full bg-accent" />
            <span class="flex-1 truncate font-semibold">{{ t.title }}</span>
            <span class="text-xs text-muted tabular-nums">{{ t.state === State.Queued ? 'Waiting' : elapsed(t.started) }}</span>
            <button class="btn h-6 px-2 text-xs" @click="Cancel(t.id)">Cancel</button>
          </div>
          <p v-if="t.note" class="text-xs text-muted">Now: {{ t.note }}</p>
          <span class="relative h-1 overflow-hidden rounded-full bg-hover">
            <span v-if="t.progress >= 0" class="absolute inset-y-0 left-0 rounded-full bg-accent transition-[width] duration-300"
                  :style="{ width: `${t.progress * 100}%` }" />
            <span v-else class="absolute inset-y-0 w-1/3 animate-[slide_1s_linear_infinite] rounded-full bg-accent" />
          </span>
        </div>
        <div v-for="d in done" :key="d.s.id" class="flex flex-col gap-1 border-b border-line px-4 py-2.5 last:border-b-0">
          <div class="flex items-center gap-2">
            <span class="size-2 shrink-0 rounded-full" :class="d.s.state === State.Failed ? 'bg-bad' : d.s.state === State.Cancelled ? 'bg-muted' : 'bg-ok'" />
            <span class="flex-1 truncate">{{ d.s.title }}</span>
            <span class="text-xs text-muted tabular-nums">{{ outcome[d.s.state] }} after {{ elapsed(d.s.started, d.at) }}</span>
          </div>
          <p v-if="d.s.error" class="text-xs text-bad select-text">{{ d.s.error }}</p>
        </div>
      </div>
    </div>
  </Transition>
  <footer class="flex h-7 shrink-0 items-center gap-4 border-t border-line bg-panel px-3 text-[12px] text-muted">
    <button class="flex min-w-0 items-center self-stretch rounded px-1 -mx-1 hover:bg-hover" :aria-expanded="open"
            title="Show tasks" @click="open = !open">
    <Transition name="page" mode="out-in">
      <div v-if="current" key="busy" class="flex min-w-0 items-center gap-2.5">
        <span class="size-2 shrink-0 animate-pulse rounded-full bg-accent" />
        <span class="truncate text-fg">{{ current.title }}<span v-if="current.note" class="text-muted">, {{ current.note }}</span></span>
        <span v-if="running.size > 1">and {{ running.size - 1 }} more</span>
        <span class="relative h-1 w-28 overflow-hidden rounded-full bg-hover">
          <span v-if="current.progress >= 0" class="absolute inset-y-0 left-0 rounded-full bg-accent transition-[width] duration-300"
                :style="{ width: `${current.progress * 100}%` }" />
          <span v-else class="absolute inset-y-0 w-1/3 animate-[slide_1s_linear_infinite] rounded-full bg-accent" />
        </span>
        <span v-if="!open" class="text-fg underline-offset-2 hover:underline">Details</span>
      </div>
      <div v-else key="idle" class="flex min-w-0 items-center gap-2">
        <span class="size-2 shrink-0 rounded-full" :class="last?.state === State.Failed ? 'bg-bad' : 'bg-ok'" />
        <span class="truncate">{{ finishedText || 'Ready' }}</span>
      </div>
    </Transition>
    </button>

    <span class="ml-auto shrink-0" :class="elevated === false ? 'text-warn' : ''">
      {{ elevated === null ? '' : elevated ? 'Administrator' : 'Limited rights' }}
    </span>
  </footer>
</template>

<style scoped>
@keyframes slide {
  from { left: -33%; }
  to { left: 100%; }
}
</style>
