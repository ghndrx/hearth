import { test as base, expect } from '@playwright/test';
import type { Page } from '@playwright/test';

// Extend test with custom fixtures
export const test = base.extend<{
  authenticatedPage: Page;
}>({
  // Custom fixture for authenticated state (if storage state exists)
  authenticatedPage: async ({ page }, use) => {
    // If we have stored auth state, it will be loaded automatically
    await use(page);
  },
});

export { expect };

// Test credentials for E2E tests (use environment variables in CI)
export const TEST_CREDENTIALS = {
  email: process.env.TEST_USER_EMAIL || 'test@example.com',
  password: process.env.TEST_USER_PASSWORD || 'testpassword123',
  username: process.env.TEST_USER_USERNAME || 'testuser',
};

// Common selectors
export const SELECTORS = {
  // Auth
  emailInput: 'input[type="email"], input[name="email"]',
  passwordInput: 'input[type="password"]',
  submitButton: 'button[type="submit"]',
  
  // Navigation
  serverList: '[data-testid="server-list"], .server-list',
  channelList: '[data-testid="channel-list"], .channel-list',
  memberList: '[data-testid="member-list"], .member-list',
  
  // Modals
  userSettings: '[data-testid="user-settings"], .user-settings',
  serverSettings: '[data-testid="server-settings"], .server-settings',
  quickSwitcher: '.quick-switcher, [aria-label="Quick Switcher"]',
  
  // Messages
  messageInput: '[data-testid="message-input"], textarea, .message-input input',
  messageList: '[data-testid="message-list"], .message-list, .messages',
  message: '[data-testid="message"], .message',
};

// Helper functions
export async function login(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.fill(SELECTORS.emailInput, email);
  await page.fill(SELECTORS.passwordInput, password);
  await page.click(SELECTORS.submitButton);
  // Wait for redirect away from login
  await page.waitForURL(/(?!.*login).*/, { timeout: 10000 });
}

export async function waitForApp(page: Page) {
  // Wait for the main app to load (server list visible)
  await page.waitForSelector('.app-container, [data-testid="app-container"]', { timeout: 15000 });
}
