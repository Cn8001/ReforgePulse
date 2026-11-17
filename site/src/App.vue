<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { route } from './router'
import SiteHeader from './components/SiteHeader.vue'
import SiteFooter from './components/SiteFooter.vue'
import HomePage from './pages/HomePage.vue'
import FeaturesPage from './pages/FeaturesPage.vue'
import HowPage from './pages/HowPage.vue'
import InstallPage from './pages/InstallPage.vue'

const pages = { home: HomePage, features: FeaturesPage, how: HowPage, install: InstallPage }
const titles = { home: 'ReforgePulse', features: 'Features', how: 'How it works', install: 'Install' }
const current = computed(() => pages[route.value.page])

watchEffect(() => {
  const t = titles[route.value.page]
  document.title = route.value.page === 'home' ? t : `${t}, ReforgePulse`
})
</script>

<template>
  <SiteHeader />
  <main>
    <component :is="current" />
  </main>
  <SiteFooter />
</template>
