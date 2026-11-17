<script setup lang="ts">
import { repo, version } from '../content'
</script>

<template>
  <footer>
    <div class="wrap row">
      <span>ReforgePulse {{ version }} is free and provided as is. Because it runs as administrator, please read the terms before using it.</span>
      <nav aria-label="Project links">
        <a :href="`${repo}/releases`">Releases</a>
        <a :href="`${repo}/blob/master/CHANGELOG.md`">Changelog</a>
        <a :href="`${repo}/blob/master/TERMS.md`">Terms</a>
        <a :href="`${repo}/blob/master/LICENSE`">MIT licence</a>
      </nav>
    </div>
  </footer>
</template>

<style scoped>
footer { border-top: 1px solid var(--line); padding-block: 24px 32px; color: var(--muted); font-size: 14px; }
.row { display: flex; justify-content: space-between; gap: 16px 28px; flex-wrap: wrap; }
nav { display: flex; gap: 18px; flex-wrap: wrap; }
</style>
