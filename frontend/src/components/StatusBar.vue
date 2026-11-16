<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import Icon from './Icon.vue'
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
  <footer class="flex h-8 shrink-0 items-center gap-3 px-4 text-xs text-muted">
    <Transition name="page" mode="out-in">
      <div v-if="current" key="busy" class="flex min-w-0 items-center gap-2.5">
        <span class="size-3 animate-spin rounded-full border-2 border-accent/30 border-t-accent" />
        <span class="truncate text-fg">{{ current.title }}<span v-if="current.note" class="text-muted"> · {{ current.note }}</span></span>
        <span v-if="running.size > 1" class="rounded-full bg-hover px-1.5">+{{ running.size - 1 }}</span>
        <span class="relative h-1 w-24 overflow-hidden rounded-full bg-hover">
          <span v-if="current.progress >= 0" class="absolute inset-y-0 left-0 rounded-full bg-accent transition-[width] duration-300"
                :style="{ width: `${current.progress * 100}%` }" />
          <span v-else class="absolute inset-y-0 w-1/3 animate-[slide_1.1s_ease-in-out_infinite] rounded-full bg-accent" />
        </span>
        <button class="rounded px-1.5 py-0.5 transition hover:bg-hover hover:text-fg" @click="Cancel(current.id)">Cancel</button>
      </div>
      <div v-else key="idle" class="flex min-w-0 items-center gap-2">
        <span class="size-1.5 rounded-full"
              :class="last?.state === State.Failed ? 'bg-[color:var(--color-red-500)]' : 'bg-[color:var(--color-emerald-500)]'" />
        <span class="truncate">{{ finishedText || 'Ready' }}</span>
      </div>
    </Transition>

    <span class="ml-auto flex shrink-0 items-center gap-1.5" :title="elevated ? 'Running as Administrator' : 'Running without administrator rights'">
      <Icon name="shield" class="size-3.5!" :class="elevated ? 'text-accent' : ''" />
      {{ elevated === null ? '' : elevated ? 'Administrator' : 'Limited' }}
    </span>
    <span class="shrink-0">Offline · nothing leaves this PC</span>
  </footer>
</template>

<style scoped>
@keyframes slide {
  from { left: -33%; }
  to { left: 100%; }
}
</style>
