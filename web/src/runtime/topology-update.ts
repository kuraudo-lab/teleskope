import type { Graph, TopologyUpdate } from './types'
export function applyTopologyUpdate(current: Graph | undefined, update: TopologyUpdate): Graph | null {
  if (update.schemaVersion !== 'teleskope.io/topology-update/v1alpha1') return null
  if (update.kind === 'full') {
    const graph = update.graph
    return graph?.clusterId === update.clusterId && graph?.revision === update.revision ? graph : null
  }
  if (!current || current.clusterId !== update.clusterId) return null
  if (current.revision === update.revision) return current
  if (current.revision !== update.baseRevision || update.revision < current.revision) return null
  if (update.kind === 'unchanged') return null
  if (update.kind !== 'delta') return null
  const nodes = new Map(current.nodes.map(node => [node.id, node]))
  const edges = new Map(current.edges.map(edge => [edge.id, edge]))
  update.deleteNodeIds?.forEach(id => nodes.delete(id))
  update.deleteEdgeIds?.forEach(id => edges.delete(id))
  update.upsertNodes?.forEach(node => nodes.set(node.id, node))
  update.upsertEdges?.forEach(edge => edges.set(edge.id, edge))
  if ([...edges.values()].some(edge => !nodes.has(edge.source) || !nodes.has(edge.target))) return null
  return { ...current, revision: update.revision, generatedAt: update.generatedAt,
    nodes: [...nodes.values()].sort((a,b) => a.id.localeCompare(b.id)),
    edges: [...edges.values()].sort((a,b) => a.id.localeCompare(b.id)), coverage: update.coverage || [] }
}
