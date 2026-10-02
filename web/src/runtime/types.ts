/** Extensible inventory records preserve provider fields across schema additions. */
export interface Resource { [key: string]: any; apiVersion?: string; kind?: string; namespace?: string; name?: string; uid?: string }
export interface Snapshot { schemaVersion?: string; collectedAt?: string; source?: {mode?: string}; kubernetes?: Record<string, any>; eks?: Record<string, any>; coverage?: Resource[] }
export interface Graph { schemaVersion?: string; clusterId: string; revision: number; nodes: Resource[]; edges: Resource[]; coverage?: Resource[]; generatedAt?: string }
export interface Endpoints { snapshot?: string; topology?: string; analyze?: string; exportSnapshot?: string; exportSummary?: string }
export interface BootConfig { mode: 'offline' | 'live'; endpoints?: Endpoints }
export interface Publication { revision: number; snapshot: Snapshot; graph?: Graph; advisor?: Record<string, any>; eksProjection?: Record<string, any>; sources?: Record<string, Resource>; events?: Resource[] }
export interface TopologyUpdate { schemaVersion: string; kind: 'full' | 'delta' | 'unchanged'; clusterId: string; revision: number; baseRevision?: number; graph?: Graph; upsertNodes?: Resource[]; upsertEdges?: Resource[]; deleteNodeIds?: string[]; deleteEdgeIds?: string[]; coverage?: Resource[]; generatedAt?: string }
export interface AnalysisRequest { revision: number; scope: { pageId: string; namespace: string; resourceType: string; selectedRefs: Resource[] } }
