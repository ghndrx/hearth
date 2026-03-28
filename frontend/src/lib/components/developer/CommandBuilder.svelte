<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { SlashCommand, CommandOption, CommandChoice } from '$lib/services/slashCommands';
	import CommandOptionEditor from '$lib/components/CommandOptionEditor.svelte';

	export let command: Partial<SlashCommand> | null = null;
	export let appId: string = '';
	export let loading = false;

	const dispatch = createEventDispatcher<{
		save: { command: Partial<SlashCommand> };
		cancel: void;
	}>();

	// Form state
	let name = command?.name || '';
	let description = command?.description || '';
	let commandType = command?.type || 1; // 1 = Slash command
	let defaultPermission = command?.default_permission ?? true;
	let options: CommandOption[] = command?.options || [];
	let serverId = command?.guild_id || '';

	// Validation errors
	let errors: Record<string, string> = {};

	const OPTION_TYPES = [
		{ value: 1, label: 'Subcommand' },
		{ value: 2, label: 'Subcommand Group' },
		{ value: 3, label: 'String' },
		{ value: 4, label: 'Integer' },
		{ value: 5, label: 'Boolean' },
		{ value: 6, label: 'User' },
		{ value: 7, label: 'Channel' },
		{ value: 8, label: 'Role' },
		{ value: 9, label: 'Mentionable' },
		{ value: 10, label: 'Number' }
	];

	function validate(): boolean {
		errors = {};

		if (!name.trim()) {
			errors.name = 'Name is required';
		} else if (name.length > 32) {
			errors.name = 'Name must be 32 characters or less';
		} else if (!/^[\w-]{1,32}$/.test(name)) {
			errors.name = 'Name can only contain letters, numbers, hyphens, and underscores';
		}

		if (!description.trim()) {
			errors.description = 'Description is required';
		} else if (description.length > 100) {
			errors.description = 'Description must be 100 characters or less';
		}

		// Validate option names
		for (const opt of options) {
			if (!/^[\w-]{1,32}$/.test(opt.name)) {
				errors[`option_${opt.name}`] = `Invalid option name: ${opt.name}`;
			}
		}

		return Object.keys(errors).length === 0;
	}

	function handleSubmit() {
		if (!validate()) return;

		const cmd: Partial<SlashCommand> = {
			name: name.trim(),
			description: description.trim(),
			type: commandType as any,
			default_permission: defaultPermission,
			options
		};

		if (serverId) {
			cmd.guild_id = serverId;
		}

		dispatch('save', { command: cmd });
	}

	function addOption() {
		options = [
			...options,
			{
				type: 3, // String by default
				name: '',
				description: '',
				required: false
			}
		];
	}

	function updateOption(index: number, updated: CommandOption) {
		options = options.map((opt, i) => (i === index ? updated : opt));
	}

	function removeOption(index: number) {
		options = options.filter((_, i) => i !== index);
	}

	function moveOption(index: number, direction: -1 | 1) {
		const newIndex = index + direction;
		if (newIndex < 0 || newIndex >= options.length) return;

		const newOptions = [...options];
		[newOptions[index], newOptions[newIndex]] = [newOptions[newIndex], newOptions[index]];
		options = newOptions;
	}

	function getOptionTypeName(typeValue: number): string {
		return OPTION_TYPES.find(t => t.value === typeValue)?.label || 'Unknown';
	}
</script>

<div class="command-builder">
	<form on:submit|preventDefault={handleSubmit}>
		<div class="form-section">
			<h3>Basic Information</h3>

			<div class="form-group">
				<label for="name">Command Name</label>
				<input
					id="name"
					type="text"
					bind:value={name}
					placeholder="command-name"
					class:error={errors.name}
					maxlength="32"
				/>
				{#if errors.name}
					<span class="error-text">{errors.name}</span>
				{/if}
				<span class="hint">Letters, numbers, hyphens, underscores. Max 32 characters.</span>
			</div>

			<div class="form-group">
				<label for="description">Description</label>
				<textarea
					id="description"
					bind:value={description}
					placeholder="What does this command do?"
					class:error={errors.description}
					maxlength="100"
					rows="2"
				></textarea>
				{#if errors.description}
					<span class="error-text">{errors.description}</span>
				{/if}
				<span class="hint">{description.length}/100 characters</span>
			</div>

			<div class="form-group">
				<label for="type">Command Type</label>
				<select id="type" bind:value={commandType}>
					<option value={1}>Slash Command</option>
					<option value={2}>User Context Menu</option>
					<option value={3}>Message Context Menu</option>
				</select>
			</div>

			<div class="form-group checkbox">
				<input
					id="defaultPermission"
					type="checkbox"
					bind:checked={defaultPermission}
				/>
				<label for="defaultPermission">
					Allow everyone to use this command
				</label>
				<span class="hint">
					Disable to set specific permissions per role/user
				</span>
			</div>

			{#if appId}
				<div class="form-group">
					<label for="serverId">Server (optional)</label>
					<input
						id="serverId"
						type="text"
						bind:value={serverId}
						placeholder="Server ID (leave empty for global command)"
					/>
					<span class="hint">Guild-specific commands are only available in the specified server</span>
				</div>
			{/if}
		</div>

		<div class="form-section">
			<div class="section-header">
				<h3>Options</h3>
				<button type="button" class="btn-secondary" on:click={addOption}>
					+ Add Option
				</button>
			</div>

			{#if options.length === 0}
				<div class="empty-options">
					<p>No options yet. Add options to let users provide additional input.</p>
				</div>
			{:else}
				<div class="options-list">
					{#each options as option, index (index)}
						<div class="option-item">
							<div class="option-header">
								<span class="option-type">{getOptionTypeName(option.type)}</span>
								<div class="option-actions">
									<button
										type="button"
										class="btn-icon"
										disabled={index === 0}
										on:click={() => moveOption(index, -1)}
									>
										▲
									</button>
									<button
										type="button"
										class="btn-icon"
										disabled={index === options.length - 1}
										on:click={() => moveOption(index, 1)}
									>
										▼
									</button>
									<button
										type="button"
										class="btn-icon danger"
										on:click={() => removeOption(index)}
									>
										×
									</button>
								</div>
							</div>
							<CommandOptionEditor
								{option}
								on:update={(e) => updateOption(index, e.detail)}
							/>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div class="form-actions">
			<button type="button" class="btn-secondary" on:click={() => dispatch('cancel')}>
				Cancel
			</button>
			<button type="submit" class="btn-primary" disabled={loading}>
				{#if loading}
					Saving...
				{:else if command?.id}
					Update Command
				{:else}
					Create Command
				{/if}
			</button>
		</div>
	</form>
</div>

<style>
	.command-builder {
		padding: 20px;
		max-width: 700px;
		margin: 0 auto;
	}

	.form-section {
		margin-bottom: 28px;
		padding-bottom: 20px;
		border-bottom: 1px solid var(--border-color, #3a3a3a);
	}

	.form-section h3 {
		font-size: 16px;
		font-weight: 600;
		margin-bottom: 16px;
		color: var(--text-primary, #ffffff);
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 16px;
	}

	.form-group {
		margin-bottom: 16px;
	}

	.form-group label {
		display: block;
		font-size: 13px;
		font-weight: 500;
		color: var(--text-secondary, #b9b9b9);
		margin-bottom: 6px;
	}

	.form-group.checkbox {
		display: flex;
		align-items: flex-start;
		gap: 8px;
	}

	.form-group.checkbox label {
		margin-bottom: 0;
	}

	.form-group.checkbox input {
		margin-top: 2px;
	}

	input[type="text"],
	input[type="number"],
	textarea,
	select {
		width: 100%;
		padding: 10px 12px;
		background: var(--background-secondary, #2a2a2a);
		border: 1px solid var(--border-color, #3a3a3a);
		border-radius: 4px;
		color: var(--text-primary, #ffffff);
		font-size: 14px;
		font-family: inherit;
	}

	input[type="text"]:focus,
	input[type="number"]:focus,
	textarea:focus,
	select:focus {
		outline: none;
		border-color: var(--accent-color, #5865f2);
	}

	input.error,
	textarea.error {
		border-color: var(--danger-color, #da373c);
	}

	textarea {
		resize: vertical;
		min-height: 60px;
	}

	.hint {
		display: block;
		font-size: 12px;
		color: var(--text-muted, #727067);
		margin-top: 4px;
	}

	.error-text {
		display: block;
		font-size: 12px;
		color: var(--danger-color, #da373c);
		margin-top: 4px;
	}

	.empty-options {
		padding: 24px;
		text-align: center;
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		color: var(--text-secondary, #b9b9b9);
	}

	.options-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.option-item {
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		padding: 12px;
	}

	.option-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
	}

	.option-type {
		font-size: 12px;
		font-weight: 500;
		color: var(--accent-color, #5865f2);
		text-transform: uppercase;
	}

	.option-actions {
		display: flex;
		gap: 4px;
	}

	.btn-icon {
		padding: 4px 8px;
		background: var(--background-tertiary, #3a3a3a);
		border: none;
		border-radius: 4px;
		color: var(--text-secondary, #b9b9b9);
		cursor: pointer;
		font-size: 10px;
		transition: background-color 0.15s;
	}

	.btn-icon:hover:not(:disabled) {
		background: var(--background-hover, #4a4a4a);
	}

	.btn-icon:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.btn-icon.danger:hover {
		background: var(--danger-color, #da373c);
		color: white;
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		padding-top: 16px;
	}

	.btn-primary {
		padding: 10px 20px;
		background: var(--accent-color, #5865f2);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--accent-color-hover, #4752c4);
	}

	.btn-primary:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-secondary {
		padding: 10px 20px;
		background: var(--background-tertiary, #3a3a3a);
		color: var(--text-primary, #ffffff);
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-secondary:hover {
		background: var(--background-hover, #4a4a4a);
	}
</style>
