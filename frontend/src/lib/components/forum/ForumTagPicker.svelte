<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { forumTags, type ForumTag } from '$lib/stores/forumTags';
	import ForumTagBadge from './ForumTagBadge.svelte';

	export let channelId: string;
	export let selectedTagIds: string[] = [];
	export let required = false;
	export let maxSelectable = 5;

	const dispatch = createEventDispatcher<{
		change: string[];
	}>();

	let tags: ForumTag[] = [];
	let loading = false;
	let error = '';
	let showDropdown = false;

	// Load tags on mount
	async function loadTags() {
		loading = true;
		error = '';
		tags = await forumTags.loadChannelTags(channelId);
		loading = false;
	}

	$: {
		// Load tags when channelId changes
		if (channelId) {
			loadTags();
		}
	}

	$: selectedTags = tags.filter(t => selectedTagIds.includes(t.id));

	$: toggleTag = (tagId: string) => {
		if (selectedTagIds.includes(tagId)) {
			selectedTagIds = selectedTagIds.filter(id => id !== tagId);
		} else if (selectedTagIds.length < maxSelectable) {
			selectedTagIds = [...selectedTagIds, tagId];
		}
		dispatch('change', selectedTagIds);
	};

	$: removeTag = (tagId: string) => {
		selectedTagIds = selectedTagIds.filter(id => id !== tagId);
		dispatch('change', selectedTagIds);
	};

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			showDropdown = false;
		}
	}
</script>

<div class="tag-picker" on:keydown={handleKeydown} role="group" aria-label="Forum tag picker">
	{#if loading}
		<span class="loading-text">Loading tags...</span>
	{:else if error}
		<span class="error-text">{error}</span>
	{:else if tags.length === 0}
		<span class="empty-text">No tags available</span>
	{:else}
		<!-- Selected tags display -->
		{#if selectedTags.length > 0}
			<div class="selected-tags" aria-label="Selected tags">
				{#each selectedTags as tag (tag.id)}
					<ForumTagBadge
						{tag}
						size="sm"
						removable={true}
						onRemove={() => removeTag(tag.id)}
					/>
				{/each}
				{#if selectedTagIds.length >= maxSelectable}
					<span class="limit-note">Max {maxSelectable} tags</span>
				{/if}
			</div>
		{/if}

		<!-- Tag selection dropdown -->
		<div class="tag-dropdown-wrapper">
			<button
				type="button"
				class="add-tag-btn"
				class:has-selection={selectedTagIds.length > 0}
				on:click={() => (showDropdown = !showDropdown)}
				disabled={selectedTagIds.length >= maxSelectable && !required}
				aria-expanded={showDropdown}
				aria-haspopup="listbox"
			>
				<span class="btn-icon">+</span>
				<span>{selectedTagIds.length === 0 ? 'Add tags' : 'More tags'}</span>
			</button>

			{#if showDropdown}
				<div
					class="tag-dropdown"
					role="listbox"
					aria-label="Available tags"
					aria-multiselectable="true"
				>
					{#each tags as tag (tag.id)}
						{@const isSelected = selectedTagIds.includes(tag.id)}
						{@const isDisabled = !isSelected && selectedTagIds.length >= maxSelectable}
						<button
							type="button"
							class="tag-option"
							class:selected={isSelected}
							class:disabled={isDisabled}
							role="option"
							aria-selected={isSelected}
							disabled={isDisabled}
							on:click={() => toggleTag(tag.id)}
						>
							<span class="option-emoji">{tag.emoji_name || '🏷️'}</span>
							<span class="option-name">{tag.name}</span>
							{#if tag.moderated}
								<span class="mod-badge">Mod only</span>
							{/if}
							{#if isSelected}
								<span class="check-mark" aria-hidden="true">✓</span>
							{/if}
						</button>
					{/each}

					{#if required && selectedTagIds.length === 0}
						<p class="required-note">At least one tag is required</p>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Click outside to close -->
		{#if showDropdown}
			<div
				class="backdrop"
				on:click={() => (showDropdown = false)}
				on:keydown={() => {}}
				role="button"
				tabindex="-1"
				aria-label="Close tag picker"
			></div>
		{/if}
	{/if}
</div>

<style>
	.tag-picker {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px;
		padding: 6px 0;
	}

	.loading-text,
	.error-text,
	.empty-text {
		font-size: var(--font-size-xs, 11px);
		color: var(--text-muted, #b5bac1);
		font-style: italic;
	}

	.error-text {
		color: var(--text-danger, #ed4245);
	}

	.selected-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		align-items: center;
	}

	.limit-note {
		font-size: var(--font-size-xs, 10px);
		color: var(--text-muted, #b5bac1);
		font-style: italic;
	}

	.tag-dropdown-wrapper {
		position: relative;
	}

	.add-tag-btn {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 8px;
		background: transparent;
		border: 1px dashed var(--border-color, #4f545c);
		border-radius: var(--radius-sm, 4px);
		color: var(--text-muted, #b5bac1);
		font-size: var(--font-size-xs, 10px);
		cursor: pointer;
		transition: background 0.15s, border-color 0.15s, color 0.15s;
	}

	.add-tag-btn:hover:not(:disabled) {
		background: var(--bg-modifier-accent, #4e5058);
		border-color: var(--text-muted, #b5bac1);
		color: var(--text-primary, #f2f3f5);
	}

	.add-tag-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-icon {
		font-size: 12px;
		font-weight: 600;
	}

	.tag-dropdown {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		z-index: 100;
		min-width: 200px;
		max-width: 280px;
		max-height: 240px;
		overflow-y: auto;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid var(--border-color, #4f545c);
		border-radius: var(--radius-md, 8px);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
		padding: 4px;
	}

	.tag-option {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 6px 8px;
		background: transparent;
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: var(--text-primary, #f2f3f5);
		font-size: var(--font-size-sm, 13px);
		text-align: left;
		cursor: pointer;
		transition: background 0.1s;
	}

	.tag-option:hover:not(:disabled) {
		background: var(--bg-modifier-accent, #4e5058);
	}

	.tag-option.selected {
		background: var(--bg-modifier-selected, #4e5058);
	}

	.tag-option.disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.option-emoji {
		font-size: 14px;
		flex-shrink: 0;
	}

	.option-name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.mod-badge {
		font-size: 9px;
		padding: 1px 4px;
		background: var(--bg-modifier-accent, #4e5058);
		border-radius: 3px;
		color: var(--text-muted, #b5bac1);
		text-transform: uppercase;
		font-weight: 600;
	}

	.check-mark {
		color: var(--brand-primary, #5865f2);
		font-weight: 700;
		flex-shrink: 0;
	}

	.required-note {
		padding: 8px;
		text-align: center;
		font-size: var(--font-size-xs, 10px);
		color: var(--text-danger, #ed4245);
		font-style: italic;
		margin: 0;
	}

	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 99;
	}
</style>
