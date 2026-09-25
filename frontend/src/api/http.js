/**
 * 极简 fetch 封装，对接 Go 更新服务统一响应体：
 *   { "code": 0, "message": "ok", "data": {...} }
 * 非 0 的 code 或非 2xx 状态统一抛 ApiError。
 */
const RAW_BASE = (import.meta.env.VITE_API_BASE_URL || '').trim()

// 没配 VITE_API_BASE_URL 时，用「相对基址」而不是写死 /api：
// 官网挂在子路径下（例如 https://host/schedule/）时，'./api/v1/...' 会解析成
// https://host/schedule/api/v1/...，与反向代理的挂载点一致；挂在根路径时同样正确。
const FALLBACK_BASE = import.meta.env.BASE_URL || '/'

export const API_BASE = (RAW_BASE || FALLBACK_BASE).replace(/\/+$/, '')

export class ApiError extends Error {
  constructor(message, status, code) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

function toQuery(query) {
  if (!query) return ''
  const parts = []
  Object.keys(query).forEach((key) => {
    const value = query[key]
    if (value === undefined || value === null || value === '') return
    parts.push(encodeURIComponent(key) + '=' + encodeURIComponent(String(value)))
  })
  return parts.length ? '?' + parts.join('&') : ''
}

export async function request(path, options) {
  const opts = options || {}
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), opts.timeout || 8000)
  try {
    const response = await fetch(API_BASE + path + toQuery(opts.query), {
      method: opts.method || 'GET',
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    })
    const text = await response.text()
    let body = null
    if (text) {
      try {
        body = JSON.parse(text)
      } catch (err) {
        throw new ApiError('接口返回的不是合法 JSON', response.status)
      }
    }
    if (!response.ok) {
      throw new ApiError((body && body.message) || 'HTTP ' + response.status, response.status, body && body.code)
    }
    if (body && typeof body.code === 'number' && body.code !== 0) {
      throw new ApiError(body.message || '接口返回错误', response.status, body.code)
    }
    return body ? body.data : null
  } catch (err) {
    if (err && err.name === 'AbortError') {
      throw new ApiError('请求超时', 0)
    }
    throw err
  } finally {
    clearTimeout(timer)
  }
}
