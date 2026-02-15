import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Button from './Button.svelte';

describe('Button', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders with default props', () => {
    const { container } = render(Button, {
      props: {}
    });

    const button = container.querySelector('button');
    expect(button).toBeInTheDocument();
    // Button uses Tailwind classes, check for primary variant pattern
    expect(button).toHaveClass('bg-blurple-500');
    expect(button).toHaveAttribute('type', 'button');
    expect(button).not.toBeDisabled();
  });

  it('renders with custom variant', () => {
    const { container } = render(Button, {
      props: { variant: 'danger' }
    });

    const button = container.querySelector('button');
    // Danger variant uses red background
    expect(button).toHaveClass('bg-red-500');
    expect(button).not.toHaveClass('bg-blurple-500');
  });

  it('renders with different sizes', () => {
    const { container: smContainer } = render(Button, {
      props: { size: 'sm' }
    });
    // Small size uses smaller padding
    expect(smContainer.querySelector('button')).toHaveClass('text-sm');

    const { container: lgContainer } = render(Button, {
      props: { size: 'lg' }
    });
    // Large size uses larger padding
    expect(lgContainer.querySelector('button')).toHaveClass('text-lg');
  });

  it('can be disabled', () => {
    const { container } = render(Button, {
      props: { disabled: true }
    });

    const button = container.querySelector('button');
    expect(button).toBeDisabled();
  });

  it('handles click events', async () => {
    const handleClick = vi.fn();
    const { container } = render(Button, {
      props: {}
    });

    const button = container.querySelector('button');
    button?.addEventListener('click', handleClick);

    await fireEvent.click(button!);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  // Skip - disabled buttons still receive click events in jsdom when using addEventListener
  it.skip('does not trigger click when disabled', async () => {
    const handleClick = vi.fn();
    const { container } = render(Button, {
      props: { disabled: true }
    });

    const button = container.querySelector('button');
    button?.addEventListener('click', handleClick);

    await fireEvent.click(button!);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('supports fullWidth prop', () => {
    const { container } = render(Button, {
      props: { fullWidth: true }
    });

    const button = container.querySelector('button');
    // fullWidth uses w-full Tailwind class
    expect(button).toHaveClass('w-full');
  });

  // Skip slot test - needs Svelte 5 migration
  it.skip('renders slot content', () => {
    // Svelte 5 uses snippets instead of slots
  });

  it('supports submit type', () => {
    const { container } = render(Button, {
      props: { type: 'submit' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveAttribute('type', 'submit');
  });

  it('applies all variant classes correctly', () => {
    const variantClasses = {
      primary: 'bg-blurple-500',
      secondary: 'bg-gray-600',
      danger: 'bg-red-500',
      ghost: 'bg-transparent'
    };
    const variants = Object.keys(variantClasses) as Array<keyof typeof variantClasses>;

    variants.forEach(variant => {
      const { container } = render(Button, { props: { variant } });
      const button = container.querySelector('button');
      expect(button).toHaveClass(variantClasses[variant]);
    });
  });
});
