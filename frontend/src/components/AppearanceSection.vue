<template>
  <section v-reveal class="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="text-center max-w-2xl mx-auto mb-12">
      <h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">日间清爽 · 夜间护眼</h2>
      <p class="mt-3 text-sm sm:text-base text-slate-600 dark:text-slate-400">
        支持系统跟随或定时自动切换，夜晚查课不刺眼。字体五档等比缩放、格子高度自由调。
      </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-8 items-center">
      <!-- 浅色方案 -->
      <div class="p-6 rounded-2xl bg-gradient-to-b from-[#DAE6F0] to-[#F3F6FA] border border-white/80 shadow-md">
        <div class="flex items-center justify-between pb-4 border-b border-slate-300/50">
          <span class="text-xs font-bold text-slate-700">浅色模式 (07:00 ~ 22:00)</span>
          <span class="text-[10px] text-blue-700 font-mono px-2 py-0.5 rounded bg-blue-100/80">晨光清爽</span>
        </div>

        <ScreenshotSlot
          :src="shot('appearanceLight')"
          alt="浅色模式课表"
          img-class="w-full h-28 object-cover rounded-xl mt-4"
        >
          <div class="p-4 mt-4 bg-white/70 rounded-xl space-y-2 border border-white/60">
            <div class="flex justify-between items-center text-xs font-bold text-slate-800">
              <span>08:00 高等数学</span>
              <span class="text-[10px] text-blue-600">君武 201</span>
            </div>
            <div class="text-[11px] text-slate-500">冷调浅灰白底底色，纯净高对比度，阳光下清晰可读。</div>
          </div>
        </ScreenshotSlot>
      </div>

      <!-- 深色方案 -->
      <div class="p-6 rounded-2xl bg-gradient-to-b from-[#1A2433] to-[#0D1117] border border-slate-700/80 shadow-md text-white">
        <div class="flex items-center justify-between pb-4 border-b border-slate-700/60">
          <span class="text-xs font-bold text-slate-200">深色模式 (22:00 ~ 07:00)</span>
          <span class="text-[10px] text-cyan-300 font-mono px-2 py-0.5 rounded bg-cyan-950/80">夜间舒适</span>
        </div>

        <ScreenshotSlot
          :src="shot('appearanceDark')"
          alt="深色模式课表"
          img-class="w-full h-28 object-cover rounded-xl mt-4"
        >
          <div class="p-4 mt-4 bg-[#151E2C]/90 rounded-xl space-y-2 border border-slate-700/50">
            <div class="flex justify-between items-center text-xs font-bold text-slate-100">
              <span>明早 08:00 高等数学</span>
              <span class="text-[10px] text-blue-400">智能闹钟已备好</span>
            </div>
            <div class="text-[11px] text-slate-400">深灰蓝基底配半透明色块，深夜熄灯后看课不刺眼。</div>
          </div>
        </ScreenshotSlot>
      </div>
    </div>

    <div class="mt-8 text-center text-xs text-slate-500 dark:text-slate-400">
      <span class="font-medium text-slate-700 dark:text-slate-300">个性化调节：</span>
      字体大小五档无级缩放 · 课程格子高度 40~120dp 自定义 · 14 款官方定制冷暖背景底色
    </div>
  </section>
</template>

<script setup>
import { shot } from '../config/site'
import ScreenshotSlot from './ScreenshotSlot.vue'
</script>
