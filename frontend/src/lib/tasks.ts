import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { List } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/taskservice'
import { State, type Status } from '../../bindings/github.com/cn8001/reforgepulse/internal/tasks/models'

// Running tasks by ID, for panels that show their own progress next to the
// status bar's. Fed by the same events.
export const tasks = ref(new Map<string, Status>())

let started = false
export function followTasks() {
  if (started) return
  started = true
  const put = (s: Status) => {
    const next = new Map(tasks.value)
    s.state === State.Queued || s.state === State.Running ? next.set(s.id, s) : next.delete(s.id)
    tasks.value = next
  }
  Events.On('task', (e) => put(e.data))
  List().then((all) => (all ?? []).forEach(put)).catch(() => {})
}

// The newest running task whose title starts with prefix.
export function taskTitled(prefix: string): Status | null {
  return [...tasks.value.values()].filter((t) => t.title.startsWith(prefix)).at(-1) ?? null
}

// A cancelled task comes back as an error; it is not one to the user.
export const cancelled = (e: unknown) => /context canceled/i.test(String(e))
