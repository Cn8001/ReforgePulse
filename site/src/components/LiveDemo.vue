<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

// The app is laid out for a 1100 x 700 window, so the demo keeps that size and
// is scaled to the space it gets instead of reflowing into a cramped layout.
const W = 1100
const H = 700
const box = ref<HTMLElement | null>(null)
const scale = ref(1)
let ro: ResizeObserver | undefined

onMounted(() => {
  ro = new ResizeObserver(([e]) => { scale.value = e.contentRect.width / W })
  if (box.value) ro.observe(box.value)
})
onBeforeUnmount(() => ro?.disconnect())
</script>

<template>
  <figure class="demo">
    <div ref="box" class="frame">
      <iframe src="demo/index.html" title="The ReforgePulse app running on a made-up PC" :width="W" :height="H"
        :style="{ transform: `scale(${scale})` }" loading="lazy" />
    </div>
    <figcaption>
      <span>The real ReforgePulse app on a made-up PC. Click around; nothing here touches your computer.</span>
      <a class="btn btn-sm" href="demo/index.html" target="_blank" rel="noopener">Open full size</a>
    </figcaption>
  </figure>
</template>

<style scoped>
.demo { margin: 0; display: grid; gap: 12px; min-width: 0; }
.frame { position: relative; aspect-ratio: 1100 / 700; max-width: 100%; overflow: hidden; border-radius: 6px; border: 1px solid var(--line); background: var(--app-bg); box-shadow: 0 24px 50px -30px rgb(0 0 0 / 0.6); }
iframe { position: absolute; top: 0; left: 0; border: 0; transform-origin: 0 0; }
figcaption { display: flex; align-items: center; justify-content: space-between; gap: 12px 20px; flex-wrap: wrap; color: var(--muted); font-size: 14px; }
</style>
