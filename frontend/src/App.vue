<template>
  <div
    class="grid-bg-pattern min-h-screen relative overflow-x-hidden selection:bg-blue-100 selection:text-blue-900 dark:selection:bg-blue-950 dark:selection:text-blue-200"
  >
    <SiteNavBar />
    <main>
      <HeroSection />
      <FeaturesSection />
      <SchedulePreviewSection />
      <WidgetsSection />
      <ToolsSection />
      <AppearanceSection />
      <OpenSourceSection />
      <DownloadSection />
      <FaqSection />
    </main>
    <SiteFooter />
  </div>
</template>

<script setup>
import SiteNavBar from './components/SiteNavBar.vue'
import HeroSection from './components/HeroSection.vue'
import FeaturesSection from './components/FeaturesSection.vue'
import SchedulePreviewSection from './components/SchedulePreviewSection.vue'
import WidgetsSection from './components/WidgetsSection.vue'
import ToolsSection from './components/ToolsSection.vue'
import AppearanceSection from './components/AppearanceSection.vue'
import OpenSourceSection from './components/OpenSourceSection.vue'
import DownloadSection from './components/DownloadSection.vue'
import FaqSection from './components/FaqSection.vue'
import SiteFooter from './components/SiteFooter.vue'
</script>
