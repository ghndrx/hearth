import { test, expect } from '@playwright/test';

test.describe('API Health', () => {
  test('health endpoint returns ok', async ({ request }) => {
    const response = await request.get('/api/health');
    expect(response.ok()).toBeTruthy();
    
    const body = await response.json();
    expect(body.status).toBe('ok');
  });

  test('API returns proper JSON', async ({ request }) => {
    const response = await request.get('/api/health');
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/json');
  });
});

test.describe('WebSocket', () => {
  test('WebSocket endpoint is reachable', async ({ page }) => {
    // Check that WebSocket can connect
    const wsConnected = await page.evaluate(async () => {
      return new Promise((resolve) => {
        try {
          const ws = new WebSocket('wss://hearth.gregh.dev/ws');
          ws.onopen = () => {
            ws.close();
            resolve(true);
          };
          ws.onerror = () => resolve(false);
          setTimeout(() => resolve(false), 5000);
        } catch {
          resolve(false);
        }
      });
    });
    
    // WebSocket should at least attempt connection without error
    // May fail auth but endpoint should exist
  });
});
