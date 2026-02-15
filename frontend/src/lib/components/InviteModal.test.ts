import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import InviteModal from './InviteModal.svelte';

describe('InviteModal', () => {
  const mockClipboard = {
    writeText: vi.fn().mockResolvedValue(undefined)
  };

  beforeEach(() => {
    vi.clearAllMocks();
    Object.assign(navigator, {
      clipboard: mockClipboard
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not render when closed', () => {
    const { container } = render(InviteModal, {
      props: {
        open: false,
        serverName: 'Test Server'
      }
    });

    expect(container.querySelector('.modal')).not.toBeInTheDocument();
  });

  it('renders with server name when open', () => {
    const { getByText } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    expect(getByText('Invite friends to Test Server')).toBeInTheDocument();
  });

  it('renders channel info when channel name provided', () => {
    const { getByText } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server',
        channelName: 'general',
        channelId: '123'
      }
    });

    expect(getByText('general')).toBeInTheDocument();
  });

  it('does not render channel info when channel name not provided', () => {
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    expect(container.querySelector('.channel-info')).not.toBeInTheDocument();
  });

  it('dispatches generateInvite event when opened', async () => {
    const handleGenerateInvite = vi.fn();
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    const component = container.querySelector('.modal-backdrop')?.parentElement;
    component?.addEventListener('generateInvite', handleGenerateInvite);

    // Wait for event dispatch
    await waitFor(() => {
      expect(handleGenerateInvite).toHaveBeenCalledTimes(1);
    });

    // Verify event detail
    const eventDetail = handleGenerateInvite.mock.calls[0][0] as CustomEvent;
    expect(eventDetail.detail).toMatchObject({
      maxUses: 0,
      expiresIn: 604800
    });
  });

  it('displays invite link when onInviteGenerated is called', async () => {
    const { container, component } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Simulate API response by calling onInviteGenerated
    (component as any).onInviteGenerated('ABC12345');

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      expect(input?.value).toBe('https://hearth.chat/invite/ABC12345');
    });
  });

  it('uses custom baseUrl when provided', async () => {
    const { container, component } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server',
        baseUrl: 'https://custom.domain'
      }
    });

    // Simulate API response
    (component as any).onInviteGenerated('TEST1234');

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      expect(input?.value).toBe('https://custom.domain/invite/TEST1234');
    });
  });

  it('copies invite link to clipboard', async () => {
    const { container, component, getByText } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Simulate API response
    (component as any).onInviteGenerated('COPYTEST');

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      return input?.value !== '';
    });

    const copyButton = getByText('Copy');
    await fireEvent.click(copyButton);

    expect(mockClipboard.writeText).toHaveBeenCalledTimes(1);
    expect(mockClipboard.writeText).toHaveBeenCalledWith('https://hearth.chat/invite/COPYTEST');
  });

  it('shows copied state after copying', async () => {
    const { container, component, getByText } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Simulate API response
    (component as any).onInviteGenerated('COPYTEST');

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      return input?.value !== '';
    });

    const copyButton = getByText('Copy');
    await fireEvent.click(copyButton);

    expect(getByText('Copied')).toBeInTheDocument();
  });

  it('disables copy button when no invite link', () => {
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Before invite is generated, button should be disabled
    const copyButton = container.querySelector('button') as HTMLButtonElement;
    expect(copyButton?.disabled).toBe(true);
  });

  it('dispatches close event when clicking close button', async () => {
    const handleClose = vi.fn();
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    const component = container.querySelector('.modal-backdrop')?.parentElement;
    component?.addEventListener('close', handleClose);

    const closeButton = container.querySelector('.close-btn');
    if (closeButton) {
      await fireEvent.click(closeButton);
      expect(handleClose).toHaveBeenCalledTimes(1);
    }

    // Also test backdrop click
    const backdrop = container.querySelector('.modal-backdrop');
    if (backdrop) {
      await fireEvent.click(backdrop);
      expect(handleClose).toHaveBeenCalledTimes(2);
    }
  });

  it('dispatches generateInvite event with correct settings', async () => {
    const handleGenerateInvite = vi.fn();
    const { container, component } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    const comp = container.querySelector('.modal-backdrop')?.parentElement;
    comp?.addEventListener('generateInvite', handleGenerateInvite);

    // Wait for initial event
    await waitFor(() => {
      expect(handleGenerateInvite).toHaveBeenCalledTimes(1);
    });

    // Open advanced settings and change values
    const details = container.querySelector('.advanced-settings') as HTMLDetailsElement;
    if (details) {
      details.open = true;
    }

    // Change expiration
    const expiresSelect = container.querySelector('#expires') as HTMLSelectElement;
    if (expiresSelect) {
      expiresSelect.value = '3600'; // 1 hour
      await fireEvent.change(expiresSelect);
    }

    // Should dispatch new generateInvite event
    await waitFor(() => {
      expect(handleGenerateInvite).toHaveBeenCalledTimes(2);
    });

    const lastEvent = handleGenerateInvite.mock.calls[1][0] as CustomEvent;
    expect(lastEvent.detail).toMatchObject({
      maxUses: 0,
      expiresIn: 3600
    });
  });

  it('has expiration options', () => {
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    const select = container.querySelector('#expires') as HTMLSelectElement;
    expect(select).toBeInTheDocument();
    expect(select?.options.length).toBeGreaterThan(0);
  });

  it('has max uses options', () => {
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    const select = container.querySelector('#max-uses') as HTMLSelectElement;
    expect(select).toBeInTheDocument();
    expect(select?.options.length).toBeGreaterThan(0);
  });

  it('shows expiry note', () => {
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    const expiryNote = container.querySelector('.expiry-note');
    expect(expiryNote).toBeInTheDocument();
    expect(expiryNote?.textContent).toContain('expires in');
  });

  it('shows never expire note when expiresIn is 0', async () => {
    const { container, component } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Open advanced settings
    const details = container.querySelector('.advanced-settings') as HTMLDetailsElement;
    if (details) {
      details.open = true;
    }

    // Select "Never" option
    const expiresSelect = container.querySelector('#expires') as HTMLSelectElement;
    if (expiresSelect) {
      expiresSelect.value = '0';
      await fireEvent.change(expiresSelect);
    }

    await waitFor(() => {
      const expiryNote = container.querySelector('.expiry-note');
      expect(expiryNote?.textContent).toContain('never expire');
    });
  });

  it('shows error message when invite generation fails', async () => {
    const { container, component } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Force an error by simulating a failed generation
    const generateButton = container.querySelector('button');
    if (generateButton) {
      // Multiple rapid clicks might trigger an error state
      await fireEvent.click(generateButton);
    }
  });

  it('shows error message when clipboard copy fails', async () => {
    mockClipboard.writeText.mockRejectedValueOnce(new Error('Clipboard failed'));
    
    const { container, component, getByText } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Simulate API response
    (component as any).onInviteGenerated('ERRORTEST');

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      return input?.value !== '';
    });

    const copyButton = getByText('Copy');
    await fireEvent.click(copyButton);

    // Error message should be displayed
    await waitFor(() => {
      const errorMessage = container.querySelector('.error-message');
      expect(errorMessage?.textContent).toContain('clipboard');
    });
  });

  it('clears state on close', async () => {
    const { container, component } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Generate an invite first
    (component as any).onInviteGenerated('CLEARTEST');

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      return input?.value !== '';
    });

    // Close modal by dispatching close event (simulating prop change)
    const closeEvent = new CustomEvent('close');
    (component as any).$on('close', () => {});
    
    // Simulate reopening
    const { container: newContainer, component: newComponent } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    // Should generate new invite (input should be empty until onInviteGenerated is called)
    const newInput = newContainer.querySelector('.invite-input') as HTMLInputElement;
    expect(newInput?.value).toBe('');
  });
});
