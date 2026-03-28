<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let style: 'primary' | 'secondary' | 'success' | 'danger' | 'link' = 'primary';
	export let label: string = '';
	export let emoji: string = '';
	export let disabled: boolean = false;
	export let url: string = '';
	export let customId: string = '';

	const dispatch = createEventDispatcher();

	let loading = false;

	const styleClasses = {
		primary: 'bg-[#5865f2] hover:bg-[#4752c4] text-white',
		secondary: 'bg-[#4f545c] hover:bg-[#3d4148] text-white',
		success: 'bg-[#3ba55c] hover:bg-[#2d804b] text-white',
		danger: 'bg-[#da373c] hover:bg-[#b92b31] text-white',
		link: 'bg-transparent hover:underline text-[#00a8fc] p-0'
	};

	async function handleClick() {
		if (disabled || loading || style === 'link') return;
		
		loading = true;
		dispatch('click', { customId });
		
		// Reset loading state after a short delay (caller should handle actual API call)
		setTimeout(() => {
			loading = false;
		}, 1000);
	}
</script>

{#if style === 'link'}
	<a
		href={url}
		target="_blank"
		rel="noopener noreferrer"
		class="inline-flex items-center justify-center gap-2 px-4 py-2 rounded text-sm font-medium transition-colors {styleClasses[style]}"
		class:opacity-50={disabled}
		class:pointer-events-none={disabled}
	>
		{#if emoji}
			<span class="emoji" aria-hidden="true">{emoji}</span>
		{/if}
		{label}
	</a>
{:else}
	<button
		type="button"
		class="inline-flex items-center justify-center gap-2 px-4 py-2 rounded text-sm font-medium transition-colors {styleClasses[style]}"
		class:opacity-50={disabled || loading}
		class:cursor-not-allowed={disabled || loading}
		{disabled}
		on:click={handleClick}
		aria-disabled={disabled || loading}
	>
		{#if loading}
			<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none" aria-hidden="true">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
		{:else if emoji}
			<span class="emoji" aria-hidden="true">{emoji}</span>
		{/if}
		{label}
	</button>
{/if}

<style>
	.emoji {
		font-size: 1em;
		line-height: 1;
	}
</style>
