import { test, expect } from '@playwright/test'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

test('same production bundle works in offline HTML without network', async ({page}) => {
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message))
 await page.route('http://**',route=>route.abort());await page.route('https://**',route=>route.abort())
 await page.goto(pathToFileURL(resolve('../docs/demo/source/index.html')).href)
 await expect(page.locator('#topologySvg')).toBeVisible()
 await page.getByRole('button',{name:'Kubernetes',exact:true}).click()
 await page.locator('[data-target="workloads"]').click()
 await expect(page.locator('#workloadsTable tbody tr').first()).toBeVisible()
 const download=page.waitForEvent('download');await page.locator('#exportReport').click();await page.locator('[data-export-format="json"]').click();expect((await download).suggestedFilename()).toMatch(/\.json$/)
 expect(errors).toEqual([])
})

test('Hub filters, search, exports and isolated cluster drilldown', async ({page}) => {
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message))
 await page.goto('http://127.0.0.1:8091')
 await expect(page.locator('#clusters tr')).toHaveCount(2)
 await page.locator('[data-target="search"]').click()
 await page.locator('#searchInput').fill('checkout')
 await page.locator('#runSearch').click()
 await expect(page.locator('#searchStatus')).toContainText('matches')
 await expect(page.locator('#searchResults tr').first()).toBeVisible()
 await page.locator('[data-target="fleet"]').click()
 const download=page.waitForEvent('download');await page.locator('#exportJSON').click();expect((await download).suggestedFilename()).toBe('teleskope-fleet.json')
 await page.locator('#clusters a').first().click()
 await expect(page.locator('#title')).toContainText('demo')
 await expect(page.locator('#topologySvg')).toBeVisible()
 expect(new URL(page.url()).searchParams.get('id')).toBeTruthy()
 expect(errors).toEqual([])
})

for(const profile of ['small','medium','large'])test(`topology ${profile} production render and interaction`, async ({page},testInfo) => {
 test.setTimeout(60000)
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message))
 await page.goto(pathToFileURL(resolve(`.fixtures/${profile}.html`)).href)
 await page.waitForFunction(()=>!!(window as any).__teleskopeScaleEvidence)
 const evidence=await page.evaluate(()=>(window as any).__teleskopeScaleEvidence)
 await testInfo.attach('scale-evidence',{body:JSON.stringify(evidence,null,2),contentType:'application/json'})
 expect(evidence.rendered.mode).toBe('graph')
 const budget=profile==='small'?500:profile==='medium'?1500:4000
 expect(evidence.renderMs.median).toBeLessThanOrEqual(budget)
 await page.locator('#topology .node').first().focus()
 await page.keyboard.press('Enter')
 await expect(page.locator('#topology .node[aria-pressed="true"]')).toHaveCount(1)
 const before=await page.locator('#topologyViewport').getAttribute('transform')
 await page.locator('#topologyZoomIn').click()
 await expect(page.locator('#topologyViewport')).not.toHaveAttribute('transform',before!)
 expect(errors).toEqual([])
})

test('live topology accepts delta, resyncs revision gaps and preserves selection',async({page})=>{
 await page.goto('http://127.0.0.1:18945')
 await page.waitForFunction(()=>(window as any).__teleskopeLiveEvidence?.phase==='complete'||(window as any).__teleskopeLiveEvidence?.phase==='failed')
 const result=await page.evaluate(()=>(window as any).__teleskopeLiveEvidence)
 expect(result.phase,result.error).toBe('complete')
 expect(result.delta.accepted).toBe(true)
 expect(result.revisionGap.fullReset).toBe(true)
 expect(result.selectionStable).toBe(true)
 expect(result.before.position).not.toBeNull()
 expect(result.layoutStable).toBe(true)
})
