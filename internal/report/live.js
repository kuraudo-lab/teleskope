// Same-origin browser updates read the shared snapshot; they never trigger scans.
let liveRevision = -1, liveETag = '', updatesPaused = false, analysisInFlight = false, lastAnalyzedRevision = -1, lastAnalyzedRequestKey = '', lastAnalyzedRequestKeys = {}, liveSources = {};
document.querySelector('.toolbar').insertAdjacentHTML('beforeend', '<button id="pause-updates" class="export-button" type="button">Pause updates</button>');
const analyzeButton = byId('analyzeSnapshot');
const pageAnalysisButtons = [...document.querySelectorAll('[data-analyze-page]')];
const scopedAnalysisPages = new Set(['eks','nodes','workloads','network','storage','security']);
function normalizedAnalysisPage(pageId) { return String(pageId || '').startsWith('eks-') ? 'eks' : pageId; }
document.querySelectorAll('[data-analysis-page]').forEach(panel => { panel.hidden = false; });
analyzeButton.hidden = false;
syncAnalyzeButtonState();
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
function syncAnalyzeButtonState() {
  if (!analyzeButton) return;
  syncOneAnalyzeButton(analyzeButton, activeSection, 'Analyze with AI', lastAnalyzedRequestKey);
  pageAnalysisButtons.forEach(button => {
    const page = button.dataset.analyzePage;
    const label = 'Analyze ' + page[0].toUpperCase() + page.slice(1);
    syncOneAnalyzeButton(button, page, label, lastAnalyzedRequestKeys[page] || '');
  });
}
function syncOneAnalyzeButton(button, page, label, lastKey) {
  if (document.body.classList.contains('live-loading') || liveRevision < 0) {
    button.disabled = true;
    button.innerHTML = `<span>${label}</span>`;
    button.title = 'Analyze is available after the first scan completes.';
  } else if (analysisInFlight) {
    button.disabled = true;
    button.innerHTML = '<span>Analyzing…</span>';
    button.title = 'Analysis is running.';
  } else if (currentAnalysisKey(currentAnalysisRequest(page)) === lastKey) {
    button.disabled = true;
    button.innerHTML = '<span>Analyzed</span>';
    button.title = 'This page scope has already been analyzed.';
  } else {
    button.disabled = false;
    button.innerHTML = `<span>${label}</span>`;
    button.title = `Analyze the current ${page} scope.`;
  }
}
analyzeButton.onclick = () => runScopedAnalysis(activeSection, true);
pageAnalysisButtons.forEach(button => { button.onclick = () => runScopedAnalysis(button.dataset.analyzePage, false); });
async function runScopedAnalysis(pageId, fromToolbar) {
  const resultPage = scopedAnalysisPages.has(normalizedAnalysisPage(pageId)) ? normalizedAnalysisPage(pageId) : 'advisor';
  const analysisRequest = currentAnalysisRequest(pageId);
  const analysisKey = currentAnalysisKey(analysisRequest);
  const lastKey = fromToolbar ? lastAnalyzedRequestKey : (lastAnalyzedRequestKeys[pageId] || '');
  if (analysisInFlight || analysisKey === lastKey || liveRevision < 0 || document.body.classList.contains('live-loading')) return;
  analysisInFlight = true;
  syncAnalyzeButtonState();
  setAnalysisMessage(`Requesting AI analysis for the current ${pageId} scope…`, 'empty', resultPage);
  if (fromToolbar && resultPage === 'advisor') selectSection('advisor');
  try {
    const result = await fetch(bootConfig.endpoints?.analyze, {method:'POST', cache:'no-store', headers:{'Content-Type':'application/json'}, body:JSON.stringify(analysisRequest)});
    if (!result.ok) throw new Error((await result.text()).trim() || ('HTTP ' + result.status));
    const data = await result.json();
    if (resultPage === 'advisor') llmAnalysis = data.analysis || null;
    else llmAnalyses[resultPage] = data.analysis || null;
    lastAnalyzedRevision = liveRevision;
    lastAnalyzedRequestKey = analysisKey;
    lastAnalyzedRequestKeys[pageId] = analysisKey;
    renderAnalysis(resultPage);
  } catch (error) {
    setAnalysisMessage('AI analysis failed: ' + error.message, 'error', resultPage);
  } finally {
    analysisInFlight = false;
    syncAnalyzeButtonState();
  }
}
function analysisResourceType(pageId) {
  pageId = normalizedAnalysisPage(pageId);
  if (scopedAnalysisPages.has(pageId)) return pageId;
  return pageId === 'overview' && resourceFilter !== 'all' ? resourceFilter : '';
}
function currentAnalysisRequest(pageId=activeSection) {
  const normalizedPage = normalizedAnalysisPage(pageId);
  const scope = {
    pageId: normalizedPage || '',
    namespace: nsFilter === 'all' ? '' : nsFilter,
    resourceType: analysisResourceType(normalizedPage),
    selectedRefs: selectedDetail ? [selectedDetail].filter(refLike) : []
  };
  return {revision: liveRevision, scope};
}
function currentAnalysisKey(request=currentAnalysisRequest()) {
  return liveRevision + ':' + JSON.stringify(request);
}
function refLike(value) {
  return value && (value.name || value.image) && (value.kind || value.apiVersion || value.namespace || value.name);
}
function detailKey(obj) {
  if (obj.uid) return 'uid:' + obj.uid;
  return JSON.stringify([obj.apiVersion, obj.kind, obj.namespace, obj.name, obj.image, obj.pod, obj.container, obj.crdName, obj.principalArn, obj.serviceAccount, obj.roleArn]);
}
function applyLiveSnapshot(next) {
  const scrolls = [...document.querySelectorAll('.scroll')].map(el => [el, el.scrollTop, el.scrollLeft]);
  snapshot = next.snapshot || next;
  eksProjection = next.eksProjection || {};
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
  syncAnalyzeButtonState();
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
  liveSources = sources || {};
  renderSourceFreshness();
  const entries = Object.entries(liveSources);
  const lines = entries.map(([name, s]) => {
    const last = s.lastSuccess ? new Date(s.lastSuccess).toLocaleString() : 'none yet';
    const state = s.mode === 'watch' ? watchStateLabel(s) : (labels[s.state] || s.state);
    return name + ': ' + state + (s.refreshing ? ' · refreshing' : '') + ' · last published ' + last;
  });
  const hasRefreshing = entries.some(([, s]) => s.refreshing || s.state === 'loading');
  const hasError = entries.some(([, s]) => s.state === 'error');
  const hasStale = entries.some(([, s]) => s.state === 'stale');
  const hasPartial = entries.some(([, s]) => s.state === 'partial');
  const icon = hasRefreshing ? '⟳' : hasError ? '×' : hasStale || hasPartial ? '!' : '✓';
  updateLiveProgress(icon, lines.join(' | ') || 'Waiting for first snapshot…', hasRefreshing);
  const eventHTML = arr(events).slice(-80).reverse().map(event => {
    const at = event.at ? new Date(event.at).toLocaleTimeString() : '-';
    const level = event.level || 'info';
    const kind = eventKind(event);
    return '<div class="live-event live-event-' + esc(level) + '">' +
      '<span class="live-event-time">' + esc(at) + '</span>' +
      '<span class="live-event-kind">' + esc(kind) + '</span>' +
      '<span class="live-event-source">' + esc(event.source || '-') + '</span>' +
      '<span class="live-event-level">' + esc(level) + '</span>' +
      '<span class="live-event-message">' + esc(event.message || '-') + '</span>' +
      '</div>';
  }).join('') || '<p class="muted">No events yet</p>';
  byId('live-event-list').innerHTML = eventHTML;
}
function renderSourceFreshness() {
  const target = byId('sourceFreshness');
  if (!target) return;
  const entries = Object.entries(liveSources || {});
  target.innerHTML = entries.map(([name, status]) => {
    const label = status.mode === 'watch' ? watchStateLabel(status) : (status.state || 'loading');
    const details = sourceFreshnessDetail(status);
    return '<span class="chip ' + esc(sourceStateClass(status.state)) + '" title="' + esc(details) + '"><strong>' + esc(name + ': ' + label) + '</strong><small>' + esc(details) + '</small></span>';
  }).join('');
}
function sourceStateClass(state) {
  if (state === 'ready') return 'ok';
  if (state === 'error') return 'bad';
  return 'warn';
}
function sourceFreshnessDetail(status) {
  const published = status.lastSuccess ? 'published ' + shortTime(status.lastSuccess) : 'not published';
  if (status.mode !== 'watch') return published;
  const eventAt = status.lastEventAt ? 'event ' + shortTime(status.lastEventAt) : 'event none';
  const fullSync = status.lastFullSyncAt ? 'full resync ' + shortTime(status.lastFullSyncAt) : 'full resync none';
  return eventAt + ' · ' + fullSync + ' · reconnects ' + (status.reconnects || 0);
}
function watchStateLabel(status) {
  if (status.state === 'ready') return 'watch connected';
  if (status.state === 'stale') return 'watch reconnecting';
  if (status.state === 'error') return 'watch error';
  if (status.state === 'partial') return 'watch partial';
  return 'watch ' + (status.state || 'loading');
}
function shortTime(value) {
  return value ? new Date(value).toLocaleTimeString() : '-';
}
function eventKind(event) {
  const source = event.source || '';
  const message = event.message || '';
  if (source === 'analysis' || message.includes('analysis')) return 'analysis';
  if (message.includes('watch ')) return 'watch';
  if (message.includes('published') || message.includes('retained previous data')) return 'publication';
  if (message.includes('refresh starting') || message.includes('collect')) return 'collection';
  return 'event';
}
function updateLiveProgress(icon, message, spinning) {
  const status = byId('eventsNavStatus');
  if (!status) return;
  const normalized = spinning ? 'loading' : icon === '×' || icon === '!' ? 'bad' : icon === '⏸' ? 'warn' : 'ready';
  status.className = 'nav-status ' + normalized;
  status.textContent = spinning ? 'Updating' : icon === '×' ? 'Error' : icon === '!' ? 'Partial' : icon === '⏸' ? 'Paused' : 'Ready';
  status.title = message;
}
async function refreshLivePage() {
  try {
    // Avoid replacing SVG nodes while a pointer drag is in progress.
    if (!updatesPaused && !topologyDrag.active) {
      const result = await fetch(bootConfig.endpoints?.snapshot, {
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
            liveRevision = data.revision;
            applyLiveSnapshot(data);
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
