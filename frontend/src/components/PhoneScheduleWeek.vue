<template>
  <div class="p-4 pt-9 text-[12px] h-full flex flex-col justify-between" :class="variant === 'hero' ? 'p-3.5' : ''">
    <div>
      <div class="flex items-center justify-between pb-2 border-b border-slate-200/70 dark:border-slate-800">
        <div class="flex items-center space-x-1.5">
          <span class="font-bold text-sm text-slate-900 dark:text-white">{{ data.title }}</span>
          <span class="text-[10px] px-1.5 py-0.5 rounded bg-blue-100 dark:bg-blue-900/60 text-blue-700 dark:text-blue-300 font-medium">
            {{ data.badge }}
          </span>
        </div>
        <div v-if="variant === 'hero'" class="flex items-center space-x-2 text-slate-500">
          <span class="text-[11px] font-mono">{{ data.meta }}</span>
          <IconDotsVertical class="w-3.5 h-3.5" />
        </div>
        <span v-else class="text-[10px] text-emerald-600 dark:text-emerald-400 font-mono">{{ data.connected }}</span>
      </div>

      <div class="grid grid-cols-5 text-center py-2 text-[10px] text-slate-500 font-medium border-b border-slate-200/50 dark:border-slate-800/60">
        <div v-for="day in data.weekdays" :key="day.label" :class="day.today ? 'text-blue-600 dark:text-blue-400 font-bold' : ''">
          {{ day.label }}<br />
          <span class="text-[9px]" :class="day.today ? '' : 'text-slate-400'">{{ day.date }}</span>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-5 gap-1.5 my-2 flex-grow overflow-hidden relative">
      <div
        v-for="(column, columnIndex) in data.columns"
        :key="columnIndex"
        class="space-y-1.5"
        :class="columnIndex === todayIndex ? todayClass : ''"
      >
        <template v-for="(block, blockIndex) in column" :key="blockIndex">
          <div v-if="block.empty" :style="{ height: block.h + 'px' }"></div>
          <div
            v-else
            class="course-card-m3 text-white transition-all duration-300 relative"
            :class="block.ring ? 'shadow-sm ring-1 ring-white/40' : ''"
            :style="cardStyle(block)"
          >
            <span
              v-if="block.overlap"
              class="absolute top-1 right-1 rounded-full bg-amber-400 ring-2 ring-white text-slate-900 font-bold flex items-center justify-center"
              :class="variant === 'hero' ? 'w-2.5 h-2.5 text-[7px]' : 'w-3 h-3 text-[7.5px]'"
            >{{ block.overlap }}</span>

            <div class="flex items-center justify-between">
              <span class="font-bold text-[10.5px]">{{ block.title }}</span>
              <span v-if="block.badge" class="text-[7.5px] px-1 bg-white/20 rounded">{{ block.badge }}</span>
            </div>
            <div v-if="block.room" class="text-[9px] opacity-90 mt-1">{{ block.room }}</div>
            <div v-if="block.slot" class="text-[8px] opacity-75 mt-0.5">{{ block.slot }}</div>
          </div>
        </template>
      </div>
    </div>

    <div v-if="variant === 'hero'" class="pt-2 border-t border-slate-200/50 dark:border-slate-800/80 flex items-center justify-around text-slate-400 text-[10px]">
      <div class="text-[#4A90E2] font-semibold flex flex-col items-center">
        <IconGridNav class="w-4 h-4" /><span>课表</span>
      </div>
      <div class="flex flex-col items-center"><IconToolsNav class="w-4 h-4" /><span>工具</span></div>
      <div class="flex flex-col items-center"><IconUserNav class="w-4 h-4" /><span>我的</span></div>
    </div>
    <div v-else class="pt-2 text-center text-[10px] text-slate-400">{{ data.footer }}</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import IconDotsVertical from './icons/IconDotsVertical.vue'
import IconGridNav from './icons/IconGridNav.vue'
import IconToolsNav from './icons/IconToolsNav.vue'
import IconUserNav from './icons/IconUserNav.vue'

const props = defineProps({
  data: { type: Object, required: true },
  /** hero：首屏手机；interactive：课表展示区中间那台 */
  variant: { type: String, default: 'hero' },
})

const todayIndex = computed(() => props.data.weekdays.findIndex((day) => day.today))

const todayClass = computed(() =>
  props.variant === 'hero'
    ? 'bg-blue-50/40 dark:bg-blue-900/10 rounded-lg p-0.5'
    : 'bg-blue-50/50 dark:bg-blue-900/20 rounded-lg p-0.5'
)

function cardStyle(block) {
  return {
    backgroundColor: block.hex || 'var(--course-c' + block.c + ')',
    height: block.h + 'px',
  }
}
</script>
