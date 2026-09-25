import { ref } from 'vue'
import { themePresets } from '../config/site'

const activeKey = ref('ocean')
let initialized = false

/** #RRGGBB -> "r g b"，用于 rgb(var(--x) / alpha) 形式叠加透明度 */
function hexToChannels(hex) {
  const raw = String(hex).replace('#', '')
  const full = raw.length === 3
    ? raw.split('').map((c) => c + c).join('')
    : raw
  const value = parseInt(full, 16)
  return ((value >> 16) & 255) + ' ' + ((value >> 8) & 255) + ' ' + (value & 255)
}

/**
 * 把预设写入全局 CSS 变量：
 *   --course-c1..c5  课程卡配色（存 r g b，配合 --course-alpha 做半透明）
 *   --app-primary*   界面强调色（顶栏日期、今日高亮、底部导航选中）
 * 全部是 CSS 变量，切换时整页平滑过渡，不需要重新渲染组件。
 */
export function applyThemePreset(key) {
  const preset = themePresets.find((item) => item.key === key) || themePresets[1]
  activeKey.value = preset.key
  if (typeof document === 'undefined') return
  const root = document.documentElement
  preset.colors.forEach((hex, index) => {
    root.style.setProperty('--course-c' + (index + 1), hexToChannels(hex))
  })
  root.style.setProperty('--app-primary-light', preset.brand)
  root.style.setProperty('--app-primary-dark', preset.brandDark)
  root.style.setProperty('--app-on-primary-dark', preset.onPrimaryDark)
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
