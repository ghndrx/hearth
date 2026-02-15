import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Mock Web Animations API for jsdom (used by Svelte transitions)
// Always override as jsdom's implementation may be incomplete
Element.prototype.animate = function() {
	const animation = {
		finished: Promise.resolve(),
		cancel: () => {},
		play: () => {},
		pause: () => {},
		reverse: () => {},
		finish: () => {},
		onfinish: null as (() => void) | null,
		oncancel: null as (() => void) | null,
		playState: 'finished',
		currentTime: 0,
		effect: null,
		timeline: null,
		startTime: null,
		playbackRate: 1,
		pending: false,
		id: '',
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => true,
		commitStyles: () => {},
		persist: () => {},
		updatePlaybackRate: () => {},
		replaceState: () => {}
	};
	// Immediately call onfinish callback if set (for Svelte transitions)
	setTimeout(() => {
		if (animation.onfinish) animation.onfinish();
	}, 0);
	return animation as unknown as Animation;
};

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
