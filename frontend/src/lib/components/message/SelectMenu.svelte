<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let customId: string = '';
	export let options: Array<{ label: string; value: string; description?: string; emoji?: string; default?: boolean }> = [];
	export let placeholder: string = 'Select an option';
	export let minValues: number = 1;
	export let maxValues: number = 1;
	export let disabled: boolean = false;

	const dispatch = createEventDispatcher();

	let selectedValues: string[] = [];
	let isOpen = false;
	let loading = false;

	function toggleDropdown() {
		if (disabled) return;
		isOpen = !isOpen;
	}

	function selectOption(value: string) {
		if (maxValues === 1) {
			selectedValues = [value];
			isOpen = false;
		} else {
			if (selectedValues.includes(value)) {
				selectedValues = selectedValues.filter(v => v !== value);
			} else if (selectedValues.length < maxValues) {
				selectedValues = [...selectedValues, value];
			}
		}
	}

	function isSelected(value: string): boolean {
		return selectedValues.includes(value);
	}

	async function submit() {
		if (disabled || selectedValues.length < minValues) return;
		
		loading = true;
		dispatch('change', { customId, values: selectedValues });
		
		setTimeout(() => {
			loading = false;
		}, 1000);
	}

	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.select-menu-container')) {
			isOpen = false;
		}
	}

	$: selectedLabels = options
		.filter(opt => selectedValues.includes(opt.value))
		.map(opt => opt.label);
	$: displayText = selectedLabels.length > 0 ? selectedLabels.join(', ') : placeholder;
</script>

<svelte:window on:click={handleClickOutside} />

<div class="select-menu-container relative inline-block" class:opacity-50={disabled}>
	<div class="flex items-center gap-2">
		<button
			type="button"
			class="flex items-center gap-2 px-3 py-2 bg-[#2b2d31] hover:bg-[#36393f] rounded text-sm text-[#dbdee1] transition-colors min-w-[150px]"
			class:bg-[#36393f]={isOpen}
			{disabled}
			on:click|stopPropagation={toggleDropdown}
			aria-haspopup="listbox"
			aria-expanded={isOpen}
		>
			<span class="flex-1 text-left truncate">{displayText}</span>
			<svg class="w-4 h-4 transition-transform" class:rotate-180={isOpen} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
				<path d="M7 10l5 5 5-5z"/>
			</svg>
		</button>
	</div>

	{#if isOpen}
		<div 
			class="absolute top-full left-0 mt-1 w-full min-w-[200px] max-h-[300px] overflow-y-auto bg-[#36393f] rounded-lg shadow-xl border border-[#202225] z-50"
			role="listbox"
			aria-multiselectable={maxValues > 1}
		>
			{#each options as option}
				<button
					type="button"
					class="w-full flex items-start gap-3 px-3 py-2 hover:bg-[#4f545c] transition-colors text-left"
					class:bg-[#5865f2]={isSelected(option.value)}
					class:bg-transparent={!isSelected(option.value)}
					on:click|stopPropagation={() => selectOption(option.value)}
					role="option"
					aria-selected={isSelected(option.value)}
				>
					{#if maxValues > 1}
						<span class="flex-shrink-0 mt-0.5">
							{#if isSelected(option.value)}
								<svg viewBox="0 0 24 24" width="18" height="18" fill="#fff" aria-hidden="true">
									<rect x="3" y="3" width="18" height="18" rx="3" fill="#5865f2"/>
									<path d="M9 12l2 2 4-4" stroke="#fff" stroke-width="2" fill="none"/>
								</svg>
							{:else}
								<svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden="true">
									<rect x="3" y="3" width="18" height="18" rx="3" stroke="#80848e" stroke-width="2"/>
								</svg>
							{/if}
						</span>
					{/if}
					{#if option.emoji}
						<span class="text-lg flex-shrink-0" aria-hidden="true">{option.emoji}</span>
					{/if}
					<div class="flex-1 min-w-0">
						<div class="text-sm font-medium text-[#dbdee1]">{option.label}</div>
						{#if option.description}
							<div class="text-xs text-[#b5bac1] truncate">{option.description}</div>
						{/if}
					</div>
				</button>
			{/each}
		</div>
	{/if}

	{#if maxValues > 1 && selectedValues.length > 0}
		<button
			type="button"
			class="mt-2 px-3 py-1.5 bg-[#5865f2] hover:bg-[#4752c4] text-white text-xs rounded transition-colors"
			class:opacity-50={loading}
			on:click|stopPropagation={submit}
		>
			{loading ? 'Submitting...' : `Submit (${selectedValues.length} selected)`}
		</button>
	{/if}
</div>
