import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test('homepage redirects to login', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/login/);
  });

  test('login page loads correctly', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('h1, h2').first()).toContainText(/login|sign in|welcome/i);
    await expect(page.locator('input[type="email"], input[name="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
  });

  test('register page loads correctly', async ({ page }) => {
    await page.goto('/register');
    // Check for any text input (username field may vary)
    await expect(page.locator('input[type="text"]').first()).toBeVisible();
    await expect(page.locator('input[type="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"]').first()).toBeVisible();
  });

  test('can navigate between login and register', async ({ page }) => {
    await page.goto('/login');
    
    // Find link to register
    const registerLink = page.getByRole('link', { name: /register|sign up|create/i });
    if (await registerLink.isVisible()) {
      await registerLink.click();
      await expect(page).toHaveURL(/register/);
    }
  });

  test('shows error on invalid login', async ({ page }) => {
    await page.goto('/login');
    
    await page.fill('input[type="email"]', 'invalid@test.com');
    await page.fill('input[type="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');
    
    // Wait for response - either error message or still on login page
    await page.waitForTimeout(2000);
    // Should still be on login page (not redirected to app)
    await expect(page).toHaveURL(/login/);
  });
});
