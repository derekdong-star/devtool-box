let _wechatHTML = '';
let _wechatFullHTML = '';
const DEFAULT_WECHAT_THEME = 'red';

async function loadWeChatThemes() {
  const select = document.getElementById('wechatTheme');
  if (!select) return;

  const selected = select.value || DEFAULT_WECHAT_THEME;
  const res = await fetch('/api/wechat/themes').then(r => r.json()).catch(() => null);
  if (!ok(res)) return;

  select.innerHTML = (res.data || [])
    .map(theme => `<option value="${esc(theme.id)}">${esc(theme.name)}</option>`)
    .join('');
  select.value = Array.from(select.options).some(opt => opt.value === selected) ? selected : DEFAULT_WECHAT_THEME;
}

async function formatWeChatMarkdown() {
  const markdown = document.getElementById('wechatMarkdown').value.trim();
  const theme = document.getElementById('wechatTheme').value || DEFAULT_WECHAT_THEME;
  const preview = document.getElementById('wechatPreview');

  if (!markdown) {
    showError(preview, '请先粘贴 Markdown 内容');
    return;
  }

  preview.innerHTML = '<div class="empty-state"><div class="spinner"></div><p>正在生成微信推文格式...</p></div>';
  const res = await post('/api/wechat/format', { markdown, theme });
  if (!ok(res)) {
    showError(preview, err(res));
    return;
  }

  _wechatHTML = res.data.html || '';
  _wechatFullHTML = res.data.full_html || '';
  preview.innerHTML = _wechatHTML;
  document.getElementById('wechatThemeHint').textContent = `当前模板：${document.getElementById('wechatTheme').selectedOptions[0]?.textContent || res.data.theme}`;
  appear(preview);
}

async function copyWeChatRichHTML() {
  if (!_wechatHTML) {
    showToast('请先格式化 Markdown');
    return;
  }

  const plainText = htmlToPlainText(_wechatHTML);
  try {
    if (navigator.clipboard && window.ClipboardItem) {
      await navigator.clipboard.write([
        new ClipboardItem({
          'text/html': new Blob([_wechatHTML], { type: 'text/html' }),
          'text/plain': new Blob([plainText], { type: 'text/plain' }),
        }),
      ]);
      showToast('已复制富文本');
      return;
    }
  } catch {}

  copyRichHTMLBySelection(_wechatHTML);
  showToast('已复制富文本');
}

function copyRichHTMLBySelection(html) {
  const holder = document.createElement('div');
  holder.contentEditable = 'true';
  holder.style.cssText = 'position:fixed;left:-9999px;top:0;width:680px;';
  holder.innerHTML = html;
  document.body.appendChild(holder);

  const range = document.createRange();
  range.selectNodeContents(holder);
  const selection = window.getSelection();
  selection.removeAllRanges();
  selection.addRange(range);
  try { document.execCommand('copy'); } catch {}
  selection.removeAllRanges();
  document.body.removeChild(holder);
}

function copyWeChatHTML() {
  if (!_wechatHTML) {
    showToast('请先格式化 Markdown');
    return;
  }
  copyString(_wechatHTML);
  showToast('已复制 HTML');
}

function downloadWeChatHTML() {
  if (!_wechatFullHTML) {
    showToast('请先格式化 Markdown');
    return;
  }

  const blob = new Blob([_wechatFullHTML], { type: 'text/html;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `wechat-article-${new Date().toISOString().slice(0, 10)}.html`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

function htmlToPlainText(html) {
  const div = document.createElement('div');
  div.innerHTML = html;
  return div.textContent || div.innerText || '';
}

function insertWeChatSample() {
  document.getElementById('wechatMarkdown').value = `# Toolbox 微信推文示例

这是一段用于验证排版效果的 Markdown。它会被转换成适合微信公众号编辑器粘贴的富文本。

## 核心能力

- 支持标题、段落、列表和引用
- 支持代码块和行内代码
- 支持表格、链接和图片

> 复制时会优先写入 text/html，便于直接粘贴到微信公众平台。

**工具变强，不等于你的人生系统自动变强。**

### 代码示例

\`\`\`go
func main() {
    fmt.Println("hello wechat")
}
\`\`\`

| 模板 | 场景 |
| --- | --- |
| 默认清爽 | 普通说明文 |
| 技术文章 | 技术博客 |
| 红绯强调 | 观点表达 |
`;
}

if (_activeTab === 'wechat') loadWeChatThemes();
