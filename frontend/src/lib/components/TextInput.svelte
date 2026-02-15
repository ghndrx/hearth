<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let value: string = '';
	export let placeholder: string = '';
	export let label: string = '';
	export let error: string = '';
	export let type: 'text' | 'password' | 'email' | 'number' = 'text';
	export let disabled: boolean = false;
	export let readonly: boolean = false;
	export let required: boolean = false;
	export let autocomplete: string | undefined = undefined;
	export let autofocus: boolean = false;
	export let id: string = '';
	export let name: string = '';
	export let min: number | undefined = undefined;
	export let max: number | undefined = undefined;
	export let minlength: number | undefined = undefined;
	export let maxlength: number | undefined = undefined;
	export let fullWidth: boolean = true;

	const dispatch = createEventDispatcher<{
		input: string;
		change: string;
		focus: FocusEvent;
		blur: FocusEvent;
		keydown: KeyboardEvent;
		keyup: KeyboardEvent;
	});

	let inputElement: HTMLInputElement;

	export function focus() {
		inputElement?.focus();
	}

	export function blur() {
		inputElement?.blur();
	}

	export function select() {
		inputElement?.select();
	}

	function handleInput(event: Event) {
		const target = event.target as HTMLInputElement;
		value = target.value;
		dispatch('input', value);
	}

	function handleChange(event: Event) {
		const target = event.target as HTMLInputElement;
		dispatch('change', target.value);
	}

	function handleKeydown(event: KeyboardEvent) {
		dispatch('keydown', event);
	}

	function handleKeyup(event: KeyboardEvent) {
		dispatch('keyup', event);
	}

	function handleFocus(event: FocusEvent) {
		dispatch('focus', event);
	}

	function handleBlur(event: FocusEvent) {
		dispatch('blur', event);
	}

	$: inputId = id || `input-${Math.random().toString(36).substr(2, 9)}`;
</script>

<div class="input-wrapper" class:full-width={fullWidth}>
	{#if label}
		<label for={inputId} class="input-label">
			{label}
			{#if required}
				<span class="required-indicator">*</span>
			{/if}
		</label>
	{/if}

	<input
		bind:this={inputElement}
		{id}
		{name}
		{type}
		{value}
		{placeholder}
		{disabled}
		{readonly}
		{required}
		{autocomplete}
		{autofocus}
		{min}
		{max}
		{minlength}
		{maxlength}
		class="text-input"
		class:has-error={!!error}
		class:disabled
		on:input={handleInput}
		on:change={handleChange}
		on:focus={handleFocus}
		on:blur={handleBlur}
		on:keydown={handleKeydown}
		on:keyup={handleKeyup}
	/>

	{#if error}
		<span class="error-message" role="alert">{error}</span>
	{/if}
</div>

<style>
	.input-wrapper {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.input-wrapper.full-width {
		width: 100%;
	}

	.input-label {
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-muted);
		letter-spacing: 0.02em;
	}

	.required-indicator {
		color: var(--red);
		margin-left: 4px;
	}

	.text-input {
		background-color: var(--bg-tertiary);
		border: none;
		border-radius: var(--radius-sm);
		padding: 10px;
		color: var(--text-normal);
		font-size: var(--font-size-md);
		font-family: var(--font-family);
		line-height: var(--line-height-normal);
		transition: box-shadow var(--transition-fast);
		width: 100%;
		box-sizing: border-box;
	}

	.text-input::placeholder {
		color: var(--text-faint);
	}

	.text-input:focus {
		outline: none;
		box-shadow: 0 0 0 2px var(--blurple);
	}

	.text-input.has-error {
		box-shadow: 0 0 0 2px var(--red);
	}

	.text-input.has-error:focus {
		box-shadow: 0 0 0 2px var(--red);
	}

	.text-input:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.text-input:disabled:hover {
		background-color: var(--bg-tertiary);
	}

	.error-message {
		font-size: var(--font-size-sm);
		color: var(--red);
		margin-top: 4px;
	}
</style>
