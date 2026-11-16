<script setup lang="ts">
import { onMounted, ref } from 'vue'

// A plain level meter that fills in when shown; it turns red past 90%.
const props = defineProps<{ ratio: number | null }>()
const fill = ref(0)
onMounted(() => requestAnimationFrame(() => (fill.value = Math.min(1, Math.max(0, props.ratio ?? 0)))))
</script>

<template>
  <div class="h-1.5 overflow-hidden rounded-full bg-hover">
    <div v-if="ratio !== null" class="h-full rounded-full transition-[width] duration-700 ease-out"
         :class="fill > 0.9 ? 'bg-bad' : 'bg-accent'" :style="{ width: `${fill * 100}%` }" />
  </div>
</template>
