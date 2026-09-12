package hub

import (
	"encoding/base64"
	"strings"

	_ "embed"
)

//go:embed teleskope-icon-128.png
var hubIconPNG []byte

// HubHTML renders the embedded multi-cluster fleet page.
func HubHTML() string {
	icon := "data:image/png;base64," + base64.StdEncoding.EncodeToString(hubIconPNG)
	page := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Teleskope hub</title>
  <style>
    :root { color-scheme: light dark; --bg:#f6f8fb; --panel:rgba(255,255,255,.88); --panel-2:rgba(248,250,252,.9); --line:rgba(15,23,42,.13); --text:#0f172a; --muted:#64748b; --blue:#2563eb; --cyan:#0891b2; --green:#10b981; --amber:#d97706; --red:#e11d48; --purple:#7c3aed; --chip:rgba(37,99,235,.1); --chip-text:#1e3a8a; --aside-bg:rgba(255,255,255,.72); --nav-hover:rgba(37,99,235,.11); --control-bg:rgba(255,255,255,.92); --card-bg:linear-gradient(180deg,rgba(255,255,255,.94),rgba(241,245,249,.86)); --table-line:rgba(15,23,42,.08); --table-head-bg:rgba(248,250,252,.96); --shadow:0 24px 80px rgba(15,23,42,.14); --radius:18px; font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; }
    @media (prefers-color-scheme: dark) { :root { color-scheme: dark; --bg:#08111f; --panel:rgba(17,29,50,.82); --panel-2:rgba(11,20,35,.88); --line:rgba(148,163,184,.18); --text:#e5eefb; --muted:#91a4bd; --blue:#60a5fa; --cyan:#22d3ee; --green:#34d399; --amber:#f59e0b; --red:#fb7185; --purple:#a78bfa; --chip:rgba(96,165,250,.12); --chip-text:#cfe5ff; --aside-bg:rgba(4,10,20,.58); --nav-hover:rgba(96,165,250,.12); --control-bg:rgba(15,23,42,.9); --card-bg:linear-gradient(180deg,rgba(30,41,59,.84),rgba(15,23,42,.76)); --table-line:rgba(148,163,184,.12); --table-head-bg:rgba(15,23,42,.96); --shadow:0 24px 80px rgba(0,0,0,.42); } }
    * { box-sizing:border-box; }
    body { margin:0; background:radial-gradient(circle at 20% -10%,rgba(34,211,238,.16),transparent 34%),radial-gradient(circle at 92% 10%,rgba(167,139,250,.20),transparent 34%),var(--bg); color:var(--text); }
    .app { min-height:100vh; display:grid; grid-template-columns:280px 1fr; }
    aside { border-right:1px solid var(--line); background:var(--aside-bg); backdrop-filter:blur(16px); padding:24px 18px; position:sticky; top:0; height:100vh; }
    .brand { display:flex; align-items:center; gap:12px; margin-bottom:24px; }
    .logo { width:42px; height:42px; border-radius:14px; box-shadow:0 0 30px rgba(34,211,238,.35); flex:none; display:block; object-fit:cover; background:var(--panel-2); }
    .brand h1 { font-size:18px; line-height:1.1; margin:0; }
    .brand p { color:var(--muted); margin:2px 0 0; font-size:12px; }
    nav button { width:100%; display:flex; align-items:center; gap:10px; background:transparent; color:var(--muted); border:0; text-align:left; padding:11px 12px; border-radius:12px; cursor:pointer; font-size:14px; }
    nav button:hover, nav button.active { background:var(--nav-hover); color:var(--text); }
    .dot { width:8px; height:8px; border-radius:999px; background:var(--blue); box-shadow:0 0 16px currentColor; }
    main { padding:28px; width:100%; }
    header { display:flex; justify-content:space-between; gap:18px; align-items:flex-start; margin-bottom:22px; }
    .eyebrow { color:var(--cyan); text-transform:uppercase; letter-spacing:.12em; font-size:12px; font-weight:700; }
    h2 { margin:4px 0 8px; font-size:34px; letter-spacing:-.04em; }
    .sub, .muted { color:var(--muted); }
    .toolbar { display:flex; gap:10px; align-items:center; flex-wrap:wrap; }
    button, select { background:var(--control-bg); color:var(--text); border:1px solid var(--line); border-radius:12px; height:42px; padding:0 12px; cursor:pointer; font:inherit; font-size:14px; }
    button:hover, select:hover { background:var(--nav-hover); }
    .cards { display:grid; grid-template-columns:repeat(6,minmax(0,1fr)); gap:16px; margin-bottom:18px; }
    .card { background:var(--card-bg); border:1px solid var(--line); border-radius:var(--radius); padding:16px; box-shadow:var(--shadow); }
    .label { color:var(--muted); font-size:12px; }
    .value { font-size:28px; font-weight:800; margin-top:6px; letter-spacing:-.04em; }
    .section { display:none; }
    .section.active { display:block; }
    .panel { background:var(--panel); border:1px solid var(--line); border-radius:var(--radius); box-shadow:var(--shadow); overflow:hidden; }
    .panel h3 { margin:0; padding:16px 18px; border-bottom:1px solid var(--line); font-size:16px; }
    .scroll { overflow-x:auto; overflow-y:visible; }
    table { width:100%; border-collapse:collapse; font-size:13px; }
    th, td { padding:10px 12px; border-bottom:1px solid var(--table-line); text-align:left; vertical-align:top; }
    th { color:var(--muted); background:var(--table-head-bg); font-weight:600; position:sticky; top:0; z-index:1; }
    tr:hover td { background:rgba(37,99,235,.06); }
    .badge { display:inline-flex; border-radius:999px; padding:3px 8px; font-size:12px; font-weight:700; white-space:nowrap; }
    .ready { color:#052e22; background:var(--green); }
    .partial, .stale { color:#422006; background:var(--amber); }
    .error { color:#450a0a; background:var(--red); }
    .sources { display:flex; gap:6px; flex-wrap:wrap; }
    a { color:var(--blue); text-decoration:none; }
    a:hover { text-decoration:underline; }
    @media (max-width:1000px) { .app { grid-template-columns:1fr; } aside { position:relative; height:auto; } header { flex-direction:column; } .cards { grid-template-columns:repeat(2,minmax(0,1fr)); } }
  </style>
</head>
<body>
<div class="app">
  <aside>
    <div class="brand"><img class="logo" src="__TELESKOPE_ICON__" alt="" aria-hidden="true" /><div><h1>Teleskope</h1><p>multi-cluster hub</p></div></div>
    <nav id="nav"></nav>
  </aside>
  <main>
    <header>
      <div><div class="eyebrow">Fleet inventory</div><h2 id="title">Teleskope hub</h2><p class="sub" id="subtitle">Waiting for cluster envelopes.</p></div>
      <div class="toolbar">
        <select id="stateFilter" aria-label="State filter"><option value="all">All states</option><option value="ready">Ready</option><option value="partial">Partial</option><option value="stale">Stale</option><option value="error">Error</option></select>
        <button id="exportJSON" type="button">JSON</button>
        <button id="exportMarkdown" type="button">Markdown</button>
      </div>
    </header>
    <section class="section active" data-section="fleet">
      <div class="cards" id="cards"></div>
      <div class="panel"><h3>Clusters</h3><div class="scroll"><table><thead><tr><th>Cluster</th><th>Provider</th><th>Region</th><th>State</th><th>Sources</th><th>Revision</th><th>Collected</th><th>Inventory</th><th>Advisor</th></tr></thead><tbody id="clusters"><tr><td colspan="9" class="muted">No clusters have reported yet.</td></tr></tbody></table></div></div>
    </section>
    <section class="section" data-section="events">
      <div class="panel"><h3>Recent events</h3><div class="scroll"><table><thead><tr><th>Time</th><th>Cluster</th><th>Source</th><th>Level</th><th>Message</th></tr></thead><tbody id="events"><tr><td colspan="5" class="muted">No events have reported yet.</td></tr></tbody></table></div></div>
    </section>
  </main>
</div>
<script>
let etag = "", fleet = {revision:0, clusters:[], events:[]};
const navItems = [["fleet","Fleet"],["events","Events"]];
const byId = id => document.getElementById(id);
const esc = v => String(v ?? "-").replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const stateClass = s => ['ready','partial','stale','error'].includes(s) ? s : 'partial';
function renderNav() { byId('nav').innerHTML = navItems.map(([id,label],i) => '<button class="' + (i === 0 ? 'active' : '') + '" data-target="' + id + '"><span class="dot"></span>' + esc(label) + '</button>').join(''); byId('nav').onclick = e => { const btn = e.target.closest('button'); if (!btn) return; selectSection(btn.dataset.target); }; }
function selectSection(id) { document.querySelectorAll('nav button').forEach(b => b.classList.toggle('active', b.dataset.target === id)); document.querySelectorAll('.section').forEach(s => s.classList.toggle('active', s.dataset.section === id)); }
function filtered() { const want = byId('stateFilter').value; return fleet.clusters.filter(c => want === 'all' || c.state === want); }
function totals(rows) { return rows.reduce((out, c) => { out.nodes += c.nodes || 0; out.workloads += c.workloads || 0; out.pods += c.pods || 0; out.images += c.images || 0; return out; }, {nodes:0, workloads:0, pods:0, images:0}); }
function sourceBadges(sources) { const entries = Object.entries(sources || {}); return entries.length ? '<div class="sources">' + entries.map(([name, status]) => '<span class="badge ' + stateClass(status.state) + '">' + esc(name) + ':' + esc(status.state || '-') + '</span>').join('') + '</div>' : '<span class="muted">-</span>'; }
function render() {
  const rows = filtered(), total = totals(fleet.clusters);
  byId('subtitle').textContent = fleet.clusters.length + ' clusters - hub revision ' + (fleet.revision || 0);
  byId('cards').innerHTML = [['Clusters', fleet.clusters.length], ['Ready', fleet.clusters.filter(c => c.state === 'ready').length], ['Needs attention', fleet.clusters.filter(c => c.state !== 'ready').length], ['Nodes', total.nodes], ['Workloads', total.workloads], ['Pods', total.pods]].map(([label, value]) => '<div class="card"><div class="label">' + esc(label) + '</div><div class="value">' + esc(value) + '</div></div>').join('');
  byId('clusters').innerHTML = rows.length ? rows.map(c => {
    const cluster = c.cluster || {};
    const link = '/cluster?id=' + encodeURIComponent(cluster.id);
    const inventory = 'nodes=' + (c.nodes || 0) + ' workloads=' + (c.workloads || 0) + ' pods=' + (c.pods || 0) + ' images=' + (c.images || 0);
    return '<tr><td><a href="' + link + '"><strong>' + esc(cluster.name || cluster.id) + '</strong></a><div class="muted">' + esc(cluster.id) + '</div></td><td>' + esc(cluster.provider) + '</td><td>' + esc(cluster.region) + '</td><td><span class="badge ' + stateClass(c.state) + '">' + esc(c.state) + '</span></td><td>' + sourceBadges(c.sources) + '</td><td>' + esc(c.revision) + '</td><td>' + esc(c.collectedAt) + '</td><td>' + esc(inventory) + '</td><td>' + esc(c.advisor || '-') + '</td></tr>';
  }).join('') : '<tr><td colspan="9" class="muted">No clusters match the current filter.</td></tr>';
  byId('events').innerHTML = (fleet.events || []).length ? fleet.events.map(item => {
    const cluster = item.cluster || {}, event = item.event || {};
    return '<tr><td>' + esc(event.at) + '</td><td>' + esc(cluster.name || cluster.id) + '</td><td>' + esc(event.source) + '</td><td><span class="badge ' + stateClass(event.level === 'error' ? 'error' : event.level === 'warn' ? 'partial' : 'ready') + '">' + esc(event.level) + '</span></td><td>' + esc(event.message) + '</td></tr>';
  }).join('') : '<tr><td colspan="5" class="muted">No events have reported yet.</td></tr>';
}
async function refresh() {
  try {
    const headers = etag ? {'If-None-Match': etag} : {};
    const response = await fetch('/api/clusters', {headers, cache:'no-store'});
    if (response.status === 304) return;
    if (!response.ok) throw new Error('HTTP ' + response.status);
    etag = response.headers.get('ETag') || '';
    fleet = await response.json();
    render();
  } catch (error) {
    byId('subtitle').textContent = 'Hub API unavailable: ' + error.message;
  } finally {
    setTimeout(refresh, 5000);
  }
}
function download(name, text, type) { const blob = new Blob([text], {type}); const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = name; document.body.appendChild(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(url), 1000); }
byId('stateFilter').addEventListener('change', render);
byId('exportJSON').onclick = async () => download('teleskope-fleet.json', await (await fetch('/api/export/fleet.json')).text(), 'application/json;charset=utf-8');
byId('exportMarkdown').onclick = async () => download('teleskope-fleet-summary.md', await (await fetch('/api/export/summary.md')).text(), 'text/markdown;charset=utf-8');
renderNav(); render(); refresh();
</script>
</body>
</html>`
	return strings.Replace(page, "__TELESKOPE_ICON__", icon, 1)
}
