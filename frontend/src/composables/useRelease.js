import { computed, onMounted, ref } from 'vue'
import { fetchLatestRelease } from '../api/release'
import { fallbackRelease, links } from '../config/site'

function formatSize(bytes) {
  if (!bytes || bytes <= 0) return ''
  const mb = bytes / 1048576
  if (mb >= 1) return mb.toFixed(1) + ' MB'
  return Math.max(1, Math.round(bytes / 1024)) + ' KB'
}

function parseChangelog(raw) {
  if (!raw) return []
  return String(raw)
    .split('\n')
    .map((line) => line.replace(/^\s*[-*•]\s*/, '').trim())
    .filter((line) => line.length > 0)
    .slice(0, 6)
}

/**
 * 最新版本信息：优先读更新服务接口，失败时回退到静态数据，
 * 保证官网在任何情况下都不会开天窗。
 */
export function useRelease() {
  const data = ref(null)
  const loading = ref(true)
  const error = ref(null)

  onMounted(async () => {
    try {
      data.value = await fetchLatestRelease()
      error.value = null
    } catch (err) {
      error.value = err
      data.value = null
    } finally {
      loading.value = false
    }
  })

  const source = computed(() => (data.value && data.value.latestVersionName ? 'api' : 'fallback'))

  const versionName = computed(() => {
    if (data.value && data.value.latestVersionName) return data.value.latestVersionName
    return fallbackRelease.versionName
  })

  const versionCode = computed(() => {
    if (data.value && data.value.latestVersionCode) return data.value.latestVersionCode
    return fallbackRelease.versionCode
  })

  const sizeText = computed(() => {
    const fromApi = data.value ? formatSize(data.value.fileSize) : ''
    if (fromApi) return '大小约 ' + fromApi
    const fromFallback = formatSize(fallbackRelease.sizeBytes)
    return fromFallback ? '大小约 ' + fromFallback : ''
  })

  const updatedText = computed(() => {
    const date = (data.value && data.value.updatedAt) || fallbackRelease.updatedAt
    return date ? '更新日期：' + date : ''
  })

  const changelog = computed(() => {
    const fromApi = data.value ? parseChangelog(data.value.changelog) : []
    return fromApi.length ? fromApi : fallbackRelease.changelog
  })

  const downloadUrl = computed(() => {
    if (data.value && data.value.downloadUrl) return data.value.downloadUrl
    return links.releases
  })

  const hasDirectDownload = computed(() => Boolean(data.value && data.value.downloadUrl))

  return {
    loading: loading,
    error: error,
    source: source,
    versionName: versionName,
    versionCode: versionCode,
    sizeText: sizeText,
    updatedText: updatedText,
    changelog: changelog,
    downloadUrl: downloadUrl,
    hasDirectDownload: hasDirectDownload,
  }
}
