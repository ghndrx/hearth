import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Mock SvelteKit's $app modules
vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
  beforeNavigate: vi.fn(),
  afterNavigate: vi.fn()
}));

vi.mock('$app/stores', () => ({
  page: {
    subscribe: vi.fn((fn) => {
      fn({ url: new URL('http://localhost'), params: {} });
      return () => {};
    })
  },
  navigating: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    })
  }
}));

// Mock $lib imports
vi.mock('$lib/stores/channels', () => ({
  channels: {
    subscribe: vi.fn((fn) => {
      fn([]);
      return () => {};
    })
  },
  currentChannel: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    }),
    set: vi.fn()
  }
}));

vi.mock('$lib/stores/servers', () => ({
  currentServer: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    }),
    set: vi.fn()
  },
  leaveServer: vi.fn()
}));

vi.mock('$lib/stores/auth', () => ({
  user: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    })
  }
}));

vi.mock('$lib/stores/settings', () => ({
  settings: {
    openServerSettings: vi.fn()
  }
}));
