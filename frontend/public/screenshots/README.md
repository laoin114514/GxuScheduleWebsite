# 真实截图占位目录

官网默认用 **纯 HTML/CSS 还原的 App 界面**（课表、课程卡、底部导航），不依赖任何外部图片，
开箱即可看到完整效果。

## 已经预生成的两张课表整屏图

| 文件 | 尺寸 | 说明 |
|---|---|---|
| `schedule-light.png` | 1080×2256 | 浅色模式周课表 |
| `schedule-dark.png` | 1080×2256 | 深色模式周课表 |

这两张是按 App 真机布局 1:1 还原的组件渲染出来的（无头 Chrome 截图，**不是真机拍摄**），
可以直接用于 README、商店页、社交分享。想让官网手机模型用它们，把路径填进
`src/config/site.js` 的 `screenshots.schedule`（会失去实时换色效果），或改用真机截图替换本目录文件。

如果你希望换成 **真实真机截图**，把图片放进本目录，然后任选一种方式启用：

## 方式一：改配置文件（推荐）

编辑 `src/config/site.js` 里的 `screenshots` 对象，填入相对路径：

```js
export const screenshots = {
  hero: './screenshots/hero.png',
  schedule: './screenshots/schedule.png',
  prev: './screenshots/week-06.png',
  next: './screenshots/week-08.png',
  appearanceLight: './screenshots/light.png',
  appearanceDark: './screenshots/dark.png',
  widgetCountdown: './screenshots/widget-countdown.png',
  widgetToday: './screenshots/widget-today.png',
  widgetUpcoming: './screenshots/widget-upcoming.png',
  widgetWeek: './screenshots/widget-week.png',
}
```

## 方式二：环境变量（适合 CI / 不想改代码）

在 `.env` 或 CI 环境变量里设置对应键即可覆盖，键名见根目录 `.env.example`：

    VITE_SHOT_HERO=./screenshots/hero.png

## 建议的截图规格

| 键名 | 用途 | 建议尺寸 / 比例 |
|---|---|---|
| hero | 首屏手机模型 | 1170×2532，9:20 竖屏 |
| schedule | 课表展示区中间大号手机 | 1170×2532，9:20 竖屏 |
| prev / next | 左右两侧辅助手机 | 同上，内容分别为上周 / 下周 |
| appearanceLight / appearanceDark | 深浅色对照区 | 1180×820 横屏截取 |
| widgetCountdown 等 4 个 | 桌面小组件卡片 | 按官方组件比例（2×1 / 2×2 / 4×1 / 4×4） |

> 图片会被 `object-fit: cover; object-position: top` 裁切进 9:20 的手机框里，
> 请保证截图本身是完整竖屏，不要带状态栏以外的系统 UI。

## 仓库里现成可用的真机截图

```
screenshots/app_main.png          课程详情（深色）
screenshots/tools_v2.png          工具页
screenshots/tools_dark_final.png  工具页（深色）
screenshots/profile_dark.png      我的页（深色）
screenshots/theme_switch_back.png 主题切换
screenshots/badge_zoom.png        课程重叠角标特写
screenshots/tools_search_empty.png 工具搜索空态
```

这些是页面级截图，与设计稿里的「课表周视图」手机模型不完全一致，
所以默认没有启用；需要时复制到本目录再按上面的方式引用即可。
