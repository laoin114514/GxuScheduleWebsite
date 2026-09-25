<template>
  <section id="faq" v-reveal class="py-16 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="text-center mb-10">
      <h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">常见问题与答疑</h2>
      <p class="mt-2 text-sm text-slate-600 dark:text-slate-400">使用中有疑问？先看看这里有没有你遇到的情况。</p>
    </div>

    <div class="space-y-3">
      <div
        v-for="(item, index) in faqs"
        :key="item.q"
        class="faq-item glass-card p-4"
        :class="{ active: index === activeIndex }"
      >
        <button
          type="button"
          class="w-full flex items-center justify-between font-semibold text-sm text-slate-900 dark:text-white text-left"
          :aria-expanded="index === activeIndex"
          :aria-controls="'faq-panel-' + index"
          @click="toggle(index)"
        >
          <span>{{ item.q }}</span>
          <IconChevronDown class="faq-icon w-4 h-4 text-slate-400 transition-transform duration-200 shrink-0 ml-3" />
        </button>
        <div :id="'faq-panel-' + index" class="faq-content text-xs text-slate-600 dark:text-slate-400 mt-2 leading-relaxed">
          {{ item.a }}
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { faqs } from '../config/site'
import IconChevronDown from './icons/IconChevronDown.vue'

const activeIndex = ref(0)

function toggle(index) {
  activeIndex.value = activeIndex.value === index ? -1 : index
}
</script>
