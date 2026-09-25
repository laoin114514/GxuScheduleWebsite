<template>
  <div class="phone-mockup" :style="frameStyle">
    <div class="phone-speaker"></div>
    <div class="phone-camera"></div>
    <div class="phone-screen">
      <ScreenshotSlot :src="screenshot" :alt="alt" img-class="object-cover object-top">
        <slot />
      </ScreenshotSlot>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import ScreenshotSlot from './ScreenshotSlot.vue'

const props = defineProps({
  width: { type: [Number, String], default: 0 },
  height: { type: [Number, String], default: 0 },
  scale: { type: [Number, String], default: 0 },
  screenshot: { type: String, default: '' },
  alt: { type: String, default: '西大课栈课表界面' },
})

const frameStyle = computed(() => {
  const style = {}
  if (props.width) style.width = typeof props.width === 'number' ? props.width + 'px' : props.width
  if (props.height) style.height = typeof props.height === 'number' ? props.height + 'px' : props.height
  if (props.scale) style.transform = 'scale(' + props.scale + ')'
  return style
})
</script>
