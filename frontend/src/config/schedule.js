/**
 * 手机模型里的课表数据 —— 结构与字段完全照搬 App「周课表」页（WeekPagerAdapter）。
 *
 * 时间表、界面文案、卡片信息顺序都取自 App 源码：
 *   - 13 节默认时间表：TimeTableEditActivity.getDefaultStartTime / getDefaultEndTime
 *   - 课程卡片文案：主课程名 + 教室 + 教师，用换行拼接，居中加粗白字
 *   - 日期栏：第一列是「年 / 年份」，后面 7 列是「一…日 / 月-日」，今天带主色圆角底
 *
 * 说明：App 内 9 色课程卡是固定色，不跟随主题；官网为了演示「五种样子」，
 * 把 color(1~5) 映射到当前主题预设的 5 个色彩槽位，切换色板时整页平滑过渡。
 */

/** 广西大学默认时间表（13 节） */
export const periods = [
  { node: 1, start: '08:00', end: '08:45' },
  { node: 2, start: '08:55', end: '09:40' },
  { node: 3, start: '10:00', end: '10:45' },
  { node: 4, start: '10:55', end: '11:40' },
  { node: 5, start: '14:30', end: '15:15' },
  { node: 6, start: '15:25', end: '16:10' },
  { node: 7, start: '16:30', end: '17:15' },
  { node: 8, start: '17:25', end: '18:10' },
  { node: 9, start: '19:00', end: '19:45' },
  { node: 10, start: '19:55', end: '20:40' },
  { node: 11, start: '20:50', end: '21:35' },
  { node: 12, start: '21:45', end: '22:30' },
  { node: 13, start: '22:40', end: '23:25' },
]

/** 节次轴 / 课程卡的基础尺寸（逻辑像素，等于 App 里的 dp） */
export const layout = {
  baseWidth: 360, // 逻辑画布宽度，等于一台 360dp 宽的手机
  statusBar: 26, // 让出刘海/状态栏高度
  axisWidth: 32, // WeekPagerAdapter: timeAxisWidth = 32dp
  cellHeight: 68, // App 默认课格子高度 68dp（可在 40~120dp 之间调）
  gap: 2, // 卡片与格子边的间距
  radius: 10, // 卡片圆角（App 为 14px @1080）
}

/**
 * 课程卡片字段：
 *   day    1~7，周一到周日
 *   start  开始节次（1 起）
 *   end    结束节次（含）
 *   color  1~5，对应当前主题预设的课程配色槽位
 *   overlap  同一天同一时段的其它课程数量，>0 时卡片右下角显示 +N
 */
const currentCourses = [
  // 周一
  { day: 1, start: 1, end: 2, color: 3, name: '高等数学A(上)', room: '6B-603', teacher: '朱光军' },
  { day: 1, start: 5, end: 6, color: 4, name: '大学物理实验', room: '物理楼B302', teacher: '黄志强' },
  { day: 1, start: 7, end: 8, color: 5, name: '马克思主义基本原理', room: '汇学堂A101', teacher: '李文静' },
  // 周二
  { day: 2, start: 1, end: 2, color: 2, name: '大学英语(三)', room: '6A-528', teacher: '陈永红' },
  { day: 2, start: 3, end: 4, color: 1, name: '数据结构与算法', room: '计电楼405', teacher: '陈晓明' },
  // 周三（1-2 节有两门课重叠，卡片右下角出现 +1 角标）
  { day: 3, start: 1, end: 2, color: 1, name: '软件工程', room: '综合楼312', teacher: '王志远', overlap: 1 },
  { day: 3, start: 3, end: 4, color: 4, name: '计算机组成原理', room: '计电楼401', teacher: '刘建华' },
  // 周四
  { day: 4, start: 3, end: 4, color: 3, name: '操作系统', room: '计电楼208', teacher: '张伟' },
  { day: 4, start: 5, end: 6, color: 2, name: '概率论与数理统计', room: '汇学楼B202', teacher: '周敏' },
  // 周五
  { day: 5, start: 1, end: 2, color: 3, name: '计算机网络', room: '计电楼301', teacher: '李海' },
  { day: 5, start: 7, end: 8, color: 5, name: '形势与政策', room: '汇学堂B201', teacher: '龙紫璐' },
  // 周六
  { day: 6, start: 7, end: 8, color: 2, name: '大学生就业与创业指导', room: '西2-501', teacher: '龙紫璐' },
  // 周日
  { day: 7, start: 3, end: 4, color: 1, name: '文献检索', room: '图书馆南楼409', teacher: '韩春晖' },
]

export const weeks = {
  /** 本周：顶栏日期与今天高亮都与真机截图一致（2026/9/13 周日） */
  current: {
    weekInfo: '第3周 (本周) · 大三上',
    date: '2026/9/13',
    year: '2026',
    days: [
      { label: '一', date: '9/7' },
      { label: '二', date: '9/8' },
      { label: '三', date: '9/9' },
      { label: '四', date: '9/10' },
      { label: '五', date: '9/11' },
      { label: '六', date: '9/12' },
      { label: '日', date: '9/13', today: true },
    ],
    courses: currentCourses,
  },

  prev: {
    weekInfo: '第2周 · 大三上',
    date: '2026/9/13',
    year: '2026',
    days: [
      { label: '一', date: '8/31' },
      { label: '二', date: '9/1' },
      { label: '三', date: '9/2' },
      { label: '四', date: '9/3' },
      { label: '五', date: '9/4' },
      { label: '六', date: '9/5' },
      { label: '日', date: '9/6' },
    ],
    courses: [
      { day: 1, start: 3, end: 4, color: 2, name: '线性代数', room: '6B-501', teacher: '周敏' },
      { day: 2, start: 1, end: 2, color: 4, name: '电路分析基础', room: '计电楼202', teacher: '邓志刚' },
      { day: 3, start: 5, end: 6, color: 1, name: '大学物理(上)', room: '6B-603', teacher: '黄志强' },
      { day: 4, start: 3, end: 4, color: 3, name: '高级语言程序设计', room: '计电机房3', teacher: '陈晓明' },
      { day: 5, start: 7, end: 8, color: 5, name: '形势与政策', room: '汇学堂B201', teacher: '龙紫璐' },
    ],
  },

  next: {
    weekInfo: '第4周 · 大三上',
    date: '2026/9/13',
    year: '2026',
    days: [
      { label: '一', date: '9/14' },
      { label: '二', date: '9/15' },
      { label: '三', date: '9/16' },
      { label: '四', date: '9/17' },
      { label: '五', date: '9/18' },
      { label: '六', date: '9/19' },
      { label: '日', date: '9/20' },
    ],
    courses: [
      { day: 1, start: 1, end: 2, color: 3, name: '高等数学A(上)', room: '6B-603', teacher: '朱光军' },
      { day: 2, start: 3, end: 4, color: 1, name: '数据结构与算法', room: '计电楼405', teacher: '陈晓明' },
      { day: 3, start: 5, end: 6, color: 2, name: '计算方法', room: '计电楼303', teacher: '李海' },
      { day: 4, start: 1, end: 2, color: 2, name: '大学英语(三)', room: '6A-528', teacher: '陈永红' },
      { day: 5, start: 3, end: 4, color: 3, name: '计算机网络', room: '计电楼301', teacher: '李海' },
    ],
  },
}
