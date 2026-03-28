<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import ActionRow from './ActionRow.svelte';
	import Button from './Button.svelte';
	import SelectMenu from './SelectMenu.svelte';
	import TextInput from './TextInput.svelte';
	import { api } from '$lib/api';

	export let components: any[] = [];
	export let messageId: string = '';
	export let channelId: string = '';

	const dispatch = createEventDispatcher();

	interface ComponentData {
		type: string;
		style?: string;
		label?: string;
		custom_id?: string;
		url?: string;
		disabled?: boolean;
		emoji?: string;
		emoji_name?: string;
		options?: Array<{
			label: string;
			value: string;
			description?: string;
			emoji?: string;
			default?: boolean;
		}>;
		min_values?: number;
		max_values?: number;
		placeholder?: string;
		required?: boolean;
		value?: string;
		min_length?: number;
		max_length?: number;
	}

	// Group components by action rows
	// Components can be:
	// - action_row (contains child components)
	// - button
	// - select_menu
	// - text_input
	let rows: ComponentData[][] = [];

	$: {
		rows = [];
		let currentRow: ComponentData[] = [];
		
		for (const comp of components) {
			if (comp.type === 'action_row') {
				// Push current row if not empty
				if (currentRow.length > 0) {
					rows.push(currentRow);
					currentRow = [];
				}
				// Action rows contain child components
				const actionRowComponents = comp.components || [];
				rows.push(actionRowComponents);
			} else {
				// Other components go directly in a row
				if (currentRow.length > 0) {
					rows.push(currentRow);
					currentRow = [];
				}
				rows.push([comp]);
			}
		}
		
		if (currentRow.length > 0) {
			rows.push(currentRow);
		}
	}

	async function handleComponentClick(customId: string) {
		dispatch('interaction', { type: 'click', customId, messageId, channelId });
		
		try {
			await api.post('/interactions/components', {
				custom_id: customId,
				message_id: messageId,
				channel_id: channelId
			});
		} catch (error) {
			console.error('Component interaction failed:', error);
		}
	}

	async function handleComponentChange(customId: string, values: string[]) {
		dispatch('interaction', { type: 'change', customId, values, messageId, channelId });
		
		try {
			await api.post('/interactions/components', {
				custom_id: customId,
				values,
				message_id: messageId,
				channel_id: channelId
			});
		} catch (error) {
			console.error('Component interaction failed:', error);
		}
	}

	async function handleTextSubmit(customId: string, value: string) {
		dispatch('interaction', { type: 'submit', customId, value, messageId, channelId });
		
		try {
			await api.post('/interactions/components', {
				custom_id: customId,
				values: [value],
				message_id: messageId,
				channel_id: channelId
			});
		} catch (error) {
			console.error('Component interaction failed:', error);
		}
	}

	function handleButtonClick(event: CustomEvent<{ customId: string }>) {
		handleComponentClick(event.detail.customId);
	}

	function handleSelectChange(event: CustomEvent<{ customId: string; values: string[] }>) {
		handleComponentChange(event.detail.customId, event.detail.values);
	}

	function handleTextSubmitWrapper(event: CustomEvent<{ customId: string; value: string }>) {
		handleTextSubmit(event.detail.customId, event.detail.value);
	}
</script>

<div class="flex flex-col gap-2 mt-2" role="group" aria-label="Interactive components">
	{#each rows as row, rowIndex}
		<div class="flex flex-wrap items-center gap-2">
			{#each row as component (component.custom_id || rowIndex)}
				{#if component.type === 'button'}
					<Button
						style={(component.style as 'primary' | 'secondary' | 'success' | 'danger' | 'link') || 'primary'}
						label={component.label || ''}
						emoji={component.emoji_name || component.emoji || ''}
						disabled={component.disabled || false}
						url={component.url || ''}
						customId={component.custom_id || ''}
						on:click={handleButtonClick}
					/>
				{:else if component.type === 'select_menu'}
					<SelectMenu
						customId={component.custom_id || ''}
						options={component.options || []}
						placeholder={component.placeholder || 'Select an option'}
						minValues={component.min_values || 1}
						maxValues={component.max_values || 1}
						disabled={component.disabled || false}
						on:change={handleSelectChange}
					/>
				{:else if component.type === 'text_input'}
					<TextInput
						customId={component.custom_id || ''}
						style={(component.style as 'short' | 'paragraph') || 'short'}
						label={component.label || ''}
						placeholder={component.placeholder || ''}
						value={component.value || ''}
						minLength={component.min_length}
						maxLength={component.max_length}
						required={component.required || false}
						disabled={component.disabled || false}
						on:submit={handleTextSubmitWrapper}
					/>
				{/if}
			{/each}
		</div>
	{/each}
</div>
