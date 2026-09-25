import { computed, onMounted, onUnmounted, ref } from 'vue'

/**
 * 下课倒计时：每秒递减，归零后回到初始值（演示用）。
 * @param {number} initialSeconds 初始剩余秒数
 */
export function useCountdown(initialSeconds) {
  const total = initialSeconds || 18 * 60 + 42
  const remaining = ref(total)
  let timer = null

  const text = computed(() => {
    const value = Math.max(remaining.value, 0)
    const mins = Math.floor(value / 60)
    const secs = value % 60
    return String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0')
  })

  onMounted(() => {
    timer = setInterval(() => {
      remaining.value = remaining.value > 0 ? remaining.value - 1 : total
    }, 1000)
  })

  onUnmounted(() => {
    if (timer) clearInterval(timer)
  })

  return { remaining: remaining, text: text }
}
