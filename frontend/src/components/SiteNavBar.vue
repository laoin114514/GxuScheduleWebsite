<template>
  <header id="navbar" class="glass-nav sticky top-0 z-50 transition-all duration-300 py-3.5">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-between">
      <a href="#hero" class="flex items-center space-x-3 group">
        <img
          :src="site.logo"
          :alt="site.name + ' Logo'"
          class="w-9 h-9 rounded-xl shadow-sm transition-transform duration-200 group-hover:scale-105"
        />
        <div class="flex flex-col">
          <span class="font-bold text-lg leading-tight tracking-tight text-slate-900 dark:text-white">{{ site.name }}</span>
          <span class="text-[11px] text-slate-500 dark:text-slate-400 font-medium">{{ site.slug }}</span>
        </div>
      </a>

      <nav class="hidden md:flex items-center space-x-8 text-sm font-medium text-slate-700 dark:text-slate-300">
        <a
          v-for="item in navLinks"
          :key="item.href"
          :href="item.href"
          class="hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
        >{{ item.label }}</a>
      </nav>

      <div class="flex items-center space-x-3">
        <button
          type="button"
          aria-label="切换深浅主题"
          :title="isDark ? '切换到浅色模式' : '切换到深色模式'"
          class="p-2 rounded-xl text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white hover:bg-slate-200/50 dark:hover:bg-slate-800/50 transition-colors"
          @click="toggle"
        >
          <IconSun v-if="isDark" class="w-5 h-5" />
          <IconMoon v-else class="w-5 h-5" />
        </button>

        <a
          :href="links.repo"
          target="_blank"
          rel="noopener noreferrer"
          class="hidden sm:inline-flex items-center space-x-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold border border-slate-300 dark:border-slate-700 text-slate-700 dark:text-slate-200 hover:bg-white/60 dark:hover:bg-slate-800/60 transition-all"
        >
          <IconGithub class="w-4 h-4" />
          <span>GitHub</span>
        </a>

        <a
          href="#download"
          class="inline-flex items-center space-x-1.5 px-4 py-2 rounded-xl text-xs font-semibold bg-[#4A90E2] text-white shadow-sm hover:bg-[#357ABD] active:translate-y-0.5 transition-all"
        >
          <IconDownload class="w-4 h-4" />
          <span>下载应用</span>
        </a>
      </div>
    </div>
  </header>
</template>

<script setup>
import { links, navLinks, site } from '../config/site'
import { useTheme } from '../composables/useTheme'
import IconSun from './icons/IconSun.vue'
import IconMoon from './icons/IconMoon.vue'
import IconGithub from './icons/IconGithub.vue'
import IconDownload from './icons/IconDownload.vue'

const { isDark, toggle } = useTheme()
</script>
