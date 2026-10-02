import { shallowReactive } from 'vue'
import type { AnalysisRequest, BootConfig, Graph, Publication, Resource, Snapshot } from './types'
import { applyTopologyUpdate } from './topology-update'

export function createReportStore(boot: BootConfig, initial: Partial<Publication> = {}, request: typeof fetch = fetch) {
  const state = shallowReactive({
    snapshot: initial.snapshot || {} as Snapshot, graph: initial.graph as Graph | undefined,
    advisor: initial.advisor || {}, eksProjection: initial.eksProjection || {},
    sources: initial.sources || {}, events: initial.events || [] as Resource[],
    revision: initial.revision ?? (boot.mode === 'offline' ? 0 : -1),
    loading: boot.mode === 'live', error: '', paused: false, analyzing: false,
  })
  let disposed = false, busy = false, etag = '', timer: ReturnType<typeof setTimeout> | undefined
  const controllers = new Set<AbortController>()
  async function http(path: string | undefined, init: RequestInit = {}) {
    if (!path) throw new Error('This operation is unavailable for this report.')
    const controller = new AbortController()
    controllers.add(controller)
    const timeout = setTimeout(() => controller.abort(), 10000)
    try { return await request(path, { ...init, signal: controller.signal, cache: 'no-store' }) }
    finally { clearTimeout(timeout); controllers.delete(controller) }
  }
  function apply(data: Publication) {
    if (data.revision < state.revision) return
    state.snapshot = data.snapshot; state.graph = data.graph
    state.advisor = data.advisor || {}; state.eksProjection = data.eksProjection || {}
    state.sources = data.sources || {}; state.events = data.events || []
    state.revision = data.revision; state.loading = false; state.error = ''
  }
  async function refresh() {
    if (disposed || busy || state.paused || boot.mode !== 'live') return
    busy = true
    try {
      const response = await http(boot.endpoints?.snapshot, {headers: etag ? {'If-None-Match': etag} : {}})
      if (response.status === 304) { state.error = ''; return }
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      const data: Publication = await response.json()
      if (boot.endpoints?.topology && state.graph && data.revision !== state.revision) {
        try {
          const url = new URL(boot.endpoints.topology, location.href)
          url.searchParams.set('since', String(state.graph.revision))
          const delta = await http(url.toString())
          if (delta.ok) {
            const graph = applyTopologyUpdate(state.graph, await delta.json())
            // The separate endpoint may already have advanced. Keep the publication atomic.
            if (graph?.revision === data.revision && graph.clusterId === data.graph?.clusterId) data.graph = graph
          }
        } catch { /* The publication contains a full resync graph. */ }
      }
      if (disposed || state.paused) return
      if (!data.snapshot || !Number.isFinite(data.revision)) throw new Error('Invalid snapshot publication')
      apply(data); etag = response.headers.get('ETag') || ''
    } catch (error) {
      etag = ''
      if (!disposed) state.error = `Connection lost — displayed data may be stale. ${String(error)}`
    } finally { busy = false }
  }
  async function poll() { await refresh(); if (!disposed) timer = setTimeout(poll, 5000) }
  async function analyze(input: AnalysisRequest) {
    if (state.analyzing) throw new Error('Analysis is already running')
    state.analyzing = true
    try {
      const response = await http(boot.endpoints?.analyze, {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(input)})
      if (!response.ok) throw new Error((await response.text()).trim() || `HTTP ${response.status}`)
      const data = await response.json()
      if (disposed || input.revision !== state.revision) throw new Error('The snapshot changed during analysis. Run analysis again.')
      return data.analysis
    } finally { state.analyzing = false }
  }
  return { state, refresh, analyze, apply,
    start() { void poll() },
    pause(value: boolean) { state.paused = value; etag = ''; if (!value) void refresh() },
    dispose() { disposed = true; clearTimeout(timer); controllers.forEach(c => c.abort()); controllers.clear() },
  }
}
