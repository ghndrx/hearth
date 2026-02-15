<script lang="ts">
	import { slide } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { createEventDispatcher } from 'svelte';

	export let name: string;
	export let collapsed = false;
	export let showAddButton = true;

	const dispatch = createEventDispatcher<{
		toggle: { collapsed: boolean };
		addChannel: void;
	}>();

	function handleToggle() {
		collapsed = !collapsed;
		dispatch('toggle', { collapsed });
	}

	function handleAddChannel(e: MouseEvent) {
		e.stopPropagation();
		dispatch('addChannel');
	}
</script>

<div class="channel-category" class:collapsed>
	<button
		class="category-header"
		on:click={handleToggle}
		aria-expanded={!collapsed}
		aria-controls="category-channels-{name}"
	>
		<svg
			viewBox="0 0 24 24"
			width="12"
			height="12"
			fill="currentColor"
			class="collapse-icon"
			class:rotated={!collapsed}
			aria-hidden="true"
		>
			<path d="M9.29 15.88L13.17 12 9.29 8.12a1 1 0 0 1 1.42-1.42l4.59 4.59a1 1 0 0 1 0 1.42l-4.59 4.59a1 1 0 0 1-1.42 0 1 1 0 0 1 0-1.42z"/>
		</svg>
		<span class="category-name">{name.toUpperCase()}</span>
	</button>

	{#if showAddButton}
		<button
			class="add-channel"
			title="Create Channel"
			aria-label="Create new channel in {name}"
			on:click={handleAddChannel}
		>
			<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden="true">
				<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
			</svg>
		</button>
	{/if}
</div>

{#if !collapsed}
	<div
		id="category-channels-{name}"
		class="category-channels"
		transition:slide={{ duration: 150, easing: cubicOut }}
	>
		<slot />
	</div>
{/if}

<style>
	.channel-category {
		display: flex;
		align-items: center;
		padding: var(--spacing-md) var(--spacing-sm) var(--spacing-xs) 2px;
		user-select: none;
	}

	.category-header {
		display: flex;
		align-items: center;
		gap: 2px;
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: var(--font-size-xs);
		font-weight: 600;
		letter-spacing: 0.02em;
		cursor: pointer;
		flex: 1;
		text-align: left;
		padding: 0;
		text-transform: uppercase;
		transition: color var(--transition-fast);
	}

	.category-header:hover {
		color: var(--text-normal);
	}

	.category-header:focus-visible {
		outline: none;
		box-shadow: 0 0 0 2px var(--blurple);
		border-radius: var(--radius-sm);
	}

	.collapse-icon {
		transition: transform var(--transition-fast);
		flex-shrink: 0;
	}

	.collapse-icon.rotated {
		transform: rotate(90deg);
	}

	.category-name {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.add-channel {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: 0;
		opacity: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		transition: opacity var(--transition-fast), color var(--transition-fast);
	}

	.add-channel:hover {
		color: var(--text-normal);
	}

	.add-channel:focus-visible {
		opacity: 1;
		outline: none;
		box-shadow: 0 0 0 2px var(--blurple);
		border-radius: var(--radius-sm);
	}

	.channel-category:hover .add-channel,
	.channel-category:focus-within .add-channel {
		opacity: 1;
	}

	.category-channels {
		display: flex;
		flex-direction: column;
	}
</style>
