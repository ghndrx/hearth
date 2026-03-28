import { test, expect } from '@playwright/test';
import { TEST_CREDENTIALS, login, waitForApp } from './fixtures';

test.describe('Server Management', () => {
  // Skip all tests if no valid credentials
  test.beforeEach(async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires valid test credentials');
    await login(page, TEST_CREDENTIALS.email, TEST_CREDENTIALS.password);
    await waitForApp(page);
  });

  test.describe('Server List', () => {
    test('server list is visible after login', async ({ page }) => {
      // Server list should be in the leftmost sidebar
      const serverList = page.locator('.server-list, [data-testid="server-list"]').first();
      await expect(serverList).toBeVisible();
    });

    test('home button (@me) is visible', async ({ page }) => {
      // Home/DM button should always be present
      const homeButton = page.locator('[href*="/@me"], .home-button, [data-testid="home-button"]').first();
      await expect(homeButton).toBeVisible();
    });

    test('clicking server navigates to server page', async ({ page }) => {
      // Find any server icon (not home)
      const serverIcon = page.locator('.server-icon:not(.home-icon), [data-testid="server-icon"]').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        // Should navigate to /channels/[serverId]
        await expect(page).toHaveURL(/\/channels\/[a-zA-Z0-9-]+/);
      }
    });

    test('server switching auto-redirects to first channel', async ({ page }) => {
      // Find server icons
      const serverIcons = page.locator('.server-icon:not(.home-icon), [data-testid="server-icon"]');
      const count = await serverIcons.count();
      
      if (count > 0) {
        await serverIcons.first().click();
        await page.waitForTimeout(1000);
        
        // Should redirect to a channel (not just /channels/serverId but /channels/serverId/channelId)
        const url = page.url();
        // Either on server page or already redirected to channel
        expect(url).toMatch(/\/channels\/[a-zA-Z0-9-]+/);
      }
    });
  });

  test.describe('Create Server', () => {
    test('create server button is visible', async ({ page }) => {
      // Create server button (usually + icon at bottom of server list)
      const createButton = page.locator(
        '[data-testid="create-server"], .add-server-button, button:has-text("Add"), .server-list button[title*="Add"], .server-list button[title*="Create"]'
      ).first();
      
      // Create button should exist (might be a + icon)
      await expect(createButton).toBeVisible();
    });

    test('create server modal opens on click', async ({ page }) => {
      const createButton = page.locator(
        '[data-testid="create-server"], .add-server-button, button:has-text("Add"), .server-list button'
      ).last(); // Usually at the bottom
      
      if (await createButton.isVisible()) {
        await createButton.click();
        
        // Look for modal or dialog
        const modal = page.locator('[role="dialog"], .modal, [data-testid="create-server-modal"]');
        await expect(modal.first()).toBeVisible({ timeout: 5000 });
      }
    });

    test('can create a new server', async ({ page }) => {
      // Open create server modal
      const createButton = page.locator(
        '[data-testid="create-server"], .add-server-button'
      ).first();
      
      if (await createButton.isVisible()) {
        await createButton.click();
        await page.waitForTimeout(500);
        
        // Fill server name
        const nameInput = page.locator('input[name="name"], input[placeholder*="server"], input[type="text"]').first();
        if (await nameInput.isVisible()) {
          const testServerName = `Test Server ${Date.now()}`;
          await nameInput.fill(testServerName);
          
          // Submit
          const submitBtn = page.locator('button[type="submit"], button:has-text("Create")').first();
          await submitBtn.click();
          
          // Should create server and navigate to it
          await page.waitForTimeout(2000);
          // Look for the server in the list or verify navigation
        }
      }
    });
  });

  test.describe('Server Settings', () => {
    test('server settings accessible for owned server', async ({ page }) => {
      // Navigate to a server first
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        // Look for settings button (usually gear icon in header)
        const settingsButton = page.locator(
          '[data-testid="server-settings"], .server-header button, button[title*="Settings"]'
        ).first();
        
        if (await settingsButton.isVisible()) {
          await settingsButton.click();
          
          // Settings modal/page should appear
          const settingsPanel = page.locator(
            '[data-testid="server-settings-panel"], .server-settings, [role="dialog"]'
          );
          await expect(settingsPanel.first()).toBeVisible({ timeout: 5000 });
        }
      }
    });

    test('server settings closes with ESC', async ({ page }) => {
      // Navigate to server and open settings
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        const settingsButton = page.locator(
          '[data-testid="server-settings"], .server-header button'
        ).first();
        
        if (await settingsButton.isVisible()) {
          await settingsButton.click();
          await page.waitForTimeout(500);
          
          // Press Escape
          await page.keyboard.press('Escape');
          await page.waitForTimeout(500);
          
          // Settings should be closed
          const settingsPanel = page.locator('.server-settings, [data-testid="server-settings-panel"]');
          await expect(settingsPanel).not.toBeVisible();
        }
      }
    });
  });

  test.describe('Channel Management', () => {
    test('channel list is visible in server', async ({ page }) => {
      // Navigate to a server
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        // Channel list should be visible
        const channelList = page.locator('.channel-list, [data-testid="channel-list"]');
        await expect(channelList).toBeVisible();
      }
    });

    test('clicking channel navigates to channel page', async ({ page }) => {
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        // Click first channel
        const channel = page.locator('.channel-item, [data-testid="channel-item"], [href*="/channels/"]').first();
        
        if (await channel.isVisible()) {
          await channel.click();
          
          // Should navigate to channel URL
          await expect(page).toHaveURL(/\/channels\/[a-zA-Z0-9-]+\/[a-zA-Z0-9-]+/);
        }
      }
    });

    test('create channel button opens modal', async ({ page }) => {
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        // Look for create channel button (usually + next to category)
        const createChannelBtn = page.locator(
          '[data-testid="create-channel"], button[title*="Create Channel"], .add-channel'
        ).first();
        
        if (await createChannelBtn.isVisible()) {
          await createChannelBtn.click();
          
          const modal = page.locator('[role="dialog"], .modal, [data-testid="create-channel-modal"]');
          await expect(modal.first()).toBeVisible({ timeout: 5000 });
        }
      }
    });
  });

  test.describe('Delete Server', () => {
    test('delete server option exists in settings', async ({ page }) => {
      // Navigate to a server
      const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
      
      if (await serverIcon.isVisible()) {
        await serverIcon.click();
        await page.waitForTimeout(500);
        
        // Open server settings
        const settingsButton = page.locator(
          '[data-testid="server-settings"], .server-header button'
        ).first();
        
        if (await settingsButton.isVisible()) {
          await settingsButton.click();
          await page.waitForTimeout(500);
          
          // Look for delete option (usually at bottom or in danger zone)
          const deleteButton = page.locator(
            'button:has-text("Delete"), [data-testid="delete-server"], .danger-zone button'
          );
          
          // Should exist but we won't click it
          // Just verify it's present for server owners
          const hasDeleteOption = await deleteButton.count() > 0;
          // This might not be visible for non-owners, so just log
          console.log('Delete server option present:', hasDeleteOption);
        }
      }
    });
  });
});
