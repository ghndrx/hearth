<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Button from './Button.svelte';
	import SelectMenu from './SelectMenu.svelte';

	export let components: any[] = [];

	const dispatch = createEventDispatcher();

	function handleButtonClick(event: CustomEvent<{ customId: string }>) {
		dispatch('componentClick', { customId: event.detail.customId });
	}

	function handleSelectChange(event: CustomEvent<{ customId: string; values: string[] }>) {
		dispatch('componentChange', { customId: event.detail.customId, values: event.detail.values });
	}
</script>

<div class="flex flex-wrap items-center gap-2" role="group" aria-label="Message components">
	{#each components as component}
		{#if component.type === 'button'}
			<Button
				style={component.style || 'primary'}
				label={component.label || ''}
				emoji={component.emoji_name || component.emoji || ''}
				disabled={component.disabled || false}
				url={component.url || ''}
				customId={component.custom_id || component.customId || ''}
				on:click={handleButtonClick}
			/>
		{:else if component.type === 'select_menu'}
			<SelectMenu
				customId={component.custom_id || component.customId || ''}
				options={component.options || []}
				placeholder={component.placeholder || 'Select an option'}
				minValues={component.min_values || component.minValues || 1}
				maxValues={component.max_values || component.maxValues || 1}
				disabled={component.disabled || false}
				on:change={handleSelectChange}
			/>
		{/if}
	{/each}
</div>
