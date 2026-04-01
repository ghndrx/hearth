<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let id: string;
	export let customId: string;
	export let options: Array<{
		label: string;
		value: string;
		description?: string;
		emoji?: string;
		default?: boolean;
	}> = [];
	export let placeholder: string = 'Select an option';
	export let minValues: number = 1;
	export let maxValues: number = 1;
	export let disabled: boolean = false;

	const dispatch = createEventDispatcher<{
		select: { customId: string; componentId: string; values: string[] };
	}>();

	let isOpen = false;
	let selectedValues: string[] = [];

	// Find default options
	$: {
		selectedValues = options.filter(o => o.default).map(o => o.value);
	}

	function toggleDropdown() {
		if (disabled) return;
		isOpen = !isOpen;
	}

	function selectOption(value: string) {
		if (maxValues === 1) {
			selectedValues = [value];
			isOpen = false;
			dispatch('select', { customId, componentId: id, values: selectedValues });
		} else {
			if (selectedValues.includes(value)) {
				selectedValues = selectedValues.filter(v => v !== value);
			} else if (selectedValues.length < maxValues) {
				selectedValues = [...selectedValues, value];
			}
		}
	}

	function handleConfirm() {
		if (selectedValues.length >= minValues) {
			dispatch('select', { customId, componentId: id, values: selectedValues });
			isOpen = false;
		}
	}

	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.select-menu-container')) {
			isOpen = false;
		}
	}

	function getSelectedLabels(): string {
		if (selectedValues.length === 0) return placeholder;
		if (selectedValues.length === 1) {
			const opt = options.find(o => o.value === selectedValues[0]);
			return opt?.label || selectedValues[0];
		}
		return `${selectedValues.length} selected`;
	}
</script>

<svelte:window on:click={handleClickOutside} />

<div class="select-menu-container" role="combobox" aria-haspopup="listbox" aria-expanded={isOpen}>
	<button
		type="button"
		class="select-trigger"
		class:disabled
		class:open={isOpen}
		on:click={toggleDropdown}
		aria-label={placeholder}
		{disabled}
	>
		<span class="select-value">{getSelectedLabels()}</span>
		<svg class="select-arrow" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
			<path d="M7 10l5 5 5-5z"/>
		</svg>
	</button>

	{#if isOpen}
		<div class="select-dropdown" role="listbox" aria-multiselectable={maxValues > 1}>
			{#each options as option (option.value)}
				<button
					type="button"
					class="select-option"
					class:selected={selectedValues.includes(option.value)}
					role="option"
					aria-selected={selectedValues.includes(option.value)}
					on:click={() => selectOption(option.value)}
				>
					{#if maxValues > 1}
						<span class="checkbox" class:checked={selectedValues.includes(option.value)}>
							{#if selectedValues.includes(option.value)}
								<svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
									<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
								</svg>
							{/if}
						</span>
					{/if}
					{#if option.emoji}
						<span class="emoji" aria-hidden="true">{option.emoji}</span>
					{/if}
					<span class="option-content">
						<span class="option-label">{option.label}</span>
						{#if option.description}
							<span class="option-description">{option.description}</span>
						{/if}
					</span>
				</button>
			{/each}

			{#if maxValues > 1}
				<div class="select-footer">
					<button
						type="button"
						class="confirm-btn"
						disabled={selectedValues.length < minValues}
						on:click={handleConfirm}
					>
						Confirm
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.select-menu-container {
		position: relative;
		display: inline-block;
	}

	.select-trigger {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		min-width: 150px;
		padding: 8px 12px;
		background-color: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-modifier-accent, #1e1f22);
		border-radius: 4px;
		color: var(--text-normal, #f2f3f5);
		font-size: 14px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.select-trigger:hover:not(.disabled) {
		background-color: var(--bg-modifier-hover, #35373c);
	}

	.select-trigger.open {
		border-color: var(--blurple, #5865f2);
	}

	.select-trigger.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.select-value {
		flex: 1;
		text-align: left;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.select-arrow {
		color: var(--text-muted, #b5bac1);
		transition: transform 0.15s ease;
	}

	.select-trigger.open .select-arrow {
		transform: rotate(180deg);
	}

	.select-dropdown {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		max-height: 300px;
		overflow-y: auto;
		background-color: var(--bg-floating, #111214);
		border: 1px solid var(--bg-modifier-accent, #1e1f22);
		border-radius: 4px;
		box-shadow: 0 8px 16px rgba(0, 0, 0, 0.24);
		z-index: 100;
	}

	.select-option {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		width: 100%;
		padding: 8px 12px;
		background: transparent;
		border: none;
		color: var(--text-normal, #dcddde);
		font-size: 14px;
		text-align: left;
		cursor: pointer;
		transition: background-color 0.1s ease;
	}

	.select-option:hover {
		background-color: var(--bg-modifier-hover, #35373c);
	}

	.select-option.selected {
		background-color: var(--bg-modifier-selected, #404249);
	}

	.checkbox {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		margin-top: 2px;
		background-color: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-modifier-accent, #1e1f22);
		border-radius: 3px;
		flex-shrink: 0;
	}

	.checkbox.checked {
		background-color: var(--blurple, #5865f2);
		border-color: var(--blurple, #5865f2);
	}

	.checkbox svg {
		color: white;
	}

	.emoji {
		font-size: 16px;
		line-height: 1;
		flex-shrink: 0;
	}

	.option-content {
		display: flex;
		flex-direction: column;
		gap: 2px;
		flex: 1;
		min-width: 0;
	}

	.option-label {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.option-description {
		font-size: 12px;
		color: var(--text-muted, #b5bac1);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.select-footer {
		display: flex;
		justify-content: flex-end;
		padding: 8px;
		border-top: 1px solid var(--bg-modifier-accent, #1e1f22);
	}

	.confirm-btn {
		padding: 6px 12px;
		background-color: var(--blurple, #5865f2);
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s ease;
	}

	.confirm-btn:hover:not(:disabled) {
		background-color: var(--blurple-hover, #4752c4);
	}

	.confirm-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
