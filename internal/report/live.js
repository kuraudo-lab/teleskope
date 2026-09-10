// Same-origin browser updates read the shared snapshot; they never trigger scans.
let liveRevision = -1, liveETag = '', updatesPaused = false;
document.querySelectorAll('.section').forEach(section => {
  const progress = document.createElement('div');
  progress.className = 'live-progress';
  progress.setAttribute('role', 'status');
  progress.setAttribute('aria-live', 'polite');
  progress.innerHTML = '<span class="live-progress-icon spinning"></span><span>Waiting for first snapshot…</span>';
  section.prepend(progress);
});
document.querySelector('.toolbar').insertAdjacentHTML('beforeend', '<button id="pause-updates" class="export-button" type="button">Pause updates</button>');
document.querySelector('.brand p').textContent = 'live cluster inventory';
byId('title').textContent = 'Waiting for first snapshot';
byId('subtitle').textContent = 'Collection runs in the background. This page retries automatically.';
byId('pause-updates').onclick = () => {
  updatesPaused = !updatesPaused;
  liveETag = '';
  byId('pause-updates').textContent = updatesPaused ? 'Resume page updates' : 'Pause page updates';
  byId('pause-updates').setAttribute('aria-pressed', String(updatesPaused));
  if (updatesPaused) updateLiveProgress('⏸', 'Page updates paused. Background collection continues.', false);
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
function renderLiveStatus(sources, events) {
  const labels = {loading:'Loading', ready:'Up to date', partial:'Partial coverage', stale:'Stale — showing previous data', error:'No data — collection failed'};
  const entries = Object.entries(sources);
  const lines = entries.map(([name, s]) => {
    const last = s.lastSuccess ? new Date(s.lastSuccess).toLocaleString() : 'none yet';
    return name + ': ' + (labels[s.state] || s.state) + (s.refreshing ? ' · refreshing' : '') + ' · last published ' + last;
  });
  const hasRefreshing = entries.some(([, s]) => s.refreshing || s.state === 'loading');
  const hasError = entries.some(([, s]) => s.state === 'error');
  const hasStale = entries.some(([, s]) => s.state === 'stale');
  const hasPartial = entries.some(([, s]) => s.state === 'partial');
  const icon = hasRefreshing ? '⟳' : hasError ? '×' : hasStale || hasPartial ? '!' : '✓';
  updateLiveProgress(icon, lines.join(' | ') || 'Waiting for first snapshot…', hasRefreshing);
  byId('live-event-list').innerHTML = arr(events).slice(-80).reverse().map(event => {
    const at = event.at ? new Date(event.at).toLocaleTimeString() : '-';
    const level = event.level || 'info';
    return '<div class="live-event live-event-' + esc(level) + '">' +
      '<span class="live-event-time">' + esc(at) + '</span>' +
      '<span class="live-event-source">' + esc(event.source || '-') + '</span>' +
      '<span class="live-event-level">' + esc(level) + '</span>' +
      '<span class="live-event-message">' + esc(event.message || '-') + '</span>' +
      '</div>';
  }).join('') || '<p class="muted">No events yet</p>';
}
function updateLiveProgress(icon, message, spinning) {
  document.querySelectorAll('.live-progress').forEach(progress => {
    progress.innerHTML = '<span class="live-progress-icon' + (spinning ? ' spinning' : '') + '">' + (spinning ? '' : esc(icon)) + '</span><span>' + esc(message) + '</span>';
  });
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
          advisorReport = data.advisor || {};
          renderAdvisor();
          if (data.snapshot && data.revision !== liveRevision) {
            applyLiveSnapshot(data.snapshot);
            liveRevision = data.revision;
          }
          renderLiveStatus(data.sources, data.events);
          liveETag = result.headers.get('ETag') || '';
        }
      }
    }
  } catch (error) {
    liveETag = ''; // Re-render status after reconnect, even if the revision is unchanged.
    if (!updatesPaused) updateLiveProgress('!', 'Connection lost — displayed data may be stale. Retrying automatically. ' + error.message, false);
  } finally {
    setTimeout(refreshLivePage, 5000);
  }
}
refreshLivePage();
