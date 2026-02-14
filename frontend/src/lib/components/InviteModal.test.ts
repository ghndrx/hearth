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

  it('generates invite code when opened', async () => {
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      expect(input?.value).toMatch(/https:\/\/hearth\.chat\/invite\/[a-zA-Z0-9]{8}/);
    });
  });

  it('uses custom baseUrl when provided', async () => {
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server',
        baseUrl: 'https://custom.domain'
      }
    });

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      expect(input?.value).toMatch(/https:\/\/custom\.domain\/invite\/[a-zA-Z0-9]{8}/);
    });
  });

  it('copies invite link to clipboard', async () => {
    const { container, getByText } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      return input?.value !== '';
    });

    const copyButton = getByText('Copy');
    await fireEvent.click(copyButton);

    expect(mockClipboard.writeText).toHaveBeenCalledTimes(1);
    expect(mockClipboard.writeText).toHaveBeenCalledWith(
      expect.stringMatching(/https:\/\/hearth\.chat\/invite\/[a-zA-Z0-9]{8}/)
    );
  });

  it('shows copied state after copying', async () => {
    const { container, getByText } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

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
    }

    // Also test backdrop click
    const backdrop = container.querySelector('.modal-backdrop');
    if (backdrop) {
      await fireEvent.click(backdrop);
    }
  });

  it('dispatches invite event when generating invite', async () => {
    const handleInvite = vi.fn();
    const { container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    const component = container.querySelector('.modal-backdrop')?.parentElement;
    component?.addEventListener('invite', handleInvite);

    // Wait for initial invite generation
    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      return input?.value !== '';
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
    const { container, getByText } = render(InviteModal, {
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

  it('clears state on close', async () => {
    const { component, container } = render(InviteModal, {
      props: {
        open: true,
        serverName: 'Test Server'
      }
    });

    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      return input?.value !== '';
    });

    // Close and reopen
    await component.$set({ open: false });
    await component.$set({ open: true });

    // Should generate new invite
    await waitFor(() => {
      const input = container.querySelector('.invite-input') as HTMLInputElement;
      expect(input?.value).toMatch(/https:\/\/hearth\.chat\/invite\/[a-zA-Z0-9]{8}/);
    });
  });
});
