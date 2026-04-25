/* ─── 全局 Modal ─────────────────────────────────────────────── */
let _modalResolve = null;

function openModal(title, defaultVal) {
  return new Promise(resolve => {
    _modalResolve = resolve;
    document.getElementById('modalTitle').textContent = title;
    const input = document.getElementById('modalInput');
    input.value = defaultVal || '';
    document.getElementById('globalModal').style.display = 'flex';
    setTimeout(() => { input.focus(); input.select(); }, 50);
    input.onkeydown = e => { if (e.key === 'Enter') confirmModal(); if (e.key === 'Escape') closeModal(); };
  });
}

function confirmModal() {
  const val = document.getElementById('modalInput').value.trim();
  document.getElementById('globalModal').style.display = 'none';
  if (_modalResolve) { _modalResolve(val || null); _modalResolve = null; }
}

function closeModal() {
  document.getElementById('globalModal').style.display = 'none';
  if (_modalResolve) { _modalResolve(null); _modalResolve = null; }
}

/* ─── 命令模板 ───────────────────────────────────────────────── */

async function saveAsTemplate(command, kind) {
  closeSqlHistory();
  closeRedisCmdHistory();

  const name = await openModal(
    `存为${kind === 'sql' ? ' SQL' : ' Redis'} 命令模板`,
    command.slice(0, 40)
  );
  if (!name) return;

  const res = await post('/api/cmd-templates/save', { name, command, kind });
  if (!ok(res)) { alert('保存失败：' + err(res)); return; }
}

async function loadTemplates() {
  const kind = document.getElementById('templateKindFilter')?.value || '';
  const url  = '/api/cmd-templates' + (kind ? '?kind=' + kind : '');
  const res  = await fetch(url).then(r => r.json());
  const el   = document.getElementById('templateList');
  if (!ok(res)) { showError(el, err(res)); return; }

  const list = res.data || [];
  if (!list.length) {
    el.innerHTML = `<div class="empty-state">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
      <p>暂无模板，在 SQL / Redis 历史记录中点击 ★ 收藏</p>
    </div>`;
    return;
  }

  el.innerHTML = list.map(t => `
    <div class="template-item">
      <div class="template-item-body" onclick="applyTemplate(${JSON.stringify(t.command)}, ${JSON.stringify(t.kind)})" title="点击回填命令">
        <div class="template-item-name">${esc(t.name)}</div>
        <div class="template-item-cmd">${esc(t.command)}</div>
      </div>
      <span class="template-item-kind">
        <span class="badge ${t.kind === 'sql' ? 'badge-cyan' : 'badge-pink'}" style="font-size:10px">${t.kind.toUpperCase()}</span>
      </span>
      <div class="template-item-del" onclick="deleteTemplate(${JSON.stringify(t.id)}, event)" title="删除">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14H6L5 6"/><path d="M9 6V4h6v2"/></svg>
      </div>
    </div>`).join('');
  appear(el);
}

function applyTemplate(command, kind) {
  copyString(command);
  if (kind === 'sql') {
    document.getElementById('dbQuery').value = command;
    const item = [...document.querySelectorAll('.nav-item')].find(n => n.dataset.target === 'db');
    if (item) item.click();
    switchDbTab('query');
  } else {
    document.getElementById('redisCommand').value = command;
    const item = [...document.querySelectorAll('.nav-item')].find(n => n.dataset.target === 'redis');
    if (item) item.click();
    switchRedisTab('exec');
  }
  showToast('已复制并回填');
}

async function deleteTemplate(id, e) {
  if (e) { e.stopPropagation(); e.preventDefault(); }
  const res = await post('/api/cmd-templates/delete', { id });
  if (!ok(res)) { alert('删除失败：' + err(res)); return; }
  loadTemplates();
}

/* ─── 模板下拉（SQL / Redis 执行区）────────────────────────────── */
async function loadAndRenderTemplates(kind, anchorBtn) {
  const res = await fetch('/api/cmd-templates?kind=' + kind).then(r => r.json());
  if (!ok(res)) return;
  const list = res.data || [];

  let html = '';
  if (!list.length) {
    html = '<div class="sql-history-empty">暂无命令模板</div>';
  } else {
    html = list.map(t => `
      <div class="sql-history-item" data-action="apply-template" data-kind="${kind}" data-command="${esc(t.command)}">
        <span class="sql-history-item-sql" title="${esc(t.command)}">${esc(t.name)} <span style="color:var(--text-tertiary)">· ${esc(t.command)}</span></span>
        <span class="sql-history-item-save" data-action="copy-template" data-command="${esc(t.command)}" title="复制">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        </span>
        <span class="sql-history-item-del" data-action="delete-template" data-id="${esc(t.id)}" data-kind="${kind}" title="删除">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6L6 18M6 6l12 12"/></svg>
        </span>
      </div>`).join('');
  }
  if (anchorBtn) _openHistoryPortal(anchorBtn, 'template_' + kind, html);
}

function toggleSqlTemplates(e, btn) {
  e.stopPropagation();
  const type = _hdPortalType;
  if (type === 'template_sql') { _closeHistoryPortal(); return; }
  _closeHistoryPortal();
  loadAndRenderTemplates('sql', btn);
}

function toggleRedisTemplates(e, btn) {
  e.stopPropagation();
  const type = _hdPortalType;
  if (type === 'template_redis') { _closeHistoryPortal(); return; }
  _closeHistoryPortal();
  loadAndRenderTemplates('redis', btn);
}

function applyTemplateDirect(kind, command) {
  copyString(command);
  if (kind === 'sql') {
    document.getElementById('dbQuery').value = command;
    switchDbTab('query');
  } else {
    document.getElementById('redisCommand').value = command;
    switchRedisTab('exec');
  }
  _closeHistoryPortal();
  showToast('已复制并回填');
}

async function deleteTemplateAndRefresh(id, kind) {
  const res = await post('/api/cmd-templates/delete', { id });
  if (!ok(res)) { alert('删除失败：' + err(res)); return; }
  loadTemplates();
  _closeHistoryPortal();
}
