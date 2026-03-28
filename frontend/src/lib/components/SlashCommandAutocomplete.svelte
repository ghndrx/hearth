<script lang="ts">
	import { createEventDispatcher, onMount, onDestroy } from 'svelte';
	import { slashCommandUI, selectedAutocompleteItem, shouldShowAutocomplete } from '$lib/stores/slashCommandsUI';
	import { getOptionTypeLabel } from '$lib/services/slashCommands';
	import type { AutocompleteResult, SlashCommand } from '$lib/services/slashCommands';
	import type { Readable } from 'svelte/store';

	export let commands: SlashCommand[] = [];
	export let position: { top: number; left: number } = { top: 0, left: 0 };
	
	const dispatch = createEventDispatcher<{
		select: { command: SlashCommand };
		execute: { command: SlashCommand; options: Record<string, unknown> };
		escape: void;
	}>();
	
	let inputEl: HTMLInputElement | null = null;
	
	$: state = $slashCommandUI;
	$: results = state.autocompleteResults;
	$: selectedIdx = state.selectedIndex;
	$: show = $shouldShowAutocomplete;
	
	function handleKeydown(e: KeyboardEvent) {
		if (!show) return;
		
		switch (e.key) {
			case 'ArrowDown':
				e.preventDefault();
				e.stopPropagation();
				slashCommandUI.selectNext(results.length);
				break;
			case 'ArrowUp':
				e.preventDefault();
				e.stopPropagation();
				slashCommandUI.selectPrev();
				break;
			case 'Enter':
				e.preventDefault();
				e.stopPropagation();
				if (results.length > 0 && results[selectedIdx]) {
					selectItem(results[selectedIdx]);
				}
				break;
			case 'Escape':
				e.preventDefault();
				e.stopPropagation();
				slashCommandUI.hideAutocomplete();
				dispatch('escape');
				break;
			case 'Tab':
				e.preventDefault();
				e.stopPropagation();
				if (results.length > 0) {
					selectItem(results[selectedIdx]);
				}
				break;
		}
	}
	
	function selectItem(item: AutocompleteResult) {
		dispatch('select', { command: item.command });
	}
	
	function handleMouseEnter(idx: number) {
		slashCommandUI.setSelectedIndex(idx);
	}
	
	function handleClick(command: SlashCommand) {
		dispatch('select', { command });
	}
	
	function getOptionPreview(cmd: SlashCommand): string {
		if (!cmd.options || cmd.options.length === 0) return '';
		
		const preview: string[] = [];
		for (const opt of cmd.options.slice(0, 3)) {
			const typeLabel = getOptionTypeLabel(opt.type as any);
			if (opt.type === 1 || opt.type === 2) {
				preview.push(opt.name);
			} else {
				const req = opt.required ? '*' : '';
				preview.push(`<${opt.name}${req}>`);
			}
		}
		
		const suffix = cmd.options.length > 3 ? '...' : '';
		return preview.join(' ') + suffix;
	}
	
	$: style = `top: ${position.top}px; left: ${position.left}px;`;
</script>

{#if show && results.length > 0}
	<div 
		class="autocomplete-dropdown"
		{style}
		role="listbox"
		aria-label="Slash command suggestions"
	>
		<div class="autocomplete-header">
			<span class="header-icon">/</span>
			<span class="header-text">Commands</span>
		</div>
		
		{#each results as item, idx}
			{@const cmd = item.command}
			<div
				class="autocomplete-item"
				class:selected={idx === selectedIdx}
				role="option"
				aria-selected={idx === selectedIdx}
				on:mouseenter={() => handleMouseEnter(idx)}
				on:click={() => handleClick(cmd)}
				on:keydown={(e) => e.key === 'Enter' && handleClick(cmd)}
				tabindex="-1"
			>
				<div class="command-name">
					<span class="command-icon">/</span>
					<span class="name">{cmd.name}</span>
				</div>
				<div class="command-desc">{cmd.description}</div>
				{#if cmd.options && cmd.options.length > 0}
					<div class="command-preview">{getOptionPreview(cmd)}</div>
				{/if}
				
				{#if item.choices && item.choices.length > 0}
					<div class="choices-grid">
						{#each item.choices.slice(0, 5) as choice}
							<button 
								class="choice-chip"
								on:click|stopPropagation={() => {
									dispatch('execute', { command: cmd, options: { [item.focusedOption?.name || '']: choice.value } });
								}}
							>
								{choice.name}
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/each}
		
		{#if results.length === 0 && state.inputValue.length > 1}
			<div class="no-results">
				No commands found matching "{state.inputValue.slice(1)}"
			</div>
		{/if}
	</div>
{/if}

<style>
	.autocomplete-dropdown {
		position: absolute;
		z-index: 1000;
		min-width: 320px;
		max-width: 420px;
		max-height: 360px;
		overflow-y: auto;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 8px;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
		padding: 4px;
	}
	
	.autocomplete-header {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 12px 6px;
		color: var(--text-muted, #949ba4);
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}
	
	.header-icon {
		font-weight: 700;
		color: var(--text-normal, #dbdee1);
	}
	
	.autocomplete-item {
		padding: 8px 12px;
		border-radius: 4px;
		cursor: pointer;
		transition: background-color 0.1s;
	}
	
	.autocomplete-item:hover,
	.autocomplete-item.selected {
		background: var(--bg-modifier-hover, #3f434a);
	}
	
	.autocomplete-item.selected {
		background: var(--bg-modifier-selected, #4e545c);
	}
	
	.command-name {
		display: flex;
		align-items: center;
		gap: 4px;
		font-weight: 500;
		color: var(--text-normal, #dbdee1);
	}
	
	.command-icon {
		color: var(--text-muted, #949ba4);
		font-weight: 600;
	}
	
	.command-desc {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
		margin-top: 2px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	
	.command-preview {
		font-size: 11px;
		color: var(--text-link, #5865f2);
		margin-top: 4px;
		font-family: monospace;
	}
	
	.choices-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-top: 6px;
	}
	
	.choice-chip {
		padding: 2px 8px;
		background: var(--bg-modifier-hover, #3f434a);
		border: 1px solid var(--bg-tertiary, #1e1f22);
		border-radius: 12px;
		font-size: 11px;
		color: var(--text-normal, #dbdee1);
		cursor: pointer;
		transition: all 0.1s;
	}
	
	.choice-chip:hover {
		background: var(--bg-modifier-selected, #4e545c);
		border-color: var(--text-link, #5865f2);
	}
	
	.no-results {
		padding: 12px;
		text-align: center;
		color: var(--text-muted, #949ba4);
		font-size: 13px;
	}
</style>
