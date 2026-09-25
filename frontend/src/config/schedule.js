/**
 * 手机模型里的课表还原数据（纯演示，与 App 真实数据无关）。
 *
 * block 字段说明：
 *   c       1~5，对应 CSS 变量 --course-c1 ~ --course-c5，会随主题预设一起变色
 *   hex     需要固定颜色的卡片直接写十六进制
 *   h       卡片高度（px），还原 App 里可调的课格子高度
 *   empty   占位空隙
 *   badge   卡片右上角小标签，如「正在上」「提醒已设」
 *   overlap 课表重叠角标文案，如「2」「重」
 *   ring    是否加高亮描边（表示正在上的课）
 */

export const heroSchedule = {
  title: '第 07 周',
  badge: '广西大学',
  meta: '2025 春',
  weekdays: [
    { label: '周一', date: '10/20' },
    { label: '周二', date: '10/21' },
    { label: '周三', date: '今天', today: true },
    { label: '周四', date: '10/23' },
    { label: '周五', date: '10/24' },
  ],
  columns: [
    [
      { c: 1, h: 96, title: '高等数学(下)', room: '君武馆 201', slot: '1-2节' },
      { c: 3, h: 86, title: '大学物理实验', room: '物电楼 B302', slot: '5-6节' },
    ],
    [
      { empty: true, h: 40 },
      { c: 2, h: 96, title: '数据结构与算法', room: '计电楼 405', slot: '3-4节' },
      { c: 4, h: 75, title: '马克思主义', room: '汇学堂 A101', slot: '7-8节' },
    ],
    [
      { c: 1, h: 96, title: '软件工程', room: '综合楼 312', slot: '1-2节 · 距下课 18m', badge: '正在上', ring: true },
      { c: 5, h: 86, title: '大学英语(IV)', room: '外语楼 108', slot: '3-4节', overlap: '2' },
    ],
    [
      { hex: '#455A64', h: 90, title: '操作系统', room: '计电楼 208', slot: '1-2节' },
      { empty: true, h: 32 },
      { hex: '#2E7D32', h: 80, title: '羽毛球(初级)', room: '东体 场地3', slot: '7-8节' },
    ],
    [
      { empty: true, h: 56 },
      { c: 2, h: 90, title: '计算机网络', room: '计电楼 301', slot: '3-4节' },
      { hex: '#7D5260', h: 85, title: '形势与政策', room: '汇学堂 B201', slot: '9-10节' },
    ],
  ],
}

export const interactiveSchedule = {
  title: '第 07 周',
  badge: '智能同步中',
  connected: '● 已连接教务',
  weekdays: [
    { label: '一', date: '10/20' },
    { label: '二', date: '10/21' },
    { label: '三', date: '今日', today: true },
    { label: '四', date: '10/23' },
    { label: '五', date: '10/24' },
  ],
  columns: [
    [
      { c: 1, h: 95, title: '软件工程概论', room: '君武馆 301', slot: '1-2 节' },
      { c: 3, h: 85, title: '数字逻辑设计', room: '计电楼 B102', slot: '5-6 节' },
    ],
    [
      { empty: true, h: 24 },
      { c: 2, h: 98, title: '面向对象编程', room: '计电机房 3', slot: '3-4 节' },
      { c: 4, h: 75, title: '马克思主义', room: '汇学堂 A101' },
    ],
    [
      { c: 1, h: 105, title: '操作系统原理', room: '综合楼 312', slot: '1-2 节 · 08:00', badge: '提醒已设', ring: true },
      { c: 5, h: 90, title: '大学英语精读', room: '外国语 204', slot: '3-4 节', overlap: '重' },
    ],
    [
      { hex: '#546E7A', h: 85, title: '计算机组成', room: '计电楼 401' },
      { empty: true, h: 32 },
      { hex: '#2E7D32', h: 80, title: '大学体育', room: '东体馆' },
    ],
    [
      { empty: true, h: 48 },
      { c: 2, h: 95, title: '概率论与数理', room: '汇学 B202', slot: '3-4 节' },
    ],
  ],
  footer: '课格子高度支持 40~120dp 自由拉伸',
}

export const miniPrevSchedule = {
  title: '第 06 周 · 上周',
  footer: '左右滑动顺畅切换周次',
  cards: [
    { c: 2, h: 75, title: '电路分析', room: '君武 102' },
    { c: 1, h: 110, title: '大学体育', room: '西体 乒乓球' },
    { c: 4, h: 65, title: '形势政策' },
    { c: 3, h: 95, title: '线性代数', room: '汇学 A301' },
  ],
}

export const miniNextSchedule = {
  title: '第 08 周 · 下周',
  footer: '内置法定节假日智能隐藏课程',
  cards: [
    { c: 5, h: 80, title: '专业英语', room: '外语 301' },
    { c: 3, h: 100, title: '算法设计', room: '计电 105' },
    { c: 1, h: 90, title: '大学物理', room: '物电 A201' },
    { c: 2, h: 70, title: '毛泽东思想' },
  ],
}
