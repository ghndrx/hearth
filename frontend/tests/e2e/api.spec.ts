import { test, expect } from '@playwright/test';

test.describe('API Health', () => {
  test.skip('health endpoint returns ok', async ({ request }) => {
    // Skipped: API tests need backend to be running
    const response = await request.get('/api/health');
    expect(response.ok()).toBeTruthy();
  });

  test.skip('API returns proper JSON', async ({ request }) => {
    // Skipped: API tests need backend to be running
    const response = await request.get('/api/health');
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');
  });
});

test.describe('WebSocket', () => {
  test.skip('WebSocket endpoint is reachable', async ({ page: _page }) => {
    // Skipped: WebSocket tests need auth
  });
});
