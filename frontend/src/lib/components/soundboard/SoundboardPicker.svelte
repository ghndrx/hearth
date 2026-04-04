<script lang="ts">
	import { createEventDispatcher, onMount, onDestroy } from 'svelte';
	import { handleListKeyboard } from '$lib/utils/keyboard';
	import { currentServer } from '$lib/stores/servers';
	import { api } from '$lib/api';

	export let show = false;

	const dispatch = createEventDispatcher<{ play: { id: string; name: string; url: string; volume: number }; close: void }>();

	interface Sound {
		id: string;
		name: string;
		emoji_name?: string;
		volume: number;
		audio_url: string;
		duration_ms: number;
		available: boolean;
		guild_id?: string;
	}

	let sounds: Sound[] = [];
	let filteredSounds: Sound[] = [];
	let searchQuery = '';
	let selectedCategory = 'all';
	let focusedIndex = -1;
	let pickerElement: HTMLDivElement;
	let searchInput: HTMLInputElement;
	let loading = false;
	let error = '';
	let playingSoundId: string | null = null;
	let audioPreview: HTMLAudioElement | null = null;

	// Categories for organization
	const categories = [
		{ id: 'all', name: 'All Sounds', icon: '🔊' },
		{ id: 'recent', name: 'Recently Used', icon: '🕒' },
		{ id: 'server', name: 'Server Sounds', icon: '🏠' },
	];

	const RECENT_SOUNDS_KEY = 'hearth_recent_sounds';
	const MAX_RECENT_SOUNDS = 24;

	let recentSounds: Sound[] = [];

	// Load recent sounds from localStorage
	function loadRecentSounds() {
		try {
			const stored = localStorage.getItem(RECENT_SOUNDS_KEY);
			if (stored) {
				recentSounds = JSON.parse(stored);
			}
		} catch (err) {
			console.error('[SoundboardPicker] Failed to load recent sounds:', err);
			recentSounds = [];
		}
	}

	// Save recent sounds to localStorage
	function saveRecentSounds() {
		try {
			localStorage.setItem(RECENT_SOUNDS_KEY, JSON.stringify(recentSounds));
		} catch (err) {
			console.error('[SoundboardPicker] Failed to save recent sounds:', err);
		}
	}

	// Add sound to recent
	function addToRecent(sound: Sound) {
		// Remove if already exists
		recentSounds = recentSounds.filter(s => s.id !== sound.id);
		// Add to beginning
		recentSounds.unshift(sound);
		// Limit to max
		if (recentSounds.length > MAX_RECENT_SOUNDS) {
			recentSounds = recentSounds.slice(0, MAX_RECENT_SOUNDS);
		}
		saveRecentSounds();
	}

	// Fetch sounds from API
	async function fetchSounds() {
		loading = true;
		error = '';

		try {
			// Get default/global sounds
			const defaultResponse = await api.get<Sound[]>('/soundboard/defaults');
			const defaultSounds: Sound[] = defaultResponse || [];

			// Get server sounds if we're in a server
			let serverSounds: Sound[] = [];
			if ($currentServer) {
				try {
					const serverResponse = await api.get<Sound[]>(`/servers/${$currentServer.id}/soundboard`);
					serverSounds = serverResponse || [];
				} catch (err) {
					console.error('[SoundboardPicker] Failed to load server sounds:', err);
					serverSounds = [];
				}
			}

			// Combine sounds
			sounds = [...defaultSounds, ...serverSounds];
			updateFilteredSounds();
		} catch (err) {
			console.error('Failed to fetch sounds:', err);
			error = 'Failed to load sounds';
			sounds = [];
		} finally {
			loading = false;
		}
	}

	function updateFilteredSounds() {
		if (searchQuery) {
			// Search by name or emoji
			const query = searchQuery.toLowerCase();
			filteredSounds = sounds.filter(s => 
				s.name.toLowerCase().includes(query) ||
				s.emoji_name?.toLowerCase().includes(query)
			);
		} else if (selectedCategory === 'recent') {
			filteredSounds = recentSounds;
		} else if (selectedCategory === 'server') {
			filteredSounds = sounds.filter(s => s.guild_id === $currentServer?.id);
		} else {
			// All sounds
			filteredSounds = sounds;
		}
	}

	$: if (searchQuery !== undefined || selectedCategory) {
		updateFilteredSounds();
	}

	function playSound(sound: Sound) {
		// Stop any currently playing preview
		if (audioPreview) {
			audioPreview.pause();
			audioPreview = null;
		}

		// Play preview
		audioPreview = new Audio(sound.audio_url);
		audioPreview.volume = sound.volume;
		audioPreview.play().catch(err => {
			console.error('Failed to play sound preview:', err);
		});

		playingSoundId = sound.id;

		// Stop preview after duration
		setTimeout(() => {
			if (playingSoundId === sound.id) {
				audioPreview?.pause();
				audioPreview = null;
				playingSoundId = null;
			}
		}, sound.duration_ms);

		addToRecent(sound);
		dispatch('play', { id: sound.id, name: sound.name, url: sound.audio_url, volume: sound.volume });
	}

	function handleClickOutside(event: MouseEvent) {
		if (show && pickerElement && !pickerElement.contains(event.target as Node)) {
			dispatch('close');
		}
	}

	function getSoundButtons(): HTMLElement[] {
		if (!pickerElement) return [];
		return Array.from(pickerElement.querySelectorAll<HTMLElement>('.sound-btn'));
	}

	function focusSoundAt(index: number) {
		const buttons = getSoundButtons();
		if (buttons[index]) {
			buttons[index].focus();
			focusedIndex = index;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			// Stop preview
			if (audioPreview) {
				audioPreview.pause();
				audioPreview = null;
				playingSoundId = null;
			}
			dispatch('close');
			return;
		}

		const buttons = getSoundButtons();
		if (buttons.length === 0) return;

		const { handled, newIndex } = handleListKeyboard(event, focusedIndex, buttons.length, {
			wrap: true,
			gridNavigation: true,
			gridColumns: 5,
			onSelect: (idx) => {
				const sound = filteredSounds[idx];
				if (sound) playSound(sound);
			},
			onEscape: () => {
				if (audioPreview) {
					audioPreview.pause();
					audioPreview = null;
					playingSoundId = null;
				}
				dispatch('close');
			}
		});

		if (handled && newIndex !== focusedIndex) {
			focusedIndex = newIndex;
		}
	}

	function handleSearchKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			focusedIndex = 0;
			setTimeout(() => focusSoundAt(0), 0);
		} else if (event.key === 'Escape') {
			dispatch('close');
		}
	}

	function handleSoundFocus(index: number) {
		focusedIndex = index;
	}

	onMount(() => {
		loadRecentSounds();
		fetchSounds();
		document.addEventListener('click', handleClickOutside);
	});

	onDestroy(() => {
		document.removeEventListener('click', handleClickOutside);
		if (audioPreview) {
			audioPreview.pause();
		}
	});

	// Focus search when picker becomes visible
	$: if (show && searchInput) {
		setTimeout(() => searchInput?.focus(), 0);
	}

	// Re-fetch when server changes
	$: if ($currentServer) {
		fetchSounds();
	}
</script>

{#if show}
	<div 
		bind:this={pickerElement} 
		class="soundboard-picker" 
		role="dialog" 
		aria-label="Soundboard picker" 
		aria-modal="true"
		on:keydown={handleKeydown}
	>
		<!-- Header -->
		<div class="header">
			<h3 class="title">Soundboard</h3>
		</div>

		<!-- Search -->
		<div class="search-container">
			<svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
				<path fill="currentColor" d="M21.707 20.293l-4.054-4.054A8.46 8.46 0 0 0 19.5 11c0-4.687-3.813-8.5-8.5-8.5S2.5 6.313 2.5 11s3.813 8.5 8.5 8.5a8.46 8.46 0 0 0 5.239-1.847l4.054 4.054a1 1 0 0 0 1.414-1.414zM11 17.5c-3.584 0-6.5-2.916-6.5-6.5S7.416 4.5 11 4.5s6.5 2.916 6.5 6.5-2.916 6.5-6.5 6.5z"/>
			</svg>
			<input
				bind:this={searchInput}
				type="text"
				placeholder="Search sounds..."
				bind:value={searchQuery}
				class="search-input"
				aria-label="Search sounds"
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

		<!-- Sounds Grid -->
		<div class="sound-grid" role="tabpanel">
			{#if loading}
				<div class="loading">
					<div class="spinner"></div>
					<span>Loading sounds...</span>
				</div>
			{:else if error}
				<div class="error">
					<span>{error}</span>
					<button class="retry-btn" on:click={fetchSounds} type="button">Retry</button>
				</div>
			{:else if filteredSounds.length === 0}
				<div class="empty">
					{#if searchQuery}
						<span>No sounds found for "{searchQuery}"</span>
					{:else if selectedCategory === 'recent'}
						<span>No recent sounds</span>
					{:else}
						<span>No sounds available</span>
					{/if}
				</div>
			{:else}
				{#each filteredSounds as sound, index}
					<button
						class="sound-btn"
						class:playing={playingSoundId === sound.id}
						on:click={() => playSound(sound)}
						on:focus={() => handleSoundFocus(index)}
						title="{sound.name}{index < 9 ? ` (${getHotkeyLabel(index)})` : ''}"
						aria-label={sound.name}
						type="button"
					>
						{#if index < 9}
							<span class="hotkey-badge">{getHotkeyLabel(index)}</span>
						{/if}
						<span class="sound-emoji" aria-hidden="true">{sound.emoji_name || '🔊'}</span>
						<span class="sound-name">{sound.name}</span>
						{#if playingSoundId === sound.id}
							<span class="playing-indicator">
								<span class="bar"></span>
								<span class="bar"></span>
								<span class="bar"></span>
							</span>
						{/if}
					</button>
				{/each}
			{/if}
		</div>
	</div>
{/if}

<style>
	.soundboard-picker {
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
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.title {
		margin: 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.hotkey-hint {
		font-size: 11px;
		color: var(--text-muted, #949ba4);
		background: var(--bg-tertiary, #1e1f22);
		padding: 2px 6px;
		border-radius: 4px;
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

	.sound-grid {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 4px;
		min-height: 100px;
	}

	.sound-btn {
		background: none;
		border: none;
		padding: 8px 4px;
		border-radius: 4px;
		cursor: pointer;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		transition: background-color 0.15s;
		gap: 4px;
		position: relative;
	}

	.sound-btn:hover {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.sound-btn:focus {
		outline: none;
		background: var(--bg-modifier-selected, rgba(79, 84, 92, 0.24));
	}

	.sound-btn.playing {
		background: var(--bg-modifier-selected, rgba(79, 84, 92, 0.24));
	}

	.sound-btn {
		position: relative;
	}

	.hotkey-badge {
		position: absolute;
		top: 2px;
		left: 2px;
		font-size: 9px;
		font-weight: 600;
		color: var(--text-muted, #949ba4);
		background: var(--bg-tertiary, #1e1f22);
		padding: 1px 3px;
		border-radius: 3px;
		line-height: 1.2;
	}

	.sound-emoji {
		font-size: 24px;
		line-height: 1;
	}

	.sound-name {
		font-size: 10px;
		color: var(--text-secondary, #b5bac1);
		text-align: center;
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.playing-indicator {
		position: absolute;
		top: 2px;
		right: 2px;
		display: flex;
		gap: 1px;
		align-items: flex-end;
		height: 8px;
	}

	.playing-indicator .bar {
		width: 2px;
		background: var(--brand-primary, #5865f2);
		animation: soundbar 0.5s ease-in-out infinite alternate;
	}

	.playing-indicator .bar:nth-child(1) {
		height: 4px;
		animation-delay: 0s;
	}

	.playing-indicator .bar:nth-child(2) {
		height: 8px;
		animation-delay: 0.15s;
	}

	.playing-indicator .bar:nth-child(3) {
		height: 5px;
		animation-delay: 0.3s;
	}

	@keyframes soundbar {
		from { transform: scaleY(0.3); }
		to { transform: scaleY(1); }
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
