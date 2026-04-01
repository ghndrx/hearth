<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import MessageButton from './MessageButton.svelte';
	import SelectMenu from './SelectMenu.svelte';

	export let components: Array<{
		id: string;
		type: string;
		style?: string;
		label?: string;
		custom_id?: string;
		disabled?: boolean;
		url?: string;
		emoji?: string;
		options?: Array<{
			label: string;
			value: string;
			description?: string;
			emoji?: string;
			default?: boolean;
		}>;
		placeholder?: string;
		min_values?: number;
		max_values?: number;
	}> = [];

	const dispatch = createEventDispatcher<{
		interaction: {
			type: string;
			customId: string;
			componentId: string;
			values?: string[];
		};
	}>();

	function handleButtonClick(event: CustomEvent<{ customId: string; componentId: string }>) {
		dispatch('interaction', {
			type: 'button',
			customId: event.detail.customId,
			componentId: event.detail.componentId,
		});
	}

	function handleSelectChange(event: CustomEvent<{ customId: string; componentId: string; values: string[] }>) {
		dispatch('interaction', {
			type: 'select',
			customId: event.detail.customId,
			componentId: event.detail.componentId,
			values: event.detail.values,
		});
	}

	// Group components by action_row if present, otherwise wrap individual components
	function getActionRows(comps: typeof components): Array<typeof components> {
		const rows: Array<typeof components> = [];
		let currentRow: typeof components = [];

		for (const comp of comps) {
			if (comp.type === 'action_row') {
				if (currentRow.length > 0) {
					rows.push(currentRow);
					currentRow = [];
				}
				// action_row is a container, skip it and process children
				continue;
			}
			currentRow.push(comp);
			// Discord limit: 5 buttons per action row, 1 select per action row
			if (currentRow.length >= 5) {
				rows.push(currentRow);
				currentRow = [];
			}
		}

		if (currentRow.length > 0) {
			rows.push(currentRow);
		}

		return rows;
	}

	$: actionRows = getActionRows(components);
</script>

<div class="message-components">
	{#each actionRows as row, rowIndex}
		<div class="action-row" role="group" aria-label="Message actions row {rowIndex + 1}">
			{#each row as component (component.id)}
				{#if component.type === 'button'}
					<MessageButton
						id={component.id}
						customId={component.custom_id || ''}
						label={component.label || ''}
						style={component.style as 'primary' | 'secondary' | 'success' | 'danger' | 'link' || 'primary'}
						disabled={component.disabled || false}
						url={component.url}
						emoji={component.emoji}
						on:click={handleButtonClick}
					/>
				{:else if component.type === 'select_menu'}
					<SelectMenu
						id={component.id}
						customId={component.custom_id || ''}
						options={component.options || []}
						placeholder={component.placeholder || 'Select an option'}
						minValues={component.min_values || 1}
						maxValues={component.max_values || 1}
						disabled={component.disabled || false}
						on:select={handleSelectChange}
					/>
				{/if}
			{/each}
		</div>
	{/each}
</div>

<style>
	.message-components {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-top: 8px;
	}

	.action-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
	}
</style>
