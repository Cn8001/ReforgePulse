<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { Cancel, List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/taskservice'
import { State, type Status } from '../../bindings/github.com/cn8001/reforgepulse/internal/tasks/models'

defineProps<{ elevated: boolean | null }>()

const running = ref(new Map<string, Status>())
// The first List can answer after a task's final event; those IDs must not
// come back as running.
const ended = new Set<string>()
const last = ref<Status | null>(null) // the latest finished task, shown for a moment
let hide: ReturnType<typeof setTimeout> | undefined

function onTask(s: Status) {
  const live = s.state === State.Queued || s.state === State.Running
  if (live && ended.has(s.id)) return
  if (!live) ended.add(s.id)
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
  return `${s.title}: done`
})
</script>

<template>
  <footer class="flex h-7 shrink-0 items-center gap-4 border-t border-line bg-panel px-3 text-[12px] text-muted">
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
        <button class="rounded px-1.5 text-fg transition hover:bg-hover" @click="Cancel(current.id)">Cancel</button>
      </div>
      <div v-else key="idle" class="flex min-w-0 items-center gap-2">
        <span class="size-2 shrink-0 rounded-full" :class="last?.state === State.Failed ? 'bg-bad' : 'bg-ok'" />
        <span class="truncate">{{ finishedText || 'Ready' }}</span>
      </div>
    </Transition>

    <span class="ml-auto shrink-0" :class="elevated === false ? 'text-warn' : ''">
      {{ elevated === null ? '' : elevated ? 'Administrator' : 'Limited rights' }}
    </span>
    <span class="shrink-0">Offline</span>
  </footer>
</template>

<style scoped>
@keyframes slide {
  from { left: -33%; }
  to { left: 100%; }
}
</style>
