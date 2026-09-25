import IconDoc from '../components/icons/IconDoc.vue'
import IconCalculator from '../components/icons/IconCalculator.vue'
import IconBuilding from '../components/icons/IconBuilding.vue'
import IconClock from '../components/icons/IconClock.vue'
import IconPencil from '../components/icons/IconPencil.vue'
import IconImage from '../components/icons/IconImage.vue'
import IconBook from '../components/icons/IconBook.vue'
import IconMap from '../components/icons/IconMap.vue'
import IconCart from '../components/icons/IconCart.vue'
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
  tagline: '为广西大学学生打造的一键导入、智能提醒、可自由定制的 Android 课表应用。',
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
  { href: '#features', label: '核心卖点' },
  { href: '#schedule-preview', label: '课表交互' },
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
    '告别每次查课都要登录教务网页的折磨。一键同步课表、四重课前提醒绝不漏课、4 类桌面小组件抬眼即知、6 套 Material 3 配色随心切换。',
  primaryCta: '立即下载 Android 安装包',
  secondaryCta: '查看开源源码',
  compatibility: ['支持 Android 8.0 ~ Android 15', '基于 Apache-2.0 协议开源', '原生 Kotlin 编写'],
  stats: [
    { value: '0', label: '广告 & 开屏弹窗' },
    { value: '100%', label: '数据本地 Room 存储' },
    { value: '4 重', label: '后台保活提醒防漏' },
  ],
}

export const features = [
  {
    num: '01',
    title: '教务一键导入',
    desc: '绑定教务账号后自动拉取全学期课表，免除繁琐手打；同时支持 CSV、Excel 与 JSON 格式导入导出。',
    footnote: '3 秒同步全部学期安排 →',
  },
  {
    num: '04',
    title: '四重课前提醒',
    desc: 'setAlarmClock + WorkManager + 前台保活服务 + 智能心跳机制，对抗国产定制系统后台杀进程，尽量不漏课。',
    footnote: '告别迟到与误旷课 →',
  },
  {
    num: '04',
    title: '四种桌面小组件',
    desc: '今日课程、下课倒计时、近日课程、一周课程。桌面直接抬眼看地点，不必每次解锁找应用图标。',
    footnote: '下课倒计时精确到秒 →',
  },
  {
    num: '06+',
    title: '完全个性化定制',
    desc: '6 套 Material 3 配色、14 款背景、课格子高度 40-120dp 自由缩放、自定义背景图与定时深色模式。',
    footnote: '课表长相，由你自己说了算 →',
  },
]

/** 5 套 M3 主题预设，与 App 内 6 套配色方案对齐（官网展示 5 套） */
export const themePresets = [
  { key: 'purple', label: '经典紫', brand: '#6750A4', dot: '#6750A4', colors: ['#6750A4', '#5E35B1', '#7D5260', '#EF6C00', '#C2185B'] },
  { key: 'ocean', label: '海洋蓝', brand: '#4A90E2', dot: '#1565C0', colors: ['#4A90E2', '#1565C0', '#00695C', '#EF6C00', '#C2185B'] },
  { key: 'teal', label: '青绿', brand: '#00695C', dot: '#00695C', colors: ['#00695C', '#00897B', '#2E7D32', '#1565C0', '#E65100'] },
  { key: 'amber', label: '暖棕', brand: '#EF6C00', dot: '#EF6C00', colors: ['#EF6C00', '#F57C00', '#6D4C41', '#1565C0', '#5E35B1'] },
  { key: 'rose', label: '玫红', brand: '#C2185B', dot: '#C2185B', colors: ['#C2185B', '#AD1457', '#6750A4', '#00695C', '#EF6C00'] },
]

export const tools = [
  { key: 'grades', title: '成绩与考试安排', icon: IconDoc, iconBox: 'bg-blue-100 dark:bg-blue-900/50', iconColor: 'text-[#4A90E2]', desc: '直连教务同步期末各科考试时间地点；考试前自动开启提醒倒数，查分快人一步。' },
  { key: 'gpa', title: '绩点计算 & 模拟试算', icon: IconCalculator, iconBox: 'bg-purple-100 dark:bg-purple-900/50', iconColor: 'text-[#6750A4]', desc: '支持广西大学保研/奖学金 GPA 计算标准；可预估期末单科得分，模拟试算最终总绩点。' },
  { key: 'classroom', title: '空闲教室检索', icon: IconBuilding, iconBox: 'bg-emerald-100 dark:bg-emerald-900/50', iconColor: 'text-[#00695C]', desc: '君武馆、汇学堂、计电楼、综合楼空闲自习室一键筛选，自习找座位不再扑空。' },
  { key: 'pomodoro', title: '专注番茄钟', icon: IconClock, iconBox: 'bg-amber-100 dark:bg-amber-900/50', iconColor: 'text-[#EF6C00]', desc: '内置 25 分钟专注 + 5 分钟休息周期；期末复习与自习专心沉浸，拒绝短视频干扰。' },
  { key: 'notes', title: '灵感与课堂速记', icon: IconPencil, iconBox: 'bg-pink-100 dark:bg-pink-900/50', iconColor: 'text-[#C2185B]', desc: '老师布置的随堂作业、分组汇报要点、闪现的灵感，课表旁一键随记，不遗漏任何任务。' },
  { key: 'export', title: '课表一键导出成图', icon: IconImage, iconBox: 'bg-sky-100 dark:bg-sky-900/50', iconColor: 'text-[#1565C0]', desc: '将当前周或全学期课表导出为超清壁纸或长图，方便设为手机锁屏或分享给同班好友。' },
  { key: 'library', title: '图书馆藏与校园卡', icon: IconBook, iconBox: 'bg-indigo-100 dark:bg-indigo-900/50', iconColor: 'text-[#5E35B1]', desc: '书籍借阅状态与续借提醒、校园卡余额变动速查，一站式集成便捷实用入口。' },
  { key: 'map', title: '西大校区手绘地图', icon: IconMap, iconBox: 'bg-teal-100 dark:bg-teal-900/50', iconColor: 'text-[#00695C]', desc: '针对西大校园大、教学楼分散的痛点，快速索引各教学楼、实验楼、宿舍与东体西体。' },
  { key: 'dining', title: '食堂档口与菜谱精选', icon: IconCart, iconBox: 'bg-orange-100 dark:bg-orange-900/50', iconColor: 'text-[#EF6C00]', desc: '东苑、西苑、南苑各大食堂招牌菜评价，纠结今天吃什么时看一眼，拯救选择困难。' },
]

export const guarantees = [
  { title: '数据本地存储', icon: IconLock, iconColor: 'text-blue-500', desc: '使用 Android 原生 Room 数据库安全存储，无云端账户依赖，断网也能飞速看课。' },
  { title: '代码完全开源', icon: IconCode, iconColor: 'text-emerald-500', desc: '遵循 Apache License 2.0，欢迎审查每一行代码，学生开发者也可随时自编译打包。' },
  { title: '永久免费无广告', icon: IconShield, iconColor: 'text-purple-500', desc: '纯粹出于解决同学日常看课痛点的学生作品，无任何商业变现考量与开屏广告。' },
]

export const techTags = [
  'Kotlin',
  'Material 3',
  'MVVM Architecture',
  'Room Database',
  'Retrofit2 / OkHttp',
  'WorkManager / AlarmManager',
  'Apache POI (Excel解析)',
]

export const faqs = [
  {
    q: '导入后课表显示为空怎么办？',
    a: '请确认教务系统当前学期是否正确。可在「课表设置」中点击「手动选择学期」重试同步，或下拉课表主界面强制刷新缓存。若教务系统维护中，可稍候再试。',
  },
  {
    q: '手机收不到上课前提醒怎么排查？',
    a: '国产系统（如 HyperOS、OriginOS、ColorOS、HarmonyOS）默认对后台限制严格。请前往系统设置：① 授予西大课栈「悬浮窗」和「通知」权限；② 将电池优化设置为「无限制」或「允许自启动」；③ 在 App 设置中开启「前台保活服务」。',
  },
  {
    q: '会不会上传或泄露我的教务账号密码？',
    a: '绝对不会。西大课栈是纯客户端开源软件，没有自建后端服务器。您的教务账号和密码仅在本地使用 Android 安全加密存储，仅用于与广西大学官方教务接口通信以获取课表，任何人皆可在 GitHub 上检视网络请求代码。',
  },
  {
    q: '换新手机后，如何快速迁移我的课表和自定义设置？',
    a: '在旧手机的「设置 - 数据备份」中点击「导出完整备份（JSON）」，发送到新手机；在新手机安装西大课栈后点击「导入文件备份」，即可瞬间完整恢复所有课程、自定义备注与色彩配置。',
  },
  {
    q: '后续会支持其他高校的教务系统吗？',
    a: '西大课栈底层架构保留了可扩展的数据解析层（基于 WakeUp 课程表分支二次开发）。欢迎其他高校的同学在 GitHub 提交 PR 或 Issue 提供教务接口适配脚本！',
  },
]

export const footerColumns = [
  {
    title: '功能导航',
    links: [
      { label: '教务同步与提醒', href: '#features' },
      { label: '主题外观定制', href: '#schedule-preview' },
      { label: 'Android 桌面小组件', href: '#widgets' },
      { label: '成绩与学习工具集', href: '#tools' },
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
  versionName: '1.7.6',
  versionCode: 107060,
  updatedAt: '2025-03',
  sizeBytes: 38689642,
  downloadUrl: '',
  changelog: [
    '优化广西大学新版教务系统验证码与单点登录兼容性',
    '新增下课倒计时 2×1 桌面小组件秒级刷新模式',
    '修复在 Android 14/15 上的前台服务保活策略',
  ],
}
