<template>
  <section
    id="download"
    v-reveal
    class="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"
    :data-release-source="source"
  >
    <div class="glass-card p-8 sm:p-12 text-center max-w-3xl mx-auto relative overflow-hidden border-blue-300/40 dark:border-blue-900/40 shadow-xl">
      <div class="flex justify-center mb-4">
        <img :src="site.logo" :alt="site.name + ' 图标'" class="w-20 h-20 rounded-2xl shadow-md" />
      </div>

      <h2 class="text-3xl font-extrabold text-slate-900 dark:text-white">下载{{ site.name }}</h2>
      <p class="text-xs sm:text-sm text-slate-600 dark:text-slate-400 mt-2">
        最新稳定版 · 适配 Android 8.0 至 Android 15 全部架构
      </p>

      <div class="flex items-center justify-center flex-wrap gap-x-3 gap-y-1 mt-4 text-xs font-mono text-slate-500">
        <span class="px-2.5 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 font-bold">
          v{{ versionName }} 稳定版
        </span>
        <template v-if="updatedText">
          <span>·</span>
          <span>{{ updatedText }}</span>
        </template>
        <template v-if="sizeText">
          <span>·</span>
          <span>{{ sizeText }}</span>
        </template>
      </div>

      <div class="flex flex-col sm:flex-row items-center justify-center gap-4 mt-8">
        <a
          :href="downloadUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="w-full sm:w-auto px-8 py-4 rounded-xl bg-[#4A90E2] text-white font-bold text-sm shadow-md hover:bg-[#357ABD] active:translate-y-0.5 transition-all flex items-center justify-center space-x-2"
        >
          <IconDownload class="w-5 h-5" />
          <span>{{ hasDirectDownload ? '下载正式版 APK' : '下载正式版 APK (Releases)' }}</span>
        </a>
        <a
          :href="links.releases"
          target="_blank"
          rel="noopener noreferrer"
          class="w-full sm:w-auto px-6 py-4 rounded-xl border border-slate-300 dark:border-slate-700 bg-white/50 dark:bg-slate-800/50 text-slate-800 dark:text-slate-200 font-semibold text-sm hover:bg-white dark:hover:bg-slate-700 transition-all flex items-center justify-center space-x-2"
        >
          <span>查看完整更新日志</span>
          <span>→</span>
        </a>
      </div>

      <div class="mt-6 text-[11px] text-slate-400 dark:text-slate-500 leading-relaxed max-w-lg mx-auto">
        提供 universal 通用包及 arm64-v8a / armeabi-v7a / x86_64 细分架构包。
        安装时如提示「未知来源应用」，选择允许来自浏览器或本来源的安装即可。
      </div>

      <div
        v-if="changelog.length"
        class="mt-6 pt-4 border-t border-slate-200/60 dark:border-slate-800 text-left text-xs text-slate-600 dark:text-slate-400 bg-slate-50/50 dark:bg-slate-900/30 p-3.5 rounded-xl"
      >
        <div class="font-bold text-slate-800 dark:text-slate-200 mb-1">近期重要更新：</div>
        <ul class="list-disc list-inside space-y-1 text-[11px]">
          <li v-for="(line, index) in changelog" :key="index">{{ line }}</li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup>
import { links, site } from '../config/site'
import { useRelease } from '../composables/useRelease'
import IconDownload from './icons/IconDownload.vue'

const {
  source,
  versionName,
  sizeText,
  updatedText,
  changelog,
  downloadUrl,
  hasDirectDownload,
} = useRelease()
</script>
