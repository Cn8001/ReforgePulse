<script setup lang="ts">
// A page whose title, actions and tabs stay put while only its content
// scrolls under them.
</script>

<template>
  <section class="flex h-full flex-col">
    <div class="mx-auto flex w-full max-w-6xl shrink-0 flex-col gap-5 px-6 pt-6 pb-4">
      <slot name="head" />
    </div>
    <div class="min-h-0 flex-1 overflow-y-auto [scrollbar-gutter:stable]">
      <div class="mx-auto flex max-w-6xl flex-col gap-5 px-6 pb-6">
        <slot />
      </div>
    </div>
  </section>
</template>
