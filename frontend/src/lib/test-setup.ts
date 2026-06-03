import '@testing-library/jest-dom';
import { vi } from 'vitest';
import fakeIndexedDB from 'fake-indexeddb';

// Polyfill IndexedDB for tests using fake-indexeddb
if (typeof indexedDB === 'undefined') {
  Object.defineProperty(globalThis, 'indexedDB', {
    value: fakeIndexedDB,
    writable: true,
    configurable: true,
  });
}

// jsdom does not implement window.matchMedia, which is needed by theme.ts
// Polyfill matchMedia for jsdom environment
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(() => true),
  })),
});

// Polyfill localStorage for test environments
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value; }),
    removeItem: vi.fn((key: string) => { delete store[key]; }),
    clear: vi.fn(() => { store = {}; }),
  };
})();
Object.defineProperty(window, 'localStorage', {
  writable: true,
  value: localStorageMock,
});

// happy-dom does not implement navigator.clipboard (or only provides a getter)
try {
  if (!navigator.clipboard || typeof navigator.clipboard.writeText !== 'function') {
    // Remove any existing non-configurable property first
    delete (navigator as unknown as Record<string, unknown>)['clipboard'];
    const mockClipboard = {
      writeText: vi.fn().mockResolvedValue(undefined),
      readText: vi.fn().mockResolvedValue(''),
    };
    Object.defineProperty(navigator, 'clipboard', {
      writable: true,
      value: mockClipboard,
    });
  }
} catch {
  // Ignore if clipboard access fails (some test environments restrict this)
}

// Mock ApiError class exported for use in tests that need to mock $lib/api
// Usage: import { MockApiError } from '../test-setup'; 
//        vi.mock('$lib/api', () => ({ api: mockApi, ApiError: MockApiError }))
export class MockApiError extends Error {
  status: number;
  data: unknown;
  constructor(message: string, status: number, data?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

// Mock Svelte transitions to be instant in tests
// This prevents elements from being held in DOM during transition animations
vi.mock('svelte/transition', () => ({
  fade: () => ({ duration: 0 }),
  fly: () => ({ duration: 0 }),
  slide: () => ({ duration: 0 }),
  scale: () => ({ duration: 0 }),
  blur: () => ({ duration: 0 }),
  draw: () => ({ duration: 0 }),
  crossfade: () => [() => ({ duration: 0 }), () => ({ duration: 0 })]
}));

// Polyfill for Web Animations API (not supported in jsdom)
// This is needed for Svelte 5 transitions
if (typeof Element !== 'undefined' && !Element.prototype.animate) {
  Element.prototype.animate = function(_keyframes: Keyframe[] | PropertyIndexedKeyframes | null, _options?: number | KeyframeAnimationOptions) {
    // Create a deferred animation mock that properly types the Promise properties
    const animationPromise = Promise.resolve() as unknown as Promise<Animation>;
    const animation = {
      finished: animationPromise,
      cancel: vi.fn(),
      finish: vi.fn(),
      pause: vi.fn(),
      play: vi.fn(),
      reverse: vi.fn(),
      updatePlaybackRate: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(() => true),
      currentTime: 0,
      playbackRate: 1,
      playState: 'finished' as AnimationPlayState,
      pending: false,
      id: '',
      oncancel: null,
      onfinish: null,
      onremove: null,
      timeline: null,
      startTime: null,
      effect: null,
      replaceState: 'active' as AnimationReplaceState,
      persist: vi.fn(),
      commitStyles: vi.fn(),
      ready: animationPromise,
    } as unknown as Animation;
    return animation;
  };
}

// Mock SvelteKit's $app modules
vi.mock('$app/environment', () => ({
  browser: true,
  dev: true,
  building: false,
  version: 'test'
}));

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
    }),
    set: vi.fn(),
    update: vi.fn()
  },
  currentChannel: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    }),
    set: vi.fn()
  },
  channelsLoading: {
    subscribe: vi.fn((fn) => {
      fn(false);
      return () => {};
    }),
    set: vi.fn()
  },
  channelsError: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    }),
    set: vi.fn()
  },
  serverChannels: {
    subscribe: vi.fn((fn) => {
      fn([]);
      return () => {};
    })
  },
  dmChannels: {
    subscribe: vi.fn((fn) => {
      fn([]);
      return () => {};
    })
  },
  categorizedChannels: {
    subscribe: vi.fn((fn) => {
      fn({ categories: [], uncategorized: [] });
      return () => {};
    })
  },
  loadServerChannels: vi.fn().mockResolvedValue([]),
  loadDMChannels: vi.fn().mockResolvedValue([]),
  createChannel: vi.fn(),
  updateChannel: vi.fn(),
  deleteChannel: vi.fn(),
  createDM: vi.fn(),
  getChannel: vi.fn()
}));

vi.mock('$lib/stores/servers', () => ({
  servers: {
    subscribe: vi.fn((fn) => {
      fn([]);
      return () => {};
    }),
    set: vi.fn(),
    update: vi.fn()
  },
  currentServer: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    }),
    set: vi.fn()
  },
  loadServers: vi.fn().mockResolvedValue([]),
  createServer: vi.fn(),
  updateServer: vi.fn(),
  deleteServer: vi.fn(),
  leaveServer: vi.fn(),
  joinServer: vi.fn(),
  updateServerIcon: vi.fn(),
  removeServerIcon: vi.fn()
}));

vi.mock('$lib/stores/auth', () => ({
  auth: {
    subscribe: vi.fn((fn) => {
      fn({ user: null, token: null, loading: false, initialized: true });
      return () => {};
    }),
    set: vi.fn(),
    update: vi.fn()
  },
  user: {
    subscribe: vi.fn((fn) => {
      fn(null);
      return () => {};
    })
  },
  isAuthenticated: {
    subscribe: vi.fn((fn) => {
      fn(false);
      return () => {};
    })
  },
  isLoading: {
    subscribe: vi.fn((fn) => {
      fn(false);
      return () => {};
    })
  },
  login: vi.fn(),
  logout: vi.fn(),
  register: vi.fn(),
  refreshToken: vi.fn(),
  updateProfile: vi.fn()
}));

// Mock $lib/stores/presence
vi.mock('$lib/stores/presence', () => ({
  presenceStore: {
    subscribe: vi.fn((fn) => {
      fn(new Map());
      return () => {};
    }),
    getPresence: vi.fn((userId: string) => ({
      userId,
      status: 'offline' as const,
      activities: [],
    })),
    setStatus: vi.fn(),
  },
  getStatusColor: vi.fn((status: string) => {
    switch (status) {
      case 'online': return '#3ba55c';
      case 'idle': return '#faa61a';
      case 'dnd': return '#ed4245';
      default: return '#747f8d';
    }
  }),
  getStatusLabel: vi.fn((status: string) => {
    switch (status) {
      case 'online': return 'Online';
      case 'idle': return 'Idle';
      case 'dnd': return 'Do Not Disturb';
      case 'invisible': return 'Invisible';
      default: return 'Offline';
    }
  }),
  getActivityLabel: vi.fn((type: number) => {
    switch (type) {
      case 0: return 'Playing';
      case 1: return 'Streaming';
      case 2: return 'Listening to';
      case 3: return 'Watching';
      case 4: return '';
      case 5: return 'Competing in';
      default: return '';
    }
  }),
}));

// NOTE: $lib/stores/settings is NOT mocked here - tests that need the real
// implementation (like settings.test.ts) require the actual module.
// If a test needs a mock settings store, it should mock it locally.
