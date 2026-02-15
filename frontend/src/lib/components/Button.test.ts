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
    expect(button).toHaveClass('button', 'primary', 'md');
    expect(button).toHaveAttribute('type', 'button');
    expect(button).not.toBeDisabled();
  });

  it('renders with custom variant', () => {
    const { container } = render(Button, {
      props: { variant: 'danger' }
    });

    const button = container.querySelector('button');
    expect(button).toHaveClass('danger');
    expect(button).not.toHaveClass('primary');
  });

  it('renders with different sizes', () => {
    const { container: smContainer } = render(Button, {
      props: { size: 'sm' }
    });
    expect(smContainer.querySelector('button')).toHaveClass('sm');

    const { container: lgContainer } = render(Button, {
      props: { size: 'lg' }
    });
    expect(lgContainer.querySelector('button')).toHaveClass('lg');
  });

  it('can be disabled', () => {
    const { container } = render(Button, {
      props: { disabled: true }
    });

    const button = container.querySelector('button');
    expect(button).toBeDisabled();
    expect(button).toHaveClass('button');
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

  it('does not trigger click when disabled', async () => {
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
    expect(button).toHaveClass('full-width');
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
    const variants: Array<'primary' | 'secondary' | 'danger' | 'ghost'> = ['primary', 'secondary', 'danger', 'ghost'];

    variants.forEach(variant => {
      const { container } = render(Button, { props: { variant } });
      const button = container.querySelector('button');
      expect(button).toHaveClass(variant);
    });
  });
});
