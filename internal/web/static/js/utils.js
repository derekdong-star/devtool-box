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

async function logout() {
  await post('/api/auth/logout', {});
  location.href = '/login';
}

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

/* ─── 复制与 Toast ───────────────────────────────────────────── */
async function copyString(text) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.cssText = 'position:fixed;top:-9999px;left:-9999px;';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); } catch {}
    document.body.removeChild(ta);
  }
}

function showToast(msg) {
  let el = document.getElementById('_toast');
  if (!el) {
    el = document.createElement('div');
    el.id = '_toast';
    el.style.cssText = 'position:fixed;bottom:24px;left:50%;transform:translateX(-50%) translateY(20px);background:rgba(26,29,46,.85);color:#fff;padding:8px 18px;border-radius:8px;font-size:12px;z-index:2000;opacity:0;transition:opacity .2s,transform .2s;pointer-events:none;white-space:nowrap;';
    document.body.appendChild(el);
  }
  el.textContent = msg;
  el.style.opacity = '1';
  el.style.transform = 'translateX(-50%) translateY(0)';
  clearTimeout(el._timer);
  el._timer = setTimeout(() => {
    el.style.opacity = '0';
    el.style.transform = 'translateX(-50%) translateY(20px)';
  }, 1800);
}

/* ─── Copy (input element) ───────────────────────────────────── */
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
