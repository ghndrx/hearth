<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { CommandOption, CommandChoice } from '$lib/services/slashCommands';
	import { getOptionTypeLabel } from '$lib/services/slashCommands';
	
	export let option: CommandOption;
	export let value: unknown = undefined;
	export let error: string | null = null;
	export let autofocus: boolean = false;
	
	const dispatch = createEventDispatcher<{
		change: { name: string; value: unknown };
		submit: void;
		escape: void;
	}>();
	
	let inputValue = '';
	let showChoices = false;
	
	$: {
		// Sync external value changes
		if (value !== undefined && value !== null) {
			inputValue = String(value);
		}
	}
	
	$: hasChoices = option.choices && option.choices.length > 0;
	$: isBoolean = option.type === 5;
	$: isChannel = option.type === 7;
	$: isUser = option.type === 6;
	$: isRole = option.type === 8;
	$: isMentionable = option.type === 9;
	$: requiresMention = isChannel || isUser || isRole || isMentionable;
	
	function handleInput(e: Event) {
		const target = e.target as HTMLInputElement;
		inputValue = target.value;
		dispatch('change', { name: option.name, value: parseValue(inputValue) });
	}
	
	function handleSelect(choice: CommandChoice) {
		inputValue = String(choice.value);
		dispatch('change', { name: option.name, value: choice.value });
		showChoices = false;
	}
	
	function handleBoolean(value: boolean) {
		inputValue = String(value);
		dispatch('change', { name: option.name, value });
	}
	
	function parseValue(val: string): unknown {
		switch (option.type) {
			case 4: // Integer
				return parseInt(val, 10) || 0;
			case 5: // Boolean
				return val === 'true';
			case 10: // Number
				return parseFloat(val) || 0;
			default:
				return val;
		}
	}
	
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			dispatch('submit');
		}
		if (e.key === 'Escape') {
			dispatch('escape');
		}
	}
	
	function formatPlaceholder(): string {
		if (option.type === 5) return 'true / false';
		if (option.type === 6) return '@username';
		if (option.type === 7) return '#channel-name';
		if (option.type === 8) return '@role-name';
		if (option.type === 9) return '@user or @role';
		if (option.type === 4) return '123';
		if (option.type === 10) return '3.14';
		return `Enter ${option.name}...`;
	}
</script>

<div class="option-input" class:has-error={error} class:boolean={isBoolean}>
	<div class="option-header">
		<span class="option-name">{option.name}</span>
		<span class="option-type">{getOptionTypeLabel(option.type)}</span>
		{#if option.required}
			<span class="required-badge">Required</span>
		{/if}
	</div>
	
	{#if isBoolean}
		<div class="boolean-input">
			<button 
				class="bool-btn"
				class:active={inputValue === 'true'}
				on:click={() => handleBoolean(true)}
			>
				True
			</button>
			<button 
				class="bool-btn"
				class:active={inputValue === 'false'}
				on:click={() => handleBoolean(false)}
			>
				False
			</button>
		</div>
	{:else if hasChoices}
		<div class="choice-input">
			<div class="input-wrapper">
				{#if requiresMention}
					<span class="mention-prefix">
						{#if isUser}@{:else if isRole}@{:else if isChannel}#{/if}
					</span>
				{/if}
				<input
					type="text"
					value={inputValue}
					placeholder={formatPlaceholder()}
					on:input={handleInput}
					on:focus={() => showChoices = true}
					on:blur={() => setTimeout(() => showChoices = false, 200)}
					on:keydown={handleKeydown}
					autocomplete="off"
					{autofocus}
				/>
			</div>
			
			{#if showChoices && option.choices}
				<div class="choices-dropdown">
					{#each option.choices as choice}
						<button
							class="choice-item"
							class:selected={inputValue === String(choice.value)}
							on:mousedown|preventDefault={() => handleSelect(choice)}
						>
							<span class="choice-name">{choice.name}</span>
							<span class="choice-value">{choice.value}</span>
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{:else}
		<div class="input-wrapper">
			{#if requiresMention}
				<span class="mention-prefix">
					{#if isUser}@{:else if isRole}@{:else if isChannel}#{/if}
				</span>
			{/if}
			<input
				type={option.type === 4 ? 'number' : option.type === 10 ? 'number' : 'text'}
				value={inputValue}
				placeholder={formatPlaceholder()}
				min={option.min_value}
				max={option.max_value}
				minlength={option.min_length}
				maxlength={option.max_length}
				step={option.type === 10 ? 'any' : undefined}
				on:input={handleInput}
				on:keydown={handleKeydown}
				autocomplete="off"
				{autofocus}
			/>
		</div>
	{/if}
	
	{#if error}
		<div class="error-message">{error}</div>
	{/if}
	
	<div class="option-description">{option.description}</div>
</div>

<style>
	.option-input {
		padding: 10px 12px;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 6px;
		transition: border-color 0.15s;
	}
	
	.option-input:focus-within {
		border-color: var(--text-link, #5865f2);
	}
	
	.option-input.has-error {
		border-color: var(--status-danger, #ed4245);
	}
	
	.option-header {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 6px;
	}
	
	.option-name {
		font-weight: 600;
		font-size: 13px;
		color: var(--text-normal, #dbdee1);
	}
	
	.option-type {
		font-size: 10px;
		color: var(--text-muted, #949ba4);
		background: var(--bg-tertiary, #1e1f22);
		padding: 1px 6px;
		border-radius: 4px;
	}
	
	.required-badge {
		font-size: 10px;
		color: var(--status-danger, #ed4245);
		font-weight: 500;
	}
	
	.input-wrapper {
		display: flex;
		align-items: center;
		background: var(--bg-primary, #313338);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		overflow: hidden;
	}
	
	.mention-prefix {
		padding: 0 8px;
		color: var(--text-muted, #949ba4);
		font-size: 13px;
		background: var(--bg-tertiary, #1e1f22);
		border-right: 1px solid var(--bg-tertiary, #1e1f22);
		height: 100%;
		display: flex;
		align-items: center;
	}
	
	input {
		flex: 1;
		background: transparent;
		border: none;
		outline: none;
		padding: 8px;
		font-size: 13px;
		color: var(--text-normal, #dbdee1);
		width: 100%;
	}
	
	input::placeholder {
		color: var(--text-muted, #949ba4);
	}
	
	input[type="number"] {
		-moz-appearance: textfield;
	}
	
	input[type="number"]::-webkit-outer-spin-button,
	input[type="number"]::-webkit-inner-spin-button {
		-webkit-appearance: none;
		margin: 0;
	}
	
	.boolean-input {
		display: flex;
		gap: 8px;
	}
	
	.bool-btn {
		flex: 1;
		padding: 8px 12px;
		background: var(--bg-primary, #313338);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		color: var(--text-muted, #949ba4);
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s;
	}
	
	.bool-btn:hover {
		background: var(--bg-modifier-hover, #3f434a);
		color: var(--text-normal, #dbdee1);
	}
	
	.bool-btn.active {
		background: var(--text-link, #5865f2);
		border-color: var(--text-link, #5865f2);
		color: white;
	}
	
	.choice-input {
		position: relative;
	}
	
	.choices-dropdown {
		position: absolute;
		top: 100%;
		left: 0;
		right: 0;
		margin-top: 4px;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 6px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		max-height: 160px;
		overflow-y: auto;
		z-index: 100;
	}
	
	.choice-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
		padding: 8px 12px;
		background: transparent;
		border: none;
		cursor: pointer;
		text-align: left;
		transition: background-color 0.1s;
	}
	
	.choice-item:hover,
	.choice-item.selected {
		background: var(--bg-modifier-hover, #3f434a);
	}
	
	.choice-name {
		font-size: 13px;
		color: var(--text-normal, #dbdee1);
	}
	
	.choice-value {
		font-size: 11px;
		color: var(--text-muted, #949ba4);
		font-family: monospace;
	}
	
	.error-message {
		font-size: 11px;
		color: var(--status-danger, #ed4245);
		margin-top: 4px;
	}
	
	.option-description {
		font-size: 11px;
		color: var(--text-muted, #949ba4);
		margin-top: 6px;
		line-height: 1.4;
	}
</style>
