<script lang="ts">
	import type { ForumTag } from '$lib/stores/forumTags';

	export let tag: ForumTag;
	export let size: 'sm' | 'md' = 'sm';
	export let removable = false;
	export let onRemove: (() => void) | null = null;

	$: displayEmoji = tag.emoji_name || null;
</script>

<span class="forum-tag" class:sm={size === 'sm'} class:md={size === 'md'}>
	{#if displayEmoji}
		<span class="tag-emoji">{displayEmoji}</span>
	{/if}
	<span class="tag-name">{tag.name}</span>
	{#if removable && onRemove}
		<button
			class="remove-btn"
			on:click|stopPropagation={onRemove}
			aria-label="Remove tag {tag.name}"
		>
			×
		</button>
	{/if}
</span>

<style>
	.forum-tag {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 8px;
		background: var(--bg-modifier-accent, #4e5058);
		border-radius: var(--radius-sm, 4px);
		font-size: var(--font-size-xs, 10px);
		font-weight: 500;
		color: var(--text-primary, #f2f3f5);
		white-space: nowrap;
		line-height: 1.4;
	}

	.forum-tag.sm {
		padding: 2px 6px;
		font-size: var(--font-size-xs, 10px);
	}

	.forum-tag.md {
		padding: 4px 10px;
		font-size: var(--font-size-sm, 12px);
	}

	.tag-emoji {
		font-size: 1em;
		line-height: 1;
		flex-shrink: 0;
	}

	.tag-name {
		max-width: 120px;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.remove-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 14px;
		height: 14px;
		padding: 0;
		margin-left: 2px;
		background: transparent;
		border: none;
		border-radius: 50%;
		color: var(--text-muted, #b5bac1);
		font-size: 12px;
		line-height: 1;
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
	}

	.remove-btn:hover {
		background: var(--bg-primary, #36393f);
		color: var(--text-primary, #f2f3f5);
	}
</style>
