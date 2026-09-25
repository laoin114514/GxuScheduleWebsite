import { request } from './http'

export const APP_KEY = import.meta.env.VITE_APP_KEY || 'schedule'

/**
 * GET /api/v1/apps/{appKey}/latest?versionCode=xxx
 * 官网只关心「最新版」，因此固定传 versionCode=0。
 *
 * data: {
 *   appKey, currentVersionCode, hasUpdate,
 *   latestVersionCode, latestVersionName, changelog,
 *   downloadUrl, sha256, fileSize, forced
 * }
 */
export function fetchLatestRelease() {
  return request('/api/v1/apps/' + encodeURIComponent(APP_KEY) + '/latest', {
    query: { versionCode: 0 },
    timeout: 8000,
  })
}
