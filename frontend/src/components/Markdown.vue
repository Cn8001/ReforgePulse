<script setup lang="ts">
import { computed } from 'vue'
import { Browser } from '@wailsio/runtime'
import { parse } from '../lib/markdown'
import MarkdownInline from './MarkdownInline.vue'

const props = defineProps<{ source: string }>()
const blocks = computed(() => parse(props.source))
</script>

<template>
  <div class="md flex flex-col gap-3 leading-relaxed select-text">
    <template v-for="(b, i) in blocks" :key="i">
      <h3 v-if="b.t === 'h' && b.level === 1" class="text-[15px] font-semibold text-fg"><MarkdownInline :parts="b.inline" @open="Browser.OpenURL" /></h3>
      <h4 v-else-if="b.t === 'h'" class="mt-1 text-[13px] font-semibold text-fg"><MarkdownInline :parts="b.inline" @open="Browser.OpenURL" /></h4>
      <p v-else-if="b.t === 'p'"><MarkdownInline :parts="b.inline" @open="Browser.OpenURL" /></p>
      <ul v-else-if="b.t === 'ul'" class="flex list-disc flex-col gap-1 pl-5">
        <li v-for="(item, j) in b.items" :key="j"><MarkdownInline :parts="item" @open="Browser.OpenURL" /></li>
      </ul>
      <ol v-else-if="b.t === 'ol'" class="flex list-decimal flex-col gap-1 pl-5">
        <li v-for="(item, j) in b.items" :key="j"><MarkdownInline :parts="item" @open="Browser.OpenURL" /></li>
      </ol>
      <pre v-else-if="b.t === 'code'" class="overflow-x-auto rounded border border-line bg-panel p-2 font-mono text-[11px]">{{ b.text }}</pre>
    </template>
  </div>
</template>
