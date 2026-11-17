<script setup lang="ts">
// Every change goes through here: the user sees what will happen and says yes.
// warning is the one line about what cannot be taken back, shown first so it
// is read before the button; danger makes it red, otherwise amber.
defineProps<{ open: boolean; title: string; confirm: string; danger?: boolean; busy?: boolean; warning?: string }>()
const emit = defineEmits<{ yes: []; no: [] }>()
</script>

<template>
  <Transition enter-from-class="opacity-0" leave-to-class="opacity-0"
              enter-active-class="transition-opacity duration-150" leave-active-class="transition-opacity duration-150">
    <div v-if="open" class="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6" @click.self="!busy && emit('no')">
      <section role="dialog" aria-modal="true" class="panel w-[480px] max-w-full shadow-2xl">
        <header class="border-b border-line px-5 py-4">
          <h2 class="text-[16px] font-semibold">{{ title }}</h2>
        </header>
        <div class="flex flex-col gap-3 px-5 py-4 text-muted">
          <p v-if="warning" class="notice font-semibold"
             :class="danger ? 'border-bad/40 bg-bad/10 text-bad' : 'border-warn/40 bg-warn/10 text-warn'">{{ warning }}</p>
          <div><slot /></div>
        </div>
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
