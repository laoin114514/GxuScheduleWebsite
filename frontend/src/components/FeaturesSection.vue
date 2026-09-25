<template>
  <section id="features" v-reveal class="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="text-center max-w-3xl mx-auto mb-12">
      <h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">硬核功能，把简单做到极致</h2>
      <p class="mt-3 text-sm sm:text-base text-slate-600 dark:text-slate-400">
        不做华而不实的冗余堆叠，只为解决在校大学生看课最痛的几个瞬间。
      </p>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
      <div v-for="item in features" :key="item.title" class="glass-card p-6 flex flex-col justify-between">
        <div>
          <div class="text-3xl sm:text-4xl font-extrabold text-[#4A90E2] tracking-tight">{{ item.num }}</div>
          <h3 class="text-lg font-bold text-slate-900 dark:text-white mt-3">{{ item.title }}</h3>
          <p class="text-xs sm:text-sm text-slate-600 dark:text-slate-400 mt-2 leading-relaxed">{{ item.desc }}</p>
        </div>
        <div class="mt-4 pt-3 border-t border-slate-200/60 dark:border-slate-800 text-[11px] text-blue-600 dark:text-blue-400 font-medium">
          {{ item.footnote }}
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { features } from '../config/site'
</script>
