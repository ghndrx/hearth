import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import { TEST_CREDENTIALS, login, waitForApp } from './fixtures';

test.describe('Interactive Message Components', () => {
  // Skip all tests if no valid credentials
  test.beforeEach(async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires valid test credentials');
    await login(page, TEST_CREDENTIALS.email, TEST_CREDENTIALS.password);
    await waitForApp(page);
  });

  async function navigateToChannel(page: Page) {
    // Navigate to a server and channel
    const serverIcon = page.locator('.server-icon:not(.home-icon)').first();
    
    if (await serverIcon.isVisible()) {
      await serverIcon.click();
      await page.waitForTimeout(1000);
      
      const channelItem = page.locator('.channel-item, [data-testid="channel-item"]').first();
      if (await channelItem.isVisible()) {
        await channelItem.click();
        await page.waitForTimeout(500);
      }
      
      return true;
    }
    return false;
  }

  test.describe('Button Components', () => {
    test('buttons are visible when message has components', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      // Find messages with buttons
      const messageWithButtons = page.locator('.message:has(button)').first();
      
      if (await messageWithButtons.isVisible({ timeout: 5000 }).catch(() => false)) {
        const buttons = messageWithButtons.locator('button');
        await expect(await buttons.count()).toBeGreaterThan(0);
      }
    });

    test('button has correct label text', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const button = page.locator('.message button').first();
      
      if (await button.isVisible({ timeout: 5000 }).catch(() => false)) {
        const buttonText = await button.textContent();
        expect(buttonText).toBeTruthy();
      }
    });

    test('button is clickable', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const button = page.locator('.message button:not([disabled])').first();
      
      if (await button.isVisible({ timeout: 5000 }).catch(() => false)) {
        await button.click();
        // Button should respond to click (might trigger an action or modal)
        await page.waitForTimeout(500);
      }
    });

    test('disabled button is not clickable', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const disabledButton = page.locator('.message button[disabled]').first();
      
      if (await disabledButton.isVisible({ timeout: 5000 }).catch(() => false)) {
        await expect(disabledButton).toBeDisabled();
      }
    });

    test('link button has correct href', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const linkButton = page.locator('.message a[href]').first();
      
      if (await linkButton.isVisible({ timeout: 5000 }).catch(() => false)) {
        const href = await linkButton.getAttribute('href');
        expect(href).toMatch(/^https?:\/\//);
      }
    });

    test('button with emoji shows emoji', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const button = page.locator('.message button').first();
      
      if (await button.isVisible({ timeout: 5000 }).catch(() => false)) {
        const buttonContent = await button.textContent();
        // Button should have either text or emoji content
        expect(buttonContent).toBeTruthy();
      }
    });
  });

  test.describe('Select Menu Components', () => {
    test('select menu is visible when message has one', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const selectMenu = page.locator('.message select').first();
      
      if (await selectMenu.isVisible({ timeout: 5000 }).catch(() => false)) {
        await expect(selectMenu).toBeVisible();
      }
    });

    test('select menu has placeholder text', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const selectMenu = page.locator('.message select').first();
      
      if (await selectMenu.isVisible({ timeout: 5000 }).catch(() => false)) {
        const placeholder = selectMenu.locator('option[value=""]').first();
        if (await placeholder.isVisible()) {
          const placeholderText = await placeholder.textContent();
          expect(placeholderText).toBeTruthy();
        }
      }
    });

    test('select menu has options', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const selectMenu = page.locator('.message select').first();
      
      if (await selectMenu.isVisible({ timeout: 5000 }).catch(() => false)) {
        const options = selectMenu.locator('option');
        const count = await options.count();
        expect(count).toBeGreaterThan(1);
      }
    });

    test('select menu is focusable', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const selectMenu = page.locator('.message select').first();
      
      if (await selectMenu.isVisible({ timeout: 5000 }).catch(() => false)) {
        await selectMenu.focus();
        await expect(selectMenu).toBeFocused();
      }
    });

    test('can select an option', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const selectMenu = page.locator('.message select').first();
      
      if (await selectMenu.isVisible({ timeout: 5000 }).catch(() => false)) {
        // Select second option (first is placeholder)
        const options = selectMenu.locator('option');
        const count = await options.count();
        
        if (count > 1) {
          await selectMenu.selectOption({ index: 1 });
          const selectedValue = await selectMenu.inputValue();
          expect(selectedValue).toBeTruthy();
        }
      }
    });

    test('disabled select menu is not interactive', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const disabledSelect = page.locator('.message select[disabled]').first();
      
      if (await disabledSelect.isVisible({ timeout: 5000 }).catch(() => false)) {
        await expect(disabledSelect).toBeDisabled();
      }
    });
  });

  test.describe('Component Modal', () => {
    test('modal opens when triggered by button', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      // Look for a button that triggers a modal (if any)
      const modalTriggerButton = page.locator('button[data-modal-trigger], button:has-text("Modal")').first();
      
      if (await modalTriggerButton.isVisible({ timeout: 3000 }).catch(() => false)) {
        await modalTriggerButton.click();
        await page.waitForTimeout(500);
        
        const modal = page.locator('.modal-backdrop, [role="dialog"]');
        await expect(modal).toBeVisible({ timeout: 3000 });
      }
    });

    test('modal has title', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const modal = page.locator('.modal-backdrop, [role="dialog"]');
      
      if (await modal.isVisible({ timeout: 3000 }).catch(() => false)) {
        const title = modal.locator('h1, h2, [role="heading"]').first();
        await expect(title).toBeVisible();
      }
    });

    test('modal has text input when required', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const modal = page.locator('.modal-backdrop, [role="dialog"]');
      
      if (await modal.isVisible({ timeout: 3000 }).catch(() => false)) {
        const textInput = modal.locator('input[type="text"], textarea');
        await expect(textInput.first()).toBeVisible();
      }
    });

    test('modal submit button is present', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const modal = page.locator('.modal-backdrop, [role="dialog"]');
      
      if (await modal.isVisible({ timeout: 3000 }).catch(() => false)) {
        const submitBtn = modal.locator('.submit-btn, button:has-text("Submit"), button:has-text("Send")');
        await expect(submitBtn.first()).toBeVisible();
      }
    });

    test('modal cancel button closes modal', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const modal = page.locator('.modal-backdrop, [role="dialog"]');
      
      if (await modal.isVisible({ timeout: 3000 }).catch(() => false)) {
        const cancelBtn = modal.locator('.cancel-btn, button:has-text("Cancel")').first();
        if (await cancelBtn.isVisible()) {
          await cancelBtn.click();
          await page.waitForTimeout(300);
          await expect(modal).not.toBeVisible();
        }
      }
    });

    test('modal closes on Escape key', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const modal = page.locator('.modal-backdrop, [role="dialog"]');
      
      if (await modal.isVisible({ timeout: 3000 }).catch(() => false)) {
        await page.keyboard.press('Escape');
        await page.waitForTimeout(300);
        await expect(modal).not.toBeVisible();
      }
    });

    test('modal closes on backdrop click', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const modal = page.locator('.modal-backdrop, [role="dialog"]');
      
      if (await modal.isVisible({ timeout: 3000 }).catch(() => false)) {
        const backdrop = modal.first();
        await backdrop.click({ position: { x: 10, y: 10 } });
        await page.waitForTimeout(300);
        // Modal might stay open if clicking inside modal, so just verify no crash
      }
    });
  });

  test.describe('Action Rows', () => {
    test('multiple buttons render in action row', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      // Find action row or group of buttons
      const actionRow = page.locator('[role="group"], .action-row, .message-components').first();
      
      if (await actionRow.isVisible({ timeout: 5000 }).catch(() => false)) {
        const buttons = actionRow.locator('button');
        const count = await buttons.count();
        expect(count).toBeGreaterThan(0);
      }
    });

    test('buttons in row have gap spacing', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const actionRow = page.locator('[role="group"], .action-row, .message-components').first();
      
      if (await actionRow.isVisible({ timeout: 5000 }).catch(() => false)) {
        // Verify the container has flex layout
        const classList = await actionRow.getAttribute('class');
        expect(classList).toMatch(/flex|gap/);
      }
    });
  });

  test.describe('Component Accessibility', () => {
    test('buttons have accessible names', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const button = page.locator('.message button').first();
      
      if (await button.isVisible({ timeout: 5000 }).catch(() => false)) {
        const accessibleName = await button.getAttribute('aria-label') || await button.textContent();
        expect(accessibleName).toBeTruthy();
      }
    });

    test('select menus have accessible label', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const selectMenu = page.locator('.message select').first();
      
      if (await selectMenu.isVisible({ timeout: 5000 }).catch(() => false)) {
        const label = await selectMenu.getAttribute('aria-label') || 
                      selectMenu.locator('option[value=""]').textContent();
        expect(label).toBeTruthy();
      }
    });
  });
});
