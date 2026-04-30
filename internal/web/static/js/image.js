/* ─── Image Config ───────────────────────────────────────────── */

let _imageConfig = { api_url: '', api_key: '', models: [] };
let _uploadedImageFile = null;

async function loadImageConfig() {
  const res = await fetch('/api/config/image').then(r => r.json());
  if (!ok(res)) {
    showToast('读取配置失败: ' + err(res));
    return;
  }
  _imageConfig = res.data || { api_url: '', api_key: '', models: [] };

  document.getElementById('imageApiUrl').value = _imageConfig.api_url || '';
  document.getElementById('imageApiKey').value = _imageConfig.api_key || '';
  document.getElementById('imageModels').value = (_imageConfig.models || []).join('\n');

  renderImageModelSelect(_imageConfig.models || []);
}

async function saveImageConfig() {
  const url   = document.getElementById('imageApiUrl').value.trim();
  const key   = document.getElementById('imageApiKey').value;
  const raw   = document.getElementById('imageModels').value;
  const models = raw.split('\n').map(s => s.trim()).filter(Boolean);

  const cfg = { api_url: url, api_key: key, models };
  const res = await post('/api/config/image/save', cfg);
  if (!ok(res)) {
    showToast('保存失败: ' + err(res));
    return;
  }
  _imageConfig = cfg;
  renderImageModelSelect(models);
  showToast('配置已保存');
}

function renderImageModelSelect(models) {
  const select = document.getElementById('imageModel');
  if (!models || !models.length) {
    select.innerHTML = '<option value="">— 未配置模型 —</option>';
    return;
  }
  select.innerHTML = '<option value="">— 选择模型 —</option>' +
    models.map(m => `<option value="${esc(m)}">${esc(m)}</option>`).join('');
}

/* ─── Image Upload ───────────────────────────────────────────── */

function handleImageUpload(input) {
  const file = input.files[0];
  if (!file) return;
  _uploadedImageFile = file;

  const reader = new FileReader();
  reader.onload = e => {
    const img = document.getElementById('imagePreview');
    img.src = e.target.result;
    img.style.display = 'block';
    document.getElementById('imageUploadHint').style.display = 'none';
    document.getElementById('imageRemoveBtn').style.display = 'inline-flex';
    document.getElementById('imageUploadZone').classList.add('has-image');
  };
  reader.readAsDataURL(file);
}

function removeUploadedImage() {
  _uploadedImageFile = null;
  document.getElementById('imageFileInput').value = '';
  document.getElementById('imagePreview').style.display = 'none';
  document.getElementById('imageUploadHint').style.display = 'flex';
  document.getElementById('imageRemoveBtn').style.display = 'none';
  document.getElementById('imageUploadZone').classList.remove('has-image');
}

// 拖拽上传 + 结果区事件委托
document.addEventListener('DOMContentLoaded', () => {
  const zone = document.getElementById('imageUploadZone');
  if (zone) {
    zone.addEventListener('dragover', e => { e.preventDefault(); zone.classList.add('dragover'); });
    zone.addEventListener('dragleave', () => zone.classList.remove('dragover'));
    zone.addEventListener('drop', e => {
      e.preventDefault();
      zone.classList.remove('dragover');
      const files = e.dataTransfer.files;
      if (files && files[0]) {
        const input = document.getElementById('imageFileInput');
        const dt = new DataTransfer();
        dt.items.add(files[0]);
        input.files = dt.files;
        handleImageUpload(input);
      }
    });
  }

  const resultArea = document.getElementById('imageResultArea');
  if (resultArea) {
    resultArea.addEventListener('click', e => {
      const btn = e.target.closest('[data-action]');
      if (!btn) return;
      const action = btn.dataset.action;
      const url    = btn.dataset.url;
      const idx    = parseInt(btn.dataset.idx || '0', 10);
      if (action === 'preview')  { previewImage(url); }
      if (action === 'download') { downloadImage(url, idx); }
      if (action === 'copy')     { copyString(url); showToast('链接已复制'); }
    });
  }
});

/* ─── Image Generation ───────────────────────────────────────── */

async function generateImage() {
  const prompt = document.getElementById('imagePrompt').value.trim();
  const model  = document.getElementById('imageModel').value;
  const size   = document.getElementById('imageSize').value;
  const resultArea = document.getElementById('imageResultArea');
  const countBadge = document.getElementById('imageResultCount');
  const btnText    = document.getElementById('imageGenBtnText');

  if (!prompt) { showToast('请输入提示词'); return; }
  if (!model)  { showToast('请选择模型'); return; }

  btnText.textContent = '生成中...';
  resultArea.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>生成中，请稍候...</p></div>`;
  countBadge.innerHTML = '';

  let res;
  try {
    if (_uploadedImageFile) {
      const form = new FormData();
      form.append('image', _uploadedImageFile);
      form.append('prompt', prompt);
      form.append('model', model);
      form.append('size', size);

      const fetchRes = await fetchWithTimeout('/api/image/generate-with-image', { method: 'POST', body: form }, 300000);
      res = await fetchRes.json();
    } else {
      res = await post('/api/image/generate', { prompt, model, size, n: 1 }, 300000);
    }
  } catch (e) {
    resultArea.innerHTML = `<div class="empty-state" style="color:var(--danger)"><p>请求失败: ${esc(e.message)}</p></div>`;
    btnText.textContent = '生成图片';
    return;
  }

  btnText.textContent = '生成图片';

  if (!ok(res)) {
    showError(resultArea, err(res));
    return;
  }

  const data = res.data || {};
  const items = data.data || [];
  if (!items.length) {
    resultArea.innerHTML = '<div class="empty-state"><p>未返回图片结果</p></div>';
    return;
  }

  countBadge.innerHTML = `<span class="badge badge-neutral">${items.length} 张</span>`;

  let html = '<div class="image-result-grid">';
  items.forEach((item, idx) => {
    const url = item.url || '';
    html += `
      <div class="image-result-card">
        <img src="${esc(url)}" alt="result-${idx}" onerror="this.src='data:image/svg+xml,%3Csvg xmlns=%27http://www.w3.org/2000/svg%27 width=%27400%27 height=%27400%27%3E%3Crect width=%27400%27 height=%27400%27 fill=%27%23f1f3f9%27/%3E%3Ctext x=%2750%25%27 y=%2750%25%27 dominant-baseline=%27middle%27 text-anchor=%27middle%27 font-size=%2714%27 fill=%27%239398b0%27%3E加载失败%3C/text%3E%3C/svg%3E'">
        <div class="image-actions">
          <button class="btn btn-secondary btn-sm" data-action="preview" data-url="${esc(url)}">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
            查看
          </button>
          <button class="btn btn-secondary btn-sm" data-action="download" data-url="${esc(url)}" data-idx="${idx}">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            下载
          </button>
          <button class="btn btn-secondary btn-sm" data-action="copy" data-url="${esc(url)}">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
            复制
          </button>
        </div>
      </div>`;
  });
  html += '</div>';
  resultArea.innerHTML = html;
  appear(resultArea);
}

function extFromDataURL(url) {
  const m = url.match(/^data:image\/(\w+);/);
  return m ? m[1] : 'png';
}

function downloadImage(url, idx) {
  const a = document.createElement('a');
  a.href = url;
  a.download = `generated-${idx + 1}.${extFromDataURL(url)}`;
  a.target = '_blank';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

function previewImage(url) {
  const modal = document.getElementById('imagePreviewModal');
  const img = document.getElementById('imagePreviewFull');
  img.src = url;
  modal.style.display = 'flex';
  document.addEventListener('keydown', handlePreviewKeydown);
}

function closeImagePreview() {
  const modal = document.getElementById('imagePreviewModal');
  modal.style.display = 'none';
  document.getElementById('imagePreviewFull').src = '';
  document.removeEventListener('keydown', handlePreviewKeydown);
}

function handlePreviewKeydown(e) {
  if (e.key === 'Escape') closeImagePreview();
}
