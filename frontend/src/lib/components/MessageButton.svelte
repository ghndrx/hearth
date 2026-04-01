<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Tooltip from './Tooltip.svelte';

	export let id: string;
	export let customId: string;
	export let label: string = '';
	export let style: 'primary' | 'secondary' | 'success' | 'danger' | 'link' = 'primary';
	export let disabled: boolean = false;
	export let url: string | undefined = undefined;
	export let emoji: string | undefined = undefined;

	const dispatch = createEventDispatcher<{
		click: { customId: string; componentId: string };
	}>();

	const styleClasses = {
		primary: 'bg-blurple-500 hover:bg-blurple-600 text-white',
		secondary: 'bg-gray-600 hover:bg-gray-700 text-white',
		success: 'bg-green-500 hover:bg-green-600 text-white',
		danger: 'bg-red-500 hover:bg-red-600 text-white',
		link: 'bg-transparent hover:underline text-blurple-400 hover:text-blurple-300'
	};

	function handleClick() {
		if (disabled) return;
		if (url && style === 'link') {
			window.open(url, '_blank');
			return;
		}
		dispatch('click', { customId, componentId: id });
	}

	$: buttonClass = `
		${styleClasses[style]}
		${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
		px-4 py-2 rounded font-medium transition-all duration-150
		focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-800
		active:scale-[0.97]
		inline-flex items-center justify-center gap-2
	`.trim();
</script>

{#if style === 'link' && url}
	<Tooltip text={label || url} position="top">
		<a
			href={url}
			target="_blank"
			rel="noopener noreferrer"
			class={buttonClass}
			aria-disabled={disabled}
		>
			{#if emoji}
				<span class="emoji" aria-hidden="true">{emoji}</span>
			{/if}
			{#if label}
				<span>{label}</span>
			{/if}
		</a>
	</Tooltip>
{:else}
	<Tooltip text={label || ''} position="top">
		<button
			type="button"
			class={buttonClass}
			{disabled}
			on:click={handleClick}
			aria-label={label}
		>
			{#if emoji}
				<span class="emoji" aria-hidden="true">{emoji}</span>
			{/if}
			{#if label}
				<span>{label}</span>
			{/if}
		</button>
	</Tooltip>
{/if}

<style>
	button, a {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		white-space: nowrap;
		font-size: 14px;
		line-height: 1;
	}

	.emoji {
		font-size: 16px;
		line-height: 1;
	}
</style>
