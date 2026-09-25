import { ref, readonly } from 'vue'

const STORAGE_KEY = 'gxu-theme'
const isDark = ref(false)
let initialized = false

function apply(dark) {
  isDark.value = dark
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', dark)
  }
  if (typeof window !== 'undefined' && window.dispatchEvent) {
    window.dispatchEvent(new CustomEvent('gxu-theme-change', { detail: { dark: dark } }))
  }
}

function prefersDark() {
  return typeof window !== 'undefined'
    && window.matchMedia
    && window.matchMedia('(prefers-color-scheme: dark)').matches
}

/** 首次挂载：localStorage 优先，其次跟随系统 */
export function initTheme() {
  if (initialized) return
  initialized = true
  let saved = null
  try {
    saved = localStorage.getItem(STORAGE_KEY)
  } catch (err) {
    saved = null
  }
  apply(saved ? saved === 'dark' : prefersDark())

  // 未手动选择过时，跟随系统变化
  if (typeof window !== 'undefined' && window.matchMedia) {
    const mql = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = (event) => {
      let manual = null
      try {
        manual = localStorage.getItem(STORAGE_KEY)
      } catch (err) {
        manual = null
      }
      if (!manual) apply(event.matches)
    }
    if (mql.addEventListener) mql.addEventListener('change', onChange)
    else if (mql.addListener) mql.addListener(onChange)
  }
}

export function useTheme() {
  function toggle() {
    apply(!isDark.value)
    try {
      localStorage.setItem(STORAGE_KEY, isDark.value ? 'dark' : 'light')
    } catch (err) {
      /* 隐私模式下忽略 */
    }
  }

  return { isDark: readonly(isDark), toggle: toggle }
}
