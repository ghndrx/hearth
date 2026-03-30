<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';

	export let value = '';
	export let placeholder = 'Search servers...';
	export let suggestions: Array<{ type: string; value: string; count?: number }> = [];
	export let loading = false;
	export let debounceMs = 300;

	const dispatch = createEventDispatcher();

	let inputElement: HTMLInputElement;
	let showSuggestions = false;
	let debounceTimer: ReturnType<typeof setTimeout>;

	$: hasSuggestions = suggestions.length > 0;

	function handleInput(event: Event) {
		const target = event.target as HTMLInputElement;
		value = target.value;

		if (debounceTimer) {
			clearTimeout(debounceTimer);
		}

		debounceTimer = setTimeout(() => {
			dispatch('search', value);
		}, debounceMs);
	}

	function handleClear() {
		value = '';
		dispatch('search', '');
		dispatch('clear');
		inputElement?.focus();
	}

	function handleFocus() {
		showSuggestions = true;
		dispatch('focus');
	}

	function handleBlur() {
		// Delay to allow clicking on suggestions
		setTimeout(() => {
			showSuggestions = false;
		}, 200);
	}

	function selectSuggestion(suggestion: { type: string; value: string }) {
		value = suggestion.value;
		showSuggestions = false;
		dispatch('select', suggestion);
		dispatch('search', value);
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			showSuggestions = false;
			inputElement?.blur();
		} else if (event.key === 'Enter' && !showSuggestions) {
			dispatch('submit', value);
		}
	}

	onMount(() => {
		return () => {
			if (debounceTimer) {
				clearTimeout(debounceTimer);
			}
		};
	});
</script>

<div class="search-bar" class:has-value={value.length > 0} class:has-suggestions={showSuggestions && hasSuggestions}>
	<div class="search-input-container">
		<svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
		</svg>
		<input
			bind:this={inputElement}
			type="text"
			{placeholder}
			bind:value
			on:input={handleInput}
			on:focus={handleFocus}
			on:blur={handleBlur}
			on:keydown={handleKeydown}
			class="search-input"
			aria-label="Search for servers"
			aria-expanded={showSuggestions && hasSuggestions}
			aria-haspopup="listbox"
		/>
		{#if loading}
			<div class="loading-spinner"></div>
		{:else if value}
			<button
				class="clear-button"
				on:click={handleClear}
				aria-label="Clear search"
				tabindex="-1"
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		{/if}
	</div>

	{#if showSuggestions && hasSuggestions}
		<div class="suggestions" role="listbox" aria-label="Search suggestions">
			{#each suggestions as suggestion}
				<button
					class="suggestion-item"
					on:mousedown|preventDefault={() => selectSuggestion(suggestion)}
					role="option"
					aria-selected="false"
				>
					<span class="suggestion-icon">
						{#if suggestion.type === 'category'}
							🏷️
						{:else if suggestion.type === 'tag'}
							🏷️
						{:else}
							🔍
						{/if}
					</span>
					<span class="suggestion-value">{suggestion.value}</span>
					{#if suggestion.count}
						<span class="suggestion-count">{suggestion.count}</span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.search-bar {
		position: relative;
		width: 100%;
		max-width: 600px;
	}

	.search-input-container {
		position: relative;
		display: flex;
		align-items: center;
	}

	.search-icon {
		position: absolute;
		left: 14px;
		width: 20px;
		height: 20px;
		color: #6d6f78;
		pointer-events: none;
		transition: color 0.15s;
	}

	.search-input {
		width: 100%;
		padding: 12px 44px;
		background: #1e1f22;
		border: 2px solid transparent;
		border-radius: 8px;
		color: #f2f3f5;
		font-size: 14px;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.search-input::placeholder {
		color: #6d6f78;
	}

	.search-input:focus {
		outline: none;
		border-color: #5865f2;
		background: #2b2d31;
	}

	.search-input:focus + .search-icon,
	.search-input-container:focus-within .search-icon {
		color: #5865f2;
	}

	.clear-button {
		position: absolute;
		right: 10px;
		width: 24px;
		height: 24px;
		padding: 0;
		border: none;
		background: transparent;
		color: #6d6f78;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 4px;
		transition: color 0.15s, background-color 0.15s;
	}

	.clear-button:hover {
		color: #f2f3f5;
		background: #35373c;
	}

	.clear-button svg {
		width: 16px;
		height: 16px;
	}

	.loading-spinner {
		position: absolute;
		right: 14px;
		width: 18px;
		height: 18px;
		border: 2px solid #3f4147;
		border-top-color: #5865f2;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Suggestions dropdown */
	.suggestions {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		background: #2b2d31;
		border: 1px solid #1e1f22;
		border-radius: 8px;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
		z-index: 100;
		overflow: hidden;
	}

	.suggestion-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 10px 14px;
		border: none;
		background: transparent;
		color: #dbdee1;
		font-size: 14px;
		cursor: pointer;
		text-align: left;
		transition: background-color 0.1s;
	}

	.suggestion-item:hover {
		background: #35373c;
	}

	.suggestion-icon {
		font-size: 14px;
		opacity: 0.7;
	}

	.suggestion-value {
		flex: 1;
	}

	.suggestion-count {
		font-size: 12px;
		color: #6d6f78;
	}

	/* Focus styles for accessibility */
	.search-input:focus-visible {
		box-shadow: 0 0 0 3px rgba(88, 101, 242, 0.3);
	}

	.clear-button:focus-visible {
		outline: 2px solid #5865f2;
		outline-offset: 2px;
	}

	.suggestion-item:focus-visible {
		outline: none;
		background: #35373c;
	}
</style>
