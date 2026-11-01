import IconDoc from '../components/icons/IconDoc.vue'
import IconCalculator from '../components/icons/IconCalculator.vue'
import IconClock from '../components/icons/IconClock.vue'
import IconLock from '../components/icons/IconLock.vue'
import IconCode from '../components/icons/IconCode.vue'
import IconShield from '../components/icons/IconShield.vue'

/** 兼容相对路径部署（GitHub Pages 子目录 / 任意静态托管） */
export function asset(path) {
  const base = import.meta.env.BASE_URL || '/'
  return base.replace(/\/$/, '') + '/' + String(path).replace(/^\//, '')
}

export const site = {
  name: '西大课栈',
  slug: 'GxuScheduleAPP',
  tagline: '为广西大学学生打造的一键导入、可自由定制的 Android 课表应用。',
  email: '2908451607@qq.com',
  logo: asset('logo.png'),
  favicon: asset('favicon.png'),
}

export const links = {
  repo: 'https://github.com/laoin114514/GxuScheduleAPP',
  releases: 'https://github.com/laoin114514/GxuScheduleAPP/releases',
  issues: 'https://github.com/laoin114514/GxuScheduleAPP/issues',
  mailto: 'mailto:2908451607@qq.com',
  license: 'Apache-2.0',
}

export const navLinks = [
  // { href: '#features', label: '核心卖点' },
  // { href: '#schedule-preview', label: '课表交互' },
  { href: '#widgets', label: '桌面小组件' },
  { href: '#tools', label: '学习工具集' },
  { href: '#open-source', label: '开源理念' },
  { href: '#faq', label: '常见问题' },
]

export const hero = {
  badges: ['开源 · 免费 · 纯净无广告', '为西大学子定制'],
  titleLead: '你的课表，',
  titleHighlight: '本该这么好看、这么省心',
  subtitle:
    '告别每次查课都要登录教务网页的折磨。一键同步课表、6 款桌面小组件抬眼即知、成绩与绩点随手查、深浅色自动切换。',
  primaryCta: '立即下载 Android 安装包',
  secondaryCta: '查看开源源码',
  compatibility: ['支持 Android 8.0 ~ Android 15', '基于 Apache-2.0 协议开源', '原生 Kotlin 编写'],
  stats: [
    { value: '0', label: '广告 & 开屏弹窗' },
    { value: '100%', label: '数据本地 Room 存储' },
    { value: '6 款', label: '桌面小组件' },
  ],
}

export const features = [
  {
    num: '01',
    title: '教务一键导入',
    desc: '绑定广西大学教务账号后，自动获取个人信息、识别当前学期并拉取课表，免除繁琐手打；支持手动刷新。',
    footnote: '绑定后自动同步当前学期 →',
  },
  {
    num: '02',
    title: '成绩、考试与绩点',
    desc: '按学期查询成绩，支持必修 / 选修 / 不及格筛选与绩点汇总；查看各科考试时间地点，并汇总全部学期计算平均学分绩点。',
    footnote: '查分、看考场、算绩点一站完成 →',
  },
  {
    num: '03',
    title: '六款桌面小组件',
    desc: '今日课程（2×2 / 4×2）、下课倒计时、近日课程、一周课程、下一门课。桌面直接抬眼看地点，可在应用内一键添加或从系统小组件列表添加。',
    footnote: '下课倒计时精确到秒 →',
  },
  {
    num: '04',
    title: '课表与外观定制',
    desc: '浅色 / 深色 / 跟随系统，可按自定义时间自动切换；五档字体缩放，课程格子高度 40–120dp、边框颜色、显示项与今日高亮均可调。',
    footnote: '课表长相，由你自己说了算 →',
  },
]

/**
 * 5 套 M3 主题预设。
 * brand / brandDark / onPrimaryDark 均取自 App M3Palettes.kt 的 LIGHT_PALETTES / DARK_PALETTES，
 * 保证手机模型里的强调色（顶栏日期、今日高亮、底部导航选中）与真机一致。
 */
export const themePresets = [
  { key: 'purple', label: '经典紫', brand: '#6750A4', brandDark: '#D0BCFF', onPrimaryDark: '#381E72', dot: '#6750A4', colors: ['#6750A4', '#5E35B1', '#7D5260', '#EF6C00', '#C2185B'] },
  { key: 'ocean', label: '海洋蓝', brand: '#1565C0', brandDark: '#AAC7FF', onPrimaryDark: '#002F5C', dot: '#1565C0', colors: ['#2979FF', '#1565C0', '#00695C', '#EF6C00', '#C2185B'] },
  { key: 'teal', label: '青绿', brand: '#006A60', brandDark: '#5CDBCB', onPrimaryDark: '#003730', dot: '#006A60', colors: ['#1DE9B6', '#00695C', '#2E7D32', '#1565C0', '#E65100'] },
  { key: 'amber', label: '暖棕', brand: '#8D5000', brandDark: '#FFB870', onPrimaryDark: '#4B2800', dot: '#8D5000', colors: ['#EF6C00', '#F57C00', '#6D4C41', '#1565C0', '#5E35B1'] },
  { key: 'rose', label: '玫红', brand: '#B0005C', brandDark: '#FFB0CB', onPrimaryDark: '#650033', dot: '#B0005C', colors: ['#C2185B', '#AD1457', '#6750A4', '#00695C', '#EF6C00'] },
]

export const tools = [
  { key: 'grades', title: '成绩查询', icon: IconDoc, iconBox: 'bg-blue-100 dark:bg-blue-900/50', iconColor: 'text-[#4A90E2]', desc: '按学期查询成绩，支持全部 / 必修 / 选修 / 不及格筛选、按分数排序与单科成绩详情。' },
  { key: 'exams', title: '考试安排', icon: IconClock, iconBox: 'bg-emerald-100 dark:bg-emerald-900/50', iconColor: 'text-[#00695C]', desc: '从教务系统同步各科考试时间、地点与考场信息，期末不再跑错教室。' },
  { key: 'gpa', title: '绩点计算 & 模拟试算', icon: IconCalculator, iconBox: 'bg-purple-100 dark:bg-purple-900/50', iconColor: 'text-[#6750A4]', desc: '汇总全部学期成绩计算平均学分绩点，也可按预期成绩模拟试算最终结果。' },
]

export const guarantees = [
  { title: '数据本地存储', icon: IconLock, iconColor: 'text-blue-500', desc: '使用 Android 原生 Room 数据库本地存储，无云端账户依赖，断网也能飞速看课。' },
  { title: '代码完全开源', icon: IconCode, iconColor: 'text-emerald-500', desc: '遵循 Apache License 2.0，欢迎审查每一行代码，学生开发者也可随时自编译打包。' },
  { title: '永久免费无广告', icon: IconShield, iconColor: 'text-purple-500', desc: '纯粹出于解决同学日常看课痛点的学生作品，无任何商业变现考量与开屏广告。' },
]

export const techTags = [
  'Kotlin',
  'Material 3',
  'MVVM Architecture',
  'Room Database',
  'OkHttp',
  'AlarmManager',
  'Apache POI (Excel 解析)',
]

export const faqs = [
  {
    q: '导入后课表显示为空怎么办？',
    a: '请先确认已完成教务账号绑定，并检查「我的 → 当前学期」是否选中了正确的学期。回到课表页点击右上角刷新按钮重新拉取；若教务系统正在维护，请稍后再试。',
  },
  {
    q: '桌面小组件不刷新怎么办？',
    a: '先启动一次 App，让系统解除安装后的 stopped 状态；再检查系统是否限制了应用后台运行。必要时删除组件后重新添加。',
  },
  {
    q: '会不会上传或泄露我的教务账号密码？',
    a: '不会上传到第三方服务器。账号密码仅保存在本机的应用私有数据库中，用于与广西大学官方教务接口通信；项目代码完全开源，网络请求可在 GitHub 上审查。请避免在共用设备上绑定账号。',
  },
  {
    q: '工具页有些功能点不开？',
    a: '工具页中的教室查询、图书馆、校园卡、校园地图、食堂菜谱、番茄钟、快速笔记与课表导出目前是占位入口，点击会提示「功能开发中」。已开放的是教务一键导入、成绩查询、考试安排与绩点计算。',
  },
  {
    q: '后续会支持其他高校的教务系统吗？',
    a: '目前仅适配广西大学的教务系统，其他学校暂未支持。欢迎在 GitHub 提交 Issue 或 PR 交流适配方案。',
  },
]

export const footerColumns = [
  {
    title: '功能导航',
    links: [
      { label: '教务一键导入', href: '#features' },
      { label: '主题与外观', href: '#appearance' },
      { label: 'Android 桌面小组件', href: '#widgets' },
      { label: '成绩与绩点', href: '#tools' },
    ],
  },
  {
    title: '开源生态',
    links: [
      { label: 'GitHub 仓库源码', href: links.repo, external: true },
      { label: 'Releases 安装包发布', href: links.releases, external: true },
      { label: '提交 Issue 适配建议', href: links.issues, external: true },
      { label: '开源协议：Apache-2.0', href: '', external: false },
    ],
  },
]

/**
 * ===========================================================================
 * 真实截图占位
 * ---------------------------------------------------------------------------
 * 这里留空时，页面使用内置的 HTML/CSS 还原界面，效果完整、无需任何图片。
 * 想换成真机截图：把图片放进 public/screenshots/，然后在下面填相对路径，
 * 或直接用环境变量覆盖（键名见 .env.example，如 VITE_SHOT_HERO）。
 * ===========================================================================
 */
export const screenshots = {
  hero: '',
  schedule: '',
  prev: '',
  next: '',
  appearanceLight: '',
  appearanceDark: '',
  widgetCountdown: '',
  widgetToday: '',
  widgetUpcoming: '',
  widgetWeek: '',
}

export function shot(name) {
  const manual = screenshots[name]
  if (manual) return manual
  const envKey = 'VITE_SHOT_' + String(name).replace(/([A-Z])/g, '_$1').toUpperCase()
  try {
    return import.meta.env[envKey] || ''
  } catch (err) {
    return ''
  }
}

/**
 * 更新服务不可用时的静态兜底数据，
 * 保证官网永远有版本号、体积与更新日志可展示。
 */
export const fallbackRelease = {
  versionName: 'v0.4.2',
  versionCode: 4020,
  updatedAt: '2026-11-01',
  sizeBytes: 21593674,
  downloadUrl: '',
  changelog: [
    '修复教务个人信息接口返回空包导致的绑定失败',
    '补充个人信息页面兜底解析与解析单元测试',
  ],
}
