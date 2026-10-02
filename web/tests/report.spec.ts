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
