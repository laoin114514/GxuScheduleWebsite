<template>
  <section id="hero" class="relative pt-12 pb-20 md:py-24 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-12 lg:gap-8 items-center">
      <div class="lg:col-span-7 flex flex-col items-start space-y-6">
        <div class="inline-flex items-center space-x-2 px-3.5 py-1.5 rounded-full bg-blue-50/90 dark:bg-blue-950/50 border border-blue-200/80 dark:border-blue-800/80 text-xs font-medium text-blue-700 dark:text-blue-300">
          <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
          <span>{{ hero.badges[0] }}</span>
          <span class="text-blue-300 dark:text-blue-700">|</span>
          <span>{{ hero.badges[1] }}</span>
        </div>

        <h1 class="text-3xl sm:text-4xl lg:text-5xl font-extrabold tracking-tight text-slate-900 dark:text-white leading-[1.25]">
          {{ hero.titleLead }}<br class="hidden sm:inline" />
          <span class="bg-gradient-to-r from-[#4A90E2] to-[#1565C0] bg-clip-text text-transparent">{{ hero.titleHighlight }}</span>
        </h1>

        <p class="text-base sm:text-lg text-slate-600 dark:text-slate-300 leading-relaxed max-w-xl">
          {{ hero.subtitle }}
        </p>

        <div class="flex flex-wrap items-center gap-4 pt-2">
          <a
            href="#download"
            class="px-6 py-3.5 rounded-xl bg-[#4A90E2] text-white font-semibold text-sm shadow-md hover:bg-[#357ABD] hover:shadow-lg transition-all flex items-center space-x-2 active:scale-95"
          >
            <IconDownload class="w-5 h-5" />
            <span>{{ hero.primaryCta }}</span>
          </a>
          <a
            :href="links.repo"
            target="_blank"
            rel="noopener noreferrer"
            class="px-5 py-3.5 rounded-xl border border-slate-300 dark:border-slate-700 bg-white/60 dark:bg-slate-800/60 text-slate-800 dark:text-slate-200 font-semibold text-sm hover:bg-white dark:hover:bg-slate-700/80 transition-all flex items-center space-x-2"
          >
            <IconGithub class="w-5 h-5" />
            <span>{{ hero.secondaryCta }}</span>
          </a>
        </div>

        <div class="text-xs text-slate-500 dark:text-slate-400 flex items-center flex-wrap gap-x-2 gap-y-1 pt-1">
          <span class="inline-block w-1.5 h-1.5 rounded-full bg-blue-500"></span>
          <template v-for="(item, index) in hero.compatibility" :key="item">
            <span>{{ item }}</span>
            <span v-if="index < hero.compatibility.length - 1">·</span>
          </template>
        </div>

        <div class="grid grid-cols-3 gap-4 pt-4 border-t border-slate-200/80 dark:border-slate-800/80 w-full max-w-lg">
          <div v-for="stat in hero.stats" :key="stat.label">
            <div class="text-2xl font-bold text-[#4A90E2]">{{ stat.value }}</div>
            <div class="text-xs text-slate-500 dark:text-slate-400">{{ stat.label }}</div>
          </div>
        </div>
      </div>

      <div class="lg:col-span-5 flex justify-center relative">
        <div class="absolute -inset-4 bg-gradient-to-r from-blue-400/20 to-sky-300/20 rounded-full blur-2xl -z-10 dark:from-blue-600/10 dark:to-cyan-500/10"></div>
        <PhoneMockup :screenshot="shot('hero')" alt="西大课栈周课表">
          <PhoneScheduleWeek :week="weeks.current" />
        </PhoneMockup>
      </div>
    </div>
  </section>
</template>

<script setup>
import { hero, links, shot } from '../config/site'
import { weeks } from '../config/schedule'
import PhoneMockup from './PhoneMockup.vue'
import PhoneScheduleWeek from './PhoneScheduleWeek.vue'
import IconDownload from './icons/IconDownload.vue'
import IconGithub from './icons/IconGithub.vue'
</script>
