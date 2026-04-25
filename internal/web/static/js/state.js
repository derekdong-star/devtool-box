/* ─── Tab State 持久化（localStorage）───────────────────────── */
const TAB_FIELDS = {
  db:     ['dbType', 'dbDsn', 'dbQuery'],
  redis:  ['redisAddr', 'redisDB', 'redisPattern', 'redisCommand'],
  cookie: ['cookieInput'],
  jwt:    ['jwtInput'],
  json:   ['jsonInput'],
  codec:  ['codecInput'],
  time:   ['tsInput'],
};
const TAB_STATE_KEY    = 'dtb_tab_state';
const ACTIVE_TAB_KEY   = 'dtb_active_tab';

function saveTabState(tabId) {
  const fields = TAB_FIELDS[tabId];
  if (!fields) return;
  try {
    const all = JSON.parse(localStorage.getItem(TAB_STATE_KEY) || '{}');
    all[tabId] = {};
    fields.forEach(id => {
      const el = document.getElementById(id);
      if (el) all[tabId][id] = el.value;
    });
    localStorage.setItem(TAB_STATE_KEY, JSON.stringify(all));
  } catch {}
}

function restoreTabState(tabId) {
  const fields = TAB_FIELDS[tabId];
  if (!fields) return;
  try {
    const all   = JSON.parse(localStorage.getItem(TAB_STATE_KEY) || '{}');
    const state = all[tabId] || {};
    fields.forEach(id => {
      const el = document.getElementById(id);
      if (el && state[id] != null) el.value = state[id];
    });
  } catch {}
}

/* ─── Tab navigation ─────────────────────────────────────────── */
const navItems = document.querySelectorAll('.nav-item');
const panels   = document.querySelectorAll('.panel');
const pageTitle = document.getElementById('pageTitle');
const pageDesc  = document.getElementById('pageDesc');

let _activeTab = localStorage.getItem(ACTIVE_TAB_KEY) || 'db';

navItems.forEach(item => {
  item.addEventListener('click', () => {
    saveTabState(_activeTab);

    navItems.forEach(n => n.classList.remove('active'));
    panels.forEach(p => p.classList.remove('active'));
    item.classList.add('active');

    const target = item.dataset.target;
    document.getElementById(target).classList.add('active');
    pageTitle.textContent = item.textContent.trim();
    pageDesc.textContent  = item.dataset.desc || '';

    restoreTabState(target);

    _activeTab = target;
    localStorage.setItem(ACTIVE_TAB_KEY, target);

    if (target === 'db')        loadSavedConns();
    if (target === 'redis')     loadRedisConns();
    if (target === 'templates') loadTemplates();
  });
});

/* ─── 历史下拉 Portal（挂在 body，绕开 overflow:hidden）──────── */
let _hdPortalType = null;

function _getPortal() { return document.getElementById('historyDropdownPortal'); }

function _openHistoryPortal(anchorBtn, type, contentHtml) {
  const portal = _getPortal();
  portal.innerHTML = `<div class="sql-history-dropdown" style="display:block;position:static;width:480px;max-width:90vw;max-height:340px;overflow-y:auto">${contentHtml}</div>`;

  const rect = anchorBtn.getBoundingClientRect();
  const pw   = 480;
  let left   = rect.right - pw;
  if (left < 8) left = 8;
  portal.style.left   = left + 'px';
  portal.style.top    = (rect.bottom + 6) + 'px';
  portal.style.display = 'block';
  _hdPortalType = type;
}

function _closeHistoryPortal() {
  _getPortal().style.display = 'none';
  _hdPortalType = null;
}

// 点击 portal 内部阻止冒泡 + 事件委托
_getPortal().addEventListener('click', e => {
  e.stopPropagation();
  const el = e.target.closest('[data-action]');
  if (!el) return;
  const action = el.dataset.action;
  const sql    = el.dataset.sql;
  const cmd    = el.dataset.cmd;
  const kind   = el.dataset.kind;

  if (action === 'apply-sql')         { applySqlHistory(sql); return; }
  if (action === 'apply-redis')       { applyRedisCmdHistory(cmd); return; }
  if (action === 'template')          { saveAsTemplate(cmd || sql, kind); return; }
  if (action === 'delete-sql')        { deleteSqlHistory(sql); return; }
  if (action === 'delete-redis')      { deleteRedisCmdHistory(cmd); return; }
  if (action === 'clear-sql')         { clearSqlHistory(); return; }
  if (action === 'clear-redis')       { clearRedisCmdHistory(); return; }
  if (action === 'apply-template')    { applyTemplateDirect(kind, el.dataset.command); return; }
  if (action === 'copy-template')     { copyString(el.dataset.command); showToast('已复制'); return; }
  if (action === 'delete-template')   { deleteTemplateAndRefresh(el.dataset.id, kind); return; }
});

document.addEventListener('click', () => {
  if (_hdPortalType !== null) _closeHistoryPortal();
});
