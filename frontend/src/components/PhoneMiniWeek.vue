<template>
  <div class="p-3 pt-8 text-[11px] h-full flex flex-col justify-between">
    <div class="text-center font-bold text-slate-700 dark:text-slate-300">{{ data.title }}</div>
    <div class="grid grid-cols-4 gap-1.5 my-3">
      <div
        v-for="(card, index) in data.cards"
        :key="index"
        class="course-card-m3 text-white"
        :style="{ backgroundColor: card.hex || 'var(--course-c' + card.c + ')', height: card.h + 'px' }"
      >
        <div class="font-bold text-[10px]">{{ card.title }}</div>
        <div v-if="card.room" class="text-[8px] opacity-80 mt-1">{{ card.room }}</div>
      </div>
    </div>
    <div class="text-center text-[10px] text-slate-400">{{ data.footer }}</div>
  </div>
</template>

<script setup>
defineProps({
  data: { type: Object, required: true },
})
</script>
