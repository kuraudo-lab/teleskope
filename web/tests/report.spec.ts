import { test, expect } from '@playwright/test'

test('recorded report navigation, workload search, details and theme', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.goto('/')
  await expect(page.locator('#title')).not.toHaveText('Waiting for first snapshot')
  await page.getByRole('button', { name: 'Kubernetes', exact: true }).click()
  await page.locator('[data-target="workloads"]').click()
  await page.locator('#workloadSearch').fill('aws-node')
  await page.locator('#workloadSearchResults button').first().click()
  await expect(page.locator('#drawer')).toHaveClass(/open/)
  await expect(page.locator('#drawerBody')).toContainText('Images')
  const theme = await page.locator('html').getAttribute('data-theme')
  await page.locator('#themeToggle').click()
  await expect(page.locator('html')).not.toHaveAttribute('data-theme', theme || '')
  expect(errors).toEqual([])
})

test('recorded API publishes replay data and export without credentials', async ({ request }) => {
  const response = await request.get('/api/snapshot')
  expect(response.ok()).toBeTruthy()
  const data = await response.json()
  expect(data.snapshot.schemaVersion).toBe('teleskope.io/snapshot/v1alpha1')
  expect(data.graph.nodes.length).toBeGreaterThan(0)
  const snapshot = await request.get('/api/export/snapshot.json')
  expect(snapshot.ok()).toBeTruthy()
  expect((await snapshot.json()).kubernetes.workloads.length).toBeGreaterThan(0)
})

test('every resource page renders and topology survives remount and URL state', async ({page},testInfo) => {
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message))
 await page.goto('/')
 await expect(page.locator('#topologySvg')).toBeVisible()
 await page.screenshot({path:testInfo.outputPath('overview.png')})
 for(const [group,ids] of [['Kubernetes',['nodes','images','crds','workloads','network','security','policies','storage']],['EKS',['eks','eks-upgrades','eks-compute','eks-network','eks-security','eks-addons']]] as const) {
  for(const id of ids){await page.getByRole('button',{name:group,exact:true}).click();await page.locator(`[data-target="${id}"]`).click();await expect(page.locator(`[data-section="${id}"] table`).first()).toBeVisible()}
 }
 await page.locator('[data-target="advisor"]').click()
 await expect(page.locator('#advisor-capabilities')).toBeVisible()
 await page.locator('[data-target="overview"]').click()
 await expect(page.locator('#topologySvg')).toBeVisible()
 await page.locator('#topology-search').fill('aws-node')
 await expect(page).toHaveURL(/topologyQuery=aws-node/)
 await page.locator('[data-topology-mode="table"]').click()
 await expect(page.locator('#topology table')).toBeVisible()
 await page.reload()
 await expect(page.locator('#topology-search')).toHaveValue('aws-node')
 await expect(page.locator('#topology table')).toBeVisible()
 expect(errors).toEqual([])
})

test('namespace absent from refreshed inventory remains selected', async ({page}) => {
 await page.goto('/?namespace=removed-namespace&page=workloads')
 await expect(page.locator('#focus')).toHaveValue('removed-namespace')
 await expect(page.locator('#workloadsTable')).toContainText('No matching data')
 await page.waitForTimeout(5200)
 await expect(page.locator('#focus')).toHaveValue('removed-namespace')
})

test('configured analysis uses the selected page scope and survives page navigation',async({page})=>{
 await page.route('http://127.0.0.1:18943/',async route=>{
  const response=await route.fetch();await route.fulfill({response,body:(await response.text()).replace('"analysisEnabled":false','"analysisEnabled":true')})
 })
 let scope:any
 await page.route('**/api/analyze',async route=>{scope=route.request().postDataJSON();await route.fulfill({json:{analysis:{summary:'Scoped result',model:'fixture',sections:[{items:[{summary:'Evidence fact',basis:'observed',recommendation:'Review configuration'}]}],limitations:['Fixture analysis']}}})})
 await page.goto('/')
 await page.getByRole('button',{name:'Kubernetes',exact:true}).click();await page.locator('[data-target="workloads"]').click()
 await page.locator('[data-analyze-page="workloads"]').click()
 await expect(page.locator('#analysisBody-workloads')).toContainText('Scoped result')
 expect(scope.scope.pageId).toBe('workloads');expect(scope.revision).toBeGreaterThan(0)
 await page.locator('[data-target="advisor"]').click()
 await page.getByRole('button',{name:'Kubernetes',exact:true}).click();await page.locator('[data-target="workloads"]').click()
 await expect(page.locator('#analysisBody-workloads')).toContainText('Scoped result')
})
