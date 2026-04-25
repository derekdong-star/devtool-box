/* ─── DB 历史表 & 历史 SQL（localStorage）────────────────────── */
const RECENT_TABLES_KEY = 'dtb_recent_tables';
const SQL_HISTORY_KEY   = 'dtb_sql_history';
const MAX_RECENT_TABLES = 20;
const MAX_SQL_HISTORY   = 50;

function connKey(type, dsn) { return `${type}::${dsn}`; }

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
    document.getElementById('sqlHistoryWrap').style.display = 'block';
    if (_hdPortalType === 'sql') _renderSqlHistoryContent();
  } catch {}
}

function deleteSqlHistory(sql) {
  try {
    const list = loadSqlHistory().filter(h => h.sql !== sql);
    localStorage.setItem(SQL_HISTORY_KEY, JSON.stringify(list));
    if (!list.length) {
      document.getElementById('sqlHistoryWrap').style.display = 'none';
      _closeHistoryPortal();
    } else {
      _renderSqlHistoryContent();
    }
  } catch {}
}

function clearSqlHistory() {
  localStorage.removeItem(SQL_HISTORY_KEY);
  document.getElementById('sqlHistoryWrap').style.display = 'none';
  _closeHistoryPortal();
}

function applySqlHistory(sql) {
  document.getElementById('dbQuery').value = sql;
  _closeHistoryPortal();
  switchDbTab('query');
}

function toggleSqlHistory(e, btn) {
  e.stopPropagation();
  if (_hdPortalType === 'sql') { _closeHistoryPortal(); return; }
  _closeHistoryPortal();
  _renderSqlHistoryContent(btn);
}

function closeSqlHistory() { if (_hdPortalType === 'sql') _closeHistoryPortal(); }

function _renderSqlHistoryContent(anchorBtn) {
  const list = loadSqlHistory();
  const anchor = anchorBtn || document.getElementById('sqlHistoryWrap')?.querySelector('button');

  let html = '';
  if (!list.length) {
    html = '<div class="sql-history-empty">暂无历史记录</div>';
  } else {
    html = list.map(h => `
      <div class="sql-history-item" data-action="apply-sql" data-sql="${esc(h.sql)}">
        <span class="sql-history-item-sql" title="${esc(h.sql)}">${esc(h.sql)}</span>
        <span class="sql-history-item-save" data-action="template" data-sql="${esc(h.sql)}" data-kind="sql" title="存为模板">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
        </span>
        <span class="sql-history-item-del" data-action="delete-sql" data-sql="${esc(h.sql)}" title="删除">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6L6 18M6 6l12 12"/></svg>
        </span>
      </div>`).join('') +
      `<div class="sql-history-footer">
        <button class="btn btn-ghost" data-action="clear-sql" style="font-size:11px;height:26px;padding:0 8px;color:var(--danger)">清空全部</button>
      </div>`;
  }
  if (anchor) _openHistoryPortal(anchor, 'sql', html);
}

(function initSqlHistory() {
  if (loadSqlHistory().length) {
    const wrap = document.getElementById('sqlHistoryWrap');
    if (wrap) wrap.style.display = 'block';
  }
})();

/* ─── Redis 命令历史 ─────────────────────────────────────────── */
const REDIS_CMD_HISTORY_KEY = 'dtb_redis_cmd_history';
const MAX_REDIS_CMD_HISTORY = 50;

function loadRedisCmdHistory() {
  try { return JSON.parse(localStorage.getItem(REDIS_CMD_HISTORY_KEY) || '[]'); }
  catch { return []; }
}

function pushRedisCmdHistory(cmd) {
  try {
    const list = loadRedisCmdHistory().filter(h => h.cmd !== cmd);
    list.unshift({ cmd, ts: Date.now() });
    localStorage.setItem(REDIS_CMD_HISTORY_KEY, JSON.stringify(list.slice(0, MAX_REDIS_CMD_HISTORY)));
    if (_hdPortalType === 'redis') _renderRedisCmdHistoryContent();
  } catch {}
}

function deleteRedisCmdHistory(cmd) {
  try {
    const list = loadRedisCmdHistory().filter(h => h.cmd !== cmd);
    localStorage.setItem(REDIS_CMD_HISTORY_KEY, JSON.stringify(list));
    if (!list.length) {
      _closeHistoryPortal();
    } else {
      _renderRedisCmdHistoryContent();
    }
  } catch {}
}

function clearRedisCmdHistory() {
  localStorage.removeItem(REDIS_CMD_HISTORY_KEY);
  _closeHistoryPortal();
}

function applyRedisCmdHistory(cmd) {
  document.getElementById('redisCommand').value = cmd;
  _closeHistoryPortal();
  switchRedisTab('exec');
}

function toggleRedisCmdHistory(e, btn) {
  e.stopPropagation();
  if (_hdPortalType === 'redis') { _closeHistoryPortal(); return; }
  _closeHistoryPortal();
  _renderRedisCmdHistoryContent(btn);
}

function closeRedisCmdHistory() { if (_hdPortalType === 'redis') _closeHistoryPortal(); }

function _renderRedisCmdHistoryContent(anchorBtn) {
  const list   = loadRedisCmdHistory();
  const anchor = anchorBtn || document.getElementById('redisCmdHistoryWrap')?.querySelector('button');

  let html = '';
  if (!list.length) {
    html = '<div class="sql-history-empty">暂无历史记录</div>';
  } else {
    html = list.map(h => `
      <div class="sql-history-item" data-action="apply-redis" data-cmd="${esc(h.cmd)}">
        <span class="sql-history-item-sql" title="${esc(h.cmd)}">${esc(h.cmd)}</span>
        <span class="sql-history-item-save" data-action="template" data-cmd="${esc(h.cmd)}" data-kind="redis" title="存为模板">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
        </span>
        <span class="sql-history-item-del" data-action="delete-redis" data-cmd="${esc(h.cmd)}" title="删除">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6L6 18M6 6l12 12"/></svg>
        </span>
      </div>`).join('') +
      `<div class="sql-history-footer">
        <button class="btn btn-ghost" data-action="clear-redis" style="font-size:11px;height:26px;padding:0 8px;color:var(--danger)">清空全部</button>
      </div>`;
  }
  if (anchor) _openHistoryPortal(anchor, 'redis', html);
}

(function initRedisCmdHistory() {
  const list = loadRedisCmdHistory();
  if (list.length) {
    const el = document.getElementById('redisCommand');
    if (el && !el.value) el.value = list[0].cmd;
  }
})();
