#!/usr/bin/env node
/**
 * 检查「下载西大课栈」按钮到底会跳到哪 —— 即官网能不能从后端拿到下载直链。
 *
 * 用法：
 *   npm run check:api
 *   npm run check:api -- --base=https://update.example.com --app-key=schedule
 *   npm run check:api -- --origin=https://laoin114514.github.io
 *
 * 退出码：0 = 接口可用（api）；1 = 其它情况（empty / 不可达 / 跨域被拦 / 不是 JSON）。
 */
import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')

// ---------- 读取 .env / .env.local（不覆盖已有的 process.env） ----------
function loadEnvFile(name) {
  const file = resolve(root, name)
  if (!existsSync(file)) return
  for (const rawLine of readFileSync(file, 'utf8').split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) continue
    const eq = line.indexOf('=')
    if (eq <= 0) continue
    const key = line.slice(0, eq).trim()
    let value = line.slice(eq + 1).trim()
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1)
    }
    if (!(key in process.env)) process.env[key] = value
  }
}
loadEnvFile('.env')
loadEnvFile('.env.local')

// ---------- 命令行参数 ----------
const args = {}
for (const arg of process.argv.slice(2)) {
  const m = /^--([^=]+)(?:=(.*))?$/.exec(arg)
  if (m) args[m[1]] = m[2] === undefined ? 'true' : m[2]
}

const base = String(args.base ?? process.env.VITE_API_BASE_URL ?? '').replace(/\/+$/, '')
const appKey = String(args['app-key'] ?? process.env.VITE_APP_KEY ?? 'schedule')
const origin = String(args.origin ?? 'https://laoin114514.github.io')
const url = base + '/api/v1/apps/' + encodeURIComponent(appKey) + '/latest?versionCode=0'

const ok = (t) => console.log('  \u2713 ' + t)
const bad = (t) => console.log('  \u2717 ' + t)
const info = (t) => console.log('    ' + t)

console.log('')
console.log('检查更新服务：' + url)
console.log('  appKey = ' + appKey + '   |   模拟来源 Origin = ' + origin)
console.log('')

if (!base) {
  console.log('VITE_API_BASE_URL 为空 —— 官网会请求同源 /api/v1/...（生产环境应由 Nginx 把 /api 反代到 Go 服务）。')
  console.log('命令行体检需要一个绝对地址，请补上 --base=https://你的更新服务地址 再跑一次，例如：')
  console.log('')
  console.log('  npm run check:api -- --base=http://localhost:8080')
  console.log('  npm run check:api -- --base=https://update.example.com')
  console.log('')
  process.exit(1)
}

function fail(headline, hints) {
  console.log('结论：' + headline)
  if (hints && hints.length) {
    console.log('')
    console.log('可能的原因 / 下一步：')
    hints.forEach((h) => console.log('  - ' + h))
  }
  console.log('')
  process.exit(1)
}

const controller = new AbortController()
const timer = setTimeout(() => controller.abort(), 8000)
let res
try {
  res = await fetch(url, {
    headers: { Accept: 'application/json', Origin: origin },
    signal: controller.signal,
  })
} catch (err) {
  clearTimeout(timer)
  fail('接口不通，官网上会兜底跳到 GitHub Releases。', [
    '后端没启动 / 地址写错：本地联调先 cd website && go run .（默认 :8080）',
    'VITE_API_BASE_URL 留空时会请求同源 /api —— 如果官网是纯静态托管（GitHub Pages 等），那里没有后端，必须填后端完整地址',
    '失败详情：' + (err && err.message ? err.message : String(err)),
  ])
}
clearTimeout(timer)

const contentType = res.headers.get('content-type') || ''
const text = await res.text()
let body = null
try {
  body = JSON.parse(text)
} catch {
  body = null
}

console.log('HTTP ' + res.status + '  ' + (contentType || '(无 content-type)'))
console.log('')

if (!res.ok) {
  fail('接口返回 HTTP ' + res.status + '，官网会兜底跳到 Releases。', [
    base === ''
      ? '当前请求的是同源 ' + url + ' —— 静态站点上不存在这个路径，把 VITE_API_BASE_URL 指向后端域名'
      : '确认后端域名、端口和反向代理配置',
    contentType.includes('text/html') ? '返回的是 HTML（静态站点的 404 页面），基本可以确定是打到了静态托管而不是后端' : null,
    '后端限流：RATE_LIMIT_RPS 太小会被 429',
  ].filter(Boolean))
}

if (!body) {
  fail('返回的不是 JSON，官网会兜底跳到 Releases。', [
    contentType.includes('text/html') ? '拿到的是一张 HTML 页面（多半是静态站点的 404 / index.html），说明请求没打到更新服务' : '检查反向代理是否把 /api 转发到了 Go 服务',
  ])
}

if (typeof body.code === 'number' && body.code !== 0) {
  fail('接口返回业务错误：' + (body.message || body.code), ['对照 website/README.md 的错误码说明'])
}

const data = body.data || {}
const sizeMB = data.fileSize > 0 ? (data.fileSize / 1048576).toFixed(1) + ' MB' : '-'
const published = data.publishedAt ? String(data.publishedAt).slice(0, 10) : '-'

console.log('接口返回：')
info('latestVersionCode = ' + (data.latestVersionCode ?? 0))
info('latestVersionName = ' + (data.latestVersionName || '(空)'))
info('publishedAt       = ' + published)
info('fileName          = ' + (data.fileName || '(空，老版本后端不返回)'))
info('fileSize          = ' + sizeMB)
info('downloadUrl       = ' + (data.downloadUrl || '(空)'))
info('changelog         = ' + (data.changelog ? String(data.changelog).split('\n').length + ' 行' : '(空)'))
console.log('')

// 浏览器跨域的真实条件：后端必须回 Access-Control-Allow-Origin
const acao = res.headers.get('access-control-allow-origin')
if (acao) {
  ok('CORS：Access-Control-Allow-Origin = ' + acao)
} else {
  bad('CORS：后端没有返回 Access-Control-Allow-Origin —— 浏览器从官网域名发起的跨域请求会被拦掉')
  info('后端 .env 里设置 CORS_ALLOW_ORIGINS=https://你的官网域名（或 *）后重启服务')
  info('（命令行 node 不受 CORS 限制，所以这里能拿到数据，但浏览器里会失败）')
}
console.log('')

if (Number(data.latestVersionCode) > 0 && data.downloadUrl) {
  console.log('结论：官网下载按钮会直接下载这个 APK —— ' + data.downloadUrl)
  console.log('')
  process.exit(acao ? 0 : 1)
}

console.log('结论：这个 appKey 下还没有发布记录，官网会兜底跳到 GitHub Releases。')
console.log('')
console.log('可能的原因 / 下一步：')
console.log('  - 还没发过版：推一个 tag 触发 .github/workflows/release.yml，它会 POST /api/v1/releases/upload')
console.log('  - workflow 里上传是「软失败」：secrets 没配 UPDATE_SERVER_URL / UPDATE_SERVER_API_KEY 时会打印')
console.log('    warning 并跳过，后端永远拿不到 schedule 的记录')
console.log('  - 上传时 appKey 必须是 schedule（与官网 VITE_APP_KEY 一致），versionCode 必须是 base*10')
console.log('  - 本地库里若只有测试数据，可以临时 VITE_APP_KEY=<有记录的 appKey> 验证链路')
console.log('')
process.exit(1)
