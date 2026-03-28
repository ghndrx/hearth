<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let customId: string = '';
	export let style: 'short' | 'paragraph' = 'short';
	export let label: string = '';
	export let placeholder: string = '';
	export let value: string = '';
	export let minLength: number | undefined = undefined;
	export let maxLength: number | undefined = undefined;
	export let required: boolean = false;
	export let disabled: boolean = false;

	const dispatch = createEventDispatcher();

	let inputValue = value;

	function handleSubmit() {
		if (disabled) return;
		if (required && !inputValue.trim()) return;
		if (minLength && inputValue.length < minLength) return;
		if (maxLength && inputValue.length > maxLength) return;
		
		dispatch('submit', { customId, value: inputValue });
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			handleSubmit();
		}
	}
</script>

<div class="flex flex-col gap-2 {disabled ? 'opacity-50' : ''}">
	{#if label}
		<label class="text-sm font-medium text-[#b5bac1]" for={customId}>{label}</label>
	{/if}
	
	{#if style === 'paragraph'}
		<textarea
			id={customId}
			bind:value={inputValue}
			{placeholder}
			minlength={minLength}
			maxlength={maxLength}
			{required}
			{disabled}
			rows="3"
			class="w-full p-3 bg-[#383a40] rounded-lg text-[#f2f3f5] text-base resize-none border-0 focus:outline-none focus:ring-2 focus:ring-[#5865f2]"
			on:keydown={handleKeydown}
		></textarea>
	{:else}
		<input
			type="text"
			id={customId}
			bind:value={inputValue}
			{placeholder}
			minlength={minLength}
			maxlength={maxLength}
			{required}
			{disabled}
			class="w-full p-3 bg-[#383a40] rounded-lg text-[#f2f3f5] text-base border-0 focus:outline-none focus:ring-2 focus:ring-[#5865f2]"
			on:keydown={handleKeydown}
		/>
	{/if}
	
	<div class="flex items-center justify-between">
		{#if minLength || maxLength}
			<span class="text-xs text-[#949ba4]">
				{#if minLength && maxLength}
					{minLength} - {maxLength} characters
				{:else if minLength}
					Min {minLength} characters
				{:else if maxLength}
					Max {maxLength} characters
				{/if}
			</span>
		{/if}
		
		<button
			type="button"
			class="px-3 py-1.5 bg-[#5865f2] hover:bg-[#4752c4] text-white text-sm rounded transition-colors"
			{disabled}
			on:click={handleSubmit}
		>
			Submit
		</button>
	</div>
</div>
