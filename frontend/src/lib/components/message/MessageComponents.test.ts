import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import MessageComponents from './MessageComponents.svelte';

describe('MessageComponents', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const buttonComponents = [
    {
      type: 'button',
      style: 'primary',
      label: 'Accept',
      custom_id: 'accept_btn',
      disabled: false
    },
    {
      type: 'button',
      style: 'secondary',
      label: 'Decline',
      custom_id: 'decline_btn',
      disabled: false
    }
  ];

  const selectMenuComponent = [
    {
      type: 'select_menu',
      custom_id: 'color_select',
      placeholder: 'Pick a color',
      options: [
        { label: 'Red', value: 'red' },
        { label: 'Green', value: 'green' },
        { label: 'Blue', value: 'blue' }
      ],
      disabled: false
    }
  ];

  it('renders with empty components', () => {
    const { container } = render(MessageComponents, {
      props: { components: [] }
    });

    const group = container.querySelector('[role="group"]');
    expect(group).toBeInTheDocument();
    // The group may contain whitespace-only text nodes, so check for button/select elements instead
    const buttons = container.querySelectorAll('button');
    const selectMenus = container.querySelectorAll('.select-menu-container');
    expect(buttons.length).toBe(0);
    expect(selectMenus.length).toBe(0);
  });

  it('renders button components', () => {
    const { container } = render(MessageComponents, {
      props: { components: buttonComponents }
    });

    const buttons = container.querySelectorAll('button');
    expect(buttons.length).toBe(2);
  });

  it('renders button with correct label', () => {
    const { container } = render(MessageComponents, {
      props: { components: buttonComponents }
    });

    expect(container).toHaveTextContent('Accept');
    expect(container).toHaveTextContent('Decline');
  });

  it('renders select menu component', () => {
    const { container } = render(MessageComponents, {
      props: { components: selectMenuComponent }
    });

    // SelectMenu uses a custom button-based dropdown, not a native <select> element
    const selectMenu = container.querySelector('.select-menu-container');
    expect(selectMenu).toBeInTheDocument();
  });



  it('renders disabled button', () => {
    const disabledComponents = [
      {
        type: 'button',
        style: 'primary',
        label: 'Disabled',
        custom_id: 'disabled_btn',
        disabled: true
      }
    ];

    const { container } = render(MessageComponents, {
      props: { components: disabledComponents }
    });

    const button = container.querySelector('button');
    expect(button).toBeDisabled();
  });

  it('renders button with emoji', () => {
    const emojiComponents = [
      {
        type: 'button',
        style: 'primary',
        label: 'Like',
        emoji_name: '👍',
        custom_id: 'like_btn',
        disabled: false
      }
    ];

    const { container } = render(MessageComponents, {
      props: { components: emojiComponents }
    });

    expect(container).toHaveTextContent('👍');
  });

  it('renders link button as anchor', () => {
    const linkComponents = [
      {
        type: 'button',
        style: 'link',
        label: 'Visit Site',
        url: 'https://example.com',
        custom_id: 'link_btn',
        disabled: false
      }
    ];

    const { container } = render(MessageComponents, {
      props: { components: linkComponents }
    });

    const anchor = container.querySelector('a');
    expect(anchor).toBeInTheDocument();
    expect(anchor).toHaveAttribute('href', 'https://example.com');
  });

  it('renders multiple component types together', () => {
    const mixedComponents = [
      {
        type: 'button',
        style: 'primary',
        label: 'OK',
        custom_id: 'ok_btn',
        disabled: false
      },
      {
        type: 'select_menu',
        custom_id: 'choice',
        options: [{ label: 'A', value: 'a' }],
        disabled: false
      }
    ];

    const { container } = render(MessageComponents, {
      props: { components: mixedComponents }
    });

    const button = container.querySelector('button');
    const selectMenu = container.querySelector('.select-menu-container');
    expect(button).toBeInTheDocument();
    expect(selectMenu).toBeInTheDocument();
  });

  it('handles components with snake_case properties', () => {
    const snakeCaseComponents = [
      {
        type: 'button',
        style: 'primary',
        custom_id: 'snake_btn',
        label: 'Snake Case',
        emoji_name: '🐍',
        disabled: false
      }
    ];

    const { container } = render(MessageComponents, {
      props: { components: snakeCaseComponents }
    });

    expect(container).toHaveTextContent('Snake Case');
  });

  it('handles components with camelCase properties', () => {
    const camelCaseComponents = [
      {
        type: 'button',
        style: 'primary',
        customId: 'camel_btn',
        label: 'Camel Case',
        emoji: '🐪',
        disabled: false
      }
    ];

    const { container } = render(MessageComponents, {
      props: { components: camelCaseComponents }
    });

    expect(container).toHaveTextContent('Camel Case');
  });

  it('renders options in select menu when dropdown opens', async () => {
    const { container } = render(MessageComponents, {
      props: { components: selectMenuComponent }
    });

    // Open the dropdown by clicking the trigger button
    const trigger = container.querySelector('button[aria-haspopup="listbox"]');
    expect(trigger).toBeInTheDocument();
    
    if (trigger) {
      await fireEvent.click(trigger);
      
      // Check that options are rendered inside the dropdown
      const options = container.querySelectorAll('button[role="option"]');
      expect(options.length).toBe(3); // 3 options from selectMenuComponent
    }
  });
});
