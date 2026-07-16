# Markdown 转微信推文格式技术方案

## 背景

Toolbox 当前已经覆盖数据库查询、Redis 查询、JSON 工具、图片生成、COS 文件上传等常用开发工具。新的需求是增加一个面向内容生产的工具：用户粘贴 Markdown 文档后，系统将其格式化为适合微信公众号编辑器粘贴的推文样式，并支持一键复制或导出，减少手动排版成本。

该功能需要支持不同模板。模板能力不能只做成固定 CSS 预览，而要输出微信公众号编辑器可接受的富文本 HTML，因此最终结果应尽量使用 inline style，保证复制到微信公众平台后台后样式可保留。

## GitHub 调研结论

已调研 GitHub 上与微信 Markdown 编辑器相关的项目：

- `doocs/md`：成熟度最高，定位就是 WeChat Markdown Editor，支持 Markdown、自定义主题、内容管理、多图床、AI 助手等。它是完整 Vue 应用，功能重，不适合直接整体引入当前 Go 工具箱，但主题和编辑器交互可以作为参考。
- `mdnice/markdown-nice`：老牌 Markdown 排版工具，核心价值是主题设计和多平台适配，支持微信公众号、知乎、掘金。其许可证为 GPL-3.0，不建议直接复制核心代码进当前项目，避免许可证污染。
- `koala9527/markdown2wechat`：形态最接近本需求，提供 Markdown 转微信公众号格式、实时预览、多主题配置，并支持自动拉取 mdnice 主题。可参考其接口和主题配置思路，但不直接依赖其实现。

结论：当前项目应实现轻量、内聚的 Markdown 转微信推文模块，借鉴成熟项目的主题模型，不引入完整前端应用或 GPL 代码。

## 目标

- 支持粘贴 Markdown 文档并生成微信公众号可粘贴的 HTML。
- 支持模板选择，第一版内置多个模板。
- 支持预览格式化结果。
- 支持一键复制富文本到剪贴板，便于粘贴到微信公众号编辑器。
- 支持导出 HTML 文件。
- 不做服务端内容存储，不保存用户粘贴的文章内容。

## 非目标

- 第一版不接入微信公众号草稿箱 API。
- 第一版不实现模板在线编辑 UI。
- 第一版不引入完整的 `doocs/md`、`mdnice` 或 Next/Vue 编辑器体系。
- 第一版不处理复杂协同编辑、文章管理、版本管理。

## 改动范围

- 新增 `internal/service/wechat.go`：Markdown 转 HTML、主题应用、导出 HTML 包装。
- 新增 `internal/handler/wechat.go`：提供格式化和主题列表接口。
- 修改 `internal/model/dto.go`：新增 WeChat Markdown 请求和响应 DTO。
- 修改 `internal/app/app.go`：注册新 handler。
- 修改 `internal/web/index.html`：侧边栏新增「微信推文」，页面新增 Markdown 输入、模板选择、预览、复制、导出。
- 新增 `internal/web/static/js/wechat.js`：前端交互、接口调用、复制和导出。
- 修改 `internal/web/static/css/main.css`：微信推文页面布局和预览样式。
- 修改 `README.md`：补充功能说明。
- 修改 `go.mod` / `go.sum`：引入 Markdown 解析依赖。

## 接口契约

### 获取模板列表

`GET /api/wechat/themes`

响应：

```json
{
  "code": 0,
  "msg": "ok",
  "data": [
    { "id": "red", "name": "红绯强调" },
    { "id": "default", "name": "默认清爽" },
    { "id": "tech", "name": "技术文章" },
  ]
}
```

### 格式化 Markdown

`POST /api/wechat/format`

请求：

```json
{
  "markdown": "# 标题\n\n内容",
  "theme": "tech"
}
```

响应：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "html": "<section style=\"...\">...</section>",
    "full_html": "<!doctype html>...",
    "theme": "tech"
  }
}
```

## 模板设计

第一版内置三个模板，默认使用 `red`：

- `default`：默认清爽，适合普通说明文和工作总结。
- `tech`：技术文章，强调代码块、引用、层级标题。
- `red`：红绯强调，适合观点表达、重点句标红、竖条标题的极简公众号排版。

模板粒度覆盖：

- 正文容器
- `h1` / `h2` / `h3`
- 段落
- 引用
- 有序列表 / 无序列表
- 代码块
- 行内代码
- 链接
- 图片
- 分割线
- 表格

模板样式应在服务端统一转换为 inline style，避免前端预览 CSS 与复制结果不一致。

## 技术实现

后端使用 `github.com/yuin/goldmark` 解析 Markdown。解析后生成标准 HTML，再对输出 HTML 应用微信模板样式。

优先保持实现简单：

- 服务端提供固定主题注册表。
- 每个主题定义元素级样式。
- 格式化接口只接收 Markdown 和主题 ID。
- 未匹配到主题时回退到 `red`。
- 空 Markdown 返回明确错误。

前端沿用当前原生 JavaScript 结构，不新增构建链：

- `wechat.js` 负责加载主题、提交格式化、渲染预览。
- 复制时优先使用 `ClipboardItem` 写入 `text/html`。
- 浏览器不支持富文本复制时，降级复制 HTML 源码。
- 导出时使用 `Blob` 生成 `.html` 文件。

## 风险与约束

- 微信公众号编辑器会清理部分 HTML 标签和 CSS，需要控制输出为基础标签和 inline style。
- 不同浏览器对富文本剪贴板 API 支持不同，必须提供降级路径。
- 如果后续要支持用户自定义模板，应将内置主题迁移到 `wechat_themes.json`，但第一版不引入该持久化复杂度。
- 不能直接复制 `mdnice/markdown-nice` GPL-3.0 代码。

## 验证计划

- 执行 `go test ./...`
- 执行 `go build ./...`
- 执行 `node --check internal/web/static/js/wechat.js`
- 手动验证标题、段落、列表、引用、代码块、表格、图片、链接的转换效果。
- 手动验证复制富文本到微信公众号编辑器的可用性。
- 手动验证导出 HTML 文件可打开且样式正常。
