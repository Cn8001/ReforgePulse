<script setup lang="ts">
// Every change goes through here: the user sees what will happen and says yes.
defineProps<{ open: boolean; title: string; confirm: string; danger?: boolean; busy?: boolean }>()
const emit = defineEmits<{ yes: []; no: [] }>()
</script>

<template>
  <Transition enter-from-class="opacity-0" leave-to-class="opacity-0"
              enter-active-class="transition-opacity duration-150" leave-active-class="transition-opacity duration-150">
    <div v-if="open" class="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6" @click.self="!busy && emit('no')">
      <section role="dialog" aria-modal="true" class="panel w-[480px] max-w-full shadow-2xl">
        <header class="border-b border-line px-5 py-4">
          <h2 class="text-[15px] font-semibold">{{ title }}</h2>
          <p v-if="danger" class="mt-1 text-xs text-bad">This cannot be undone.</p>
        </header>
        <div class="px-5 py-4 text-muted"><slot /></div>
        <footer class="flex justify-end gap-2 border-t border-line px-5 py-3">
          <button class="btn" :disabled="busy" @click="emit('no')">Cancel</button>
          <button :class="danger ? 'btn-danger' : 'btn-primary'" :disabled="busy" @click="emit('yes')">
            {{ busy ? 'Working…' : confirm }}
          </button>
        </footer>
      </section>
    </div>
  </Transition>
</template>
