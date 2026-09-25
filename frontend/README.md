# 西大课栈 · 官网介绍页

Vue 3 + Vite + Tailwind CSS 实现的单页介绍网站，视觉与 Android App 课表页同源
（冷调蓝灰渐变 + 磨砂玻璃卡片 + Material 3 主题色）。

设计原稿来自 Stitch 导出，见 [`design/stitch/code.html`](design/stitch/code.html) 与
[`design/stitch/screen.png`](design/stitch/screen.png)；实现后的实际渲染见
[`design/preview/`](design/preview)。

## 技术栈

| 项 | 选择 |
|---|---|
| 框架 | Vue 3（`<script setup>` 组合式 API） |
| 构建 | Vite 6 |
| 样式 | Tailwind CSS 3（`darkMode: 'class'`）+ 少量全局 CSS 变量 |
| 图标 | 全部为内置 SVG 组件，**零字体、零图标库依赖** |
| 依赖 | 运行时只有 `vue`，无路由/状态管理库 |

## 目录结构

```
frontend/
├─ index.html                 入口 HTML（含首屏防闪白的主题前置脚本）
├─ vite.config.js             base './'、@ 别名、/api 开发代理
├─ tailwind.config.js         品牌色、深色模式、动效
├─ public/
│  ├─ logo.png                站点 Logo（取自 App 启动图标）
│  ├─ favicon.png
│  └─ screenshots/            真实截图占位目录（默认留空）
├─ design/
│  ├─ stitch/                 Stitch 设计原稿（HTML + 效果图）
│  └─ preview/                本实现的实际渲染截图
└─ src/
   ├─ App.vue                 页面装配（11 个区块）
   ├─ style.css               设计令牌 / 玻璃质感 / 手机模型 / 动画
   ├─ config/
   │  ├─ site.js              文案、链接、工具集、FAQ、主题预设、截图占位
   │  └─ schedule.js          手机模型里的课表还原数据
   ├─ api/
   │  ├─ http.js              fetch 封装（统一响应体、超时、错误）
   │  └─ release.js           最新版本接口
   ├─ composables/            useTheme / useThemePreset / useCountdown / useRelease
   ├─ directives/reveal.js    v-reveal 滚动入场
   └─ components/
      ├─ 11 个区块组件 + PhoneMockup / PhoneScheduleWeek / PhoneMiniWeek / ScreenshotSlot
      └─ icons/               23 个 SVG 图标组件
```

## 快速开始

需要 Node.js 18+（开发时使用 Node 22）。

```bash
cd website/frontend
npm install
npm run dev        # http://localhost:5173，/api 自动代理到 http://localhost:8080
npm run build      # 产物输出到 dist/
npm run preview    # 本地预览 dist/
```

## 接口对接

官网只读一个公开接口，用于展示最新版本号、安装包体积、更新日志与下载直链：

    GET {VITE_API_BASE_URL}/api/v1/apps/{appKey}/latest?versionCode=0

响应（后端 `internal/handler` 的 `latestView`）：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "appKey": "schedule",
    "currentVersionCode": 0,
    "hasUpdate": true,
    "latestVersionCode": 107060,
    "latestVersionName": "1.7.6",
    "changelog": "优化…\n新增…",
    "downloadUrl": "https://cdn.example.com/release/schedule/107060/xxx.apk",
    "sha256": "…",
    "fileSize": 38689642,
    "forced": false
  }
}
```

### 环境变量

复制 `.env.example` 为 `.env`：

| 变量 | 默认 | 说明 |
|---|---|---|
| `VITE_API_BASE_URL` | 空（同源 `/api`） | 更新服务地址；留空时走同源，生产环境建议用 Nginx 反代 |
| `VITE_APP_KEY` | `schedule` | 与上传 workflow 的 `appKey` 保持一致 |
| `VITE_PROXY_TARGET` | `http://localhost:8080` | **仅开发用**，Vite dev server 的代理目标 |
| `VITE_SHOT_*` | 空 | 真实截图覆盖，见下节 |

### 兜底策略

`useRelease()` 在接口失败（离线、后端未部署、CORS、超时 8s）时**自动回退**到
`src/config/site.js` 的 `fallbackRelease`，页面照常展示版本号、体积与更新日志，
不会出现空白或报错。区块根节点带 `data-release-source="api|fallback"`，方便排查。

> 官网请求 `versionCode=0`，因此后端永远返回最新版本，无需与 App 上报口径对齐。

## 真实截图怎么填

页面**默认不需要任何图片**：课表、课程卡、桌面小组件、深浅色对照全部是 HTML/CSS 还原的，
开箱即完整。想换成真机截图时二选一：

1. **改配置**：把图片放进 `public/screenshots/`，在 `src/config/site.js` 的 `screenshots` 里填相对路径；
2. **用环境变量**：设置 `VITE_SHOT_HERO`、`VITE_SHOT_SCHEDULE`、`VITE_SHOT_WIDGET_TODAY` 等（键名见 `.env.example`）。

可填的位置共 10 处：首屏手机（hero）、课表展示区三台手机（schedule / prev / next）、
深浅色对照（appearanceLight / appearanceDark）、四张小组件卡片（widgetCountdown / widgetToday /
widgetUpcoming / widgetWeek）。图片会按 `object-fit: cover; object-position: top` 裁进 9:20 手机框，
详情与建议尺寸见 [`public/screenshots/README.md`](public/screenshots/README.md)。

## 想改文案 / 数据

- 文案、导航、卖点、工具集、FAQ、页脚、外链、主题预设 → [`src/config/site.js`](src/config/site.js)
- 手机模型里的课程数据（颜色、高度、教室、角标） → [`src/config/schedule.js`](src/config/schedule.js)
- 设计令牌（渐变、玻璃、圆角） → [`src/style.css`](src/style.css) 顶部 `:root` 与 `html.dark`

课表卡片颜色用 CSS 变量 `--course-c1` ~ `--course-c5`，切换主题色板时整页平滑过渡；
后端接口无需参与配色。

## 部署

### 1. 由更新服务的 Nginx 一起托管（推荐）

```bash
npm ci && npm run build   # 产出 dist/
```

```nginx
server {
    listen 443 ssl;
    server_name update.example.com;

    root /var/www/gxuschedule;      # dist/ 的内容放这里
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;   # Go 更新服务
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

这样前端用同源 `/api`，`VITE_API_BASE_URL` 留空即可，不存在跨域问题。

### 2. GitHub Pages

`vite.config.js` 里 `base: './'`，产物可直接放到任意子路径（如
`https://<user>.github.io/GxuScheduleAPP/`）。把 `dist/` 内容提交到 Pages 分支，
或改用 GitHub Actions 构建后上传 `dist`。跨域时把 `VITE_API_BASE_URL` 指向更新服务域名
（后端需允许该来源，或直接用方案 1）。

## 与设计稿的差异（有意为之）

| 项 | 设计原稿 | 本实现 | 原因 |
|---|---|---|---|
| 首屏主按钮「立即下载」 | 直接跳 GitHub Releases | 跳页面内 `#download` 区块 | 让用户先看到版本号/体积/更新日志，再由区块按钮落到接口直链或 Releases |
| 依赖 | Tailwind CDN + 内联脚本 | 本地构建 + Vue 组件 | 可离线、可 tree-shake、可维护 |
| Logo / 图标 | Google 临时图床链接 | 本地 App 图标 + 内置 SVG 组件 | 原链接会失效，且不应外链第三方 |
| 版本号/体积/日志 | 写死 v1.7.6 / 16.8 MB | 接口获取，失败回退到静态值 | 发版后官网自动更新；体积按真实字节计算 |
