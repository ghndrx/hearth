import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import EmojiPicker from './EmojiPicker.svelte';

describe('EmojiPicker', () => {
  beforeEach(() => {
    // Mock localStorage
    const localStorageMock = {
      getItem: vi.fn(),
      setItem: vi.fn(),
      clear: vi.fn(),
    };
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      writable: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('visibility', () => {
    it('renders nothing when show is false', () => {
      const { container } = render(EmojiPicker, { props: { show: false } });
      expect(container.querySelector('.emoji-picker')).toBeNull();
    });

    it('renders picker when show is true', () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      expect(container.querySelector('.emoji-picker')).not.toBeNull();
    });
  });

  describe('search functionality', () => {
    it('renders search input', () => {
      render(EmojiPicker, { props: { show: true } });
      const searchInput = screen.getByPlaceholderText('Search emoji');
      expect(searchInput).toBeInTheDocument();
    });

    it('filters emojis based on search query', async () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const searchInput = screen.getByPlaceholderText('Search emoji');
      
      await fireEvent.input(searchInput, { target: { value: '😀' } });
      
      // Should show search results label
      expect(container.textContent).toContain('Search Results');
    });

    it('shows no results message when search has no matches', async () => {
      render(EmojiPicker, { props: { show: true } });
      const searchInput = screen.getByPlaceholderText('Search emoji');
      
      await fireEvent.input(searchInput, { target: { value: 'xyznonexistent' } });
      
      expect(screen.getByText(/No emoji found/)).toBeInTheDocument();
    });
  });

  describe('category navigation', () => {
    it('renders category buttons', () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const categoryButtons = container.querySelectorAll('.category-btn');
      
      // Should have at least 9 categories (excluding recent if empty)
      expect(categoryButtons.length).toBeGreaterThanOrEqual(9);
    });

    it('switches category when clicking category button', async () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const categoryButtons = container.querySelectorAll('.category-btn');
      
      // Click on a different category (e.g., People & Body)
      await fireEvent.click(categoryButtons[1]);
      
      // Check that the category label changed
      expect(container.textContent).toContain('People & Body');
    });

    it('shows category labels', () => {
      render(EmojiPicker, { props: { show: true } });
      
      // Default category should be Smileys & Emotion
      expect(screen.getByText('Smileys & Emotion')).toBeInTheDocument();
    });
  });

  describe('skin tone selector', () => {
    it('renders skin tone button', () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const skinToneButton = container.querySelector('.skin-tone-button');
      
      expect(skinToneButton).toBeInTheDocument();
    });

    it('opens skin tone picker when clicking button', async () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const skinToneButton = container.querySelector('.skin-tone-button') as HTMLElement;
      
      await fireEvent.click(skinToneButton);
      
      const skinTonePicker = container.querySelector('.skin-tone-picker');
      expect(skinTonePicker).toBeInTheDocument();
    });

    it('shows all 6 skin tone options', async () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const skinToneButton = container.querySelector('.skin-tone-button') as HTMLElement;
      
      await fireEvent.click(skinToneButton);
      
      const skinToneOptions = container.querySelectorAll('.skin-tone-option');
      expect(skinToneOptions.length).toBe(6);
    });

    it('selects skin tone and closes picker', async () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const skinToneButton = container.querySelector('.skin-tone-button') as HTMLElement;
      
      await fireEvent.click(skinToneButton);
      
      const skinToneOptions = container.querySelectorAll('.skin-tone-option');
      await fireEvent.click(skinToneOptions[2]); // Select Medium-Light
      
      // Picker should close
      const skinTonePicker = container.querySelector('.skin-tone-picker');
      expect(skinTonePicker).toBeNull();
    });
  });

  describe('emoji selection', () => {
    it('renders emoji buttons', () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const emojiButtons = container.querySelectorAll('.emoji-btn');
      
      // Should have many emoji buttons
      expect(emojiButtons.length).toBeGreaterThan(10);
    });

    it('dispatches select event when clicking emoji', async () => {
      const { container, component } = render(EmojiPicker, { props: { show: true } });
      
      let selectedEmoji = '';
      component.$on('select', (e: CustomEvent<string>) => {
        selectedEmoji = e.detail;
      });
      
      const emojiButtons = container.querySelectorAll('.emoji-btn');
      await fireEvent.click(emojiButtons[0]);
      
      expect(selectedEmoji).not.toBe('');
    });
  });

  describe('recent emojis', () => {
    it('loads recent emojis from localStorage on mount', () => {
      const recentEmojis = ['😀', '😎', '🎉'];
      vi.mocked(localStorage.getItem).mockReturnValue(JSON.stringify(recentEmojis));
      
      render(EmojiPicker, { props: { show: true } });
      
      expect(localStorage.getItem).toHaveBeenCalledWith('hearth_recent_emojis');
    });

    it('saves emoji to recent when selected', async () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      
      const emojiButtons = container.querySelectorAll('.emoji-btn');
      await fireEvent.click(emojiButtons[0]);
      
      expect(localStorage.setItem).toHaveBeenCalled();
    });
  });

  describe('keyboard navigation', () => {
    it('closes picker on Escape key', async () => {
      const { component } = render(EmojiPicker, { props: { show: true } });
      
      let closed = false;
      component.$on('close', () => {
        closed = true;
      });
      
      await fireEvent.keyDown(document, { key: 'Escape' });
      
      expect(closed).toBe(true);
    });

    it('closes skin tone picker first on Escape', async () => {
      const { container, component } = render(EmojiPicker, { props: { show: true } });
      
      // Open skin tone picker
      const skinToneButton = container.querySelector('.skin-tone-button') as HTMLElement;
      await fireEvent.click(skinToneButton);
      
      let closed = false;
      component.$on('close', () => {
        closed = true;
      });
      
      await fireEvent.keyDown(document, { key: 'Escape' });
      
      // Skin tone picker should close, but main picker should stay open
      const skinTonePicker = container.querySelector('.skin-tone-picker');
      expect(skinTonePicker).toBeNull();
      expect(closed).toBe(false);
    });
  });

  describe('footer preview', () => {
    it('renders footer with preview emoji', () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      const footer = container.querySelector('.footer');
      
      expect(footer).toBeInTheDocument();
    });

    it('shows preview info text', () => {
      render(EmojiPicker, { props: { show: true } });
      
      expect(screen.getByText('Hover to preview')).toBeInTheDocument();
    });
  });

  describe('styling and animations', () => {
    it('has proper structure with header, categories, emojis, and footer', () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      
      expect(container.querySelector('.header')).toBeInTheDocument();
      expect(container.querySelector('.categories')).toBeInTheDocument();
      expect(container.querySelector('.category-label')).toBeInTheDocument();
      expect(container.querySelector('.emojis')).toBeInTheDocument();
      expect(container.querySelector('.footer')).toBeInTheDocument();
    });

    it('has active class on selected category', () => {
      const { container } = render(EmojiPicker, { props: { show: true } });
      
      const activeCategory = container.querySelector('.category-btn.active');
      expect(activeCategory).toBeInTheDocument();
    });
  });
});
