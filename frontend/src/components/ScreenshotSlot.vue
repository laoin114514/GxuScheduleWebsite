<template>
  <!-- 有真实截图就渲染图片，否则回退到内置的 HTML 还原界面 -->
  <img
    v-if="src"
    :src="src"
    :alt="alt"
    class="w-full h-full"
    :class="imgClass"
    loading="lazy"
    decoding="async"
  />
  <slot v-else />
</template>

<script setup>
defineProps({
  src: { type: String, default: '' },
  alt: { type: String, default: '西大课栈界面截图' },
  imgClass: { type: String, default: 'object-cover object-top' },
})
</script>
