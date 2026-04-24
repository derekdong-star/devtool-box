/* ─── Tab State 持久化（localStorage）───────────────────────── */
// 每个 Tab 需要持久化的 input/textarea/select 字段
const TAB_FIELDS = {
  db:     ['dbType', 'dbDsn', 'dbQuery'],
  redis:  ['redisAddr', 'redisDB', 'redisPattern', 'redisCommand'],
  cookie: ['cookieInput'],
  jwt:    ['jwtInput'],
  json:   ['jsonInput'],
  codec:  ['codecInput'],
  time:   ['tsInput'],
};
const TAB_STATE_KEY    = 'dtb_tab_state';    // { tabId: { fieldId: value } }
const ACTIVE_TAB_KEY   = 'dtb_active_tab';   // 'db' | 'redis' | ...

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
    // 切换前保存当前 Tab 的输入状态
    saveTabState(_activeTab);

    navItems.forEach(n => n.classList.remove('active'));
    panels.forEach(p => p.classList.remove('active'));
    item.classList.add('active');

    const target = item.dataset.target;
    document.getElementById(target).classList.add('active');
    pageTitle.textContent = item.textContent.trim();
    pageDesc.textContent  = item.dataset.desc || '';

    // 恢复新 Tab 的输入状态
    restoreTabState(target);

    _activeTab = target;
    localStorage.setItem(ACTIVE_TAB_KEY, target);

    // 切换面板时刷新对应历史连接
    if (target === 'db')    loadSavedConns();
    if (target === 'redis') loadRedisConns();
  });
});

/* ─── HTTP helper ────────────────────────────────────────────── */
async function post(path, body) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  }).then(r => r.json());
  return res;
}

function ok(res)  { return res && res.code === 0; }
function err(res) { return (res && res.msg) ? res.msg : 'unknown error'; }

/* ─── Result helpers ─────────────────────────────────────────── */
function showError(el, msg) {
  el.innerHTML = `<div class="empty-state" style="color:var(--danger)">
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M15 9l-6 6M9 9l6 6"/></svg>
    <p>${esc(msg)}</p>
  </div>`;
}

function appear(el) {
  el.classList.remove('result-appear');
  void el.offsetWidth; // reflow
  el.classList.add('result-appear');
}

/* ─── Cookie ─────────────────────────────────────────────────── */
async function parseCookie() {
  const val = document.getElementById('cookieInput').value.trim();
  const el  = document.getElementById('cookieResult');
  if (!val) { showError(el, '请先粘贴 Cookie 字符串'); return; }

  const res = await post('/api/parse-cookie', { cookie: val });
  if (!ok(res)) { showError(el, err(res)); return; }
  if (!res.data || !res.data.length) {
    showError(el, '未解析到任何字段');
    return;
  }

  let html = '<table><thead><tr><th>Key</th><th>Value</th></tr></thead><tbody>';
  res.data.forEach(([k, v]) => {
    html += `<tr><td>${esc(k)}</td><td>${esc(v)}</td></tr>`;
  });
  html += '</tbody></table>';
  el.innerHTML = html;
  appear(el);
}

/* ─── JWT ────────────────────────────────────────────────────── */
async function parseJWT() {
  const val = document.getElementById('jwtInput').value.trim();
  const el  = document.getElementById('jwtResult');
  if (!val) { showError(el, '请先粘贴 JWT Token'); return; }

  const res = await post('/api/parse-jwt', { token: val });
  if (!ok(res)) { showError(el, err(res)); return; }

  const d = res.data;
  let html = '';

  // Header
  html += buildSection('Header', d.header);
  // Payload — exp/iat 额外显示可读时间
  html += buildPayloadSection('Payload', d.payload);
  // Signature
  html += `<div style="margin-top:12px">
    <div class="card-title" style="margin-bottom:6px">Signature</div>
    <div style="font-family:var(--font-mono);font-size:11px;color:var(--text-muted);word-break:break-all;padding:8px;background:var(--surface-2);border-radius:var(--radius)">${esc(d.signature)}</div>
  </div>`;

  el.innerHTML = html;
  appear(el);
}

function buildSection(title, obj) {
  if (!obj) return '';
  let rows = '';
  Object.entries(obj).forEach(([k, v]) => {
    rows += `<tr><td style="color:var(--accent);white-space:nowrap;width:1%">${esc(k)}</td><td>${esc(String(v))}</td></tr>`;
  });
  return `<div style="margin-bottom:10px">
    <div class="card-title" style="margin-bottom:6px">${title}</div>
    <table><thead><tr><th>Field</th><th>Value</th></tr></thead><tbody>${rows}</tbody></table>
  </div>`;
}

function buildPayloadSection(title, obj) {
  if (!obj) return '';
  let rows = '';
  Object.entries(obj).forEach(([k, v]) => {
    let display = esc(String(v));
    if ((k === 'exp' || k === 'iat' || k === 'nbf') && typeof v === 'number') {
      const d = new Date(v * 1000);
      display = `${esc(String(v))} <span style="color:var(--text-muted);font-size:11px">(${d.toLocaleString()})</span>`;
    }
    rows += `<tr><td style="color:var(--accent);white-space:nowrap;width:1%">${esc(k)}</td><td>${display}</td></tr>`;
  });
  return `<div style="margin-bottom:10px">
    <div class="card-title" style="margin-bottom:6px">${title}</div>
    <table><thead><tr><th>Field</th><th>Value</th></tr></thead><tbody>${rows}</tbody></table>
  </div>`;
}

/* ─── Session Secret 历史（localStorage）────────────────────── */
const SECRET_HISTORY_KEY = 'dtb_session_secrets';
const MAX_SECRET_HISTORY = 20;

function loadSecretHistory() {
  try { return JSON.parse(localStorage.getItem(SECRET_HISTORY_KEY) || '[]'); }
  catch { return []; }
}

function pushSecretHistory(secret) {
  if (!secret) return;
  const list = loadSecretHistory().filter(s => s !== secret);
  list.unshift(secret); // 最近使用的排最前
  localStorage.setItem(SECRET_HISTORY_KEY, JSON.stringify(list.slice(0, MAX_SECRET_HISTORY)));
  renderSecretHistory();
}

function renderSecretHistory() {
  const list   = loadSecretHistory();
  const row    = document.getElementById('secretHistoryRow');
  const select = document.getElementById('secretHistorySelect');
  if (!list.length) { row.style.display = 'none'; return; }

  row.style.display = 'flex';
  select.innerHTML = '<option value="">— 选择已保存的 Secret —</option>' +
    list.map(s => {
      // 展示时脱敏：显示前4位 + *** + 后4位
      const display = s.length > 10
        ? s.slice(0, 4) + '***' + s.slice(-4)
        : '***';
      return `<option value="${esc(s)}">${esc(display)}</option>`;
    }).join('');
}

function applySecretHistory(val) {
  if (!val) return;
  document.getElementById('sessionSecret').value = val;
}

function deleteSecretHistory() {
  const val = document.getElementById('secretHistorySelect').value;
  if (!val) return;
  const list = loadSecretHistory().filter(s => s !== val);
  localStorage.setItem(SECRET_HISTORY_KEY, JSON.stringify(list));
  document.getElementById('sessionSecret').value = '';
  renderSecretHistory();
}

/* ─── Session ────────────────────────────────────────────────── */
async function parseSession() {
  const cookie = document.getElementById('sessionInput').value.trim();
  const secret = document.getElementById('sessionSecret').value;
  const el     = document.getElementById('sessionResult');
  if (!cookie) { showError(el, '请先粘贴 Cookie 字符串'); return; }

  const res = await post('/api/parse-session', { cookie, secret });
  if (!ok(res)) { showError(el, err(res)); return; }

  // 解析成功后保存 secret（有值才存）
  if (secret) {
    pushSecretHistory(secret);
    // 同步下拉框选中当前使用的 secret
    document.getElementById('secretHistorySelect').value = secret;
  }

  el.innerHTML = renderObject(res.data, 0);
  appear(el);
}

function renderObject(obj, depth) {
  if (obj === null || obj === undefined) return '<span style="color:var(--text-subtle)">null</span>';
  if (typeof obj !== 'object') return `<span class="mono">${esc(String(obj))}</span>`;
  if (Array.isArray(obj)) {
    if (!obj.length) return '<span style="color:var(--text-subtle)">[]</span>';
    return '<ol style="margin:2px 0;padding-left:18px">' +
      obj.map(v => `<li style="padding:2px 0">${renderObject(v, depth + 1)}</li>`).join('') +
      '</ol>';
  }
  const entries = Object.entries(obj);
  if (!entries.length) return '<span style="color:var(--text-subtle)">{}</span>';
  let html = '<table><thead><tr><th>Field</th><th>Value</th></tr></thead><tbody>';
  entries.forEach(([k, v]) => {
    const isNested = v !== null && typeof v === 'object';
    html += `<tr>
      <td style="color:var(--accent);white-space:nowrap;width:1%;vertical-align:top">${esc(k)}</td>
      <td style="vertical-align:top">${isNested ? renderObject(v, depth + 1) : `<span class="mono">${esc(String(v))}</span>`}</td>
    </tr>`;
  });
  html += '</tbody></table>';
  return html;
}

/* ─── JSON ───────────────────────────────────────────────────── */
async function formatJSON(mode) {
  const val = document.getElementById('jsonInput').value.trim();
  const el  = document.getElementById('jsonResult');
  if (!val) return;

  const res = await post('/api/format-json', { json: val, mode });
  el.value = ok(res) ? res.data : err(res);
}

/* ─── Codec ──────────────────────────────────────────────────── */
async function codec(mode) {
  const val = document.getElementById('codecInput').value;
  const api = mode.startsWith('b64') ? 'base64' : 'urlcodec';
  const res = await post(`/api/${api}`, { text: val, mode });
  document.getElementById('codecResult').value = ok(res) ? res.data : err(res);
}

/* ─── DB 历史连接 ─────────────────────────────────────────────── */
let _savedConns = [];  // 缓存，避免重复请求

// 拉取历史连接并渲染下拉框
async function loadSavedConns() {
  const res = await fetch('/api/db/conns').then(r => r.json());
  if (!ok(res)) return;
  _savedConns = res.data || [];

  const row    = document.getElementById('dbSavedRow');
  const select = document.getElementById('dbSavedConns');

  if (!_savedConns.length) {
    row.style.display = 'none';
    return;
  }

  row.style.display = 'flex';
  select.innerHTML = '<option value="">— 选择已保存的连接 —</option>' +
    _savedConns.map(c =>
      `<option value="${esc(c.id)}">${esc(c.name)}</option>`
    ).join('');
}

// 选中一条历史连接 → 回填表单
function applyConn(id) {
  if (!id) return;
  const conn = _savedConns.find(c => c.id === id);
  if (!conn) return;
  document.getElementById('dbType').value = conn.type;
  document.getElementById('dbDsn').value  = conn.dsn;
  // 同步 select 的 value（onchange 可能已设好，这里确保一致）
  document.getElementById('dbSavedConns').value = id;
}

// 删除当前选中的历史连接
async function deleteConn() {
  const id = document.getElementById('dbSavedConns').value;
  if (!id) return;
  const conn = _savedConns.find(c => c.id === id);
  if (!confirm(`确认删除连接「${conn ? conn.name : id}」？`)) return;

  const res = await post('/api/db/conns/delete', { id });
  if (!ok(res)) { alert('删除失败：' + err(res)); return; }
  await loadSavedConns();
  // 清空已删除的回填
  document.getElementById('dbSavedConns').value = '';
}

/* ─── DB 历史表 & 历史 SQL（localStorage）────────────────────── */
const RECENT_TABLES_KEY = 'dtb_recent_tables'; // { connKey: [tableName,...] }
const SQL_HISTORY_KEY   = 'dtb_sql_history';   // [{ sql, ts }]
const MAX_RECENT_TABLES = 20;
const MAX_SQL_HISTORY   = 50;

function connKey(type, dsn) { return `${type}::${dsn}`; }

// ── 历史表 ──────────────────────────────────────────────────────
function loadRecentTables(type, dsn) {
  try {
    const map = JSON.parse(localStorage.getItem(RECENT_TABLES_KEY) || '{}');
    return map[connKey(type, dsn)] || [];
  } catch { return []; }
}

function pushRecentTable(type, dsn, table) {
  try {
    const map = JSON.parse(localStorage.getItem(RECENT_TABLES_KEY) || '{}');
    const key  = connKey(type, dsn);
    const list = (map[key] || []).filter(t => t !== table);
    list.unshift(table);
    map[key] = list.slice(0, MAX_RECENT_TABLES);
    localStorage.setItem(RECENT_TABLES_KEY, JSON.stringify(map));
  } catch {}
}

function clearRecentTablesStorage(type, dsn) {
  try {
    const map = JSON.parse(localStorage.getItem(RECENT_TABLES_KEY) || '{}');
    delete map[connKey(type, dsn)];
    localStorage.setItem(RECENT_TABLES_KEY, JSON.stringify(map));
  } catch {}
}

function renderRecentTableList(type, dsn) {
  const tables = loadRecentTables(type, dsn);
  const el = document.getElementById('dbRecentTableList');
  if (!tables.length) {
    el.innerHTML = `<div class="empty-state">
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
      <p>点击过的表会记录在此</p></div>`;
    return;
  }
  el.innerHTML = tables.map(t => `
    <div class="db-table-item${t === _dbActiveTable ? ' active' : ''}" onclick="selectTable('${esc(t)}')" title="${esc(t)}">
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M3 15h18M9 3v18"/>
      </svg>
      ${esc(t)}
    </div>`).join('');
}

function clearRecentTables() {
  const { type, dsn } = dbConn();
  clearRecentTablesStorage(type, dsn);
  renderRecentTableList(type, dsn);
}

// ── 历史 SQL ────────────────────────────────────────────────────
function loadSqlHistory() {
  try { return JSON.parse(localStorage.getItem(SQL_HISTORY_KEY) || '[]'); }
  catch { return []; }
}

function pushSqlHistory(sql) {
  try {
    const list = loadSqlHistory().filter(h => h.sql !== sql);
    list.unshift({ sql, ts: Date.now() });
    localStorage.setItem(SQL_HISTORY_KEY, JSON.stringify(list.slice(0, MAX_SQL_HISTORY)));
    renderSqlHistory();
    // 有历史记录时显示按钮
    document.getElementById('sqlHistoryWrap').style.display = 'block';
  } catch {}
}

function deleteSqlHistory(sql, e) {
  e.stopPropagation();
  try {
    const list = loadSqlHistory().filter(h => h.sql !== sql);
    localStorage.setItem(SQL_HISTORY_KEY, JSON.stringify(list));
    renderSqlHistory();
    if (!list.length) {
      document.getElementById('sqlHistoryWrap').style.display = 'none';
      closeSqlHistory();
    }
  } catch {}
}

function clearSqlHistory() {
  localStorage.removeItem(SQL_HISTORY_KEY);
  renderSqlHistory();
  document.getElementById('sqlHistoryWrap').style.display = 'none';
  closeSqlHistory();
}

function applySqlHistory(sql) {
  document.getElementById('dbQuery').value = sql;
  closeSqlHistory();
  switchDbTab('query');
}

function toggleSqlHistory() {
  const dd = document.getElementById('sqlHistoryDropdown');
  dd.classList.toggle('open');
  if (dd.classList.contains('open')) renderSqlHistory();
}

function closeSqlHistory() {
  document.getElementById('sqlHistoryDropdown').classList.remove('open');
}

function renderSqlHistory() {
  const list = loadSqlHistory();
  const dd   = document.getElementById('sqlHistoryDropdown');
  if (!list.length) {
    dd.innerHTML = '<div class="sql-history-empty">暂无历史记录</div>';
    return;
  }
  dd.innerHTML = list.map(h => `
    <div class="sql-history-item" onclick="applySqlHistory(${JSON.stringify(h.sql)})">
      <span class="sql-history-item-sql" title="${esc(h.sql)}">${esc(h.sql)}</span>
      <span class="sql-history-item-del" onclick="deleteSqlHistory(${JSON.stringify(h.sql)}, event)" title="删除">
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6L6 18M6 6l12 12"/></svg>
      </span>
    </div>`).join('') +
    `<div class="sql-history-footer">
      <button class="btn btn-ghost" onclick="clearSqlHistory()" style="font-size:11px;height:26px;padding:0 8px;color:var(--danger)">
        清空全部
      </button>
    </div>`;
}

// 点击外部关闭历史下拉
document.addEventListener('click', e => {
  const wrap = document.getElementById('sqlHistoryWrap');
  if (wrap && !wrap.contains(e.target)) closeSqlHistory();
});

// 初始化：如果已有历史则显示按钮
(function initSqlHistory() {
  if (loadSqlHistory().length) {
    const wrap = document.getElementById('sqlHistoryWrap');
    if (wrap) wrap.style.display = 'block';
  }
})();

/* ─── DB ─────────────────────────────────────────────────────── */
let _dbAllTables = [];   // 全量表名缓存，供搜索用
let _dbActiveTable = ''; // 当前选中的表名

function dbConn() {
  return {
    type: document.getElementById('dbType').value,
    dsn:  document.getElementById('dbDsn').value.trim(),
  };
}

// 按数据库类型返回正确的标识符引号
// MySQL: 反引号  PostgreSQL/SQLite: 双引号
function quoteIdent(name) {
  const type = document.getElementById('dbType').value;
  if (type === 'mysql') return '`' + name + '`';
  return '"' + name + '"';
}

// 连接并加载表列表
// 设置连接按钮为"已连接"绿色状态，或恢复为默认状态
function setConnectBtnState(btnId, connected, label) {
  const btn = document.getElementById(btnId);
  if (!btn) return;
  const iconConnected = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>`;
  const iconDefault   = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M5 12h14M12 5l7 7-7 7"/></svg>`;
  if (connected) {
    btn.classList.add('btn-connected');
    btn.innerHTML = iconConnected + ' ' + (label || '已连接');
  } else {
    btn.classList.remove('btn-connected');
    btn.innerHTML = iconDefault + ' 连接';
  }
}

async function loadTables() {
  const { type, dsn } = dbConn();
  const listEl  = document.getElementById('dbTableList');
  const countEl = document.getElementById('dbTableCount');

  // 连接前先重置按钮状态（避免旧连接残留绿色）
  setConnectBtnState('dbConnectBtn', false);
  listEl.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>连接中...</p></div>`;
  countEl.innerHTML = '';

  const res = await post('/api/db/tables', { type, dsn });
  if (!ok(res)) {
    listEl.innerHTML = `<div class="empty-state" style="color:var(--danger)">
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M15 9l-6 6M9 9l6 6"/></svg>
      <p>${esc(err(res))}</p>
    </div>`;
    return;
  }

  // 连接成功 → 按钮变绿
  setConnectBtnState('dbConnectBtn', true);

  _dbAllTables = res.data || [];
  countEl.innerHTML = `<span class="badge badge-neutral">${_dbAllTables.length} 张表</span>`;
  renderTableList(_dbAllTables);
  document.getElementById('dbTableSearch').value = '';
  renderRecentTableList(type, dsn);

  await loadSavedConns();
  const hint = document.getElementById('dbConnHint');
  hint.style.display = 'flex';
  clearTimeout(hint._timer);
  hint._timer = setTimeout(() => { hint.style.display = 'none'; }, 2000);
}

// 渲染表列表
function renderTableList(tables) {
  const listEl = document.getElementById('dbTableList');
  if (!tables.length) {
    listEl.innerHTML = '<div class="empty-state"><p>暂无数据表</p></div>';
    return;
  }
  listEl.innerHTML = tables.map(t => `
    <div class="db-table-item${t === _dbActiveTable ? ' active' : ''}" onclick="selectTable('${esc(t)}')" title="${esc(t)}">
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M3 15h18M9 3v18"/>
      </svg>
      ${esc(t)}
    </div>`).join('');
}

// 搜索过滤
function filterTables() {
  const q = document.getElementById('dbTableSearch').value.toLowerCase();
  const items = document.querySelectorAll('.db-table-item');
  items.forEach(el => {
    el.classList.toggle('hidden', !el.title.toLowerCase().includes(q));
  });
}

// 选中表 → 自动切换到"表结构"Tab 并加载
async function selectTable(name) {
  _dbActiveTable = name;

  // 记录到历史表
  const { type, dsn } = dbConn();
  pushRecentTable(type, dsn, name);
  renderRecentTableList(type, dsn);

  // 高亮选中项（全部表 + 历史表两列都更新）
  document.querySelectorAll('.db-table-item').forEach(el => {
    el.classList.toggle('active', el.title === name);
  });

  // 显示当前选中表名 badge
  const badge = document.getElementById('dbActiveTable');
  badge.textContent = name;
  badge.classList.add('visible');

  // 切换到结构 Tab
  switchDbTab('structure');

  // 加载表结构
  await describeTable(name);

  // 自动填充 SQL
  document.getElementById('dbQuery').value = `SELECT * FROM ${quoteIdent(name)} LIMIT 20`;
}

// 查看表结构
async function describeTable(name) {
  const { type, dsn } = dbConn();
  const el = document.getElementById('dbStructureResult');
  el.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>加载表结构...</p></div>`;

  const res = await post('/api/db/describe', { type, dsn, table: name });
  if (!ok(res)) { showError(el, err(res)); return; }

  const cols = res.data || [];
  if (!cols.length) {
    el.innerHTML = '<div class="empty-state"><p>未获取到列信息</p></div>';
    return;
  }

  let html = `<table>
    <thead><tr>
      <th>列名</th><th>类型</th><th>可空</th><th>键</th><th>默认值</th><th>备注</th>
    </tr></thead><tbody>`;

  cols.forEach(c => {
    const keyBadge  = c.key     ? `<span class="col-key-pri">${esc(c.key)}</span>`  : '';
    const nullClass = c.nullable === 'NO' ? 'col-nullable-no' : 'col-nullable-yes';
    html += `<tr>
      <td style="font-weight:500;color:var(--text)">${esc(c.name)}</td>
      <td><span class="col-type">${esc(c.type)}</span></td>
      <td><span class="${nullClass}">${c.nullable === 'NO' ? 'NOT NULL' : 'NULL'}</span></td>
      <td>${keyBadge}</td>
      <td class="mono" style="color:var(--text-secondary)">${esc(c.default || '')}</td>
      <td style="color:var(--text-tertiary)">${esc(c.extra || '')}</td>
    </tr>`;
  });
  html += '</tbody></table>';
  el.innerHTML = html;
  appear(el);
}

// 切换 DB 内部 Tab
function switchDbTab(tab) {
  document.querySelectorAll('.db-tab').forEach(el => {
    el.classList.toggle('active', el.dataset.dbtab === tab);
  });
  document.querySelectorAll('.db-tab-panel').forEach(el => {
    el.classList.toggle('active', el.id === (tab === 'structure' ? 'dbTabStructure' : 'dbTabQuery'));
  });
}

// 执行 SQL 查询
async function queryDB() {
  const { type, dsn } = dbConn();
  const query = document.getElementById('dbQuery').value.trim();
  const el    = document.getElementById('dbResult');
  const badge = document.getElementById('dbRowCount');

  badge.innerHTML = '';
  if (!query) { showError(el, '请输入 SQL 查询语句'); return; }

  // 确保在查询 Tab
  switchDbTab('query');

  el.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>执行中...</p></div>`;

  const res = await post('/api/query-db', { type, dsn, query });
  if (!ok(res)) { showError(el, err(res)); return; }

  // 执行成功后记录 SQL 历史
  pushSqlHistory(query);

  const rows = res.data || [];
  if (!rows.length) {
    el.innerHTML = '<div class="empty-state"><p style="color:var(--text-secondary)">查询成功，无返回数据</p></div>';
    badge.innerHTML = '<span class="badge badge-success">0 行</span>';
    return;
  }

  badge.innerHTML = `<span class="badge badge-success">${rows.length} 行</span>`;
  const cols = Object.keys(rows[0]);
  let html = '<table><thead><tr>' + cols.map(c => `<th>${esc(c)}</th>`).join('') + '</tr></thead><tbody>';
  rows.forEach(row => {
    html += '<tr>' + cols.map(c => `<td>${esc(String(row[c] ?? ''))}</td>`).join('') + '</tr>';
  });
  html += '</tbody></table>';
  el.innerHTML = html;
  appear(el);
}

/* ─── Redis ──────────────────────────────────────────────────── */
const REDIS_CMD_HISTORY_KEY = 'dtb_redis_cmd_history';
const MAX_REDIS_CMD_HISTORY = 50;
let _redisSavedConns  = [];
let _redisActiveKey   = '';

// ── 连接辅助 ────────────────────────────────────────────────────
function redisConn() {
  return {
    addr:     document.getElementById('redisAddr').value.trim(),
    password: document.getElementById('redisPassword').value,
    db:       parseInt(document.getElementById('redisDB').value || '0', 10),
  };
}

// ── 历史连接 ────────────────────────────────────────────────────
async function loadRedisConns() {
  const res = await fetch('/api/redis/conns').then(r => r.json());
  if (!ok(res)) return;
  _redisSavedConns = res.data || [];

  const row    = document.getElementById('redisSavedRow');
  const select = document.getElementById('redisSavedConns');
  if (!_redisSavedConns.length) { row.style.display = 'none'; return; }

  row.style.display = 'flex';
  select.innerHTML = '<option value="">— 选择已保存的连接 —</option>' +
    _redisSavedConns.map(c => `<option value="${esc(c.id)}">${esc(c.name)}</option>`).join('');
}

function applyRedisConn(id) {
  if (!id) return;
  const conn = _redisSavedConns.find(c => c.id === id);
  if (!conn) return;
  // DSN 格式: addr?db=N[&password=xxx]
  const qIdx    = conn.dsn.indexOf('?');
  const addr    = qIdx >= 0 ? conn.dsn.slice(0, qIdx) : conn.dsn;
  const params  = new URLSearchParams(qIdx >= 0 ? conn.dsn.slice(qIdx + 1) : '');
  document.getElementById('redisAddr').value     = addr;
  document.getElementById('redisDB').value       = params.get('db') || '0';
  document.getElementById('redisPassword').value = params.get('password') || '';
}

async function deleteRedisConn() {
  const id   = document.getElementById('redisSavedConns').value;
  if (!id) return;
  const conn = _redisSavedConns.find(c => c.id === id);
  if (!confirm(`确认删除连接「${conn ? conn.name : id}」？`)) return;
  const res = await post('/api/db/conns/delete', { id });
  if (!ok(res)) { alert('删除失败：' + err(res)); return; }
  await loadRedisConns();
}

// ── Ping / 连接 ─────────────────────────────────────────────────
async function redisPing() {
  const { addr, password, db } = redisConn();
  const hint = document.getElementById('redisConnHint');

  // 连接前先重置
  setConnectBtnState('redisPingBtn', false);

  const res = await post('/api/redis/ping', { addr, password, db });
  if (!ok(res)) {
    hint.style.display = 'none';
    alert('连接失败：' + err(res));
    return;
  }

  // 连接成功 → 按钮变绿，显示版本
  const version = (res.data && res.data.version) ? res.data.version : '';
  setConnectBtnState('redisPingBtn', true, `已连接${version ? ' v' + version : ''}`);

  await loadRedisConns();
  hint.style.display = 'none'; // 版本已在按钮上显示，不再需要 hint
}

// ── Key 搜索 ─────────────────────────────────────────────────────
async function redisSearchKeys() {
  const { addr, password, db } = redisConn();
  const pattern = document.getElementById('redisPattern').value.trim() || '*';
  const listEl  = document.getElementById('redisKeyList');
  const countEl = document.getElementById('redisKeyCount');

  listEl.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>搜索中...</p></div>`;
  countEl.innerHTML = '';

  const res = await post('/api/redis/keys', { addr, password, db, pattern, count: 200 });
  if (!ok(res)) {
    listEl.innerHTML = `<div class="empty-state" style="color:var(--danger)"><p>${esc(err(res))}</p></div>`;
    return;
  }

  const keys = res.data || [];
  countEl.innerHTML = `<span class="badge badge-neutral">${keys.length} 个</span>`;

  if (!keys.length) {
    listEl.innerHTML = '<div class="empty-state"><p>无匹配 Key</p></div>';
    return;
  }

  const typeIcon = {
    string: '🔤', list: '📋', hash: '🗂', set: '🔵', zset: '📊', none: '❓'
  };
  listEl.innerHTML = keys.map(k => `
    <div class="db-table-item${k.key === _redisActiveKey ? ' active' : ''}"
         onclick="redisGetKey('${esc(k.key)}')" title="${esc(k.key)}">
      <span style="font-size:11px;flex-shrink:0">${typeIcon[k.type] || '•'}</span>
      <span style="flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(k.key)}</span>
      ${k.ttl >= 0 ? `<span class="badge badge-amber" style="font-size:9px;flex-shrink:0">${k.ttl}s</span>` : ''}
    </div>`).join('');
}

// ── 获取 Key 详情 ───────────────────────────────────────────────
async function redisGetKey(key) {
  _redisActiveKey = key;

  // 高亮
  document.querySelectorAll('#redisKeyList .db-table-item').forEach(el => {
    el.classList.toggle('active', el.title === key);
  });

  // 显示 badge
  const badge = document.getElementById('redisActiveKey');
  badge.textContent = key;
  badge.classList.add('visible');

  switchRedisTab('value');

  const { addr, password, db } = redisConn();
  const el  = document.getElementById('redisValueResult');
  el.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>加载中...</p></div>`;

  const res = await post('/api/redis/get', { addr, password, db, key });
  if (!ok(res)) { showError(el, err(res)); return; }

  const d = res.data;
  const ttlStr = d.ttl === -1 ? '永不过期' : d.ttl === -2 ? 'Key 不存在' : `${d.ttl}s`;

  let valueHtml = '';
  if (typeof d.value === 'string') {
    // string 类型：尝试 JSON 格式化
    let pretty = d.value;
    try {
      pretty = JSON.stringify(JSON.parse(d.value), null, 2);
    } catch {}
    valueHtml = `<pre style="background:var(--surface-alt);padding:10px;border-radius:var(--radius);font-size:12px;max-height:400px;overflow:auto">${esc(pretty)}</pre>`;
  } else if (Array.isArray(d.value)) {
    // list / set
    valueHtml = `<table><thead><tr><th>#</th><th>Value</th></tr></thead><tbody>` +
      d.value.map((v, i) => `<tr><td style="color:var(--text-tertiary)">${i}</td><td>${esc(String(v))}</td></tr>`).join('') +
      `</tbody></table>`;
  } else if (d.value && typeof d.value === 'object') {
    // hash / zset
    const entries = Object.entries(d.value);
    if (Array.isArray(d.value)) {
      valueHtml = `<table><thead><tr><th>Member</th><th>Score</th></tr></thead><tbody>` +
        d.value.map(z => `<tr><td>${esc(z.member)}</td><td>${z.score}</td></tr>`).join('') +
        `</tbody></table>`;
    } else {
      valueHtml = `<table><thead><tr><th>Field</th><th>Value</th></tr></thead><tbody>` +
        entries.map(([k, v]) => `<tr><td style="color:var(--accent);white-space:nowrap">${esc(k)}</td><td>${esc(String(v))}</td></tr>`).join('') +
        `</tbody></table>`;
    }
  }

  el.innerHTML = `
    <div style="display:flex;gap:8px;flex-wrap:wrap;margin-bottom:10px">
      <span class="badge badge-cyan">${esc(d.type)}</span>
      <span class="badge badge-neutral">TTL: ${ttlStr}</span>
      <span class="mono" style="font-size:11px;color:var(--text-secondary);align-self:center">${esc(key)}</span>
    </div>
    ${valueHtml}`;
  appear(el);
}

// ── 命令执行 ────────────────────────────────────────────────────
async function redisExec() {
  const { addr, password, db } = redisConn();
  const command = document.getElementById('redisCommand').value.trim();
  const el = document.getElementById('redisExecResult');
  if (!command) { showError(el, '请输入命令'); return; }

  el.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>执行中...</p></div>`;

  const res = await post('/api/redis/exec', { addr, password, db, command });
  if (!ok(res)) { showError(el, err(res)); return; }

  // 执行成功后记录历史
  pushRedisCmdHistory(command);

  const val = res.data;
  let display = '';
  if (val === null || val === undefined) {
    display = '<span style="color:var(--text-tertiary)">(nil)</span>';
  } else if (Array.isArray(val)) {
    display = `<table><thead><tr><th>#</th><th>Value</th></tr></thead><tbody>` +
      val.map((v, i) => `<tr><td style="color:var(--text-tertiary)">${i + 1}</td><td class="mono">${esc(String(v ?? ''))}</td></tr>`).join('') +
      `</tbody></table>`;
  } else {
    display = `<pre style="background:var(--surface-alt);padding:10px;border-radius:var(--radius);font-size:12px">${esc(String(val))}</pre>`;
  }
  el.innerHTML = display;
  appear(el);
}

// ── Tab 切换 ─────────────────────────────────────────────────────
function switchRedisTab(tab) {
  document.querySelectorAll('.db-tab[data-redistab]').forEach(el => {
    el.classList.toggle('active', el.dataset.redistab === tab);
  });
  document.getElementById('redisTabValue').classList.toggle('active', tab === 'value');
  document.getElementById('redisTabExec').classList.toggle('active',  tab === 'exec');
}

// ── Redis 命令历史（同 SQL 历史，独立 key）──────────────────────
function loadRedisCmdHistory() {
  try { return JSON.parse(localStorage.getItem(REDIS_CMD_HISTORY_KEY) || '[]'); }
  catch { return []; }
}

function pushRedisCmdHistory(cmd) {
  try {
    const list = loadRedisCmdHistory().filter(h => h.cmd !== cmd);
    list.unshift({ cmd, ts: Date.now() });
    localStorage.setItem(REDIS_CMD_HISTORY_KEY, JSON.stringify(list.slice(0, MAX_REDIS_CMD_HISTORY)));
    renderRedisCmdHistory();
    document.getElementById('redisCmdHistoryWrap').style.display = 'block';
  } catch {}
}

function deleteRedisCmdHistory(cmd, e) {
  e.stopPropagation();
  try {
    const list = loadRedisCmdHistory().filter(h => h.cmd !== cmd);
    localStorage.setItem(REDIS_CMD_HISTORY_KEY, JSON.stringify(list));
    renderRedisCmdHistory();
    if (!list.length) { document.getElementById('redisCmdHistoryWrap').style.display = 'none'; closeRedisCmdHistory(); }
  } catch {}
}

function clearRedisCmdHistory() {
  localStorage.removeItem(REDIS_CMD_HISTORY_KEY);
  document.getElementById('redisCmdHistoryWrap').style.display = 'none';
  closeRedisCmdHistory();
}

function applyRedisCmdHistory(cmd) {
  document.getElementById('redisCommand').value = cmd;
  closeRedisCmdHistory();
  switchRedisTab('exec');
}

function toggleRedisCmdHistory() {
  const dd = document.getElementById('redisCmdHistoryDropdown');
  dd.classList.toggle('open');
  if (dd.classList.contains('open')) renderRedisCmdHistory();
}

function closeRedisCmdHistory() {
  document.getElementById('redisCmdHistoryDropdown').classList.remove('open');
}

function renderRedisCmdHistory() {
  const list = loadRedisCmdHistory();
  const dd   = document.getElementById('redisCmdHistoryDropdown');
  if (!list.length) { dd.innerHTML = '<div class="sql-history-empty">暂无历史记录</div>'; return; }
  dd.innerHTML = list.map(h => `
    <div class="sql-history-item" onclick="applyRedisCmdHistory(${JSON.stringify(h.cmd)})">
      <span class="sql-history-item-sql" title="${esc(h.cmd)}">${esc(h.cmd)}</span>
      <span class="sql-history-item-del" onclick="deleteRedisCmdHistory(${JSON.stringify(h.cmd)}, event)">
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6L6 18M6 6l12 12"/></svg>
      </span>
    </div>`).join('') +
    `<div class="sql-history-footer">
      <button class="btn btn-ghost" onclick="clearRedisCmdHistory()" style="font-size:11px;height:26px;padding:0 8px;color:var(--danger)">清空全部</button>
    </div>`;
}

document.addEventListener('click', e => {
  const wrap = document.getElementById('redisCmdHistoryWrap');
  if (wrap && !wrap.contains(e.target)) closeRedisCmdHistory();
});

// 初始化 Redis 命令历史按钮可见性
(function initRedisCmdHistory() {
  if (loadRedisCmdHistory().length) {
    const wrap = document.getElementById('redisCmdHistoryWrap');
    if (wrap) wrap.style.display = 'block';
  }
})();

/* ─── Time ───────────────────────────────────────────────────── */
function refreshTime() {
  post('/api/timestamp', {}).then(res => {
    if (!ok(res)) return;
    const d = res.data;
    document.getElementById('tsSec').value = d.sec;
    document.getElementById('tsMs').value  = d.ms;
    document.getElementById('tsRFC').value = d.rfc;
    document.getElementById('nowBadge').textContent = d.rfc;
  });
}

function convertTs() {
  const v = document.getElementById('tsInput').value.trim();
  const el = document.getElementById('tsResult');
  if (!v) return;
  post('/api/timestamp', { ts: v }).then(res => {
    if (!ok(res)) {
      el.innerHTML = `<span style="color:var(--danger)">${esc(err(res))}</span>`;
      return;
    }
    const d = res.data;
    el.innerHTML = `<span style="color:var(--success)">${esc(d.rfc)}</span>
      <span style="color:var(--text-muted);margin-left:8px">秒: ${d.sec} · 毫秒: ${d.ms}</span>`;
  });
}

/* ─── UUID ───────────────────────────────────────────────────── */
function genUUID() {
  post('/api/uuid', { n: 5 }).then(res => {
    document.getElementById('uuidResult').value = ok(res) ? (res.data || []).join('\n') : err(res);
  });
}

/* ─── Copy ───────────────────────────────────────────────────── */
function copyText(id, btnId) {
  const el = document.getElementById(id);
  if (!el.value) return;
  navigator.clipboard.writeText(el.value).catch(() => {
    el.select();
    document.execCommand('copy');
  });
  if (btnId) {
    const btn = document.getElementById(btnId);
    const orig = btn.innerHTML;
    btn.innerHTML = `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M20 6L9 17l-5-5"/></svg> 已复制`;
    btn.classList.add('flash-success');
    setTimeout(() => { btn.innerHTML = orig; btn.classList.remove('flash-success'); }, 1800);
  }
}

/* ─── Escape ─────────────────────────────────────────────────── */
function esc(s) {
  if (s == null) return '';
  return String(s).replace(/[&<>"']/g, function(c) {
    return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
  });
}

/* ─── Init ───────────────────────────────────────────────────── */
refreshTime();
setInterval(refreshTime, 30000);

// 恢复上次激活的 Tab 及其输入内容
(function initTabRestore() {
  const savedTab = localStorage.getItem(ACTIVE_TAB_KEY) || 'db';
  const targetItem = [...navItems].find(n => n.dataset.target === savedTab);

  if (targetItem && savedTab !== 'db') {
    // 重置所有 active 状态
    navItems.forEach(n => n.classList.remove('active'));
    panels.forEach(p => p.classList.remove('active'));
    // 激活目标 Tab
    targetItem.classList.add('active');
    document.getElementById(savedTab).classList.add('active');
    pageTitle.textContent = targetItem.textContent.trim();
    pageDesc.textContent  = targetItem.dataset.desc || '';
    _activeTab = savedTab;
  }

  // 恢复当前激活 Tab 的输入内容
  restoreTabState(_activeTab);
})();

// 初始化历史连接（根据当前激活的 Tab 决定加载哪个）
if (_activeTab === 'db')    loadSavedConns();
if (_activeTab === 'redis') loadRedisConns();
// 非 db/redis Tab 也预加载，方便切换时有数据
if (_activeTab !== 'db')    loadSavedConns();
if (_activeTab !== 'redis') loadRedisConns();

// 初始化 Session Secret 历史
(function initSecretHistory() {
  renderSecretHistory();
  const list = loadSecretHistory();
  if (list.length > 0 && !document.getElementById('sessionSecret').value) {
    document.getElementById('sessionSecret').value = list[0];
  }
})();
