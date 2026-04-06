<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let hasRaisedHand: boolean = false;
	export let disabled: boolean = false;
	export let loading: boolean = false;

	const dispatch = createEventDispatcher();

	function handleClick() {
		if (disabled || loading) return;
		dispatch('toggle', { raised: !hasRaisedHand });
	}
</script>

<button
	class="hand-raise-btn"
	class:raised={hasRaisedHand}
	{disabled}
	on:click={handleClick}
	aria-label={hasRaisedHand ? 'Lower hand' : 'Raise hand'}
	title={hasRaisedHand ? 'Lower hand' : 'Raise hand'}
>
	{#if loading}
		<svg class="spinner" viewBox="0 0 24 24" width="20" height="20">
			<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" fill="none" stroke-dasharray="30 70" />
		</svg>
	{:else if hasRaisedHand}
		<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
			<path d="M12.09 13.91c.27.27.7.27.97 0l2.47-2.47c.39-.39.11-1.06-.44-1.06h-1.77l1.89-5.66c.11-.33.05-.71-.17-.97-.22-.26-.55-.36-.87-.25L10.5 5.3c-.11-.05-.23-.08-.36-.08H7.5C6.67 5.22 6 5.89 6 6.71V13c0 .55.45 1 1 1h2.44c.55 0 1-.45 1-1v-.09H12l-.91.91zM19 12.5c0-1.5-.5-3-2-4s-3-2-3.5-4c-.11.33-.5.5-1 .5H7c-1.1 0-2 .9-2 2v2c0 1.1.9 2 2 2h5.5c1.93 0 3.5-1.57 3.5-3.5 0-.5-.11-.99-.31-1.44l.31-.06z"/>
		</svg>
	{:else}
		<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
			<path d="M12.09 13.91c.27.27.7.27.97 0l2.47-2.47c.39-.39.11-1.06-.44-1.06h-1.77l1.89-5.66c.11-.33.05-.71-.17-.97-.22-.26-.55-.36-.87-.25L10.5 5.3c-.11-.05-.23-.08-.36-.08H7.5C6.67 5.22 6 5.89 6 6.71V13c0 .55.45 1 1 1h2.44c.55 0 1-.45 1-1v-.09H12l-.91.91zM19 12.5c0-1.5-.5-3-2-4s-3-2-3.5-4c-.11.33-.5.5-1 .5H7c-1.1 0-2 .9-2 2v2c0 1.1.9 2 2 2h5.5c1.93 0 3.5-1.57 3.5-3.5 0-.5-.11-.99-.31-1.44l.31-.06z"/>
		</svg>
	{/if}
	<span class="btn-label">{hasRaisedHand ? 'Lower Hand' : 'Raise Hand'}</span>
</button>

<style>
	.hand-raise-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		background-color: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s ease, color 0.15s ease;
	}

	.hand-raise-btn:hover:not(:disabled) {
		background-color: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
		color: var(--text-normal, #f2f3f5);
	}

	.hand-raise-btn.raised {
		background-color: var(--yellow, #f0b232);
		color: #000;
	}

	.hand-raise-btn.raised:hover:not(:disabled) {
		background-color: var(--yellow-hover, #dfa02e);
	}

	.hand-raise-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-label {
		line-height: 1;
	}

	.spinner {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}
</style>
