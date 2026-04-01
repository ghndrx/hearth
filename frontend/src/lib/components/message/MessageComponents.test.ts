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
    expect(group).toBeEmptyDOMElement();
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

    const select = container.querySelector('select');
    expect(select).toBeInTheDocument();
  });

  it('dispatches componentClick event when button is clicked', async () => {
    const { container, component } = render(MessageComponents, {
      props: { components: buttonComponents, messageId: 'msg123', channelId: 'ch123' }
    });

    const clickHandler = vi.fn();
    component.addEventListener('componentClick', clickHandler);

    const buttons = container.querySelectorAll('button');
    await fireEvent.click(buttons[0]);

    expect(clickHandler).toHaveBeenCalled();
  });

  it('dispatches componentChange event when select changes', async () => {
    const { container, component } = render(MessageComponents, {
      props: { components: selectMenuComponent, messageId: 'msg123', channelId: 'ch123' }
    });

    const changeHandler = vi.fn();
    component.addEventListener('componentChange', changeHandler);

    const select = container.querySelector('select');
    select!.value = 'red';
    await fireEvent.change(select!);

    expect(changeHandler).toHaveBeenCalled();
  });

  it('passes custom_id in button click event', async () => {
    const { container, component } = render(MessageComponents, {
      props: { components: buttonComponents }
    });

    const clickHandler = vi.fn();
    component.addEventListener('componentClick', clickHandler);

    const buttons = container.querySelectorAll('button');
    await fireEvent.click(buttons[0]);

    expect(clickHandler).toHaveBeenCalled();
    expect(clickHandler.mock.calls[0][0].detail.customId).toBe('accept_btn');
  });

  it('passes custom_id in select change event', async () => {
    const { container, component } = render(MessageComponents, {
      props: { components: selectMenuComponent }
    });

    const changeHandler = vi.fn();
    component.addEventListener('componentChange', changeHandler);

    const select = container.querySelector('select');
    select!.value = 'green';
    await fireEvent.change(select!);

    expect(changeHandler).toHaveBeenCalled();
    expect(changeHandler.mock.calls[0][0].detail.customId).toBe('color_select');
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
    const select = container.querySelector('select');
    expect(button).toBeInTheDocument();
    expect(select).toBeInTheDocument();
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

  it('renders options in select menu', () => {
    const { container } = render(MessageComponents, {
      props: { components: selectMenuComponent }
    });

    const options = container.querySelectorAll('option');
    expect(options.length).toBe(4); // 3 options + 1 placeholder
  });
});
