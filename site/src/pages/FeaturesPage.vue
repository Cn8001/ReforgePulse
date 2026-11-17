<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { allFeatures, groups } from '../content'
import { href, route } from '../router'

const feature = computed(() => allFeatures.find((f) => f.id === route.value.sub) ?? allFeatures[0])
const shot = ref(0)
watch(feature, () => { shot.value = 0 })
const current = computed(() => feature.value.shots[shot.value] ?? feature.value.shots[0])

function pick(e: Event) {
  location.hash = href('features', (e.target as HTMLSelectElement).value)
}
</script>

<template>
  <section class="page">
    <div class="wrap">
      <div class="page-head">
        <h1>Features</h1>
        <p>ReforgePulse works with the information Windows already records. Pick a part of the app to see it.</p>
      </div>

      <div class="layout">
        <nav class="side" aria-label="Features">
          <div v-for="g in groups" :key="g.title" class="group">
            <h2>{{ g.title }}</h2>
            <ul class="plain">
              <li v-for="f in g.features" :key="f.id">
                <a :href="href('features', f.id)" :aria-current="f.id === feature.id ? 'page' : undefined">{{ f.title }}</a>
              </li>
            </ul>
          </div>
        </nav>

        <label class="picker">
          <span>Part of the app</span>
          <select id="feature-picker" :value="feature.id" @change="pick">
            <optgroup v-for="g in groups" :key="g.title" :label="g.title">
              <option v-for="f in g.features" :key="f.id" :value="f.id">{{ f.title }}</option>
            </optgroup>
          </select>
        </label>

        <article class="detail">
          <div class="text">
            <h2>{{ feature.title }}</h2>
            <p v-for="(p, i) in feature.body" :key="i">{{ p }}</p>
            <ul v-if="feature.facts" class="plain facts" aria-label="What it reads">
              <li v-for="f in feature.facts" :key="f">{{ f }}</li>
            </ul>
          </div>
          <div class="shots">
            <div v-if="feature.shots.length > 1" class="tabs" role="tablist" aria-label="Screenshots">
              <button v-for="(s, i) in feature.shots" :key="s.src" class="tab" role="tab" :aria-selected="i === shot" @click="shot = i">{{ s.label }}</button>
            </div>
            <a class="shot-frame" :href="current.src" target="_blank" rel="noopener">
              <img :key="current.src" :src="current.src" :alt="current.alt" width="2200" height="1400">
            </a>
            <p class="note">Screenshots show a made-up PC. Select one to open it full size.</p>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<style scoped>
.layout { display: grid; grid-template-columns: 220px minmax(0, 1fr); gap: 40px; align-items: start; }
.side { position: sticky; top: calc(var(--head-h) + 24px); display: grid; gap: 16px; }
.group h2 { font: 600 13px/1.3 var(--f-body); color: var(--muted); margin-bottom: 6px; }
.group a { display: block; text-decoration: none; font-size: 15px; line-height: 1.4; padding: 4px 10px; margin-left: -10px; border-radius: 4px; color: var(--muted); border-left: 2px solid transparent; }
.group a:hover { color: var(--ink); background: var(--panel); }
.group a[aria-current="page"] { color: var(--ink); border-left-color: var(--signal); border-radius: 0 4px 4px 0; background: var(--panel); }
.picker { display: none; }

.detail { display: grid; grid-template-columns: minmax(0, 0.75fr) minmax(0, 1.25fr); gap: 36px; align-items: start; }
.text { display: grid; gap: 14px; }
.text h2 { font-size: 32px; }
.text p { color: var(--muted); max-width: 52ch; }
.facts { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 4px; }
.facts li { font-family: var(--f-mono); font-size: 12px; padding: 3px 10px; border: 1px solid var(--line); border-radius: 999px; color: var(--ink); }
.shots { display: grid; gap: 12px; min-width: 0; }
.shots img { width: 100%; animation: show .25s ease; }
@keyframes show { from { opacity: 0; } }
.note { font-size: 13px; color: var(--muted); }

@media (max-width: 1180px) { .detail { grid-template-columns: minmax(0, 1fr); gap: 24px; } }
@media (max-width: 760px) {
  .layout { grid-template-columns: minmax(0, 1fr); gap: 20px; }
  .side { display: none; }
  .picker { display: grid; gap: 6px; font-size: 14px; color: var(--muted); }
  .picker select { font: 500 16px var(--f-body); color: var(--ink); background: var(--panel); border: 1px solid var(--line); border-radius: 4px; padding: 10px 12px; }
}
</style>
