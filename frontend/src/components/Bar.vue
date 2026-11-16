<script setup lang="ts">
import { onMounted, ref } from 'vue'

// A segmented meter, like a hardware gauge; it fills in when shown.
const props = defineProps<{ ratio: number | null; tint: string }>()
const fill = ref(0)
onMounted(() => requestAnimationFrame(() => (fill.value = Math.min(1, Math.max(0, props.ratio ?? 0)))))
</script>

<template>
  <div class="meter h-2.5 bg-hover">
    <div v-if="ratio !== null" class="h-full transition-[width] duration-700 ease-out"
         :style="{ width: `${fill * 100}%`, background: fill > 0.9 ? 'var(--bad)' : tint }" />
  </div>
</template>

<style scoped>
.meter {
  mask: repeating-linear-gradient(90deg, #000 0 5px, transparent 5px 7px);
}
</style>
