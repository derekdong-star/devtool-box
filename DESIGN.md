# Toolbox Brand Refresh

## 1. Visual Theme & Atmosphere

Toolbox 的视觉主题延续现有后台工具产品的清爽、克制和高密度信息感，但品牌表达从 “开发工具集合” 收敛为更清晰的 “专业工具工作台”。整体气质是：干净、精确、可信、轻微锋利。

关键词：
- modular
- precise
- calm
- technical
- bright

一句话定调：
`Toolbox 是一个面向后端与接口调试场景的轻量工作台，视觉上强调模块感与可控性。`

## 2. Color Palette & Roles

沿用现有浅色后台体系，保留紫色为主强调色，并加入更偏蓝灰的品牌承托色。

```css
:root {
  --bg: #ffffff;
  --surface: #f8f9fc;
  --surface-alt: #f1f3f9;
  --surface-hover: #eef0f8;

  --border: #e4e6ef;
  --border-hover: #c7cbe0;

  --text: #1a1d2e;
  --text-secondary: #4b5068;
  --text-tertiary: #9398b0;

  --brand-primary: #7c3aed;
  --brand-secondary: #4f46e5;
  --brand-tertiary: #0f172a;
  --brand-soft: rgba(124, 58, 237, 0.12);
  --brand-glow: rgba(79, 70, 229, 0.18);
}
```

## 3. Typography Rules

- 主字体继续使用 `Plus Jakarta Sans`
- 等宽字体继续使用 `JetBrains Mono`
- 品牌名使用更紧凑的字距与更高字重
- 副标题使用 11px 到 12px，小写或短句，不抢主层级

```css
.brand-name {
  font-size: 15px;
  font-weight: 800;
  letter-spacing: -0.04em;
}

.brand-subtitle {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
```

禁止：
- 不使用花哨衬线
- 不使用过大的彩虹渐变标题
- 不把工具页做成营销页

## 4. Component Stylings

- Sidebar 品牌区：改为品牌卡片式头部，左侧是独立方形 mark，中间是两行品牌文案，右侧保留退出按钮
- Topbar：增加紧凑品牌胶囊，与当前页面标题并列
- Login Card：顶部放置大尺寸 mark 与品牌名，形成清晰进入感
- Favicon：使用可在 16px 下识别的方形图形，不使用细线复杂图案

## 5. Layout Principles

- 品牌区高度略增加，形成应用级头部感
- Topbar 左侧先品牌、后页面信息，保持信息顺序清晰
- 不改变内容区的主网格结构
- 维持工具优先，品牌只是建立识别，不占据主要操作面积

## 6. Depth & Elevation

- Sidebar mark 使用柔和渐变与低强度阴影
- Topbar 品牌胶囊使用浅色描边与轻微内外层次
- Login 页品牌 mark 使用比主页面更明显的辉光

## 7. Animation & Interaction

- 保持现有静态工具站节奏，不引入重动画
- Sidebar 品牌 mark 和 topbar 胶囊允许 120ms 到 180ms 的 hover 过渡
- 不增加滚动驱动动画

## 8. Do's and Don'ts

- Do: 保持品牌表达简洁，优先可读性
- Do: 让 favicon、sidebar、login 页视觉一致
- Do: 让 Toolbox 在页面中显得像产品名，而不是目录名
- Do: 控制品牌区宽度，避免压缩导航
- Don’t: 改动后端服务名、Docker 服务名、module 名
- Don’t: 用复杂插画替代 logo
- Don’t: 在工具页头部加入营销式大 banner
- Don’t: 用过强的渐变覆盖文本

## 9. Responsive Behavior

- 小屏下 sidebar 品牌区继续可隐藏，但 logo 图形在登录页和浏览器标签仍要保留
- Topbar 品牌胶囊在窄屏下压缩为单行
- 品牌副标题在窄屏优先隐藏，不影响页面标题阅读
