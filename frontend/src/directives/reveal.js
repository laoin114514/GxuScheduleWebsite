/**
 * v-reveal：滚动进入视口时淡入上移，只触发一次。
 * 不支持 IntersectionObserver 时直接展示内容，绝不隐藏信息。
 */
export const reveal = {
  mounted(el) {
    if (typeof window === 'undefined' || !('IntersectionObserver' in window)) {
      el.classList.add('fade-in-section', 'is-visible')
      return
    }
    el.classList.add('fade-in-section')
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('is-visible')
            observer.unobserve(entry.target)
          }
        })
      },
      { threshold: 0.12, rootMargin: '0px 0px -40px 0px' }
    )
    observer.observe(el)
    el.__revealObserver = observer
  },
  unmounted(el) {
    if (el.__revealObserver) {
      el.__revealObserver.disconnect()
      el.__revealObserver = null
    }
  },
}
