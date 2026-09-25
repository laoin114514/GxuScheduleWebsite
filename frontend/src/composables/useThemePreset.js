import { ref } from 'vue'
import { themePresets } from '../config/site'

const activeKey = ref('ocean')
let initialized = false

/** 把预设写入全局 CSS 变量，课程卡与强调色会一起平滑过渡 */
export function applyThemePreset(key) {
  const preset = themePresets.find((item) => item.key === key) || themePresets[1]
  activeKey.value = preset.key
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.style.setProperty('--course-c1', preset.colors[0])
  root.style.setProperty('--course-c2', preset.colors[1])
  root.style.setProperty('--course-c3', preset.colors[2])
  root.style.setProperty('--course-c4', preset.colors[3])
  root.style.setProperty('--course-c5', preset.colors[4])
  root.style.setProperty('--theme-active-color', preset.brand)
}

export function initThemePreset() {
  if (initialized) return
  initialized = true
  applyThemePreset(activeKey.value)
}

export function useThemePreset() {
  return {
    presets: themePresets,
    activeKey: activeKey,
    select: applyThemePreset,
  }
}
