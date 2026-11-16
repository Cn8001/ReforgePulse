<script setup lang="ts">
defineProps<{ on: boolean; disabled?: boolean; label?: string }>()
defineEmits<{ toggle: [] }>()
</script>

<template>
  <button role="switch" :aria-checked="on" :aria-label="label" :disabled="disabled"
          class="relative h-5 w-9 shrink-0 rounded-full border transition-colors disabled:opacity-50"
          :class="on ? 'border-accent bg-accent' : 'border-line bg-hover'" @click="$emit('toggle')">
    <span class="absolute top-[3px] size-3 rounded-full bg-white transition-transform duration-200"
          :class="on ? 'translate-x-[19px]' : 'translate-x-[3px]'" />
  </button>
</template>
