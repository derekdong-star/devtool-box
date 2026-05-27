let _uploadConfig = {
  secret_id: '',
  secret_key: '',
  bucket: '',
  region: '',
  domain: '',
  public_base_url: '',
  path_prefix: 'uploads',
  use_signed_url: false,
  signed_url_expire_seconds: 3600,
};
let _selectedCOSFile = null;

const UPLOAD_DIRECTORY_HISTORY_KEY = 'dtb_upload_directory_history';
const MAX_UPLOAD_DIRECTORY_HISTORY = 8;

if (typeof removeTabState === 'function') {
  removeTabState('upload');
}

async function loadUploadConfig() {
  const res = await fetch('/api/config/upload').then(r => r.json());
  if (!ok(res)) {
    showToast('读取 COS 配置失败: ' + err(res));
    return;
  }
  _uploadConfig = res.data || _uploadConfig;
  document.getElementById('uploadSecretId').value = _uploadConfig.secret_id || '';
  document.getElementById('uploadSecretKey').value = _uploadConfig.secret_key || '';
  document.getElementById('uploadBucket').value = _uploadConfig.bucket || '';
  document.getElementById('uploadRegion').value = _uploadConfig.region || '';
  document.getElementById('uploadDomain').value = _uploadConfig.domain || '';
  document.getElementById('uploadPublicBaseUrl').value = _uploadConfig.public_base_url || '';
  document.getElementById('uploadPathPrefix').value = _uploadConfig.path_prefix || 'uploads';
  document.getElementById('uploadSignedURLExpire').value = _uploadConfig.signed_url_expire_seconds || 3600;
  document.getElementById('uploadUseSignedUrl').checked = !!_uploadConfig.use_signed_url;
  renderUploadDirectoryHistory();
}

async function saveUploadConfig() {
  const cfg = {
    secret_id: document.getElementById('uploadSecretId').value.trim(),
    secret_key: document.getElementById('uploadSecretKey').value.trim(),
    bucket: document.getElementById('uploadBucket').value.trim(),
    region: document.getElementById('uploadRegion').value.trim(),
    domain: document.getElementById('uploadDomain').value.trim(),
    public_base_url: document.getElementById('uploadPublicBaseUrl').value.trim(),
    path_prefix: document.getElementById('uploadPathPrefix').value.trim(),
    use_signed_url: document.getElementById('uploadUseSignedUrl').checked,
    signed_url_expire_seconds: parseInt(document.getElementById('uploadSignedURLExpire').value || '3600', 10) || 3600,
  };

  const res = await post('/api/config/upload/save', cfg);
  if (!ok(res)) {
    showToast('保存失败: ' + err(res));
    return;
  }
  _uploadConfig = cfg;
  showToast('COS 配置已保存');
}

function handleCOSFileSelect(input) {
  const file = input.files && input.files[0];
  if (!file) return;
  _selectedCOSFile = file;

  const zone = document.getElementById('cosUploadZone');
  const hint = document.getElementById('cosUploadHint');
  const selected = document.getElementById('cosUploadSelected');
  const removeBtn = document.getElementById('cosRemoveBtn');

  selected.innerHTML = `
    <div class="upload-selected-name">${esc(file.name)}</div>
    <div class="upload-selected-meta">${formatUploadFileSize(file.size)} · ${esc(file.type || 'application/octet-stream')}</div>
  `;
  selected.style.display = 'flex';
  hint.style.display = 'none';
  removeBtn.style.display = 'inline-flex';
  zone.classList.add('has-image');
}

function clearCOSFile() {
  _selectedCOSFile = null;
  document.getElementById('cosFileInput').value = '';
  document.getElementById('cosUploadSelected').style.display = 'none';
  document.getElementById('cosUploadHint').style.display = 'flex';
  document.getElementById('cosRemoveBtn').style.display = 'none';
  document.getElementById('cosUploadZone').classList.remove('has-image');
}

function loadUploadDirectoryHistory() {
  try {
    const raw = JSON.parse(localStorage.getItem(UPLOAD_DIRECTORY_HISTORY_KEY) || '[]');
    if (!Array.isArray(raw)) return [];
    return raw
      .map(item => sanitizeUploadDirectoryValue(item))
      .filter(Boolean)
      .slice(0, MAX_UPLOAD_DIRECTORY_HISTORY);
  } catch {
    return [];
  }
}

function saveUploadDirectoryHistory(history) {
  localStorage.setItem(
    UPLOAD_DIRECTORY_HISTORY_KEY,
    JSON.stringify(history.slice(0, MAX_UPLOAD_DIRECTORY_HISTORY)),
  );
}

function rememberUploadDirectory(directory) {
  const normalized = sanitizeUploadDirectoryValue(directory);
  if (!normalized) return;

  const nextHistory = [
    normalized,
    ...loadUploadDirectoryHistory().filter(item => item !== normalized),
  ].slice(0, MAX_UPLOAD_DIRECTORY_HISTORY);

  saveUploadDirectoryHistory(nextHistory);
  renderUploadDirectoryHistory(nextHistory);
}

function renderUploadDirectoryHistory(history = loadUploadDirectoryHistory()) {
  const datalist = document.getElementById('uploadDirectoryHistory');
  if (!datalist) return;

  datalist.innerHTML = history
    .map(item => `<option value="${esc(item)}"></option>`)
    .join('');
}

function sanitizeUploadDirectoryValue(value) {
  return String(value || '').trim();
}

async function uploadFileToCOS() {
  if (!_selectedCOSFile) {
    showToast('请先选择文件');
    return;
  }

  const btnText = document.getElementById('uploadBtnText');
  const resultArea = document.getElementById('uploadResultArea');
  const directory = sanitizeUploadDirectoryValue(document.getElementById('uploadDirectory').value);
  btnText.textContent = '上传中...';
  resultArea.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>上传中，请稍候...</p></div>`;

  const form = new FormData();
  form.append('file', _selectedCOSFile);
  form.append('directory', directory);
  form.append('name', document.getElementById('uploadObjectName').value.trim());
  form.append('use_signed_url', String(document.getElementById('uploadForceSignedUrl').checked));

  let res;
  try {
    const fetchRes = await fetchWithTimeout('/api/upload/file', { method: 'POST', body: form }, 300000);
    res = await fetchRes.json();
  } catch (e) {
    resultArea.innerHTML = `<div class="empty-state" style="color:var(--danger)"><p>请求失败: ${esc(describeRequestError(e))}</p></div>`;
    btnText.textContent = '上传到 COS';
    return;
  }

  btnText.textContent = '上传到 COS';
  if (!ok(res)) {
    showError(resultArea, err(res));
    return;
  }

  rememberUploadDirectory(directory);
  renderUploadResult(res.data || {});
}

function renderUploadResult(data) {
  const resultArea = document.getElementById('uploadResultArea');
  resultArea.innerHTML = `
    <div class="upload-result-card result-appear">
      <div class="upload-result-row">
        <span class="upload-result-label">文件名</span>
        <span class="upload-result-value mono">${esc(data.original_name || '')}</span>
      </div>
      <div class="upload-result-row">
        <span class="upload-result-label">对象 Key</span>
        <span class="upload-result-value mono">${esc(data.key || '')}</span>
      </div>
      <div class="upload-result-row">
        <span class="upload-result-label">类型</span>
        <span class="upload-result-value mono">${esc(data.content_type || '')}</span>
      </div>
      <div class="upload-result-row">
        <span class="upload-result-label">大小</span>
        <span class="upload-result-value">${esc(formatUploadFileSize(data.size || 0))}</span>
      </div>
      <div class="upload-result-url-block">
        <div class="card-title" style="margin-bottom:6px">资源链接</div>
        <textarea id="uploadResultUrl" readonly>${esc(data.url || '')}</textarea>
      </div>
      <div class="toolbar">
        <button class="btn btn-secondary" onclick="copyText('uploadResultUrl')">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
          复制链接
        </button>
        <button class="btn btn-primary" onclick="window.open(document.getElementById('uploadResultUrl').value, '_blank', 'noopener')">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 3h7v7"/><path d="M10 14L21 3"/><path d="M21 14v7h-7"/><path d="M3 10V3h7"/><path d="M3 21l7-7"/></svg>
          打开链接
        </button>
      </div>
    </div>
  `;
}

function formatUploadFileSize(size) {
  const n = Number(size) || 0;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(2)} MB`;
}

document.addEventListener('DOMContentLoaded', () => {
  const zone = document.getElementById('cosUploadZone');
  const input = document.getElementById('cosFileInput');
  const removeBtn = document.getElementById('cosRemoveBtn');
  renderUploadDirectoryHistory();
  if (!zone || !input) return;

  zone.addEventListener('click', e => {
    if (e.target === input || e.target.closest('#cosRemoveBtn')) return;
    input.value = '';
    input.click();
  });

  if (removeBtn) {
    removeBtn.addEventListener('click', e => e.stopPropagation());
  }

  zone.addEventListener('dragover', e => { e.preventDefault(); zone.classList.add('dragover'); });
  zone.addEventListener('dragleave', () => zone.classList.remove('dragover'));
  zone.addEventListener('drop', e => {
    e.preventDefault();
    zone.classList.remove('dragover');
    const files = e.dataTransfer.files;
    if (files && files[0]) {
      const dt = new DataTransfer();
      dt.items.add(files[0]);
      input.files = dt.files;
      handleCOSFileSelect(input);
    }
  });
});
