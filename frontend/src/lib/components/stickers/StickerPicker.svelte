<script lang="ts">
	import { createEventDispatcher, onMount, onDestroy } from 'svelte';
	import { handleListKeyboard } from '$lib/utils/keyboard';
	import { currentServer } from '$lib/stores/servers';
	import { api } from '$lib/api';

	export let show = false;

	const dispatch = createEventDispatcher<{ select: { id: string; name: string; url: string }; close: void }>();

	interface Sticker {
		id: string;
		name: string;
		tags: string[];
		url: string;
		format: string;
		guild_id?: string;
	}

	let stickers: Sticker[] = [];
	let filteredStickers: Sticker[] = [];
	let searchQuery = '';
	let selectedCategory = 'all';
	let focusedIndex = -1;
	let pickerElement: HTMLDivElement;
	let searchInput: HTMLInputElement;
	let loading = false;
	let error = '';

	// Categories for organization
	const categories = [
		{ id: 'all', name: 'All Stickers', icon: '📦' },
		{ id: 'recent', name: 'Recently Used', icon: '🕒' },
		{ id: 'server', name: 'Server Stickers', icon: '🏠' },
	];

	const RECENT_STICKERS_KEY = 'hearth_recent_stickers';
	const MAX_RECENT_STICKERS = 24;

	let recentStickers: Sticker[] = [];

	// Load recent stickers from localStorage
	function loadRecentStickers() {
		try {
			const stored = localStorage.getItem(RECENT_STICKERS_KEY);
			if (stored) {
				recentStickers = JSON.parse(stored);
			}
		} catch (err) {
			console.error('[StickerPicker] Failed to load recent stickers:', err);
			recentStickers = [];
		}
	}

	// Save recent stickers to localStorage
	function saveRecentStickers() {
		try {
			localStorage.setItem(RECENT_STICKERS_KEY, JSON.stringify(recentStickers));
		} catch (err) {
			console.error('[StickerPicker] Failed to save recent stickers:', err);
		}
	}

	// Add sticker to recent
	function addToRecent(sticker: Sticker) {
		// Remove if already exists
		recentStickers = recentStickers.filter(s => s.id !== sticker.id);
		// Add to beginning
		recentStickers.unshift(sticker);
		// Limit to max
		if (recentStickers.length > MAX_RECENT_STICKERS) {
			recentStickers = recentStickers.slice(0, MAX_RECENT_STICKERS);
		}
		saveRecentStickers();
	}

	// Fetch stickers from API
	async function fetchStickers() {
		loading = true;
		error = '';

		try {
			// Get global stickers
			const globalResponse = await api.get<Sticker[]>('/stickers');
			const globalStickers: Sticker[] = globalResponse || [];

			// Get server stickers if we're in a server
			let serverStickers: Sticker[] = [];
			if ($currentServer) {
				try {
					const serverResponse = await api.get<Sticker[]>(`/servers/${$currentServer.id}/stickers`);
					serverStickers = serverResponse || [];
				} catch (err) {
					console.error('[StickerPicker] Failed to load server stickers:', err);
					serverStickers = [];
				}
			}

			// Combine stickers
			stickers = [...globalStickers, ...serverStickers];
			updateFilteredStickers();
		} catch (err) {
			console.error('Failed to fetch stickers:', err);
			error = 'Failed to load stickers';
			stickers = [];
		} finally {
			loading = false;
		}
	}

	function updateFilteredStickers() {
		if (searchQuery) {
			// Search by name or tags
			const query = searchQuery.toLowerCase();
			filteredStickers = stickers.filter(s => 
				s.name.toLowerCase().includes(query) ||
				s.tags?.some((tag: string) => tag.toLowerCase().includes(query))
			);
		} else if (selectedCategory === 'recent') {
			filteredStickers = recentStickers;
		} else if (selectedCategory === 'server') {
			filteredStickers = stickers.filter(s => s.guild_id === $currentServer?.id);
		} else {
			// All stickers
			filteredStickers = stickers;
		}
	}

	$: if (searchQuery !== undefined || selectedCategory) {
		updateFilteredStickers();
	}

	function selectSticker(sticker: Sticker) {
		addToRecent(sticker);
		dispatch('select', { id: sticker.id, name: sticker.name, url: sticker.url });
	}

	function handleClickOutside(event: MouseEvent) {
		if (show && pickerElement && !pickerElement.contains(event.target as Node)) {
			dispatch('close');
		}
	}

	function getStickerButtons(): HTMLElement[] {
		if (!pickerElement) return [];
		return Array.from(pickerElement.querySelectorAll<HTMLElement>('.sticker-btn'));
	}

	function focusStickerAt(index: number) {
		const buttons = getStickerButtons();
		if (buttons[index]) {
			buttons[index].focus();
			focusedIndex = index;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			dispatch('close');
			return;
		}

		const buttons = getStickerButtons();
		if (buttons.length === 0) return;

		const { handled, newIndex } = handleListKeyboard(event, focusedIndex, buttons.length, {
			wrap: true,
			gridNavigation: true,
			gridColumns: 5,
			onSelect: (idx) => {
				const sticker = filteredStickers[idx];
				if (sticker) selectSticker(sticker);
			},
			onEscape: () => dispatch('close')
		});

		if (handled && newIndex !== focusedIndex) {
			focusedIndex = newIndex;
		}
	}

	function handleSearchKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			focusedIndex = 0;
			setTimeout(() => focusStickerAt(0), 0);
		} else if (event.key === 'Escape') {
			dispatch('close');
		}
	}

	function handleStickerFocus(index: number) {
		focusedIndex = index;
	}

	onMount(() => {
		loadRecentStickers();
		fetchStickers();
		document.addEventListener('click', handleClickOutside);
	});

	onDestroy(() => {
		document.removeEventListener('click', handleClickOutside);
	});

	// Focus search when picker becomes visible
	$: if (show && searchInput) {
		setTimeout(() => searchInput?.focus(), 0);
	}

	// Re-fetch when server changes
	$: if ($currentServer) {
		fetchStickers();
	}
</script>

{#if show}
	<div 
		bind:this={pickerElement} 
		class="sticker-picker" 
		role="dialog" 
		aria-label="Sticker picker" 
		aria-modal="true"
		on:keydown={handleKeydown}
	>
		<!-- Header -->
		<div class="header">
			<h3 class="title">Stickers</h3>
		</div>

		<!-- Search -->
		<div class="search-container">
			<svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
				<path fill="currentColor" d="M21.707 20.293l-4.054-4.054A8.46 8.46 0 0 0 19.5 11c0-4.687-3.813-8.5-8.5-8.5S2.5 6.313 2.5 11s3.813 8.5 8.5 8.5a8.46 8.46 0 0 0 5.239-1.847l4.054 4.054a1 1 0 0 0 1.414-1.414zM11 17.5c-3.584 0-6.5-2.916-6.5-6.5S7.416 4.5 11 4.5s6.5 2.916 6.5 6.5-2.916 6.5-6.5 6.5z"/>
			</svg>
			<input
				bind:this={searchInput}
				type="text"
				placeholder="Search stickers..."
				bind:value={searchQuery}
				class="search-input"
				aria-label="Search stickers"
				on:keydown={handleSearchKeydown}
			/>
		</div>

		<!-- Categories -->
		<div class="categories" role="tablist">
			{#each categories as category}
				<button
					class="category-btn"
					class:active={selectedCategory === category.id}
					role="tab"
					aria-selected={selectedCategory === category.id}
					on:click={() => selectedCategory = category.id}
					type="button"
				>
					<span class="category-icon" aria-hidden="true">{category.icon}</span>
					<span class="category-name">{category.name}</span>
				</button>
			{/each}
		</div>

		<!-- Stickers Grid -->
		<div class="sticker-grid" role="tabpanel">
			{#if loading}
				<div class="loading">
					<div class="spinner"></div>
					<span>Loading stickers...</span>
				</div>
			{:else if error}
				<div class="error">
					<span>{error}</span>
					<button class="retry-btn" on:click={fetchStickers} type="button">Retry</button>
				</div>
			{:else if filteredStickers.length === 0}
				<div class="empty">
					{#if searchQuery}
						<span>No stickers found for "{searchQuery}"</span>
					{:else if selectedCategory === 'recent'}
						<span>No recent stickers</span>
					{:else}
						<span>No stickers available</span>
					{/if}
				</div>
			{:else}
				{#each filteredStickers as sticker, index}
					<button
						class="sticker-btn"
						on:click={() => selectSticker(sticker)}
						on:focus={() => handleStickerFocus(index)}
						title={sticker.name}
						aria-label={sticker.name}
						type="button"
					>
						<img 
							src={sticker.url} 
							alt={sticker.name}
							class="sticker-image"
							class:animated={sticker.format === 'GIF' || sticker.format === 'APNG'}
						/>
					</button>
				{/each}
			{/if}
		</div>
	</div>
{/if}

<style>
	.sticker-picker {
		position: absolute;
		bottom: 100%;
		right: 0;
		width: 352px;
		max-height: 400px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		box-shadow: 0 8px 16px rgba(0, 0, 0, 0.24);
		overflow: hidden;
		display: flex;
		flex-direction: column;
		margin-bottom: 8px;
		z-index: 100;
	}

	.header {
		padding: 12px 16px 8px;
	}

	.title {
		margin: 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.search-container {
		position: relative;
		padding: 0 12px 8px;
	}

	.search-icon {
		position: absolute;
		left: 22px;
		top: 50%;
		transform: translateY(-50%);
		color: var(--text-muted, #949ba4);
		pointer-events: none;
	}

	.search-input {
		width: 100%;
		padding: 8px 12px 8px 36px;
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: 4px;
		color: var(--text-primary, #f2f3f5);
		font-size: 13px;
		outline: none;
	}

	.search-input::placeholder {
		color: var(--text-muted, #949ba4);
	}

	.search-input:focus {
		box-shadow: 0 0 0 2px var(--brand-primary, #5865f2);
	}

	.categories {
		display: flex;
		padding: 0 8px;
		gap: 4px;
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}

	.category-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 8px 12px;
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		color: var(--text-secondary, #b5bac1);
		font-size: 12px;
		cursor: pointer;
		transition: color 0.15s, border-color 0.15s;
	}

	.category-btn:hover {
		color: var(--text-primary, #f2f3f5);
	}

	.category-btn.active {
		color: var(--text-primary, #f2f3f5);
		border-bottom-color: var(--brand-primary, #5865f2);
	}

	.category-icon {
		font-size: 14px;
	}

	.category-name {
		white-space: nowrap;
	}

	.sticker-grid {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 4px;
		min-height: 100px;
	}

	.sticker-btn {
		background: none;
		border: none;
		padding: 4px;
		border-radius: 4px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: background-color 0.15s;
	}

	.sticker-btn:hover {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.sticker-btn:focus {
		outline: none;
		background: var(--bg-modifier-selected, rgba(79, 84, 92, 0.24));
	}

	.sticker-image {
		width: 48px;
		height: 48px;
		object-fit: contain;
		image-rendering: -webkit-optimize-contrast;
	}

	.sticker-image.animated {
		image-rendering: -webkit-optimize-contrast;
	}

	.loading,
	.error,
	.empty {
		grid-column: 1 / -1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 24px;
		color: var(--text-muted, #949ba4);
		font-size: 13px;
		gap: 8px;
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--bg-tertiary, #1e1f22);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.retry-btn {
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: 4px;
		padding: 6px 12px;
		color: white;
		font-size: 12px;
		cursor: pointer;
	}

	.retry-btn:hover {
		opacity: 0.9;
	}
</style>
