import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import ComponentModal from './ComponentModal.svelte';

describe('ComponentModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const defaultRows = [
    {
      components: [
        {
          id: 'text1',
          type: 'text_input',
          custom_id: 'feedback_text',
          label: 'Feedback',
          placeholder: 'Tell us what you think...',
          required: true,
          style: 'paragraph',
          min_length: 10,
          max_length: 500
        }
      ]
    }
  ];

  it('does not render when open is false', () => {
    const { container } = render(ComponentModal, {
      props: { open: false, title: 'Test Modal', rows: defaultRows }
    });

    const backdrop = container.querySelector('.modal-backdrop');
    expect(backdrop).not.toBeInTheDocument();
  });

  it('renders when open is true', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, title: 'Feedback Form', rows: defaultRows }
    });

    const backdrop = container.querySelector('.modal-backdrop');
    expect(backdrop).toBeInTheDocument();
  });

  it('displays title', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, title: 'My Modal Title', rows: defaultRows }
    });

    expect(container).toHaveTextContent('My Modal Title');
  });

  it('renders text input components', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, rows: defaultRows }
    });

    const textarea = container.querySelector('textarea');
    expect(textarea).toBeInTheDocument();
  });

  it('renders text input with label', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, rows: defaultRows }
    });

    const label = container.querySelector('label');
    expect(label).toHaveTextContent('Feedback');
  });

  it('renders text input with placeholder', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, rows: defaultRows }
    });

    const textarea = container.querySelector('textarea');
    expect(textarea).toHaveAttribute('placeholder', 'Tell us what you think...');
  });

  it('renders required indicator', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, rows: defaultRows }
    });

    const required = container.querySelector('.required');
    expect(required).toBeInTheDocument();
  });

  it('renders short style as text input', () => {
    const shortStyleRows = [
      {
        components: [
          {
            id: 'text1',
            type: 'text_input',
            custom_id: 'name',
            label: 'Your Name',
            style: 'short',
            required: true
          }
        ]
      }
    ];

    const { container } = render(ComponentModal, {
      props: { open: true, rows: shortStyleRows }
    });

    const input = container.querySelector('input[type="text"]');
    expect(input).toBeInTheDocument();
    const textarea = container.querySelector('textarea');
    expect(textarea).not.toBeInTheDocument();
  });

  it('closes on Cancel button click', async () => {
    const closeHandler = vi.fn();
    const { container } = render(ComponentModal, {
      props: { open: true, title: 'Test', rows: defaultRows },
      events: { close: closeHandler }
    } as any);

    const cancelBtn = container.querySelector('.cancel-btn');
    await fireEvent.click(cancelBtn!);

    expect(closeHandler).toHaveBeenCalled();
  });

  it('closes on backdrop click', async () => {
    const closeHandler = vi.fn();
    const { container } = render(ComponentModal, {
      props: { open: true, title: 'Test', rows: defaultRows },
      events: { close: closeHandler }
    } as any);

    const backdrop = container.querySelector('.modal-backdrop');
    await fireEvent.click(backdrop!);

    expect(closeHandler).toHaveBeenCalled();
  });

  it('closes on Escape key', async () => {
    const closeHandler = vi.fn();
    render(ComponentModal, {
      props: { open: true, title: 'Test', rows: defaultRows },
      events: { close: closeHandler }
    } as any);

    await fireEvent.keyDown(window, { key: 'Escape' });

    expect(closeHandler).toHaveBeenCalled();
  });

  it('dispatches submit event with customId and values', async () => {
    const submitHandler = vi.fn();
    const { container } = render(ComponentModal, {
      props: { 
        open: true, 
        title: 'Test', 
        customId: 'modal_1',
        rows: defaultRows 
      },
      events: { submit: submitHandler }
    } as any);

    const textarea = container.querySelector('textarea');
    await fireEvent.input(textarea!, { target: { value: 'Great product!' } });

    const submitBtn = container.querySelector('.submit-btn');
    await fireEvent.click(submitBtn!);

    expect(submitHandler).toHaveBeenCalled();
    expect(submitHandler.mock.calls[0][0].detail.customId).toBe('modal_1');
  });

  it('renders multiple rows', () => {
    const multiRow = [
      { components: [{ id: '1', type: 'text_input', custom_id: 'name', label: 'Name' }] },
      { components: [{ id: '2', type: 'text_input', custom_id: 'email', label: 'Email' }] }
    ];

    const { container } = render(ComponentModal, {
      props: { open: true, rows: multiRow }
    });

    const labels = container.querySelectorAll('label');
    expect(labels.length).toBe(2);
  });

  it('applies danger style to submit button', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, modalType: 'danger', rows: defaultRows }
    });

    const submitBtn = container.querySelector('.submit-btn.danger');
    expect(submitBtn).toBeInTheDocument();
  });

  it('applies primary style to submit button by default', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, modalType: 'primary', rows: defaultRows }
    });

    const submitBtn = container.querySelector('.submit-btn.primary');
    expect(submitBtn).toBeInTheDocument();
  });

  it('closes on X button click', async () => {
    const closeHandler = vi.fn();
    const { container } = render(ComponentModal, {
      props: { open: true, title: 'Test', rows: defaultRows },
      events: { close: closeHandler }
    } as any);

    const closeBtn = container.querySelector('.close-btn');
    await fireEvent.click(closeBtn!);

    expect(closeHandler).toHaveBeenCalled();
  });

  it('respects min_length attribute', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, rows: defaultRows }
    });

    const textarea = container.querySelector('textarea');
    expect(textarea).toHaveAttribute('minlength', '10');
  });

  it('respects max_length attribute', () => {
    const { container } = render(ComponentModal, {
      props: { open: true, rows: defaultRows }
    });

    const textarea = container.querySelector('textarea');
    expect(textarea).toHaveAttribute('maxlength', '500');
  });
});
