import { test, expect } from '@playwright/test';

interface Message {
  id: string;
  content: string;
  author_id: string;
  channel_id: string;
  created_at: string;
}

/**
 * End-to-end test for complete message flow
 * This test creates a user, server, channel, sends a message, and verifies it appears
 */
test.describe('Message Flow E2E', () => {
  const testUser = {
    username: `testuser_${Date.now()}`,
    email: `test_${Date.now()}@hearth.local`,
    password: 'TestPassword123!',
  };

  let authToken: string;
  let serverId: string;
  let channelId: string;

  test.beforeAll(async ({ request }) => {
    // Register a test user via API
    const registerResponse = await request.post('/api/v1/auth/register', {
      data: testUser,
    });
    
    if (registerResponse.ok()) {
      const data = await registerResponse.json();
      authToken = data.token;
    } else {
      // User might already exist, try login
      const loginResponse = await request.post('/api/v1/auth/login', {
        data: {
          email: testUser.email,
          password: testUser.password,
        },
      });
      
      if (loginResponse.ok()) {
        const data = await loginResponse.json();
        authToken = data.token;
      }
    }

    // Create a test server
    if (authToken) {
      const serverResponse = await request.post('/api/v1/servers', {
        headers: { Authorization: `Bearer ${authToken}` },
        data: { name: `Test Server ${Date.now()}` },
      });
      
      if (serverResponse.ok()) {
        const serverData = await serverResponse.json();
        serverId = serverData.id;
        
        // Get the default channel (created with server)
        const channelsResponse = await request.get(`/api/v1/servers/${serverId}/channels`, {
          headers: { Authorization: `Bearer ${authToken}` },
        });
        
        if (channelsResponse.ok()) {
          const channels = await channelsResponse.json();
          if (channels.length > 0) {
            channelId = channels[0].id;
          }
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    // Cleanup: delete test server
    if (authToken && serverId) {
      await request.delete(`/api/v1/servers/${serverId}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      });
    }
  });

  test('complete message send and receive flow', async ({ page }) => {
    test.skip(!authToken, 'Could not authenticate test user');
    test.skip(!serverId || !channelId, 'Could not create test server/channel');

    // Login via UI
    await page.goto('/login');
    await page.fill('input[type="email"], input[name="email"]', testUser.email);
    await page.fill('input[type="password"], input[name="password"]', testUser.password);
    await page.click('button[type="submit"]');

    // Wait for redirect to channels
    await page.waitForURL(/\/channels/, { timeout: 10000 });

    // Navigate to the test server
    await page.goto(`/channels/${serverId}/${channelId}`);
    await page.waitForLoadState('networkidle');

    // Wait for message input to be ready
    const messageInput = page.locator('textarea, [contenteditable="true"], input[placeholder*="message"]').first();
    await expect(messageInput).toBeVisible({ timeout: 10000 });

    // Send a unique test message
    const testMessage = `E2E Test Message - ${Date.now()}`;
    await messageInput.fill(testMessage);
    await page.keyboard.press('Enter');

    // Wait for message to be sent and appear
    await page.waitForTimeout(2000);

    // Verify message appears in the chat
    const messageInChat = page.locator(`.message-content:has-text("${testMessage}"), [data-testid="message"]:has-text("${testMessage}")`);
    await expect(messageInChat).toBeVisible({ timeout: 10000 });

    // Verify message input was cleared
    await expect(messageInput).toHaveValue('');
  });

  test('message appears via WebSocket in real-time', async ({ page, context }) => {
    test.skip(!authToken, 'Could not authenticate test user');
    test.skip(!serverId || !channelId, 'Could not create test server/channel');

    // Open two browser tabs to simulate two users
    const page2 = await context.newPage();

    // Login in both tabs
    for (const p of [page, page2]) {
      await p.goto('/login');
      await p.fill('input[type="email"], input[name="email"]', testUser.email);
      await p.fill('input[type="password"], input[name="password"]', testUser.password);
      await p.click('button[type="submit"]');
      await p.waitForURL(/\/channels/, { timeout: 10000 });
      await p.goto(`/channels/${serverId}/${channelId}`);
      await p.waitForLoadState('networkidle');
    }

    // Send message from page 1
    const testMessage = `WebSocket Test - ${Date.now()}`;
    const messageInput = page.locator('textarea, [contenteditable="true"]').first();
    await messageInput.fill(testMessage);
    await page.keyboard.press('Enter');

    // Verify message appears in page 2 (via WebSocket)
    const messageInPage2 = page2.locator(`text="${testMessage}"`);
    await expect(messageInPage2).toBeVisible({ timeout: 10000 });

    await page2.close();
  });

  test('message persists after page reload', async ({ page }) => {
    test.skip(!authToken, 'Could not authenticate test user');
    test.skip(!serverId || !channelId, 'Could not create test server/channel');

    // Login
    await page.goto('/login');
    await page.fill('input[type="email"], input[name="email"]', testUser.email);
    await page.fill('input[type="password"], input[name="password"]', testUser.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/channels/, { timeout: 10000 });

    // Navigate to channel
    await page.goto(`/channels/${serverId}/${channelId}`);
    await page.waitForLoadState('networkidle');

    // Send message
    const testMessage = `Persistence Test - ${Date.now()}`;
    const messageInput = page.locator('textarea, [contenteditable="true"]').first();
    await messageInput.fill(testMessage);
    await page.keyboard.press('Enter');
    await page.waitForTimeout(2000);

    // Reload page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Verify message still appears
    const messageAfterReload = page.locator(`text="${testMessage}"`);
    await expect(messageAfterReload).toBeVisible({ timeout: 10000 });
  });

  test('API: send message directly and verify', async ({ request }) => {
    test.skip(!authToken, 'Could not authenticate test user');
    test.skip(!channelId, 'Could not create test channel');

    const testMessage = `API Test Message - ${Date.now()}`;

    // Send message via API
    const sendResponse = await request.post(`/api/v1/channels/${channelId}/messages`, {
      headers: { Authorization: `Bearer ${authToken}` },
      data: { content: testMessage },
    });

    expect(sendResponse.ok()).toBeTruthy();
    const sentMessage = await sendResponse.json();
    expect(sentMessage.content).toBe(testMessage);
    expect(sentMessage.id).toBeTruthy();

    // Fetch messages and verify it's there
    const messagesResponse = await request.get(`/api/v1/channels/${channelId}/messages`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });

    expect(messagesResponse.ok()).toBeTruthy();
    const messages = await messagesResponse.json();
    
    const foundMessage = messages.find((m: Message) => m.content === testMessage);
    expect(foundMessage).toBeTruthy();
  });
});
