<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { CommandOption, OptionType } from '$lib/services/slashCommands';
	import { getOptionTypeLabel } from '$lib/services/slashCommands';
	
	export let option: CommandOption;
	
	const dispatch = createEventDispatcher<{
		update: CommandOption;
		remove: void;
	}>();
	
	const optionTypes: { value: OptionType; label: string }[] = [
		{ value: 1, label: 'Subcommand' },
		{ value: 2, label: 'Subcommand Group' },
		{ value: 3, label: 'String' },
		{ value: 4, label: 'Integer' },
		{ value: 5, label: 'Boolean' },
		{ value: 6, label: 'User' },
		{ value: 7, label: 'Channel' },
		{ value: 8, label: 'Role' },
		{ value: 9, label: 'Mentionable' },
		{ value: 10, label: 'Number' },
		{ value: 11, label: 'Attachment' }
	];
	
	let expanded = true;
	let showChoiceInput = false;
	let newChoiceName = '';
	let newChoiceValue = '';
	
	function update(changes: Partial<CommandOption>) {
		dispatch('update', { ...option, ...changes });
	}
	
	function handleTypeChange(e: Event) {
		const value = parseInt((e.target as HTMLSelectElement).value) as OptionType;
		update({ type: value });
	}
	
	function handleAddChoice() {
		if (!newChoiceName.trim() || !newChoiceValue.trim()) return;
		
		const choices = [
			...(option.choices || []),
			{ name: newChoiceName.trim(), value: newChoiceValue.trim() }
		];
		update({ choices });
		
		newChoiceName = '';
		newChoiceValue = '';
		showChoiceInput = false;
	}
	
	function handleRemoveChoice(index: number) {
		const choices = (option.choices || []).filter((_, i) => i !== index);
		update({ choices });
	}
	
	function handleAddNestedOption() {
		const nestedOptions = [
			...(option.options || []),
			{ type: 3 as OptionType, name: '', description: '', required: false }
		];
		update({ options: nestedOptions });
	}
</script>

<div class="option-editor" class:collapsed={!expanded}>
	<div class="option-header" on:click={() => expanded = !expanded} on:keydown={(e) => e.key === 'Enter' && (expanded = !expanded)} role="button" tabindex="0">
		<button class="expand-btn" aria-label={expanded ? 'Collapse' : 'Expand'}>
			<svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor" class:rotated={expanded}>
				<path d="M4.5 2L9 6l-4.5 4V2z"/>
			</svg>
		</button>
		
		<div class="option-title">
			<span class="option-name">{option.name || '(unnamed)'}</span>
			<span class="option-type-badge">{getOptionTypeLabel(option.type)}</span>
		</div>
		
		<label class="required-toggle" on:click|stopPropagation>
			<input
				type="checkbox"
				checked={option.required}
				on:change={(e) => update({ required: (e.target as HTMLInputElement).checked })}
			/>
			Required
		</label>
		
		<button class="remove-btn" on:click|stopPropagation={() => dispatch('remove')} aria-label="Remove option">
			<svg width="14" height="14" viewBox="0 0 14 14" fill="currentColor">
				<path d="M4.293 4.293a.5.5 0 01.708 0L7 6.293l2.293-2.293a.5.5 0 11.708.708L7.707 7l2.293 2.293a.5.5 0 01-.708.708L7 7.707l-2.293 2.293a.5.5 0 01-.708-.708L6.293 7 3.999 4.707a.5.5 0 010-.708z"/>
			</svg>
		</button>
	</div>
	
	{#if expanded}
		<div class="option-body">
			<div class="form-row">
				<div class="form-group">
					<label>Name</label>
					<input
						type="text"
						value={option.name}
						on:input={(e) => update({ name: (e.target as HTMLInputElement).value })}
						placeholder="option-name"
						maxlength="32"
					/>
				</div>
				
				<div class="form-group">
					<label>Type</label>
					<select value={option.type} on:change={handleTypeChange}>
						{#each optionTypes as type}
							<option value={type.value}>{type.label}</option>
						{/each}
					</select>
				</div>
			</div>
			
			<div class="form-group">
				<label>Description</label>
				<input
					type="text"
					value={option.description}
					on:input={(e) => update({ description: (e.target as HTMLInputElement).value })}
					placeholder="What this option does..."
					maxlength="100"
				/>
			</div>
			
			<!-- Choices (for string/integer/number types) -->
			{#if option.type === 3 || option.type === 4 || option.type === 10}
				<div class="choices-section">
					<div class="choices-header">
						<label>Choices (optional)</label>
						<button class="add-choice-btn" on:click={() => showChoiceInput = !showChoiceInput}>
							{showChoiceInput ? 'Cancel' : '+ Add Choice'}
						</button>
					</div>
					
					{#if showChoiceInput}
						<div class="choice-input-row">
							<input
								type="text"
								bind:value={newChoiceName}
								placeholder="Choice name"
							/>
							<input
								type="text"
								bind:value={newChoiceValue}
								placeholder="Value"
							/>
							<button class="add-btn" on:click={handleAddChoice}>Add</button>
						</div>
					{/if}
					
					{#if option.choices && option.choices.length > 0}
						<div class="choices-list">
							{#each option.choices as choice, index}
								<div class="choice-item">
									<span class="choice-name">{choice.name}</span>
									<span class="choice-value">{choice.value}</span>
									<button class="remove-choice-btn" on:click={() => handleRemoveChoice(index)}>×</button>
								</div>
							{/each}
						</div>
					{/if}
					
					<label class="autocomplete-toggle">
						<input
							type="checkbox"
							checked={option.autocomplete}
							on:change={(e) => update({ autocomplete: (e.target as HTMLInputElement).checked })}
						/>
						Enable autocomplete (disables choices)
					</label>
				</div>
			{/if}
			
			<!-- Min/Max for numbers -->
			{#if option.type === 4 || option.type === 10}
				<div class="form-row">
					<div class="form-group">
						<label>Min Value</label>
						<input
							type="number"
							value={option.min_value}
							on:input={(e) => update({ min_value: parseFloat((e.target as HTMLInputElement).value) || undefined })}
							placeholder="None"
						/>
					</div>
					<div class="form-group">
						<label>Max Value</label>
						<input
							type="number"
							value={option.max_value}
							on:input={(e) => update({ max_value: parseFloat((e.target as HTMLInputElement).value) || undefined })}
							placeholder="None"
						/>
					</div>
				</div>
			{/if}
			
			<!-- Min/Max length for strings -->
			{#if option.type === 3}
				<div class="form-row">
					<div class="form-group">
						<label>Min Length</label>
						<input
							type="number"
							value={option.min_length}
							min="0"
							on:input={(e) => update({ min_length: parseInt((e.target as HTMLInputElement).value) || undefined })}
							placeholder="None"
						/>
					</div>
					<div class="form-group">
						<label>Max Length</label>
						<input
							type="number"
							value={option.max_length}
							min="1"
							on:input={(e) => update({ max_length: parseInt((e.target as HTMLInputElement).value) || undefined })}
							placeholder="None"
						/>
					</div>
				</div>
			{/if}
			
			<!-- Channel types -->
			{#if option.type === 7}
				<div class="form-group">
					<label>Allowed Channel Types</label>
					<div class="channel-types">
						{#each [
							{ value: 0, label: 'Text' },
							{ value: 1, label: 'Voice' },
							{ value: 2, label: 'Category' },
							{ value: 3, label: 'Announcement' },
							{ value: 4, label: 'Thread' },
							{ value: 5, label: 'Stage' },
							{ value: 6, label: 'Forum' }
						] as chType}
							<label class="checkbox-label">
								<input
									type="checkbox"
									checked={option.channel_types?.includes(chType.value)}
									on:change={(e) => {
										const checked = (e.target as HTMLInputElement).checked;
										const types = option.channel_types || [];
										if (checked) {
											update({ channel_types: [...types, chType.value] });
										} else {
											update({ channel_types: types.filter(t => t !== chType.value) });
										}
									}}
								/>
								{chType.label}
							</label>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.option-editor {
		background: var(--bg-primary, #313338);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 6px;
		margin-bottom: 8px;
	}
	
	.option-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		cursor: pointer;
		user-select: none;
	}
	
	.expand-btn {
		background: transparent;
		border: none;
		color: var(--text-muted, #949ba4);
		cursor: pointer;
		padding: 2px;
		display: flex;
		align-items: center;
	}
	
	.expand-btn svg {
		transition: transform 0.15s;
	}
	
	.expand-btn svg.rotated {
		transform: rotate(90deg);
	}
	
	.option-title {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 8px;
	}
	
	.option-name {
		font-weight: 600;
		font-size: 13px;
		color: var(--text-normal, #dbdee1);
	}
	
	.option-type-badge {
		font-size: 10px;
		background: var(--bg-tertiary, #1e1f22);
		color: var(--text-muted, #949ba4);
		padding: 2px 6px;
		border-radius: 4px;
	}
	
	.required-toggle {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 11px;
		color: var(--text-muted, #949ba4);
		cursor: pointer;
	}
	
	.required-toggle input {
		accent-color: var(--text-link, #5865f2);
	}
	
	.remove-btn {
		background: transparent;
		border: none;
		color: var(--text-muted, #949ba4);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		transition: all 0.15s;
	}
	
	.remove-btn:hover {
		background: var(--status-danger, #ed4245);
		color: white;
	}
	
	.option-body {
		padding: 12px;
		border-top: 1px solid var(--bg-tertiary, #1e1f22);
	}
	
	.form-row {
		display: flex;
		gap: 12px;
		margin-bottom: 12px;
	}
	
	.form-group {
		flex: 1;
	}
	
	.form-group label {
		display: block;
		font-size: 11px;
		font-weight: 500;
		color: var(--text-muted, #949ba4);
		margin-bottom: 4px;
	}
	
	.form-group input,
	.form-group select {
		width: 100%;
		padding: 8px 10px;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		color: var(--text-normal, #dbdee1);
		font-size: 13px;
	}
	
	.form-group input:focus,
	.form-group select:focus {
		outline: none;
		border-color: var(--text-link, #5865f2);
	}
	
	.choices-section {
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid var(--bg-tertiary, #1e1f22);
	}
	
	.choices-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 8px;
	}
	
	.choices-header label {
		font-size: 11px;
		font-weight: 500;
		color: var(--text-muted, #949ba4);
	}
	
	.add-choice-btn {
		padding: 4px 8px;
		background: transparent;
		border: 1px solid var(--text-link, #5865f2);
		border-radius: 4px;
		color: var(--text-link, #5865f2);
		font-size: 11px;
		cursor: pointer;
	}
	
	.add-choice-btn:hover {
		background: var(--text-link, #5865f2);
		color: white;
	}
	
	.choice-input-row {
		display: flex;
		gap: 8px;
		margin-bottom: 8px;
	}
	
	.choice-input-row input {
		flex: 1;
		padding: 6px 8px;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		color: var(--text-normal, #dbdee1);
		font-size: 12px;
	}
	
	.add-btn {
		padding: 6px 12px;
		background: var(--text-link, #5865f2);
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 12px;
		cursor: pointer;
	}
	
	.choices-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
		margin-bottom: 8px;
	}
	
	.choice-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 4px;
	}
	
	.choice-name {
		flex: 1;
		font-size: 12px;
		color: var(--text-normal, #dbdee1);
	}
	
	.choice-value {
		font-size: 11px;
		color: var(--text-muted, #949ba4);
		font-family: monospace;
	}
	
	.remove-choice-btn {
		background: transparent;
		border: none;
		color: var(--text-muted, #949ba4);
		cursor: pointer;
		font-size: 16px;
		line-height: 1;
		padding: 0 4px;
	}
	
	.remove-choice-btn:hover {
		color: var(--status-danger, #ed4245);
	}
	
	.autocomplete-toggle {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 11px;
		color: var(--text-muted, #949ba4);
		cursor: pointer;
	}
	
	.autocomplete-toggle input {
		accent-color: var(--text-link, #5865f2);
	}
	
	.channel-types {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	
	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--text-normal, #dbdee1);
		cursor: pointer;
	}
	
	.checkbox-label input {
		accent-color: var(--text-link, #5865f2);
	}
</style>
