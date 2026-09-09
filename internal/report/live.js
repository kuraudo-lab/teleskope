// Same-origin browser updates read the shared snapshot; they never trigger scans.
let liveRevision = -1, liveETag = '', updatesPaused = false;
const liveStatus = byId('live-status');
liveStatus.innerHTML = '<div id="live-message" role="status" aria-atomic="true">Waiting for first snapshot…</div><details id="live-coverage"><summary>Collection details</summary><div id="live-details"></div></details><button id="pause-updates" type="button">Pause page updates</button>';
document.querySelector('.brand p').textContent = 'live cluster inventory';
byId('title').textContent = 'Waiting for first snapshot';
byId('subtitle').textContent = 'Collection runs in the background. This page retries automatically.';
byId('pause-updates').onclick = () => {
  updatesPaused = !updatesPaused;
  liveETag = '';
  byId('pause-updates').textContent = updatesPaused ? 'Resume page updates' : 'Pause page updates';
  byId('pause-updates').setAttribute('aria-pressed', String(updatesPaused));
  if (updatesPaused) byId('live-message').textContent = 'Page updates paused. Background collection continues.';
};
function detailKey(obj) {
  if (obj.uid) return 'uid:' + obj.uid;
  return JSON.stringify([obj.apiVersion, obj.kind, obj.namespace, obj.name, obj.image, obj.pod, obj.container, obj.crdName, obj.principalArn, obj.serviceAccount, obj.roleArn]);
}
function applyLiveSnapshot(next) {
  const scrolls = [...document.querySelectorAll('.scroll')].map(el => [el, el.scrollTop, el.scrollLeft]);
  snapshot = next;
  k = snapshot.kubernetes || {};
  eks = snapshot.eks || {};
  cluster = eks.cluster || {};
  renderHeader();
  // Keep a vanished namespace selected so deletion never silently widens scope.
  if (nsFilter !== 'all' && ![...byId('focus').options].some(o => o.value === nsFilter)) {
    const option = document.createElement('option');
    option.value = nsFilter;
    option.textContent = nsFilter + ' (no longer present)';
    byId('focus').appendChild(option);
  }
  byId('focus').value = nsFilter;
  byId('resourceType').value = resourceFilter;
  document.body.classList.remove('live-loading');
  renderAll();
  syncResourceFilterState();
  scrolls.forEach(([el, top, left]) => { el.scrollTop = top; el.scrollLeft = left; });
  if (selectedDetail) {
    const key = detailKey(selectedDetail);
    const element = [...document.querySelectorAll('[data-detail]')].find(el => detailKey(JSON.parse(el.dataset.detail)) === key);
    if (element) showText(element.dataset.detail);
    else {
      byId('drawerTitle').textContent = 'Selection no longer in this view';
      byId('drawerBody').textContent = 'This item was removed or no longer matches the current view.';
    }
  }
}
function renderLiveStatus(sources) {
  const labels = {loading:'Loading', ready:'Up to date', partial:'Partial coverage', stale:'Stale — showing previous data', error:'No data — collection failed'};
  const lines = Object.entries(sources).map(([name, s]) => {
    const last = s.lastSuccess ? new Date(s.lastSuccess).toLocaleString() : 'none yet';
    return name + ': ' + (labels[s.state] || s.state) + (s.refreshing ? ' · refreshing' : '') + ' · last published ' + last;
  });
  const message = lines.join('\n');
  if (byId('live-message').textContent !== message) byId('live-message').textContent = message;
  byId('live-message').style.whiteSpace = 'pre-line';
  byId('live-details').innerHTML = Object.entries(sources).map(([name, s]) => {
    const errors = (s.coverage || []).filter(c => c.status !== 'complete');
    return '<p><strong>' + esc(name) + '</strong>' + (s.error ? ': ' + esc(s.error) : '') +
      (s.nextAttempt ? ' · next attempt ' + esc(new Date(s.nextAttempt).toLocaleTimeString()) : '') + '</p>' +
      errors.map(c => '<p>' + esc(c.resource) + ': ' + esc(c.status) + ' — ' + esc(c.reason) + '</p>').join('');
  }).join('');
}
async function refreshLivePage() {
  try {
    // Avoid replacing SVG nodes while a pointer drag is in progress.
    if (!updatesPaused && !topologyDrag.active) {
      const result = await fetch('/api/snapshot', {
        cache: 'no-store',
        headers: liveETag ? {'If-None-Match': liveETag} : {},
        signal: AbortSignal.timeout(10000)
      });
      if (result.status !== 304) {
        if (!result.ok) throw new Error('HTTP ' + result.status);
        const data = await result.json();
        if (!updatesPaused && !topologyDrag.active) {
          if (data.snapshot && data.revision !== liveRevision) {
            applyLiveSnapshot(data.snapshot);
            liveRevision = data.revision;
          }
          renderLiveStatus(data.sources);
          liveETag = result.headers.get('ETag') || '';
        }
      }
    }
  } catch (error) {
    liveETag = ''; // Re-render status after reconnect, even if the revision is unchanged.
    if (!updatesPaused) byId('live-message').textContent = 'Connection lost — displayed data may be stale. Retrying automatically. ' + error.message;
  } finally {
    setTimeout(refreshLivePage, 5000);
  }
}
refreshLivePage();
