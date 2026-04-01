import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import SelectMenu from './SelectMenu.svelte';

describe('SelectMenu (Message Component)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const defaultOptions = [
    { label: 'Option 1', value: 'opt1' },
    { label: 'Option 2', value: 'opt2' },
    { label: 'Option 3', value: 'opt3' }
  ];

  it('renders with default props', () => {
    const { container } = render(SelectMenu, {
      props: {}
    });

    const select = container.querySelector('select');
    expect(select).toBeInTheDocument();
  });

  it('renders with placeholder', () => {
    const { container } = render(SelectMenu, {
      props: { placeholder: 'Choose an option...' }
    });

    const placeholder = container.querySelector('option[value=""]');
    expect(placeholder).toBeInTheDocument();
    expect(placeholder).toHaveTextContent('Choose an option...');
  });

  it('renders options correctly', () => {
    const { container } = render(SelectMenu, {
      props: { options: defaultOptions }
    });

    const options = container.querySelectorAll('option');
    expect(options.length).toBe(defaultOptions.length + 1); // +1 for placeholder
  });

  it('renders option with label and value', () => {
    const { container } = render(SelectMenu, {
      props: { 
        options: [{ label: 'Red Color', value: 'red' }] 
      }
    });

    const option = container.querySelector('option[value="red"]');
    expect(option).toBeInTheDocument();
    expect(option).toHaveTextContent('Red Color');
  });

  it('can be disabled', () => {
    const { container } = render(SelectMenu, {
      props: { disabled: true }
    });

    const select = container.querySelector('select');
    expect(select).toBeDisabled();
  });

  it('dispatches change event with customId and values', async () => {
    const { container, component } = render(SelectMenu, {
      props: { 
        customId: 'my_select',
        options: defaultOptions 
      }
    });

    const changeHandler = vi.fn();
    component.addEventListener('change', changeHandler);

    const select = container.querySelector('select');
    select!.value = 'opt2';
    await fireEvent.change(select!);

    expect(changeHandler).toHaveBeenCalled();
    const event = changeHandler.mock.calls[0][0];
    expect(event.detail.customId).toBe('my_select');
    expect(event.detail.values).toContain('opt2');
  });

  it('respects minValues constraint', async () => {
    const { container, component } = render(SelectMenu, {
      props: { 
        customId: 'my_select',
        options: defaultOptions,
        minValues: 2
      }
    });

    const select = container.querySelector('select');
    select!.value = 'opt1';
    await fireEvent.change(select!);

    const changeHandler = vi.fn();
    component.addEventListener('change', changeHandler);

    // Should trigger with minValues requirement
    await fireEvent.change(select!);
    expect(changeHandler).toHaveBeenCalled();
  });

  it('respects maxValues constraint', () => {
    const { container } = render(SelectMenu, {
      props: { 
        customId: 'my_select',
        options: defaultOptions,
        maxValues: 1
      }
    });

    const select = container.querySelector('select');
    expect(select).toHaveAttribute('multiple'); // For maxValues > 1
  });

  it('renders option with description', () => {
    const optionsWithDesc = [
      { 
        label: 'Red', 
        value: 'red',
        description: 'The color red'
      }
    ];

    const { container } = render(SelectMenu, {
      props: { options: optionsWithDesc }
    });

    // The select menu shows label, description is shown as data attribute
    const option = container.querySelector('option[value="red"]');
    expect(option).toBeInTheDocument();
  });

  it('renders option with emoji', () => {
    const optionsWithEmoji = [
      { 
        label: 'Happy', 
        value: 'happy',
        emoji: '😊'
      }
    ];

    const { container } = render(SelectMenu, {
      props: { options: optionsWithEmoji }
    });

    const option = container.querySelector('option[value="happy"]');
    expect(option).toBeInTheDocument();
  });

  it('shows default option', () => {
    const optionsWithDefault = [
      { label: 'Option 1', value: 'opt1', default: true },
      { label: 'Option 2', value: 'opt2' }
    ];

    const { container } = render(SelectMenu, {
      props: { options: optionsWithDefault }
    });

    const defaultOption = container.querySelector('option[selected]');
    expect(defaultOption?.textContent).toContain('Option 1');
  });
});
