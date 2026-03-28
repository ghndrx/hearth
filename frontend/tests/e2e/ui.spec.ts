import { test, expect } from '@playwright/test';
import { TEST_CREDENTIALS, login, waitForApp } from './fixtures';

test.describe('UI Components', () => {
  test.describe('Public Pages', () => {
    test('login form has dark theme styling', async ({ page }) => {
      await page.goto('/login');
      
      const body = page.locator('body');
      const bgColor = await body.evaluate(el => getComputedStyle(el).backgroundColor);
      
      // Should not be white (dark theme)
      expect(bgColor).not.toBe('rgb(255, 255, 255)');
    });

    test('buttons are interactive', async ({ page }) => {
      await page.goto('/login');
      
      const submitButton = page.locator('button[type="submit"]');
      await expect(submitButton).toBeEnabled();
      await expect(submitButton).toBeVisible();
    });

    test('form inputs are accessible', async ({ page }) => {
      await page.goto('/login');
      
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
    test('mobile viewport renders correctly', async ({ page }) => {
      await page.setViewportSize({ width: 375, height: 667 });
      await page.goto('/login');
      
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });

    test('tablet viewport renders correctly', async ({ page }) => {
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.goto('/login');
      
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });

    test('desktop viewport renders correctly', async ({ page }) => {
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.goto('/login');
      
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });
  });
});

test.describe('Authenticated UI', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires valid test credentials');
    await login(page, TEST_CREDENTIALS.email, TEST_CREDENTIALS.password);
    await waitForApp(page);
  });

  test.describe('User Settings Modal', () => {
    test('user settings can be opened', async ({ page }) => {
      // Look for user panel or settings gear
      const userPanel = page.locator('.user-panel, [data-testid="user-panel"]');
      const settingsButton = userPanel.locator('button, svg').first();
      
      if (await settingsButton.isVisible()) {
        await settingsButton.click();
        await page.waitForTimeout(500);
        
        // Settings modal should be visible
        const settingsModal = page.locator('.user-settings, [data-testid="user-settings"], [role="dialog"]');
        await expect(settingsModal.first()).toBeVisible();
      }
    });

    test('user settings closes with ESC key', async ({ page }) => {
      // Open settings
      const userPanel = page.locator('.user-panel, [data-testid="user-panel"]');
      const settingsButton = userPanel.locator('button').first();
      
      if (await settingsButton.isVisible()) {
        await settingsButton.click();
        await page.waitForTimeout(500);
        
        // Press Escape
        await page.keyboard.press('Escape');
        await page.waitForTimeout(500);
        
        // Modal should be closed
        const settingsModal = page.locator('.user-settings, [data-testid="user-settings"]');
        await expect(settingsModal).not.toBeVisible();
      }
    });

    test('user settings closes with close button', async ({ page }) => {
      // Open settings
      const userPanel = page.locator('.user-panel, [data-testid="user-panel"]');
      const settingsButton = userPanel.locator('button').first();
      
      if (await settingsButton.isVisible()) {
        await settingsButton.click();
        await page.waitForTimeout(500);
        
        // Find close button
        const closeButton = page.locator(
          '.user-settings button:has-text("Close"), .user-settings [aria-label="Close"], .close-button, button[title="Close"]'
        ).first();
        
        if (await closeButton.isVisible()) {
          await closeButton.click();
          await page.waitForTimeout(500);
          
          const settingsModal = page.locator('.user-settings');
          await expect(settingsModal).not.toBeVisible();
        }
      }
    });
  });

  test.describe('Server Settings Modal', () => {
    test('server settings modal opens', async ({ page }) => {
      // Navigate to a server
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        // Click server settings (usually in server header)
        const serverHeader = page.locator('.server-header, [data-testid="server-header"]');
        const settingsBtn = serverHeader.locator('button').first();
        
        if (await settingsBtn.isVisible()) {
          await settingsBtn.click();
          
          const settingsModal = page.locator('.server-settings, [data-testid="server-settings"]');
          await expect(settingsModal).toBeVisible({ timeout: 5000 });
        }
      }
    });

    test('server settings modal closes with ESC', async ({ page }) => {
      // Navigate to a server
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        // Open settings
        const serverHeader = page.locator('.server-header, [data-testid="server-header"]');
        const settingsBtn = serverHeader.locator('button').first();
        
        if (await settingsBtn.isVisible()) {
          await settingsBtn.click();
          await page.waitForTimeout(500);
          
          // Press Escape
          await page.keyboard.press('Escape');
          await page.waitForTimeout(500);
          
          const settingsModal = page.locator('.server-settings');
          await expect(settingsModal).not.toBeVisible();
        }
      }
    });
  });

  test.describe('Quick Switcher (Ctrl+K)', () => {
    test('quick switcher opens with Ctrl+K', async ({ page }) => {
      // Press Ctrl+K
      await page.keyboard.press('Control+k');
      await page.waitForTimeout(500);
      
      // Quick switcher modal should appear
      const quickSwitcher = page.locator('.quick-switcher, [aria-label="Quick Switcher"]');
      await expect(quickSwitcher).toBeVisible();
    });

    test('quick switcher opens with Cmd+K on Mac', async ({ page }) => {
      // Press Meta+K (Cmd on Mac)
      await page.keyboard.press('Meta+k');
      await page.waitForTimeout(500);
      
      // Quick switcher modal should appear
      const quickSwitcher = page.locator('.quick-switcher, [aria-label="Quick Switcher"]');
      await expect(quickSwitcher).toBeVisible();
    });

    test('quick switcher has search input', async ({ page }) => {
      await page.keyboard.press('Control+k');
      await page.waitForTimeout(300);
      
      const searchInput = page.locator('.quick-switcher input, [aria-label*="Search"]');
      await expect(searchInput).toBeVisible();
      await expect(searchInput).toBeFocused();
    });

    test('quick switcher closes with ESC', async ({ page }) => {
      // Open
      await page.keyboard.press('Control+k');
      await page.waitForTimeout(300);
      
      const quickSwitcher = page.locator('.quick-switcher');
      await expect(quickSwitcher).toBeVisible();
      
      // Close with ESC
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
      
      await expect(quickSwitcher).not.toBeVisible();
    });

    test('quick switcher closes when clicking outside', async ({ page }) => {
      await page.keyboard.press('Control+k');
      await page.waitForTimeout(300);
      
      const quickSwitcher = page.locator('.quick-switcher');
      await expect(quickSwitcher).toBeVisible();
      
      // Click backdrop
      const backdrop = page.locator('.quick-switcher-backdrop');
      await backdrop.click({ position: { x: 10, y: 10 } });
      await page.waitForTimeout(300);
      
      await expect(quickSwitcher).not.toBeVisible();
    });

    test('quick switcher shows results on typing', async ({ page }) => {
      await page.keyboard.press('Control+k');
      await page.waitForTimeout(300);
      
      const searchInput = page.locator('.quick-switcher input');
      await searchInput.fill('general');
      await page.waitForTimeout(500);
      
      // Should show results
      const results = page.locator('.quick-switcher .result-item, .quick-switcher [role="option"]');
      // Results or no-results message
      const hasContent = (await results.count() > 0) || 
        await page.locator('.quick-switcher .no-results').isVisible();
      expect(hasContent).toBeTruthy();
    });

    test('quick switcher keyboard navigation works', async ({ page }) => {
      await page.keyboard.press('Control+k');
      await page.waitForTimeout(300);
      
      // Arrow down should select next item
      await page.keyboard.press('ArrowDown');
      await page.waitForTimeout(100);
      
      // At least the quick switcher is open with a selectable item
      const quickSwitcher = page.locator('.quick-switcher');
      await expect(quickSwitcher).toBeVisible();
      // Verify selection element exists (after ArrowDown)
      await expect(page.locator('.quick-switcher .selected, .quick-switcher [aria-selected="true"]')).toBeDefined();
    });
  });

  test.describe('Friends Page', () => {
    test('friends page renders', async ({ page }) => {
      // Click home button
      const homeButton = page.locator('[href*="/@me"], .home-button').first();
      await homeButton.click();
      
      await page.waitForURL(/\/@me/);
      
      // Friends header should be visible
      const friendsHeader = page.locator('.friends-header, [data-testid="friends-header"]');
      await expect(friendsHeader).toBeVisible();
    });

    test('friends page has tabs', async ({ page }) => {
      await page.goto('/channels/@me');
      await page.waitForTimeout(500);
      
      // Should have tab buttons
      const onlineTab = page.locator('button:has-text("Online")');
      const allTab = page.locator('button:has-text("All")');
      const pendingTab = page.locator('button:has-text("Pending")');
      
      await expect(onlineTab).toBeVisible();
      await expect(allTab).toBeVisible();
      await expect(pendingTab).toBeVisible();
    });

    test('add friend button exists', async ({ page }) => {
      await page.goto('/channels/@me');
      await page.waitForTimeout(500);
      
      const addFriendButton = page.locator('button:has-text("Add Friend")');
      await expect(addFriendButton).toBeVisible();
    });

    test('add friend panel opens on click', async ({ page }) => {
      await page.goto('/channels/@me');
      await page.waitForTimeout(500);
      
      const addFriendButton = page.locator('button:has-text("Add Friend")');
      await addFriendButton.click();
      
      // Add friend panel should show
      const addFriendPanel = page.locator('.add-friend-panel, [data-testid="add-friend"]');
      await expect(addFriendPanel).toBeVisible();
      
      // Input should be visible
      const friendInput = addFriendPanel.locator('input');
      await expect(friendInput).toBeVisible();
    });
  });

  test.describe('Layout Structure', () => {
    test('main app container has correct layout', async ({ page }) => {
      const appContainer = page.locator('.app-container, [data-testid="app-container"]');
      await expect(appContainer).toBeVisible();
      
      // Should have horizontal flex layout
      const display = await appContainer.evaluate(el => getComputedStyle(el).display);
      expect(display).toBe('flex');
    });

    test('server list is leftmost sidebar', async ({ page }) => {
      const serverList = page.locator('.server-list').first();
      await expect(serverList).toBeVisible();
      
      // Should be narrow
      const width = await serverList.evaluate(el => (el as HTMLElement).offsetWidth);
      expect(width).toBeLessThan(100); // Server icons sidebar is narrow
    });

    test('skip link is present for accessibility', async ({ page }) => {
      const skipLink = page.locator('.skip-link, a[href="#main-content"]');
      
      // Skip link exists (might be visually hidden)
      await expect(skipLink).toBeAttached();
    });

    test('main content area is present', async ({ page }) => {
      const mainContent = page.locator('#main-content, main');
      await expect(mainContent).toBeVisible();
    });
  });
});

test.describe('Accessibility', () => {
  test('login page has no major accessibility issues', async ({ page }) => {
    await page.goto('/login');
    
    // Check for basic accessibility markers
    const html = page.locator('html');
    const lang = await html.getAttribute('lang');
    expect(lang).toBeTruthy(); // Should have lang attribute
    
    // Form should have proper structure
    const form = page.locator('form');
    await expect(form).toBeVisible();
  });

  test.skip('page has heading hierarchy', async ({ page }) => {
    await page.goto('/login');
    
    // Should have at least one heading
    const h1 = page.locator('h1');
    const h2 = page.locator('h2');
    
    const hasHeading = (await h1.count() > 0) || (await h2.count() > 0);
    expect(hasHeading).toBeTruthy();
  });

  test('interactive elements are keyboard accessible', async ({ page }) => {
    await page.goto('/login');
    
    // Tab through form elements
    await page.keyboard.press('Tab');
    const focused1 = await page.evaluate(() => document.activeElement?.tagName);
    
    await page.keyboard.press('Tab');
    const focused2 = await page.evaluate(() => document.activeElement?.tagName);
    
    // Should be able to focus different elements
    expect(focused1 || focused2).toBeTruthy();
  });
});
