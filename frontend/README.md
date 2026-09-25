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
│  └─ screenshots/            截图占位目录（含两张预生成的课表整屏图）
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
      ├─ 11 个区块组件 + PhoneMockup / PhoneScheduleWeek / ScreenshotSlot
      └─ icons/               25 个 SVG 图标组件
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
    "forced": false,
    "publishedAt": "2026-08-31T15:00:47Z",
    "fileName": "GxuScheduleAPP-release-1.7.6-universal.apk"
  }
}
```

下载区展示的每个字段都来自这里：版本号 → `latestVersionName`，更新日期 → `publishedAt`（格式化成
`YYYY-MM-DD`），体积 → `fileSize`，更新日志 → `changelog`，下载按钮直链 → `downloadUrl`，
安装包名 → `fileName`。

> `publishedAt` / `fileName` 是后端 v1.1 起新增的字段（见 [`website/README.md`](../README.md)）。
> 老版本后端不返回它们时页面不会报错，只是不显示日期与文件名。

### 环境变量

复制 `.env.example` 为 `.env`：

| 变量 | 默认 | 说明 |
|---|---|---|
| `VITE_API_BASE_URL` | 空（同源 `/api`） | 更新服务地址；留空时走同源，生产环境建议用 Nginx 反代 |
| `VITE_APP_KEY` | `schedule` | 与上传 workflow 的 `appKey` 保持一致 |
| `VITE_PROXY_TARGET` | `http://localhost:8080` | `npm run dev` 与 `npm run preview` 的 `/api` 代理目标 |
| `VITE_SHOT_*` | 空 | 真实截图覆盖，见下节 |

`.env.local`（已被 gitignore 忽略）优先级高于 `.env`，本地临时换 appKey / 指向别的后端时用它。

### 三种数据来源与兜底

`useRelease()` 把结果分成三种，组件根节点上的 `data-release-source` 会如实标出，方便排查：

| source | 触发条件 | 页面表现 |
|---|---|---|
| `api` | 接口通了，且该 appKey 有发布记录（`latestVersionCode > 0`） | 版本号 / 日期 / 体积 / 更新日志 / 下载直链**全部以接口为准** |
| `empty` | 接口通了，但该 appKey 还没发过版（全新 appKey） | 退回 `src/config/site.js` 的内置信息，按钮指向 GitHub Releases |
| `fallback` | 接口不通（离线 / 未部署 / 跨域被拦 / 超时 5s） | 同上 |

两条原则：

1. **接口活着时，接口没给的字段就不展示。** 比如后端没填 `changelog`、老版本不返回 `publishedAt`，
   页面会隐藏对应内容，而不是拿内置数据去凑 —— 否则会出现「版本号是新的、日期是旧的」这种错配。
2. **接口不可达时绝不开天窗。** 内置数据保证官网永远有可点的下载按钮（落到 Releases 页）。
   接口加载中时，版本信息行显示「正在获取最新版本…」。

跨域提醒：官网与更新服务不同域时（如 GitHub Pages + 独立后端），后端需要放行来源，
见 [`website/README.md`](../README.md) 的 `CORS_ALLOW_ORIGINS`。

### 为什么下载按钮跳到了 GitHub Releases

按钮指向 Releases，**只可能是前端没拿到可用的 `downloadUrl`**。一条命令定位：

```bash
npm run check:api -- --base=https://你的更新服务地址
```

它会打印接口返回的每个字段、CORS 头以及最终结论。四种典型结果与对应处理：

| 检查输出 | 原因 | 处理 |
|---|---|---|
| `HTTP 404 text/html` | 请求打到了静态托管（GitHub Pages 等），那里没有后端 | 构建时设 `VITE_API_BASE_URL=https://更新服务域名` 后重新构建部署；或改用 Nginx 同源反代 |
| `latestVersionCode = 0` | 接口通了，但该 appKey 还没有发布记录 | 推 tag 触发 `release.yml` 上传；并确认 secrets 里配了 `UPDATE_SERVER_URL` / `UPDATE_SERVER_API_KEY`（未配会打 warning 直接跳过） |
| 缺 `Access-Control-Allow-Origin` | 后端没放行官网域名 | 后端 `.env` 设 `CORS_ALLOW_ORIGINS=https://官网域名` 后重启服务 |
| 接口不通 / 超时 | 后端没起、域名写错、被限流 | 本地先 `cd website && go run .`；确认 `RATE_LIMIT_RPS` 不是太小 |

页面上也留了线索：下载区根节点带 `data-release-source="api|empty|fallback"`；
下载按钮的 `title` 会写明「安装包来自更新服务」或「跳转 GitHub Releases」；
回退时浏览器控制台会打一条 `[西大课栈] 更新服务不可用…` 警告。

### 本地联调

```bash
# 终端 1：起更新服务（需要 MySQL 与 OSS 配置）
cd website && go run .

# 终端 2：起官网，/api 自动代理到 8080
cd website/frontend && npm run dev     # 或 npm run preview（也已配好代理）
```

打开页面后看下载区：`data-release-source` 为 `api` 即代表真的读到了后端数据
（用 `.env.local` 写 `VITE_APP_KEY=<有发布记录的 appKey>` 可以在本地库里没有 `schedule` 记录时验证）。

> 官网请求 `versionCode=0`，因此后端永远返回最新版本，无需与 App 上报口径对齐。

## 手机模型里的课表（按真机 1:1 还原）

首屏和「同一张课表，五种样子」里的课表界面不是图片，而是照着 App 源码复刻的 Vue 组件
（[`PhoneScheduleWeek.vue`](src/components/PhoneScheduleWeek.vue)），结构与 `WeekPagerAdapter` 一一对应：

| App 实现 | 官网还原 |
|---|---|
| 逻辑画布 360dp，实际宽度按屏幕换算 | `.app-screen` 固定 360px，用 `transform: scale()` 缩到手机模型宽度 |
| 顶栏：`tv_week_info`（12sp）+ `tv_date`（18sp 粗体、colorPrimary）+ 两个 32dp 圆形按钮 | `.app-header` / `.app-date` / `.app-circle-btn` |
| 日期栏：32dp 的「年 / 年份」列 + 7 列「一…日 / 月-日」；今天套 `bg_date_selected`（主色 + 4dp 圆角） | `.app-daterow` / `.app-date-num.is-today` |
| 节次轴 32dp、每格 68dp：节次号（12sp 粗体）+ 开始 / 结束时间（10sp） | `.app-axis` / `.app-axis-cell` |
| 7 等分日列，课程卡绝对定位：左 = (day-1)×列宽，上 = (start-1)×格子高，高 = 节数×格子高 | `cardStyle()` |
| 卡片：圆角 + 2dp 白色描边 + 主题色 50%（深色 75%）半透明底 + 白色粗体居中「课程名 / 教室 / 教师」 | `.app-course` |
| 同一天同一时段多门课 → 右下角 `+N` 角标 | `.app-overlap` |
| 底部导航：激活项 = 40dp 圆形主色块 + 22dp 白图标 + 主色 10sp 文字 | `.app-nav` |
| 13 节默认时间表（08:00~23:25） | `src/config/schedule.js` 的 `periods` |
| 网格线 `gridColor = Color.TRANSPARENT`（**不画线**） | `--app-grid-line: transparent`，想开虚线网格只改这一个变量 |

课程与配色数据都在 [`src/config/schedule.js`](src/config/schedule.js)：字段是
`{ day, start, end, color, name, room, teacher, overlap? }`，`color` 取 1~5，
对应当前主题预设的 5 个色彩槽位（真机是 9 色固定值，官网为了演示色板切换做了映射，
见文件顶部注释）。三周示例数据是 `weeks.current / prev / next`。

## 真实截图怎么填

课表界面已按真机还原（见上一节），另外仓库里预生成了两张可直接使用的整屏图：

- [`public/screenshots/schedule-light.png`](public/screenshots/schedule-light.png)（1080×2256）
- [`public/screenshots/schedule-dark.png`](public/screenshots/schedule-dark.png)

它们是上面那个组件的真实渲染结果（无头 Chrome 截的），可直接用于 README、商店页或分享图。
想让官网手机模型改用它，把路径填进 `screenshots.schedule` 即可 —— 注意该槽位会因此变成静态图，
失去「五种样子」的实时换色效果。

其余位置仍然**默认不需要任何图片**：课表、课程卡、桌面小组件、深浅色对照全部是 HTML/CSS 还原的，
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

### 2. GitHub Pages 等纯静态托管

`vite.config.js` 里 `base: './'`，产物可直接放到任意子路径（如
`https://<user>.github.io/GxuScheduleAPP/`）。

> ⚠️ 静态托管**没有** `/api` 反向代理。若 `VITE_API_BASE_URL` 留空，`/api/v1/...` 会打到 GitHub Pages
> 自己（返回 404 + HTML），下载按钮就永远只能跳 Releases。在静态托管上必须三步都做：
>
> 1. 构建时指定后端地址：`VITE_API_BASE_URL=https://你的更新服务域名 npm run build`
> 2. 后端 `.env` 放行该来源：`CORS_ALLOW_ORIGINS=https://<user>.github.io`，然后重启服务
> 3. 部署后用 `npm run check:api -- --base=https://你的更新服务域名`，并在浏览器里确认下载区
>    `data-release-source="api"`
>
> 注意 `VITE_API_BASE_URL` 是**构建期**注入的，改了必须重新构建再部署。

不想处理跨域就用方案 1（Nginx 同源反代），官网与接口同域最省事。

## 与设计稿的差异（有意为之）

| 项 | 设计原稿 | 本实现 | 原因 |
|---|---|---|---|
| 首屏主按钮「立即下载」 | 直接跳 GitHub Releases | 跳页面内 `#download` 区块 | 让用户先看到版本号/体积/更新日志，再由区块按钮落到接口直链或 Releases |
| 依赖 | Tailwind CDN + 内联脚本 | 本地构建 + Vue 组件 | 可离线、可 tree-shake、可维护 |
| Logo / 图标 | Google 临时图床链接 | 本地 App 图标 + 内置 SVG 组件 | 原链接会失效，且不应外链第三方 |
| 版本号/体积/日志 | 写死 v1.7.6 / 16.8 MB | 接口获取，失败回退到静态值 | 发版后官网自动更新；体积按真实字节计算 |
| 课表界面 | 5 列示意课表（虚构课程、没有节次轴） | 按真机 360dp 布局 1:1 还原：周次顶栏 + 日期栏 + 13 节次轴 + 7 天 + 底部导航 | 让手机模型看起来就是一张真机截图 |
| 课程卡配色 | 写死 5 组主题色 | 真机 9 色固定 → 官网映射到当前主题的 5 个槽位 | 保住「五种样子」的实时换色演示 |
| 网格线 | 设计稿画了虚线网格 | 不画（跟随真机 `Color.TRANSPARENT`） | 与当前版本 App 视觉一致 |
