import { ref } from 'vue'
import {
  Check, CheckOnStart, Download, OnStart, Restart, SetCheckOnStart,
} from '../../bindings/github.com/cn8001/reforgepulse/internal/app/updateservice'
import type { Release } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'

// One update state for Settings and the start-up prompt, so a download
// begun in one shows in the other.
export type UpdateState = 'idle' | 'checking' | 'latest' | 'available' | 'downloading' | 'ready'
export const update = ref<UpdateState>('idle')
export const release = ref<Release | null>(null)
export const updateError = ref('')
export const prompting = ref(false) // the start-up prompt is open
export const restarting = ref(false) // the restart confirmation is open
export const checkOnStart = ref(true)

export async function check() {
  update.value = 'checking'
  updateError.value = ''
  try {
    release.value = await Check()
    update.value = release.value ? 'available' : 'latest'
  } catch (e) {
    updateError.value = String(e)
    update.value = 'idle'
  }
}

// The quiet check when the window opens. A failure says nothing: no
// network, or no release yet, is not worth interrupting anyone for.
export async function checkAtStart() {
  checkOnStart.value = await CheckOnStart().catch(() => true)
  const rel = await OnStart().catch(() => null)
  if (!rel || update.value !== 'idle') return
  release.value = rel
  update.value = 'available'
  prompting.value = true
}

export async function download() {
  update.value = 'downloading'
  updateError.value = ''
  try {
    await Download(release.value!.version)
    update.value = 'ready'
  } catch (e) {
    updateError.value = String(e)
    update.value = 'available'
  }
}

export const askRestart = () => (restarting.value = true)
export const cancelRestart = () => (restarting.value = false)
export const closePrompt = () => (prompting.value = false)

export async function restart() {
  try {
    await Restart()
  } catch (e) {
    updateError.value = String(e)
    restarting.value = false
  }
}

export async function setCheckOnStart(on: boolean) {
  checkOnStart.value = on
  await SetCheckOnStart(on).catch((e) => {
    checkOnStart.value = !on
    updateError.value = String(e)
  })
}
