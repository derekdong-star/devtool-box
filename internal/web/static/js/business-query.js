const BUSINESS_QUERY_ENDPOINT = '/api/business-query/tokenrouter/user-overview';

function businessQueryReq() {
  return {
    identifier: document.getElementById('businessQueryIdentifier').value.trim(),
    activeSessionsOnly: document.getElementById('businessQueryActiveSessionsOnly').checked,
    recentSessionLimit: parseInt(document.getElementById('businessQuerySessionLimit').value || '20', 10),
  };
}

async function runTokenRouterUserOverview() {
  const req = businessQueryReq();
  const resultEl = document.getElementById('businessQueryResult');
  const btn = document.getElementById('businessQueryRunBtn');

  if (!req.identifier) {
    showError(resultEl, '请输入用户邮箱或用户 UUID');
    return;
  }

  btn.disabled = true;
  resultEl.innerHTML = `<div class="empty-state"><div class="spinner"></div><p>正在聚合查询 TokenRouter 数据...</p></div>`;

  try {
    const res = await post(BUSINESS_QUERY_ENDPOINT, req, 90000);
    if (!ok(res)) {
      showError(resultEl, err(res));
      return;
    }
    renderBusinessQueryResult(res.data);
    saveTabState('business-query');
  } catch (e) {
    showError(resultEl, describeRequestError(e));
  } finally {
    btn.disabled = false;
  }
}

function renderBusinessQueryResult(data) {
  const resultEl = document.getElementById('businessQueryResult');
  const sections = data.sections || [];
  if (!sections.length) {
    resultEl.innerHTML = '<div class="empty-state"><p>查询成功，但没有返回分组结果</p></div>';
    return;
  }

  resultEl.innerHTML = sections.map(renderBusinessQuerySection).join('');
  appear(resultEl);
}

function renderBusinessQuerySection(section) {
  const rows = section.rows || [];
  const columns = section.columns || [];
  const badgeClass = rows.length ? 'badge-success' : 'badge-neutral';
  return `
    <div class="card business-query-section">
      <div class="card-header">
        <div class="card-icon card-icon-violet">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M4 6h16M4 12h16M4 18h16"/></svg>
        </div>
        <span class="card-title">${esc(section.title || section.key)}</span>
        <span class="badge ${badgeClass}" style="margin-left:auto">${rows.length} 行</span>
      </div>
      ${rows.length ? renderBusinessQueryTable(columns, rows) : '<div class="empty-state"><p>无匹配数据</p></div>'}
    </div>`;
}

function renderBusinessQueryTable(columns, rows) {
  if (!columns.length && rows.length) columns = Object.keys(rows[0]);
  return `<div class="business-query-table-wrap">
    <table>
      <thead><tr>${columns.map(c => `<th>${esc(c)}</th>`).join('')}</tr></thead>
      <tbody>
        ${rows.map(row => `<tr>${columns.map(c => `<td>${formatBusinessQueryCell(row[c])}</td>`).join('')}</tr>`).join('')}
      </tbody>
    </table>
  </div>`;
}

function formatBusinessQueryCell(value) {
  if (value == null) return '<span style="color:var(--text-tertiary)">NULL</span>';
  if (typeof value === 'boolean') {
    return value
      ? '<span class="badge badge-success">true</span>'
      : '<span class="badge badge-neutral">false</span>';
  }
  return esc(String(value));
}
