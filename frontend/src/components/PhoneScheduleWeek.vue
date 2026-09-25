<template>
  <!-- 按 App「周课表」页 1:1 还原：360dp 逻辑画布 + transform 缩放到手机模型宽度 -->
  <div ref="host" class="app-screen-host">
    <div class="app-screen" :style="screenStyle">
      <!-- 顶栏 -->
      <div class="app-header">
        <div class="app-header-main">
          <div class="app-week-info">{{ week.weekInfo }}</div>
          <div class="app-date">{{ week.date }}</div>
        </div>
        <div class="app-circle-btn"><IconRefresh /></div>
        <div class="app-circle-btn"><IconDotsVertical /></div>
      </div>

      <!-- 日期栏：周几 + 日期 -->
      <div class="app-daterow">
        <div class="app-year-cell"><span class="app-year-label">年</span></div>
        <div v-for="day in week.days" :key="'w' + day.label" class="app-weekday">{{ day.label }}</div>
      </div>
      <div class="app-daterow">
        <div class="app-year-cell"><span class="app-year-value">{{ week.year }}</span></div>
        <div v-for="day in week.days" :key="'d' + day.label" class="app-date-cell">
          <span class="app-date-num" :class="{ 'is-today': day.today }">{{ day.date }}</span>
        </div>
      </div>

      <!-- 课表网格：节次轴 + 7 天 -->
      <div class="app-grid">
        <div class="app-axis">
          <div v-for="period in periods" :key="period.node" class="app-axis-cell">
            <span class="app-axis-node">{{ period.node }}</span>
            <span class="app-axis-time">{{ period.start }}</span>
            <span class="app-axis-time">{{ period.end }}</span>
          </div>
        </div>

        <div class="app-days">
          <!-- 虚线网格 -->
          <div v-for="column in 7" :key="'c' + column" class="app-day">
            <div v-for="period in periods" :key="'g' + period.node" class="app-cell"></div>
          </div>

          <!-- 课程卡片 -->
          <div
            v-for="(course, index) in week.courses"
            :key="'k' + index"
            class="app-course"
            :style="cardStyle(course)"
          >
            <span>{{ courseText(course) }}</span>
            <span v-if="course.overlap" class="app-overlap">+{{ course.overlap }}</span>
          </div>
        </div>
      </div>

      <!-- 底部导航 -->
      <nav v-if="variant !== 'plain'" class="app-nav">
        <div
          v-for="tab in tabs"
          :key="tab.key"
          class="app-nav-item"
          :class="{ 'is-active': tab.key === 'schedule' }"
        >
          <span class="app-nav-holder">
            <span class="app-nav-circle"></span>
            <component :is="tab.icon" class="app-nav-icon" />
          </span>
          <span class="app-nav-label">{{ tab.label }}</span>
        </div>
      </nav>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { layout, periods } from '../config/schedule'
import IconCalendar from './icons/IconCalendar.vue'
import IconGridNav from './icons/IconGridNav.vue'
import IconUserNav from './icons/IconUserNav.vue'
import IconRefresh from './icons/IconRefresh.vue'
import IconDotsVertical from './icons/IconDotsVertical.vue'

const props = defineProps({
  week: { type: Object, required: true },
  /** full：带底部导航；plain：只保留课表（左右两侧的辅助手机用） */
  variant: { type: String, default: 'full' },
})

const tabs = [
  { key: 'schedule', label: '课表', icon: IconCalendar },
  { key: 'tools', label: '工具', icon: IconGridNav },
  { key: 'profile', label: '我的', icon: IconUserNav },
]

const host = ref(null)
const scale = ref(1)
const hostHeight = ref(0)
let observer = null

function measure() {
  if (!host.value) return
  // 必须用 offsetWidth/offsetHeight（布局尺寸，不受祖先 transform 影响）：
  // 左右两侧的辅助手机带 rotate/scale，用 getBoundingClientRect 会被旋转后的包围盒带偏。
  const width = host.value.offsetWidth
  const height = host.value.offsetHeight
  if (width > 0) scale.value = width / layout.baseWidth
  hostHeight.value = height
}

onMounted(() => {
  measure()
  if (typeof ResizeObserver !== 'undefined') {
    observer = new ResizeObserver(measure)
    observer.observe(host.value)
  } else {
    window.addEventListener('resize', measure)
  }
})

onUnmounted(() => {
  if (observer) observer.disconnect()
  else window.removeEventListener('resize', measure)
})

const screenStyle = computed(() => {
  const s = scale.value > 0 ? scale.value : 1
  const height = hostHeight.value > 0 ? hostHeight.value / s : 1000
  return { transform: 'scale(' + s + ')', height: height + 'px' }
})

/** 与 WeekPagerAdapter 一致：按节次定位，宽 = 1/7 列宽 - 间距，高 = 节数 * 格子高 - 间距 */
function cardStyle(course) {
  const columnWidth = 100 / 7
  const gap = layout.gap
  const top = (course.start - 1) * layout.cellHeight + gap
  const height = (course.end - course.start + 1) * layout.cellHeight - gap * 2
  return {
    left: (course.day - 1) * columnWidth + '%',
    width: 'calc(' + columnWidth + '% - ' + gap * 2 + 'px)',
    top: top + 'px',
    height: height + 'px',
    backgroundColor: 'rgb(var(--course-c' + course.color + ') / var(--course-alpha))',
  }
}

/** App 卡片文案：课程名 + 教室 + 教师，换行居中 */
function courseText(course) {
  return [course.name, course.room, course.teacher].filter(Boolean).join('\n')
}
</script>
