import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ChannelCategory from './ChannelCategory.svelte';

describe('ChannelCategory', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('renders with default props', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels'
			}
		});

		expect(container.querySelector('.channel-category')).toBeInTheDocument();
		expect(screen.getByText('TEXT CHANNELS')).toBeInTheDocument();
	});

	it('renders in expanded state by default', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels'
			}
		});

		const channels = container.querySelector('.category-channels');
		expect(channels).toBeInTheDocument();
	});

	it('can be initially collapsed', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels',
				collapsed: true
			}
		});

		const channels = container.querySelector('.category-channels');
		expect(channels).not.toBeInTheDocument();
	});

	it('toggles collapse state when clicked', async () => {
		const handleToggle = vi.fn();
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels'
			}
		});

		const header = container.querySelector('.category-header');
		expect(header).toHaveAttribute('aria-expanded', 'true');

		header?.addEventListener('toggle', handleToggle);
		await fireEvent.click(header!);

		// The component toggles its internal state and dispatches the event
		expect(handleToggle).toHaveBeenCalled();
	});

	it('shows add button by default', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels'
			}
		});

		const addButton = container.querySelector('.add-channel');
		expect(addButton).toBeInTheDocument();
		expect(addButton).toHaveAttribute('title', 'Create Channel');
		expect(addButton).toHaveAttribute('aria-label', 'Create new channel in Text Channels');
	});

	it('can hide add button', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels',
				showAddButton: false
			}
		});

		const addButton = container.querySelector('.add-channel');
		expect(addButton).not.toBeInTheDocument();
	});

	it('dispatches addChannel event when add button clicked', async () => {
		const handleAddChannel = vi.fn();
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels'
			}
		});

		const addButton = container.querySelector('.add-channel');
		addButton?.addEventListener('addChannel', handleAddChannel);
		await fireEvent.click(addButton!);

		expect(handleAddChannel).toHaveBeenCalled();
	});

	it('converts name to uppercase', () => {
		render(ChannelCategory, {
			props: {
				name: 'voice channels'
			}
		});

		expect(screen.getByText('VOICE CHANNELS')).toBeInTheDocument();
	});

	it('rotates icon when expanded', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels',
				collapsed: false
			}
		});

		const icon = container.querySelector('.collapse-icon');
		expect(icon).toHaveClass('rotated');
	});

	it('does not rotate icon when collapsed', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels',
				collapsed: true
			}
		});

		const icon = container.querySelector('.collapse-icon');
		expect(icon).not.toHaveClass('rotated');
	});

	it('has proper accessibility attributes', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels'
			}
		});

		const header = container.querySelector('.category-header');
		expect(header).toHaveAttribute('aria-expanded', 'true');
		expect(header).toHaveAttribute('aria-controls', 'category-channels-Text Channels');
	});

	it('renders children in slot', () => {
		const { container } = render(ChannelCategory, {
			props: {
				name: 'Text Channels'
			}
		});

		// The slot content container should exist when expanded
		const channels = container.querySelector('.category-channels');
		expect(channels).toBeInTheDocument();
	});
});
