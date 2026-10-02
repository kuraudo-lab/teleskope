import { expect, it, vi } from 'vitest'
import { createReportStore } from './report-store'
it('does not publish an in-flight response after pause or disposal', async () => {
 let resolve!: (response: Response) => void
 const request = vi.fn(() => new Promise<Response>(r => resolve=r))
 const store = createReportStore({mode:'live',endpoints:{snapshot:'/snapshot'}},{},request)
 const refresh = store.refresh(); store.pause(true)
 resolve(Response.json({revision:1,snapshot:{}})); await refresh
 expect(store.state.revision).toBe(-1)
 store.dispose()
})
it('rejects analysis completed after its original revision', async () => {
 let resolve!: (response: Response) => void
 const store = createReportStore({mode:'live',endpoints:{analyze:'/analyze'}},{revision:1},vi.fn(() => new Promise<Response>(r => resolve=r)))
 const result = store.analyze({revision:1,scope:{pageId:'workloads',namespace:'app',resourceType:'workloads',selectedRefs:[]}})
 store.apply({revision:2,snapshot:{}}); resolve(Response.json({analysis:{summary:'old'}}))
 await expect(result).rejects.toThrow('snapshot changed')
 store.dispose()
})
