<template>
  <section id="open-source" v-reveal class="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="glass-card p-8 sm:p-12 relative overflow-hidden">
      <div class="max-w-3xl">
        <div class="inline-flex items-center space-x-2 px-3 py-1 rounded-full bg-emerald-100/80 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 text-xs font-medium mb-4">
          <IconCheckCircle class="w-3.5 h-3.5" />
          <span>无服务器中间人 · 数据 100% 留存在手机本地</span>
        </div>

        <h2 class="text-2xl sm:text-3xl font-extrabold text-slate-900 dark:text-white">
          课表是你的，数据也留在你的手机里
        </h2>
        <p class="mt-3 text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
          没有商业打扰，没有弹窗推销。基于历史优秀开源项目二次开发，所有代码完全公开透明，
          教务账密直接在手机本地加密存入 Room 数据库与教务对接，不经任何第三方中转。
        </p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mt-8">
        <div
          v-for="item in guarantees"
          :key="item.title"
          class="p-4 rounded-xl bg-white/40 dark:bg-slate-800/40 border border-slate-200/50 dark:border-slate-700/50"
        >
          <div class="text-sm font-bold text-slate-900 dark:text-white flex items-center space-x-1.5">
            <component :is="item.icon" class="w-4 h-4" :class="item.iconColor" />
            <span>{{ item.title }}</span>
          </div>
          <p class="text-xs text-slate-500 dark:text-slate-400 mt-2">{{ item.desc }}</p>
        </div>
      </div>

      <div class="mt-8 pt-6 border-t border-slate-200/60 dark:border-slate-700/60">
        <div class="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-3">原生技术底座：</div>
        <div class="flex flex-wrap gap-2 text-[11px] font-mono">
          <span
            v-for="tag in techTags"
            :key="tag"
            class="px-2.5 py-1 rounded-lg bg-blue-100/60 dark:bg-slate-800 text-slate-700 dark:text-slate-300"
          >{{ tag }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { guarantees, techTags } from '../config/site'
import IconCheckCircle from './icons/IconCheckCircle.vue'
</script>
