<script setup lang="ts">
// Sized in px, not rem: the knob slides a fixed distance, and the root font
// size (13px) would shrink a rem track under it.
defineProps<{ on: boolean; disabled?: boolean; label?: string }>()
defineEmits<{ toggle: [] }>()
</script>

<template>
  <button role="switch" :aria-checked="on" :aria-label="label" :disabled="disabled"
          class="relative h-[20px] w-[36px] shrink-0 rounded-full border transition-colors disabled:opacity-50"
          :class="on ? 'border-accent bg-accent' : 'border-line bg-hover'" @click="$emit('toggle')">
    <span class="absolute top-[3px] left-0 size-[12px] rounded-full bg-white transition-transform duration-200"
          :class="on ? 'translate-x-[19px]' : 'translate-x-[3px]'" />
  </button>
</template>
