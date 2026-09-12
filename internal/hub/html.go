package hub

import "strings"

// HubHTML renders the embedded multi-cluster fleet page.
func HubHTML() string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Teleskope hub</title>
  <style>
    :root { color-scheme: light dark; --bg:#f6f8fb; --panel:#ffffff; --panel-2:#f8fafc; --line:#d8e0ea; --text:#0f172a; --muted:#64748b; --blue:#2563eb; --green:#10b981; --amber:#d97706; --red:#e11d48; font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; }
    @media (prefers-color-scheme: dark) { :root { --bg:#08111f; --panel:#101b2f; --panel-2:#0b1423; --line:#26364f; --text:#e5eefb; --muted:#91a4bd; --blue:#60a5fa; --green:#34d399; --amber:#f59e0b; --red:#fb7185; } }
    * { box-sizing:border-box; }
    body { margin:0; background:var(--bg); color:var(--text); }
    main { padding:28px; max-width:1440px; margin:0 auto; }
    header { display:flex; justify-content:space-between; gap:18px; align-items:flex-start; margin-bottom:20px; }
    h1 { margin:0 0 6px; font-size:30px; letter-spacing:-.02em; }
    .sub, .muted { color:var(--muted); }
    .toolbar { display:flex; gap:10px; flex-wrap:wrap; }
    button, select { height:40px; border:1px solid var(--line); border-radius:8px; background:var(--panel); color:var(--text); padding:0 12px; font:inherit; }
    button { cursor:pointer; }
    button:hover { border-color:var(--blue); }
    .cards { display:grid; grid-template-columns:repeat(6,minmax(0,1fr)); gap:12px; margin-bottom:16px; }
    .card, .panel { background:var(--panel); border:1px solid var(--line); border-radius:8px; }
    .card { padding:14px; }
    .label { color:var(--muted); font-size:12px; }
    .value { font-size:28px; font-weight:800; margin-top:4px; }
    .panel { overflow:hidden; }
    table { width:100%; border-collapse:collapse; font-size:13px; }
    th, td { padding:11px 12px; border-bottom:1px solid var(--line); text-align:left; vertical-align:top; }
    th { color:var(--muted); background:var(--panel-2); font-weight:600; }
    tr:hover td { background:var(--panel-2); }
    .badge { display:inline-flex; border-radius:999px; padding:3px 8px; font-size:12px; font-weight:700; }
    .stack { display:grid; gap:16px; }
    .sources { display:flex; gap:6px; flex-wrap:wrap; }
    .event-list { max-height:320px; overflow:auto; }
    .ready { color:#052e22; background:var(--green); }
    .partial, .stale { color:#422006; background:var(--amber); }
    .error { color:#450a0a; background:var(--red); }
    a { color:var(--blue); text-decoration:none; }
    a:hover { text-decoration:underline; }
    @media (max-width:900px) { main { padding:18px; } header { flex-direction:column; } .cards { grid-template-columns:repeat(2,minmax(0,1fr)); } .panel { overflow:auto; } }
  </style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>Teleskope hub</h1>
      <p class="sub" id="subtitle">Waiting for cluster envelopes.</p>
    </div>
    <div class="toolbar">
      <select id="stateFilter" aria-label="State filter"><option value="all">All states</option><option value="ready">Ready</option><option value="partial">Partial</option><option value="stale">Stale</option><option value="error">Error</option></select>
      <button id="exportJSON" type="button">JSON</button>
      <button id="exportMarkdown" type="button">Markdown</button>
    </div>
  </header>
  <section class="cards" id="cards"></section>
  <section class="panel" style="margin-bottom:16px">
    <table>
      <thead><tr><th>Cluster</th><th>Provider</th><th>Region</th><th>State</th><th>Sources</th><th>Revision</th><th>Collected</th><th>Inventory</th><th>Advisor</th></tr></thead>
      <tbody id="clusters"><tr><td colspan="9" class="muted">No clusters have reported yet.</td></tr></tbody>
    </table>
  </section>
  <section class="panel">
    <table>
      <thead><tr><th>Time</th><th>Cluster</th><th>Source</th><th>Level</th><th>Message</th></tr></thead>
      <tbody id="events"><tr><td colspan="5" class="muted">No events have reported yet.</td></tr></tbody>
    </table>
  </section>
</main>
<script>
let etag = "", fleet = {revision:0, clusters:[]};
const byId = id => document.getElementById(id);
const esc = v => String(v ?? "-").replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const stateClass = s => ['ready','partial','stale','error'].includes(s) ? s : 'partial';
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
render(); refresh();
</script>
</body>
</html>`
}

func escapeScriptText(value string) string {
	return strings.ReplaceAll(value, "</", "<\\/")
}
