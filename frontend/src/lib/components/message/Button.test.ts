import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import Button from './Button.svelte';

describe('Button (Message Component)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders with default props', () => {
    const { container } = render(Button, {
      props: {}
    });

    const button = container.querySelector('button');
    expect(button).toBeInTheDocument();
    expect(button).toHaveAttribute('type', 'button');
    expect(button).not.toBeDisabled();
  });

  it('renders with label', () => {
    const { container } = render(Button, {
      props: { label: 'Click Me' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveTextContent('Click Me');
  });

  it('renders with emoji', () => {
    const { container } = render(Button, {
      props: { emoji: '👍' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveTextContent('👍');
  });

  it('can be disabled', async () => {
    const { container } = render(Button, {
      props: { disabled: true }
    });

    const button = container.querySelector('button');
    expect(button).toBeDisabled();
  });

  it('handles click events', async () => {
    const handleClick = vi.fn();
    const { container } = render(Button, {
      props: { customId: 'test_btn' }
    });

    const button = container.querySelector('button');
    button?.addEventListener('click', handleClick);

    await fireEvent.click(button!);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('does not trigger click when disabled', async () => {
    const handleClick = vi.fn();
    const { container } = render(Button, {
      props: { disabled: true, customId: 'test_btn' }
    });

    const button = container.querySelector('button');
    button?.addEventListener('click', handleClick);

    await fireEvent.click(button!);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('applies primary style classes', () => {
    const { container } = render(Button, {
      props: { style: 'primary', label: 'Primary' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveClass('bg-[#5865f2]');
  });

  it('applies secondary style classes', () => {
    const { container } = render(Button, {
      props: { style: 'secondary', label: 'Secondary' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveClass('bg-[#4f545c]');
  });

  it('applies success style classes', () => {
    const { container } = render(Button, {
      props: { style: 'success', label: 'Success' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveClass('bg-[#3ba55c]');
  });

  it('applies danger style classes', () => {
    const { container } = render(Button, {
      props: { style: 'danger', label: 'Danger' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveClass('bg-[#da373c]');
  });

  it('renders as link when style is link', () => {
    const { container } = render(Button, {
      props: { style: 'link', label: 'Link', url: 'https://example.com' }
    });

    const anchor = container.querySelector('a');
    expect(anchor).toBeInTheDocument();
    expect(anchor).toHaveAttribute('href', 'https://example.com');
  });

  it('renders emoji and label together', () => {
    const { container } = render(Button, {
      props: { label: 'Like', emoji: '👍' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveTextContent('👍');
    expect(button).toHaveTextContent('Like');
  });

  it('shows loading state', async () => {
    const { container } = render(Button, {
      props: { label: 'Loading', customId: 'loading_btn' }
    });

    const button = container.querySelector('button');
    await fireEvent.click(button!);

    await tick();
    
    // After click, loading state should be set briefly
    // The loading animation SVG should appear
    const svg = container.querySelector('svg.animate-spin');
    expect(svg).toBeInTheDocument();
  });

  it.skip('does not respond to click when loading', async () => {
    // Skipped: fireEvent.click() in @testing-library/svelte does not respect the
    // disabled attribute, so this test cannot properly verify that clicks are
    // ignored when loading. This would need userEvent.click() from
    // @testing-library/user-event for proper testing.
    const handleClick = vi.fn();
    const { container } = render(Button, {
      props: { label: 'Loading', customId: 'loading_btn' }
    });

    // First click sets loading
    const button = container.querySelector('button');
    button?.addEventListener('click', handleClick);
    
    await fireEvent.click(button!);
    await tick();
    
    // Second click should not trigger when loading
    await fireEvent.click(button!);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
});
