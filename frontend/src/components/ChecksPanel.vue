<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Results, Run } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/checkservice'
import type { CheckResult } from '../../bindings/github.com/cn8001/reforgepulse/internal/domain/models'
import { checks } from '../lib/checks'
import { when } from '../lib/format'
import CheckResultView from './CheckResultView.vue'

// The last result of each check, saved ones first, so a check that took
// twenty minutes is still there after leaving the page.
const results = ref<Record<string, CheckResult>>({})
const busy = ref('')
const error = ref('')

onMounted(async () => {
  for (const r of ((await Results().catch(() => null)) ?? []).slice().reverse()) results.value[r.name] = r
})

async function run(id: string) {
  busy.value = id
  error.value = ''
  try {
    results.value[id] = await Run(id)
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = ''
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <p class="text-muted">Windows' own health checks. They only read and report; none of them repairs anything.</p>
    <p v-if="error" class="notice border-bad/40 bg-bad/10 text-bad">{{ error }}</p>
    <article v-for="c in checks" :key="c.id" class="panel">
      <div class="flex items-center gap-4 px-4 py-3">
        <div class="min-w-0 flex-1">
          <p class="font-semibold">{{ c.title }} <span class="text-xs font-normal text-muted">takes {{ c.time }}</span></p>
          <p class="text-[14px] text-muted">{{ c.what }}</p>
        </div>
        <button class="btn-primary" :disabled="!!busy" @click="run(c.id)">{{ busy === c.id ? 'Running…' : 'Run' }}</button>
      </div>
      <div v-if="results[c.id]" class="border-t border-line bg-raised px-4 py-3">
        <p class="mb-2 text-xs text-muted">
          <template v-if="results[c.id].at">Last run {{ when(results[c.id].at) }}.</template>
          <template v-if="results[c.id].id && !results[c.id].fromWindows"> Saved under Reports.</template>
        </p>
        <CheckResultView :r="results[c.id]" />
      </div>
    </article>
  </div>
</template>
