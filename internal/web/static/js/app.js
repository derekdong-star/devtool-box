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
  html += buildSection('Header', d.header);
  html += buildPayloadSection('Payload', d.payload);
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
  list.unshift(secret);
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
      const display = s.length > 10 ? s.slice(0, 4) + '***' + s.slice(-4) : '***';
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

  if (secret) {
    pushSecretHistory(secret);
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
let _savedConns = [];

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
    _savedConns.map(c => `<option value="${esc(c.id)}">${esc(c.name)}</option>`).join('');
}

function applyConn(id) {
  if (!id) return;
  const conn = _savedConns.find(c => c.id === id);
  if (!conn) return;
  document.getElementById('dbType').value = conn.type;
  document.getElementById('dbDsn').value  = conn.dsn;
  document.getElementById('dbSavedConns').value = id;
}

async function deleteConn() {
  const id = document.getElementById('dbSavedConns').value;
  if (!id) return;
  const conn = _savedConns.find(c => c.id === id);
  if (!confirm(`确认删除连接「${conn ? conn.name : id}」？`)) return;

  const res = await post('/api/db/conns/delete', { id });
  if (!ok(res)) { alert('删除失败：' + err(res)); return; }
  await loadSavedConns();
  document.getElementById('dbSavedConns').value = '';
}

/* ─── DB ─────────────────────────────────────────────────────── */
let _dbAllTables = [];
let _dbActiveTable = '';

function dbConn() {
  return {
    type: document.getElementById('dbType').value,
    dsn:  document.getElementById('dbDsn').value.trim(),
  };
}

function quoteIdent(name) {
  const type = document.getElementById('dbType').value;
  if (type === 'mysql') return '`' + name + '`';
  return '"' + name + '"';
}

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

function filterTables() {
  const q = document.getElementById('dbTableSearch').value.toLowerCase();
  document.querySelectorAll('.db-table-item').forEach(el => {
    el.classList.toggle('hidden', !el.title.toLowerCase().includes(q));
  });
}

async function selectTable(name) {
  _dbActiveTable = name;
  const { type, dsn } = dbConn();
  pushRecentTable(type, dsn, name);
  renderRecentTableList(type, dsn);

  document.querySelectorAll('.db-table-item').forEach(el => {
    el.classList.toggle('active', el.title === name);
  });

  const badge = document.getElementById('dbActiveTable');
  badge.textContent = name;
  badge.classList.add('visible');

  switchDbTab('structure');
  await describeTable(name);
  document.getElementById('dbQuery').value = `SELECT * FROM ${quoteIdent(name)} LIMIT 20`;
}

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
    <thead><tr><th>列名</th><th>类型</th><th>可空</th><th>键</th><th>默认值</th><th>备注</th></tr></thead><tbody>`;
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

function switchDbTab(tab) {
  document.querySelectorAll('.db-tab').forEach(el => {
    el.classList.toggle('active', el.dataset.dbtab === tab);
  });
  document.querySelectorAll('.db-tab-panel').forEach(el => {
    el.classList.toggle('active', el.id === (tab === 'structure' ? 'dbTabStructure' : 'dbTabQuery'));
  });
}

async function queryDB() {
  const { type, dsn } = dbConn();
  const query = document.getElementById('dbQuery').value.trim();
  const el    = document.getElementById('dbResult');
  const badge = document.getElementById('dbRowCount');

  badge.innerHTML = '';
  if (!query) { showError(el, '请输入 SQL 查询语句'); return; }
  switchDbTab('query');

  el.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>执行中...</p></div>`;

  const res = await post('/api/query-db', { type, dsn, query });
  if (!ok(res)) { showError(el, err(res)); return; }

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
let _redisSavedConns  = [];
let _redisActiveKey   = '';

function redisConn() {
  return {
    addr:     document.getElementById('redisAddr').value.trim(),
    password: document.getElementById('redisPassword').value,
    db:       parseInt(document.getElementById('redisDB').value || '0', 10),
  };
}

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

async function redisPing() {
  const { addr, password, db } = redisConn();
  const hint = document.getElementById('redisConnHint');
  setConnectBtnState('redisPingBtn', false);

  const res = await post('/api/redis/ping', { addr, password, db });
  if (!ok(res)) {
    hint.style.display = 'none';
    alert('连接失败：' + err(res));
    return;
  }

  const version = (res.data && res.data.version) ? res.data.version : '';
  setConnectBtnState('redisPingBtn', true, `已连接${version ? ' v' + version : ''}`);

  await loadRedisConns();
  hint.style.display = 'none';
}

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

async function redisGetKey(key) {
  _redisActiveKey = key;
  document.querySelectorAll('#redisKeyList .db-table-item').forEach(el => {
    el.classList.toggle('active', el.title === key);
  });

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
    let pretty = d.value;
    try { pretty = JSON.stringify(JSON.parse(d.value), null, 2); } catch {}
    valueHtml = `<pre style="background:var(--surface-alt);padding:10px;border-radius:var(--radius);font-size:12px;max-height:400px;overflow:auto">${esc(pretty)}</pre>`;
  } else if (Array.isArray(d.value)) {
    valueHtml = `<table><thead><tr><th>#</th><th>Value</th></tr></thead><tbody>` +
      d.value.map((v, i) => `<tr><td style="color:var(--text-tertiary)">${i}</td><td>${esc(String(v))}</td></tr>`).join('') +
      `</tbody></table>`;
  } else if (d.value && typeof d.value === 'object') {
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

async function redisExec() {
  const { addr, password, db } = redisConn();
  const command = document.getElementById('redisCommand').value.trim();
  const el = document.getElementById('redisExecResult');
  if (!command) { showError(el, '请输入命令'); return; }

  el.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>执行中...</p></div>`;

  const res = await post('/api/redis/exec', { addr, password, db, command });
  if (!ok(res)) { showError(el, err(res)); return; }

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

function switchRedisTab(tab) {
  document.querySelectorAll('.db-tab[data-redistab]').forEach(el => {
    el.classList.toggle('active', el.dataset.redistab === tab);
  });
  document.getElementById('redisTabValue').classList.toggle('active', tab === 'value');
  document.getElementById('redisTabExec').classList.toggle('active',  tab === 'exec');
}

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

/* ─── Init ───────────────────────────────────────────────────── */
refreshTime();
setInterval(refreshTime, 30000);

(function initTabRestore() {
  const savedTab = localStorage.getItem(ACTIVE_TAB_KEY) || 'db';
  const targetItem = [...navItems].find(n => n.dataset.target === savedTab);

  if (targetItem && savedTab !== 'db') {
    navItems.forEach(n => n.classList.remove('active'));
    panels.forEach(p => p.classList.remove('active'));
    targetItem.classList.add('active');
    document.getElementById(savedTab).classList.add('active');
    pageTitle.textContent = targetItem.textContent.trim();
    pageDesc.textContent  = targetItem.dataset.desc || '';
    _activeTab = savedTab;
  }

  restoreTabState(_activeTab);
})();

if (_activeTab === 'db')     loadSavedConns();
if (_activeTab === 'redis')  loadRedisConns();
if (_activeTab === 'image')  loadImageConfig();
if (_activeTab !== 'db')     loadSavedConns();
if (_activeTab !== 'redis')  loadRedisConns();

(function initSecretHistory() {
  renderSecretHistory();
  const list = loadSecretHistory();
  if (list.length > 0 && !document.getElementById('sessionSecret').value) {
    document.getElementById('sessionSecret').value = list[0];
  }
})();

/* ─── Responsive Breakpoints ────────────────────────────────── */
// Keep in sync with CSS @media breakpoints in main.css
const BREAKPOINT_MOBILE   = 700;
const BREAKPOINT_TABLET   = 1100;

/* ─── Resizer ────────────────────────────────────────────────── */
// Grid gap (16px × 2) + resizer column (10px) in .image-workspace
const IMAGE_WORKSPACE_CHROME = 42;

const SIDEBAR_WIDTH_KEY        = 'dtb_sidebar_width';
const SIDEBAR_MIN_WIDTH        = 220;
const SIDEBAR_MAX_WIDTH        = 420;

const IMAGE_CONTROLS_WIDTH_KEY = 'dtb_image_controls_width';
const IMAGE_CONTROLS_MIN_WIDTH = 320;
const IMAGE_CONTROLS_MAX_WIDTH = 620;

/**
 * Initialise a drag-to-resize handle.
 *
 * @param {object} opts
 * @param {string}      opts.resizerId     - id of the resizer DOM element
 * @param {string}      opts.storageKey    - localStorage key for persisting width
 * @param {string}      opts.cssVar        - CSS custom property to update (e.g. '--sidebar-width')
 * @param {number}      opts.minWidth
 * @param {number}      opts.maxWidth
 * @param {string}      opts.bodyClass     - class added to <body> while dragging
 * @param {number}      opts.breakpoint    - skip init when innerWidth <= this value
 * @param {Function}   [opts.getClientX]   - maps PointerEvent → desired width; defaults to e.clientX
 * @param {Function}   [opts.getMaxWidth]  - dynamic upper bound called on each move; defaults to opts.maxWidth
 */
function initResizer(opts) {
  const resizer = document.getElementById(opts.resizerId);
  if (!resizer || window.innerWidth <= opts.breakpoint) return;

  const root = document.documentElement;

  const clamp = (width, lo, hi) => Math.min(hi, Math.max(lo, width));

  const savedWidth = parseInt(localStorage.getItem(opts.storageKey) || '', 10);
  if (!Number.isNaN(savedWidth)) {
    root.style.setProperty(opts.cssVar, `${clamp(savedWidth, opts.minWidth, opts.maxWidth)}px`);
  }

  const applyWidth = (rawWidth) => {
    const hi = opts.getMaxWidth ? opts.getMaxWidth() : opts.maxWidth;
    const nextWidth = clamp(rawWidth, opts.minWidth, hi);
    root.style.setProperty(opts.cssVar, `${nextWidth}px`);
    localStorage.setItem(opts.storageKey, String(nextWidth));
  };

  let dragging = false;

  const onPointerMove = (e) => {
    if (!dragging) return;
    applyWidth(opts.getClientX ? opts.getClientX(e) : e.clientX);
  };

  const stopDragging = () => {
    if (!dragging) return;
    dragging = false;
    document.body.classList.remove(opts.bodyClass);
    document.body.style.userSelect = '';
    window.removeEventListener('pointermove', onPointerMove);
    window.removeEventListener('pointerup', stopDragging);
  };

  resizer.addEventListener('pointerdown', (e) => {
    dragging = true;
    document.body.classList.add(opts.bodyClass);
    document.body.style.userSelect = 'none';
    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', stopDragging);
    e.preventDefault();
  });
}

/* ─── Sidebar Resize ────────────────────────────────────────── */
initResizer({
  resizerId:  'sidebarResizer',
  storageKey: SIDEBAR_WIDTH_KEY,
  cssVar:     '--sidebar-width',
  minWidth:   SIDEBAR_MIN_WIDTH,
  maxWidth:   SIDEBAR_MAX_WIDTH,
  bodyClass:  'is-resizing-sidebar',
  breakpoint: BREAKPOINT_MOBILE,
});

/* ─── Image Workspace Resize ────────────────────────────────── */
(function () {
  const workspace = document.querySelector('.image-workspace');
  initResizer({
    resizerId:   'imageWorkspaceResizer',
    storageKey:  IMAGE_CONTROLS_WIDTH_KEY,
    cssVar:      '--image-controls-width',
    minWidth:    IMAGE_CONTROLS_MIN_WIDTH,
    maxWidth:    IMAGE_CONTROLS_MAX_WIDTH,
    bodyClass:   'is-resizing-image-workspace',
    breakpoint:  BREAKPOINT_TABLET,
    getClientX:  (e) => e.clientX - (workspace ? workspace.getBoundingClientRect().left : 0),
    getMaxWidth: () => workspace
      ? Math.min(IMAGE_CONTROLS_MAX_WIDTH, workspace.getBoundingClientRect().width - IMAGE_CONTROLS_MIN_WIDTH - IMAGE_WORKSPACE_CHROME)
      : IMAGE_CONTROLS_MAX_WIDTH,
  });
})();
