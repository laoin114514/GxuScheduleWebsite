<template>
  <footer class="glass-nav mt-12 py-12 border-t border-slate-200/80 dark:border-slate-800">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
        <div class="space-y-3">
          <div class="flex items-center space-x-2">
            <img :src="site.logo" :alt="site.name + ' Logo'" class="w-8 h-8 rounded-lg" />
            <span class="font-bold text-base text-slate-900 dark:text-white">{{ site.name }}</span>
          </div>
          <p class="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">{{ site.tagline }}</p>
          <div class="text-[11px] text-slate-400">
            问题反馈：<a :href="links.mailto" class="hover:text-blue-500 underline font-mono">{{ site.email }}</a>
          </div>
        </div>

        <div v-for="column in footerColumns" :key="column.title">
          <h4 class="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">{{ column.title }}</h4>
          <ul class="mt-3 space-y-2 text-xs text-slate-600 dark:text-slate-400">
            <li v-for="link in column.links" :key="link.label">
              <a
                v-if="link.href"
                :href="link.href"
                :target="link.external ? '_blank' : undefined"
                :rel="link.external ? 'noopener noreferrer' : undefined"
                class="hover:text-blue-600"
              >{{ link.label }}</a>
              <span v-else class="text-slate-400">{{ link.label }}</span>
            </li>
          </ul>
        </div>

        <div>
          <h4 class="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">血统与致谢</h4>
          <p class="mt-3 text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
            基于开源项目 <span class="font-mono text-slate-700 dark:text-slate-300">Yngu196/Schedule</span> 二次开发，
            最早血统来自 <span class="font-mono text-slate-700 dark:text-slate-300">WakeUp 课程表</span>。
            感谢所有为校园开源生态奉献的开发者。
          </p>
        </div>
      </div>

      <div class="mt-8 pt-6 border-t border-slate-200/50 dark:border-slate-800 text-center text-xs text-slate-400 flex flex-col sm:flex-row justify-between items-center space-y-2 sm:space-y-0">
        <div>© 2026 {{ site.name }} · 基于 Yngu196/Schedule 二次开发，遵循 Apache License 2.0</div>
        <div>专为 Android 打造 · 广西大学学子非官方校园开源工具</div>
      </div>
    </div>
  </footer>
</template>

<script setup>
import { footerColumns, links, site } from '../config/site'
</script>
