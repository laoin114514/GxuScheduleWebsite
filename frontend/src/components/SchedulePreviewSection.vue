<template>
  <section id="schedule-preview" v-reveal class="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="text-center max-w-2xl mx-auto mb-10">
      <h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">同一张课表，五种样子</h2>
      <p class="mt-3 text-sm sm:text-base text-slate-600 dark:text-slate-400">
        点击下方色板，感受课程卡与界面强调色的平滑过渡。真实 App 支持深浅色自动跟随与 14 款定制背景。
      </p>

      <div class="flex items-center justify-center space-x-4 mt-6 p-2 rounded-2xl bg-white/50 dark:bg-slate-800/50 backdrop-blur-md inline-flex border border-slate-200/80 dark:border-slate-700/80">
        <span class="text-xs font-medium text-slate-500 mr-1">预设主题：</span>
        <button
          v-for="preset in presets"
          :key="preset.key"
          type="button"
          class="color-dot w-7 h-7 rounded-full"
          :class="{ active: preset.key === activeKey }"
          :style="{ backgroundColor: preset.dot }"
          :title="preset.label"
          :aria-label="preset.label"
          :aria-pressed="preset.key === activeKey"
          @click="select(preset.key)"
        ></button>
      </div>
    </div>

    <div class="relative flex items-center justify-center py-6 overflow-hidden">
      <div class="hidden md:block transform -rotate-6 scale-90 opacity-60 hover:opacity-90 transition-all duration-300 -mr-16 z-0">
        <PhoneMockup :scale="0.92" :screenshot="shot('prev')" alt="上周课表">
          <PhoneMiniWeek :data="miniPrevSchedule" />
        </PhoneMockup>
      </div>

      <div class="z-10 shadow-2xl rounded-[36px] transition-all duration-300 transform hover:scale-[1.02]">
        <PhoneMockup :width="310" :height="630" :screenshot="shot('schedule')" alt="课表周视图">
          <PhoneScheduleWeek :data="interactiveSchedule" variant="interactive" />
        </PhoneMockup>
      </div>

      <div class="hidden md:block transform rotate-6 scale-90 opacity-60 hover:opacity-90 transition-all duration-300 -ml-16 z-0">
        <PhoneMockup :scale="0.92" :screenshot="shot('next')" alt="下周课表">
          <PhoneMiniWeek :data="miniNextSchedule" />
        </PhoneMockup>
      </div>
    </div>
  </section>
</template>

<script setup>
import { shot } from '../config/site'
import { interactiveSchedule, miniNextSchedule, miniPrevSchedule } from '../config/schedule'
import { useThemePreset } from '../composables/useThemePreset'
import PhoneMockup from './PhoneMockup.vue'
import PhoneScheduleWeek from './PhoneScheduleWeek.vue'
import PhoneMiniWeek from './PhoneMiniWeek.vue'

const { presets, activeKey, select } = useThemePreset()
</script>
