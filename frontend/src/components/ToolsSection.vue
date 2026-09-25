<template>
  <section id="tools" v-reveal class="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="text-center max-w-2xl mx-auto mb-12">
      <h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">一揽子校园学习工具集</h2>
      <p class="mt-3 text-sm sm:text-base text-slate-600 dark:text-slate-400">
        围绕西大学生核心动线，成绩、绩点、教室、借书，通通装进一个口袋。
      </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div
        v-for="item in tools"
        :key="item.key"
        class="glass-card p-6 group hover:-translate-y-1 hover:border-blue-400/60 transition-all"
      >
        <div class="w-10 h-10 rounded-xl flex items-center justify-center font-bold" :class="[item.iconBox, item.iconColor]">
          <component :is="item.icon" class="w-5 h-5" />
        </div>
        <h3 class="text-base font-bold text-slate-900 dark:text-white mt-4 group-hover:text-blue-600 transition-colors">
          {{ item.title }}
        </h3>
        <p class="text-xs text-slate-600 dark:text-slate-400 mt-2 leading-relaxed">{{ item.desc }}</p>
      </div>
    </div>
  </section>
</template>

<script setup>
import { tools } from '../config/site'
</script>
