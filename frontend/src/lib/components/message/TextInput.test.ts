import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import TextInput from './TextInput.svelte';

describe('TextInput (Message Component)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders with default props', () => {
    const { container } = render(TextInput, {
      props: {}
    });

    const input = container.querySelector('input[type="text"]');
    expect(input).toBeInTheDocument();
  });

  it('renders with label', () => {
    const { container } = render(TextInput, {
      props: { label: 'Your Name' }
    });

    const label = container.querySelector('label');
    expect(label).toHaveTextContent('Your Name');
  });

  it('renders with placeholder', () => {
    const { container } = render(TextInput, {
      props: { placeholder: 'Enter your name...' }
    });

    const input = container.querySelector('input');
    expect(input).toHaveAttribute('placeholder', 'Enter your name...');
  });

  it('renders short style as text input', () => {
    const { container } = render(TextInput, {
      props: { style: 'short' }
    });

    const input = container.querySelector('input[type="text"]');
    expect(input).toBeInTheDocument();
    const textarea = container.querySelector('textarea');
    expect(textarea).not.toBeInTheDocument();
  });

  it('renders paragraph style as textarea', () => {
    const { container } = render(TextInput, {
      props: { style: 'paragraph' }
    });

    const textarea = container.querySelector('textarea');
    expect(textarea).toBeInTheDocument();
    const input = container.querySelector('input[type="text"]');
    expect(input).not.toBeInTheDocument();
  });

  it('can be disabled', () => {
    const { container } = render(TextInput, {
      props: { disabled: true }
    });

    const input = container.querySelector('input');
    expect(input).toBeDisabled();
  });

  it('respects minLength constraint', () => {
    const { container } = render(TextInput, {
      props: { minLength: 5 }
    });

    const input = container.querySelector('input');
    expect(input).toHaveAttribute('minlength', '5');
  });

  it('respects maxLength constraint', () => {
    const { container } = render(TextInput, {
      props: { maxLength: 100 }
    });

    const input = container.querySelector('input');
    expect(input).toHaveAttribute('maxlength', '100');
  });

  it('shows required indicator', () => {
    const { container } = render(TextInput, {
      props: { required: true, label: 'Required Field' }
    });

    const input = container.querySelector('input');
    expect(input).toHaveAttribute('required');
  });

  it('does not show required attribute when not required', () => {
    const { container } = render(TextInput, {
      props: { required: false, label: 'Optional Field' }
    });

    const input = container.querySelector('input');
    expect(input).not.toHaveAttribute('required');
  });

  it('dispatches submit event on button click', async () => {
    const submitHandler = vi.fn();
    const { container } = render(TextInput, {
      props: { customId: 'my_text_input', onsubmit: submitHandler }
    });

    const input = container.querySelector('input');
    await fireEvent.input(input!, { target: { value: 'Test value' } });

    const submitBtn = container.querySelector('button[type="button"]');
    await fireEvent.click(submitBtn!);

    expect(submitHandler).toHaveBeenCalled();
  });

  it('dispatches submit event on Enter key', async () => {
    const submitHandler = vi.fn();
    const { container } = render(TextInput, {
      props: { customId: 'my_text_input', onsubmit: submitHandler }
    });

    const input = container.querySelector('input');
    await fireEvent.input(input!, { target: { value: 'Test value' } });
    await fireEvent.keyDown(input!, { key: 'Enter' });

    expect(submitHandler).toHaveBeenCalled();
    const event = submitHandler.mock.calls[0][0];
    expect(event.customId).toBe('my_text_input');
    expect(event.value).toBe('Test value');
  });

  it('does not submit empty value when required', async () => {
    const submitHandler = vi.fn();
    const { container } = render(TextInput, {
      props: { customId: 'my_text_input', required: true, onsubmit: submitHandler }
    });

    const input = container.querySelector('input');
    await fireEvent.input(input!, { target: { value: '' } });

    const submitBtn = container.querySelector('button[type="button"]');
    await fireEvent.click(submitBtn!);

    expect(submitHandler).not.toHaveBeenCalled();
  });

  it('does not submit when below minLength', async () => {
    const submitHandler = vi.fn();
    const { container } = render(TextInput, {
      props: { customId: 'my_text_input', minLength: 10, onsubmit: submitHandler }
    });

    const input = container.querySelector('input');
    await fireEvent.input(input!, { target: { value: 'short' } });

    const submitBtn = container.querySelector('button[type="button"]');
    await fireEvent.click(submitBtn!);

    expect(submitHandler).not.toHaveBeenCalled();
  });

  it('submits when value meets minLength', async () => {
    const submitHandler = vi.fn();
    const { container } = render(TextInput, {
      props: { customId: 'my_text_input', minLength: 5, onsubmit: submitHandler }
    });

    const input = container.querySelector('input');
    await fireEvent.input(input!, { target: { value: 'long enough value' } });

    const submitBtn = container.querySelector('button[type="button"]');
    await fireEvent.click(submitBtn!);

    expect(submitHandler).toHaveBeenCalled();
  });

  it('respects disabled state for submission', async () => {
    const submitHandler = vi.fn();
    const { container } = render(TextInput, {
      props: { customId: 'my_text_input', disabled: true, onsubmit: submitHandler }
    });

    const input = container.querySelector('input');
    await fireEvent.input(input!, { target: { value: 'Test' } });

    const submitBtn = container.querySelector('button[type="button"]');
    await fireEvent.click(submitBtn!);

    expect(submitHandler).not.toHaveBeenCalled();
  });

  it('applies initial value', () => {
    const { container } = render(TextInput, {
      props: { value: 'Initial text' }
    });

    const input = container.querySelector('input');
    expect(input).toHaveValue('Initial text');
  });

  it('shows character count for minLength', () => {
    const { container } = render(TextInput, {
      props: { minLength: 10 }
    });

    const charCount = container.querySelector('.text-xs');
    expect(charCount).toHaveTextContent('Min 10 characters');
  });

  it('shows character count for maxLength', () => {
    const { container } = render(TextInput, {
      props: { maxLength: 100 }
    });

    const charCount = container.querySelector('.text-xs');
    expect(charCount).toHaveTextContent('Max 100 characters');
  });

  it('shows character count for both min and max', () => {
    const { container } = render(TextInput, {
      props: { minLength: 5, maxLength: 100 }
    });

    const charCount = container.querySelector('.text-xs');
    expect(charCount).toHaveTextContent('5 - 100 characters');
  });
});
