<script setup lang="ts">
import { href, route, type Page } from '../router'
import { repo } from '../content'

const links: { page: Page; label: string }[] = [
  { page: 'features', label: 'Features' },
  { page: 'how', label: 'How it works' },
  { page: 'install', label: 'Install' },
]
</script>

<template>
  <header class="head">
    <div class="wrap row">
      <a class="brand" :href="href('home')"><img :src="'icon.png'" alt="" width="30" height="30"><span>ReforgePulse</span></a>
      <nav aria-label="Pages">
        <a v-for="l in links" :key="l.page" :href="href(l.page)" :aria-current="route.page === l.page ? 'page' : undefined">{{ l.label }}</a>
        <a :href="repo" class="gh">GitHub</a>
      </nav>
    </div>
  </header>
</template>

<style scoped>
.head { position: sticky; top: 0; z-index: 5; background: color-mix(in srgb, var(--bg) 88%, transparent); backdrop-filter: blur(8px); border-bottom: 1px solid var(--line); }
.row { height: var(--head-h); display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.brand { display: flex; align-items: center; gap: 10px; text-decoration: none; font: 600 20px/1 var(--f-display); flex: none; }
.brand img { width: 30px; height: 30px; }
nav { display: flex; gap: 4px; overflow-x: auto; scrollbar-width: none; }
nav::-webkit-scrollbar { display: none; }
nav a { flex: none; text-decoration: none; color: var(--muted); font-size: 15px; padding: 6px 10px; border-radius: 4px; }
nav a:hover { color: var(--ink); background: color-mix(in srgb, var(--panel) 80%, transparent); }
nav a[aria-current="page"] { color: var(--ink); box-shadow: inset 0 -2px 0 var(--signal); border-radius: 0; }
@media (max-width: 560px) { .gh { display: none; } nav a { padding: 6px 7px; font-size: 14px; } }
@media (max-width: 480px) { .brand span { display: none; } }
</style>
