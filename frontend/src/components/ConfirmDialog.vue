<script setup lang="ts">
// Every change goes through here: the user sees what will happen and says yes.
defineProps<{ open: boolean; title: string; confirm: string; danger?: boolean; busy?: boolean }>()
const emit = defineEmits<{ yes: []; no: [] }>()
</script>

<template>
  <Transition enter-from-class="opacity-0" leave-to-class="opacity-0"
              enter-active-class="transition-opacity duration-150" leave-active-class="transition-opacity duration-150">
    <div v-if="open" class="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6" @click.self="!busy && emit('no')">
      <section class="w-[480px] max-w-full border border-line bg-panel shadow-2xl">
        <header class="border-b border-line px-5 py-3">
          <p class="caption" :class="danger ? 'text-bad!' : 'text-warn!'">{{ danger ? 'cannot be undone' : 'confirm change' }}</p>
          <h2 class="mt-1 font-semibold">{{ title }}</h2>
        </header>
        <div class="px-5 py-4 text-muted"><slot /></div>
        <footer class="flex justify-end gap-2 border-t border-line px-5 py-3">
          <button class="h-8 border border-line px-4 font-mono text-xs transition hover:bg-hover" :disabled="busy" @click="emit('no')">cancel</button>
          <button class="h-8 px-4 font-mono text-xs font-bold text-white transition hover:brightness-110 disabled:opacity-60"
                  :class="danger ? 'bg-bad' : 'bg-accent'" :disabled="busy" @click="emit('yes')">
            {{ busy ? 'working…' : confirm }}
          </button>
        </footer>
      </section>
    </div>
  </Transition>
</template>
