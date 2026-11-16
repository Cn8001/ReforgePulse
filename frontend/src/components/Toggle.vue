<script setup lang="ts">
defineProps<{ on: boolean; disabled?: boolean }>()
defineEmits<{ toggle: [] }>()
</script>

<template>
  <button role="switch" :aria-checked="on" :disabled="disabled"
          class="relative h-5 w-9 shrink-0 border transition-colors disabled:opacity-50"
          :class="on ? 'border-accent bg-accent' : 'border-line bg-hover'" @click="$emit('toggle')">
    <span class="absolute top-0.5 size-3.5 bg-white transition-transform duration-200"
          :class="on ? 'translate-x-[18px]' : 'translate-x-0.5'" />
  </button>
</template>
