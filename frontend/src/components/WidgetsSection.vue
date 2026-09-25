<template>
  <section id="widgets" v-reveal class="py-16 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="text-center max-w-2xl mx-auto mb-12">
      <h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">桌面小组件专区</h2>
      <p class="mt-3 text-sm sm:text-base text-slate-600 dark:text-slate-400">
        不用点开 App，抬眼就知道下节课在哪。4 款精雕细琢的 Android 原生微件。
      </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      <!-- 组件 1：下课倒计时（实时跳动） -->
      <div class="glass-card p-5 relative overflow-hidden flex flex-col justify-between border-blue-200/50 dark:border-blue-900/50">
        <div class="flex items-center justify-between text-xs text-slate-500 font-medium">
          <span>Widget · 2×1</span>
          <span class="px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300 text-[10px]">实时计算</span>
        </div>

        <ScreenshotSlot :src="shot('widgetCountdown')" alt="下课倒计时小组件" img-class="object-cover object-top rounded-xl w-full h-32">
          <div class="my-4">
            <div class="text-[11px] text-slate-500 dark:text-slate-400 font-medium">当前课程：软件工程</div>
            <div class="mt-1 flex items-baseline space-x-2">
              <span class="text-xs text-slate-600 dark:text-slate-300">距下课还有</span>
              <span class="text-3xl font-extrabold font-mono text-[#4A90E2] animate-pulse-subtle">{{ countdown }}</span>
            </div>
            <div class="mt-2 text-xs text-slate-600 dark:text-slate-400 flex items-center space-x-1.5">
              <IconLocation class="w-3.5 h-3.5 text-slate-400" />
              <span>君武馆 301 · 教室</span>
            </div>
          </div>
        </ScreenshotSlot>

        <div class="text-[11px] text-slate-400 pt-2 border-t border-slate-200/60 dark:border-slate-800">下课倒计时组件</div>
      </div>

      <!-- 组件 2：今日课程 -->
      <div class="glass-card p-5 flex flex-col justify-between">
        <div class="flex items-center justify-between text-xs text-slate-500 font-medium">
          <span>Widget · 2×2</span>
          <span class="text-[11px] font-mono text-slate-400">周三 · 3 门课</span>
        </div>

        <ScreenshotSlot :src="shot('widgetToday')" alt="今日课程小组件" img-class="object-cover object-top rounded-xl w-full h-32">
          <div class="my-3 space-y-2">
            <div class="p-2 rounded-xl bg-blue-50/80 dark:bg-blue-950/40 border border-blue-100 dark:border-blue-900/60">
              <div class="flex justify-between text-[11px] font-bold text-slate-900 dark:text-white">
                <span>08:00 软件工程</span>
                <span class="text-[9px] text-blue-600">已上完</span>
              </div>
              <div class="text-[10px] text-slate-500 mt-0.5">君武 301</div>
            </div>

            <div class="p-2 rounded-xl bg-purple-50/80 dark:bg-purple-950/40 border border-purple-100 dark:border-purple-900/60">
              <div class="flex justify-between text-[11px] font-bold text-slate-900 dark:text-white">
                <span>10:10 大学英语</span>
                <span class="text-[9px] text-purple-600 font-bold">下一节</span>
              </div>
              <div class="text-[10px] text-slate-500 mt-0.5">外国语 204 · 3-4节</div>
            </div>

            <div class="p-2 rounded-xl bg-emerald-50/80 dark:bg-emerald-950/40 border border-emerald-100 dark:border-emerald-900/60">
              <div class="flex justify-between text-[11px] font-bold text-slate-900 dark:text-white">
                <span>14:30 算法设计</span>
                <span class="text-[9px] text-slate-400">未开始</span>
              </div>
              <div class="text-[10px] text-slate-500 mt-0.5">计电 105 · 5-6节</div>
            </div>
          </div>
        </ScreenshotSlot>

        <div class="text-[11px] text-slate-400 pt-2 border-t border-slate-200/60 dark:border-slate-800">今日课程概览组件</div>
      </div>

      <!-- 组件 3：近日课程 -->
      <div class="glass-card p-5 flex flex-col justify-between">
        <div class="flex items-center justify-between text-xs text-slate-500 font-medium">
          <span>Widget · 4×1</span>
          <span class="text-[10px] text-slate-400">紧凑条形</span>
        </div>

        <ScreenshotSlot :src="shot('widgetUpcoming')" alt="近日课程小组件" img-class="object-cover object-top rounded-xl w-full h-32">
          <div class="my-4">
            <div class="text-xs font-semibold text-slate-900 dark:text-white flex items-center justify-between">
              <span>下一门课提醒</span>
              <span class="text-[10px] px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 text-amber-800 dark:text-amber-200">20 分钟后</span>
            </div>
            <div class="mt-2 text-sm font-bold text-slate-900 dark:text-white">大学英语(IV)</div>
            <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">10:10 ~ 11:50 · 外国语楼 204</div>
          </div>
        </ScreenshotSlot>

        <div class="text-[11px] text-slate-400 pt-2 border-t border-slate-200/60 dark:border-slate-800">近日课程轻量条</div>
      </div>

      <!-- 组件 4：一周课程全览 -->
      <div class="glass-card p-5 flex flex-col justify-between">
        <div class="flex items-center justify-between text-xs text-slate-500 font-medium">
          <span>Widget · 4×4</span>
          <span class="text-[10px] text-slate-400">全览小窗</span>
        </div>

        <ScreenshotSlot :src="shot('widgetWeek')" alt="一周课程小组件" img-class="object-cover object-top rounded-xl w-full h-32">
          <div class="my-3">
            <div class="text-xs font-bold text-slate-900 dark:text-white mb-2">周视图微缩地图</div>
            <div class="grid grid-cols-5 gap-1 text-[9px] text-center font-mono">
              <div class="p-1 rounded bg-blue-100/70 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200">一<br />2节</div>
              <div class="p-1 rounded bg-blue-100/70 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200">二<br />3节</div>
              <div class="p-1 rounded bg-blue-500 text-white font-bold">三<br />3节</div>
              <div class="p-1 rounded bg-blue-100/70 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200">四<br />2节</div>
              <div class="p-1 rounded bg-blue-100/70 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200">五<br />1节</div>
            </div>
            <p class="text-[11px] text-slate-500 mt-3 leading-relaxed">直接在主屏幕拖拽滑动，双指还可局部缩放课格密度。</p>
          </div>
        </ScreenshotSlot>

        <div class="text-[11px] text-slate-400 pt-2 border-t border-slate-200/60 dark:border-slate-800">一周课程微缩矩阵</div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { shot } from '../config/site'
import { useCountdown } from '../composables/useCountdown'
import ScreenshotSlot from './ScreenshotSlot.vue'
import IconLocation from './icons/IconLocation.vue'

const { text: countdown } = useCountdown(18 * 60 + 42)
</script>
