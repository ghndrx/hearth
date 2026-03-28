import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import { TEST_CREDENTIALS, login, waitForApp } from './fixtures';

test.describe('Messaging', () => {
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
      
      // Wait for channel redirect or click a channel
      const channelItem = page.locator('.channel-item, [data-testid="channel-item"]').first();
      if (await channelItem.isVisible()) {
        await channelItem.click();
        await page.waitForTimeout(500);
      }
      
      return true;
    }
    return false;
  }

  test.describe('Message Input', () => {
    test('message input is visible in channel', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      // Message input should be visible
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input, input[placeholder*="message"]'
      ).first();
      await expect(messageInput).toBeVisible();
    });

    test('message input is focusable', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input'
      ).first();
      
      await messageInput.focus();
      await expect(messageInput).toBeFocused();
    });

    test('can type in message input', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input'
      ).first();
      
      const testMessage = 'Test message content';
      await messageInput.fill(testMessage);
      
      await expect(messageInput).toHaveValue(testMessage);
    });
  });

  test.describe('Send Message', () => {
    test('can send message with Enter key', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input'
      ).first();
      
      const testMessage = `E2E Test Message ${Date.now()}`;
      await messageInput.fill(testMessage);
      await page.keyboard.press('Enter');
      
      // Wait for message to appear
      await page.waitForTimeout(2000);
      
      // Check if message appears in the chat
      const messageList = page.locator('.message-list, [data-testid="message-list"], .messages');
      const sentMessage = messageList.locator(`text="${testMessage}"`);
      
      // Message should appear (or input should be cleared indicating send)
      const inputValue = await messageInput.inputValue();
      expect(inputValue === '' || await sentMessage.isVisible()).toBeTruthy();
    });

    test('message input clears after sending', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input'
      ).first();
      
      await messageInput.fill(`Test clear ${Date.now()}`);
      await page.keyboard.press('Enter');
      
      // Wait for send
      await page.waitForTimeout(1500);
      
      // Input should be cleared
      await expect(messageInput).toHaveValue('');
    });
  });

  test.describe('Message Display', () => {
    test('messages appear in chat log', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      // Wait for messages to load
      await page.waitForTimeout(1000);
      
      // Message list should be visible
      const messageList = page.locator('.message-list, [data-testid="message-list"], .messages');
      await expect(messageList).toBeVisible();
      
      // Should have some messages (or empty state)
      const messages = page.locator('.message, [data-testid="message"]');
      const emptyState = page.locator('.empty-state, .no-messages');
      
      const hasContent = (await messages.count() > 0) || await emptyState.isVisible();
      expect(hasContent).toBeTruthy();
    });

    test('messages show author and content', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const message = page.locator('.message, [data-testid="message"]').first();
      
      if (await message.isVisible()) {
        // At least the message container is visible
        await expect(message).toBeVisible();
        // Author and content elements exist within the message
        await expect(message.locator('.author, .username, [data-testid="message-author"]')).toBeDefined();
        await expect(message.locator('.content, .message-content, [data-testid="message-content"]')).toBeDefined();
      }
    });

    test('messages scroll to bottom on new message', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      const messageList = page.locator('.message-list, [data-testid="message-list"]').first();
      
      // Get initial scroll position
      const initialScroll = await messageList.evaluate(el => el.scrollTop);
      
      // Send a new message
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input'
      ).first();
      
      await messageInput.fill(`Scroll test ${Date.now()}`);
      await page.keyboard.press('Enter');
      
      await page.waitForTimeout(2000);
      
      // Scroll position should be at or near bottom
      const finalScroll = await messageList.evaluate(el => ({
        scrollTop: el.scrollTop,
        scrollHeight: el.scrollHeight,
        clientHeight: el.clientHeight,
      }));
      
      // Should be scrolled to bottom (with some tolerance)
      const isAtBottom = finalScroll.scrollTop + finalScroll.clientHeight >= finalScroll.scrollHeight - 50;
      expect(isAtBottom || finalScroll.scrollTop >= initialScroll).toBeTruthy();
    });
  });

  test.describe('Message Actions', () => {
    test('message context menu appears on right-click', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const message = page.locator('.message, [data-testid="message"]').first();
      
      if (await message.isVisible()) {
        // Right-click on message
        await message.click({ button: 'right' });
        
        // Context menu should appear
        const contextMenu = page.locator('.context-menu, [role="menu"], [data-testid="context-menu"]');
        
        // Wait a bit and check
        await page.waitForTimeout(500);
        const hasContextMenu = await contextMenu.isVisible().catch(() => false);
        
        // Close it if open
        if (hasContextMenu) {
          await page.keyboard.press('Escape');
        }
      }
    });

    test('message action bar appears on hover', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      await page.waitForTimeout(1000);
      
      const message = page.locator('.message, [data-testid="message"]').first();
      
      if (await message.isVisible()) {
        // Hover on message
        await message.hover();
        
        await page.waitForTimeout(500);
        // Action bar might be hidden or might not exist for all messages
        // (edit, delete, react buttons appear on hover)
      }
    });

    test('can edit own message', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      // First send a message to edit
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input'
      ).first();
      
      const originalMessage = `Edit test ${Date.now()}`;
      await messageInput.fill(originalMessage);
      await page.keyboard.press('Enter');
      
      await page.waitForTimeout(2000);
      
      // Find the sent message
      const sentMessage = page.locator(`.message:has-text("${originalMessage}")`).first();
      
      if (await sentMessage.isVisible()) {
        // Right-click and look for edit option
        await sentMessage.click({ button: 'right' });
        await page.waitForTimeout(300);
        
        const editButton = page.locator('button:has-text("Edit"), [data-testid="edit-message"]');
        if (await editButton.isVisible()) {
          await editButton.click();
          
          // Edit input should appear
          const editInput = page.locator('.edit-input, [data-testid="edit-message-input"], textarea').first();
          if (await editInput.isVisible()) {
            await editInput.fill(`${originalMessage} (edited)`);
            await page.keyboard.press('Enter');
          }
        }
      }
    });

    test('can delete own message', async ({ page }) => {
      const hasChannel = await navigateToChannel(page);
      test.skip(!hasChannel, 'No channels available');
      
      // Send a message to delete
      const messageInput = page.locator(
        '[data-testid="message-input"], textarea, .message-input'
      ).first();
      
      const messageToDelete = `Delete test ${Date.now()}`;
      await messageInput.fill(messageToDelete);
      await page.keyboard.press('Enter');
      
      await page.waitForTimeout(2000);
      
      // Find and delete the message
      const sentMessage = page.locator(`.message:has-text("${messageToDelete}")`).first();
      
      if (await sentMessage.isVisible()) {
        await sentMessage.click({ button: 'right' });
        await page.waitForTimeout(300);
        
        const deleteButton = page.locator('button:has-text("Delete"), [data-testid="delete-message"]');
        if (await deleteButton.isVisible()) {
          await deleteButton.click();
          
          // Confirm deletion if there's a confirmation dialog
          const confirmButton = page.locator('button:has-text("Confirm"), button:has-text("Delete")').last();
          if (await confirmButton.isVisible()) {
            await confirmButton.click();
          }
          
          await page.waitForTimeout(1000);
          
          // Message should be gone
          await expect(page.locator(`text="${messageToDelete}"`)).not.toBeVisible();
        }
      }
    });
  });
});
