# DESIGN.md — DevToolbox (Dopamine Edition)

> 多巴胺配色 × 工具台克制——明亮、活跃，但每个像素都有意义。

## 1. Visual Theme & Atmosphere

**Style**: Dopamine Minimal — Playful Creative × Minimal Pure 混搭
**Keywords**: 明亮、饱和、活力、简洁、高效、糖果、精准
**Tone**: 像一盒彩色圆珠笔摆在白色办公桌上 — NOT 卡通、NOT 暗沉、NOT 复杂
**Feel**: 打开就想用，用完就想截图分享

**Interaction Tier**: L1（精致静态 — 工具台不喧闹，但每次操作都有愉悦感）
**Dependencies**: CSS only

---

## 2. Color Palette & Roles

```css
:root {
  /* ── Backgrounds ── */
  --bg:            #ffffff;
  --bg-rgb:        255, 255, 255;
  --surface:       #f8f9fc;          /* 侧栏、卡片底 */
  --surface-alt:   #f1f3f9;          /* 输入框、嵌入区 */
  --surface-hover: #eef0f8;          /* hover 态 */

  /* ── Borders ── */
  --border:        #e4e6ef;
  --border-hover:  #c7cbe0;

  /* ── Text ── */
  --text:          #1a1d2e;          /* 主文字，深蓝近黑 */
  --text-secondary:#4b5068;          /* 次级，偏蓝灰 */
  --text-tertiary: #9398b0;          /* 占位符、hint */

  /* ── Dopamine Accent Palette ── */
  --violet:        #7c3aed;          /* 主强调：紫 */
  --violet-rgb:    124, 58, 237;
  --violet-light:  #ede9fe;
  --violet-border: rgba(124, 58, 237, 0.25);

  --pink:          #ec4899;          /* 辅色：粉 */
  --pink-rgb:      236, 72, 153;
  --pink-light:    #fce7f3;

  --cyan:          #06b6d4;          /* 辅色：青 */
  --cyan-rgb:      6, 182, 212;
  --cyan-light:    #cffafe;

  --amber:         #f59e0b;          /* 辅色：琥珀 */
  --amber-rgb:     245, 158, 11;
  --amber-light:   #fef3c7;

  --emerald:       #10b981;          /* 成功色 */
  --emerald-rgb:   16, 185, 129;
  --emerald-light: #d1fae5;

  /* ── Accent alias (默认用 violet) ── */
  --accent:        var(--violet);
  --accent-rgb:    var(--violet-rgb);
  --accent-light:  var(--violet-light);
  --accent-border: var(--violet-border);
  --accent-hover:  #6d28d9;

  /* ── Semantic ── */
  --success:       var(--emerald);
  --success-rgb:   var(--emerald-rgb);
  --success-light: var(--emerald-light);
  --danger:        #ef4444;
  --danger-rgb:    239, 68, 68;
  --danger-light:  #fee2e2;
  --warning:       var(--amber);
  --warning-light: var(--amber-light);

  /* ── Radius ── */
  --radius-sm:  4px;
  --radius:     8px;
  --radius-md:  10px;
  --radius-lg:  14px;
  --radius-xl:  20px;

  /* ── Shadow ── */
  --shadow-sm:  0 1px 3px rgba(26,29,46,0.06), 0 1px 2px rgba(26,29,46,0.04);
  --shadow-md:  0 4px 12px rgba(26,29,46,0.08), 0 2px 4px rgba(26,29,46,0.04);
  --shadow-lg:  0 8px 24px rgba(26,29,46,0.10), 0 4px 8px rgba(26,29,46,0.05);
  --shadow-color: 0 4px 16px rgba(var(--accent-rgb), 0.22);
}
```

**Color Rules:**
- 所有颜色通过 CSS 变量引用，禁止硬编码 hex
- 同一功能区只用一个强调色，页面整体不超过 3 个强调色同时出现
- 背景保持白色，多巴胺感来自强调色和 badge，不做彩色大背景
- 错误用 `--danger`，成功用 `--success`，警告用 `--warning`，语义严格

---

## 3. Typography Rules

```css
@import url('https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap');

:root {
  --font-sans: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
}
```

| Role | Font | Size | Weight | Line Height | Letter Spacing |
|------|------|------|--------|-------------|----------------|
| 品牌名 | Plus Jakarta Sans | 14px | 700 | — | -0.3px |
| 导航项 | Plus Jakarta Sans | 13px | 500 | — | — |
| 页面标题 | Plus Jakarta Sans | 15px | 700 | — | -0.2px |
| 卡片标题 | Plus Jakarta Sans | 11px | 700 | — | 0.5px (uppercase) |
| 正文 | Plus Jakarta Sans | 13px | 400 | 1.6 | — |
| 代码/输出 | JetBrains Mono | 12px | 400 | 1.65 | — |
| hint | Plus Jakarta Sans | 12px | 400 | — | — |

**Typography Rules:**
- 卡片标题用 `text-transform: uppercase` + `letter-spacing: 0.5px`，营造专业感
- 代码输出区必须用 `--font-mono`
- **NEVER use**: 系统 serif 字体，Roboto，Comic Sans

**Text Decoration:**
- 品牌名：无渐变（简洁工具台）
- 强调关键词（如 Tab active 状态）：`color: var(--accent)` 即可，不做渐变流动

---

## 4. Component Stylings

### Buttons

```css
.btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 16px;
  height: 34px;
  border-radius: var(--radius);
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 600;
  border: 1.5px solid transparent;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s ease, transform 0.12s ease,
              box-shadow 0.15s ease, border-color 0.15s ease;
  user-select: none;
}

/* Primary — violet 渐变 */
.btn-primary {
  background: linear-gradient(135deg, #7c3aed 0%, #6d28d9 100%);
  color: #fff;
  border-color: transparent;
  box-shadow: 0 2px 8px rgba(124, 58, 237, 0.3);
}
.btn-primary:hover {
  background: linear-gradient(135deg, #6d28d9 0%, #5b21b6 100%);
  transform: translateY(-1px);
  box-shadow: 0 4px 14px rgba(124, 58, 237, 0.4);
}
.btn-primary:active { transform: translateY(0) scale(0.97); box-shadow: none; }
.btn-primary:focus-visible { outline: 2px solid var(--violet); outline-offset: 2px; }
.btn-primary:disabled { opacity: 0.45; cursor: not-allowed; transform: none; box-shadow: none; }

/* Secondary */
.btn-secondary {
  background: var(--bg);
  color: var(--text-secondary);
  border-color: var(--border);
  box-shadow: var(--shadow-sm);
}
.btn-secondary:hover {
  background: var(--surface-hover);
  border-color: var(--border-hover);
  color: var(--text);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}
.btn-secondary:active { transform: scale(0.97); box-shadow: none; }

/* Ghost */
.btn-ghost {
  background: transparent;
  color: var(--text-tertiary);
  border-color: transparent;
}
.btn-ghost:hover { background: var(--surface-alt); color: var(--text-secondary); }
```

### Cards

```css
.card {
  background: var(--bg);
  border: 1.5px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  box-shadow: var(--shadow-sm);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
.card:hover {
  border-color: var(--border-hover);
  box-shadow: var(--shadow-md);
}
```

### Navigation

```css
.sidebar { background: var(--surface); border-right: 1.5px solid var(--border); }
.nav-item { color: var(--text-secondary); border-left: 2px solid transparent; }
.nav-item:hover { background: var(--surface-hover); color: var(--text); }
.nav-item.active {
  border-left-color: var(--violet);
  background: var(--violet-light);
  color: var(--violet);
  font-weight: 600;
}
```

### Input / Textarea / Select

```css
input, textarea, select {
  background: var(--surface-alt);
  color: var(--text);
  border: 1.5px solid var(--border);
  border-radius: var(--radius);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}
input:focus, textarea:focus, select:focus {
  background: var(--bg);
  border-color: var(--violet);
  box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.12);
}
input::placeholder, textarea::placeholder { color: var(--text-tertiary); }
```

### Tags / Badges

```css
.badge {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 2px 8px; border-radius: var(--radius-xl);
  font-size: 11px; font-weight: 600;
  background: var(--violet-light); color: var(--violet);
  border: 1px solid var(--violet-border);
}
.badge-success { background: var(--emerald-light); color: var(--emerald); border-color: rgba(16,185,129,.25); }
.badge-danger  { background: var(--danger-light);  color: var(--danger);  border-color: rgba(239,68,68,.25); }
.badge-pink    { background: var(--pink-light);    color: var(--pink);    border-color: rgba(236,72,153,.25); }
.badge-cyan    { background: var(--cyan-light);    color: var(--cyan);    border-color: rgba(6,182,212,.25); }
.badge-amber   { background: var(--amber-light);   color: var(--amber);   border-color: rgba(245,158,11,.25); }
.badge-neutral { background: var(--surface-alt);   color: var(--text-secondary); border-color: var(--border); }
```

### Table

```css
thead th { background: var(--surface); color: var(--text-tertiary); font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; font-size: 11px; }
tbody td { color: var(--text); border-bottom: 1px solid var(--border); }
tbody tr:hover td { background: var(--surface); }
```

---

## 5. Layout Principles

- **侧栏宽度**: 240px
- **内容区 padding**: 20px
- **卡片 gap**: 16px
- **Grid**: 2 列等宽
- **border-radius 整体偏大**: 体现多巴胺圆润感（卡片 14px，按钮 8px，badge 20px）

---

## 6. Depth & Elevation

| Level | Treatment | Use |
|-------|-----------|-----|
| Flat | 无阴影 | 输入框内嵌区域 |
| Subtle | `var(--shadow-sm)` | 卡片默认态 |
| Elevated | `var(--shadow-md)` | 卡片 hover、按钮 hover |
| Pop | `var(--shadow-color)` | 主按钮 hover（带色彩阴影）|

---

## 7. Animation & Interaction — L1

**Motion Philosophy**: 轻快但不浮夸，每次操作都有明确的视觉回应

```css
/* 结果区出现 */
@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: translateY(0); }
}
.result-appear { animation: fadeInUp 0.22s cubic-bezier(0.16, 1, 0.3, 1); }

/* 复制成功绿色闪烁 */
@keyframes flashSuccess {
  0%, 100% { border-color: var(--border); box-shadow: none; }
  40%       { border-color: var(--emerald); box-shadow: 0 0 0 3px var(--emerald-light); }
}
.flash-success { animation: flashSuccess 0.7s ease; }

/* 按钮 hover 上浮 */
.btn:hover { transform: translateY(-1px); }
.btn:active { transform: scale(0.97); }

/* 卡片 hover 阴影加深 */
.card:hover { box-shadow: var(--shadow-md); }

/* 输入框 focus glow */
input:focus, textarea:focus { box-shadow: 0 0 0 3px rgba(124,58,237,0.12); }
```

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 8. Do's and Don'ts

### Do
- ✅ 用圆润的 border-radius（卡片 14px，badge pill 形）
- ✅ 强调色用 violet 渐变按钮，视觉重心明确
- ✅ 用彩色 badge 区分不同状态（成功/错误/信息）
- ✅ 代码/输出区用 `--surface-alt`（浅灰）区分于白色背景
- ✅ hover 时阴影加深，给予立体感和可点击暗示

### Don't
- ❌ 禁止黑色/深灰大背景（白底是多巴胺风格的基础）
- ❌ 禁止超过 3 个强调色在同一视图中竞争注意力
- ❌ 禁止 `border-radius: 0`（直角与整体风格冲突）
- ❌ 禁止用灰色按钮作为主操作按钮
- ❌ 禁止大面积背景色块（如整个卡片填充紫色）
- ❌ 禁止硬编码 hex 颜色值
- ❌ 禁止卡片内部出现多种不同字体
- ❌ 禁止 `text-transform: uppercase` 用于正文内容（仅限卡片标题/表头）

---

## 9. Responsive Behavior

| Name | Width | Key Changes |
|------|-------|-------------|
| Desktop | > 900px | 2 列 Grid，侧栏 240px |
| Tablet | 700-900px | 2 列 → 1 列 Grid |
| Mobile | < 700px | 侧栏变顶部横向 Tab Bar |

**Touch Targets:** minimum 44×44px
**Collapsing Strategy:** 侧栏折叠为底部横向 Tab，内容区占满屏幕
