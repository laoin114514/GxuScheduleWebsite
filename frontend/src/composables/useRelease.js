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

/** 后端发版时间（RFC3339 / MySQL 时间戳）-> YYYY-MM-DD；解析不了就原样返回 */
function formatDate(value) {
  if (!value) return ''
  const text = String(value)
  const date = new Date(text)
  if (Number.isNaN(date.getTime())) return text
  const pad = (n) => String(n).padStart(2, '0')
  return date.getFullYear() + '-' + pad(date.getMonth() + 1) + '-' + pad(date.getDate())
}

/**
 * 最新版本信息：优先走更新服务接口
 *   GET /api/v1/apps/{appKey}/latest?versionCode=0
 *
 * 三种数据来源（组件根节点上有 data-release-source 便于排查）：
 *   api       接口通了，且拿到了发布记录 —— 版本号 / 体积 / 更新日志 / 下载直链全部以接口为准
 *   empty     接口通了，但这个 appKey 还没发过版 —— 退回内置信息，避免页面开天窗
 *   fallback  接口不通（离线 / 未部署 / 超时）—— 同样退回内置信息
 *
 * 注意：接口活着时，接口没给的字段（如未填 changelog、未返回 publishedAt）
 * 就【不展示】，不会拿内置数据去凑，避免版本号和时间对不上。
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
      // 生产环境也留一条日志，方便运维在控制台定位「为什么下载按钮跳到了 Releases」；
      // 更完整的体检用 npm run check:api
      // eslint-disable-next-line no-console
      console.warn(
        '[西大课栈] 更新服务不可用，下载按钮回退到 GitHub Releases：',
        (err && err.message) || err
      )
    } finally {
      loading.value = false
    }
  })

  /** 只有真正读到发布记录时才用接口数据（latestVersionCode > 0） */
  const apiData = computed(() => {
    if (data.value && Number(data.value.latestVersionCode) > 0) return data.value
    return null
  })

  const source = computed(() => {
    if (apiData.value) return 'api'
    if (data.value) return 'empty'
    return 'fallback'
  })

  /** 按钮来源说明，挂在 title 上，平时看不见、排查时有用 */
  const sourceHint = computed(() => {
    if (source.value === 'api') return '安装包来自更新服务'
    if (source.value === 'empty') return '更新服务里还没有发布记录，跳转 GitHub Releases'
    return '更新服务不可用，跳转 GitHub Releases'
  })

  const versionName = computed(() =>
    apiData.value ? apiData.value.latestVersionName : fallbackRelease.versionName
  )

  const versionCode = computed(() =>
    apiData.value ? apiData.value.latestVersionCode : fallbackRelease.versionCode
  )

  const sizeText = computed(() => {
    const text = formatSize(apiData.value ? apiData.value.fileSize : fallbackRelease.sizeBytes)
    return text ? '大小约 ' + text : ''
  })

  const updatedText = computed(() => {
    const date = apiData.value
      ? formatDate(apiData.value.publishedAt)
      : fallbackRelease.updatedAt
    return date ? '更新日期：' + date : ''
  })

  const changelog = computed(() =>
    apiData.value ? parseChangelog(apiData.value.changelog) : fallbackRelease.changelog
  )

  const rawDownloadUrl = computed(() =>
    apiData.value ? String(apiData.value.downloadUrl || '') : ''
  )

  /** 有直链就用直链（OSS / CDN），否则落到 GitHub Releases 页 */
  const downloadUrl = computed(() =>
    /^https?:\/\//i.test(rawDownloadUrl.value) ? rawDownloadUrl.value : links.releases
  )

  const hasDirectDownload = computed(() => /^https?:\/\//i.test(rawDownloadUrl.value))

  const fileName = computed(() => (apiData.value ? apiData.value.fileName || '' : ''))

  return {
    loading: loading,
    error: error,
    source: source,
    sourceHint: sourceHint,
    versionName: versionName,
    versionCode: versionCode,
    sizeText: sizeText,
    updatedText: updatedText,
    changelog: changelog,
    downloadUrl: downloadUrl,
    hasDirectDownload: hasDirectDownload,
    fileName: fileName,
  }
}
