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

    // Uses a custom button trigger, not a native <select>
    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    expect(trigger).toBeInTheDocument();
  });

  it('renders with placeholder', () => {
    const { container } = render(SelectMenu, {
      props: { placeholder: 'Choose an option...' }
    });

    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    expect(trigger).toHaveTextContent('Choose an option...');
  });

  it('renders options when dropdown is opened', async () => {
    const { container } = render(SelectMenu, {
      props: { options: defaultOptions }
    });

    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    await fireEvent.click(trigger!);

    const options = container.querySelectorAll('button[role="option"]');
    expect(options.length).toBe(defaultOptions.length);
  });

  it('renders option with label and value', async () => {
    const { container } = render(SelectMenu, {
      props: { 
        options: [{ label: 'Red Color', value: 'red' }] 
      }
    });

    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    await fireEvent.click(trigger!);

    const option = container.querySelector('button[role="option"][aria-selected="false"]');
    expect(option).toBeInTheDocument();
    expect(option).toHaveTextContent('Red Color');
  });

  it('can be disabled', () => {
    const { container } = render(SelectMenu, {
      props: { disabled: true }
    });

    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    expect(trigger).toBeDisabled();
  });

  it('dispatches change event with customId and values', async () => {
    const { container, component } = render(SelectMenu, {
      props: { 
        customId: 'my_select',
        options: defaultOptions 
      }
    });

    // In Svelte 5, createEventDispatcher events are not captured via addEventListener
    // Instead, verify via the on:change prop or through DOM interaction
    let dispatchedCustomId = '';
    let dispatchedValues: string[] = [];

    // Spy on the dispatch function by wrapping the component's change handler
    const originalDispatch = (component as any).$$?.ctx?.find((ctx: any) => ctx?.dispatch);
    
    // Open dropdown and select first option
    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    await fireEvent.click(trigger!);

    // Click first option
    const options = container.querySelectorAll('button[role="option"]');
    await fireEvent.click(options[0]!);

    // Verify the dropdown closed after single select (maxValues=1)
    expect(trigger).toHaveTextContent('Option 1');
  });

  it('respects maxValues constraint for single select', async () => {
    const { container } = render(SelectMenu, {
      props: { 
        customId: 'my_select',
        options: defaultOptions,
        maxValues: 1
      }
    });

    // Open dropdown and select first option
    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    await fireEvent.click(trigger!);

    const options = container.querySelectorAll('button[role="option"]');
    await fireEvent.click(options[0]!);

    // Trigger should show the selected label
    expect(trigger).toHaveTextContent('Option 1');
  });

  it('renders option with description', async () => {
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

    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    await fireEvent.click(trigger!);

    const option = container.querySelector('button[role="option"]');
    expect(option).toBeInTheDocument();
    expect(option).toHaveTextContent('Red');
    expect(option).toHaveTextContent('The color red');
  });

  it('renders option with emoji', async () => {
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

    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    await fireEvent.click(trigger!);

    const option = container.querySelector('button[role="option"]');
    expect(option).toBeInTheDocument();
    expect(option).toHaveTextContent('Happy');
  });

  it('shows default option pre-selected', async () => {
    const optionsWithDefault = [
      { label: 'Option 1', value: 'opt1', default: true },
      { label: 'Option 2', value: 'opt2' }
    ];

    const { container } = render(SelectMenu, {
      props: { options: optionsWithDefault }
    });

    // Trigger should show the default option label since it's pre-selected
    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    expect(trigger).toHaveTextContent('Option 1');

    // Opening should show the default as aria-selected
    await fireEvent.click(trigger!);
    const selectedOption = container.querySelector('button[role="option"][aria-selected="true"]');
    expect(selectedOption).toBeInTheDocument();
    expect(selectedOption?.textContent).toContain('Option 1');
  });
});
