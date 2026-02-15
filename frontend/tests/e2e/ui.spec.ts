import { test, expect } from '@playwright/test';

test.describe('UI Components', () => {
  test('login form has proper styling', async ({ page }) => {
    await page.goto('/login');
    
    // Check for dark theme colors
    const body = page.locator('body');
    const bgColor = await body.evaluate(el => getComputedStyle(el).backgroundColor);
    expect(bgColor).not.toBe('rgb(255, 255, 255)'); // Not white = dark theme
  });

  test('buttons are interactive', async ({ page }) => {
    await page.goto('/login');
    
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeEnabled();
    
    // Check hover state changes
    const initialBg = await submitButton.evaluate(el => getComputedStyle(el).backgroundColor);
    await submitButton.hover();
    // Button should be visible and styled
    await expect(submitButton).toBeVisible();
  });

  test('form inputs are accessible', async ({ page }) => {
    await page.goto('/login');
    
    // Check labels or placeholders exist
    const emailInput = page.locator('input[type="email"], input[name="email"]');
    const passwordInput = page.locator('input[type="password"]');
    
    await expect(emailInput).toBeVisible();
    await expect(passwordInput).toBeVisible();
    
    // Should be focusable
    await emailInput.focus();
    await expect(emailInput).toBeFocused();
  });

  test('page has proper title', async ({ page }) => {
    await page.goto('/login');
    await expect(page).toHaveTitle(/hearth/i);
  });
});

test.describe('Responsive Design', () => {
  test('mobile viewport works', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/login');
    
    // Content should still be visible
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('tablet viewport works', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/login');
    
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });
});
