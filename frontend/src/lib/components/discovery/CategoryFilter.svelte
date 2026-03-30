<script lang="ts">
	export let categories: Array<{
		id?: string;
		name: string;
		slug: string;
		icon?: string;
		server_count?: number;
		total_members?: number;
		avg_member_count?: number;
	}> = [];

	export let selectedCategory = 'all';
	export let showCounts = true;

	const defaultCategories: Array<{ id: string; name: string; slug: string; icon: string; server_count?: number }> = [
		{ id: 'all', name: 'All', slug: 'all', icon: '🏠', server_count: 0 },
		{ id: 'gaming', name: 'Gaming', slug: 'gaming', icon: '🎮', server_count: 0 },
		{ id: 'music', name: 'Music', slug: 'music', icon: '🎵', server_count: 0 },
		{ id: 'technology', name: 'Technology', slug: 'technology', icon: '💻', server_count: 0 },
		{ id: 'art', name: 'Art & Design', slug: 'art', icon: '🎨', server_count: 0 },
		{ id: 'education', name: 'Education', slug: 'education', icon: '📚', server_count: 0 },
		{ id: 'entertainment', name: 'Entertainment', slug: 'entertainment', icon: '🎬', server_count: 0 },
		{ id: 'social', name: 'Social', slug: 'social', icon: '💬', server_count: 0 },
		{ id: 'sports', name: 'Sports', slug: 'sports', icon: '⚽', server_count: 0 },
		{ id: 'anime', name: 'Anime & Manga', slug: 'anime', icon: '🍜', server_count: 0 },
		{ id: 'science', name: 'Science', slug: 'science', icon: '🔬', server_count: 0 },
		{ id: 'fashion', name: 'Fashion', slug: 'fashion', icon: '👗', server_count: 0 },
		{ id: 'food', name: 'Food & Cooking', slug: 'food', icon: '🍳', server_count: 0 },
		{ id: 'business', name: 'Business', slug: 'business', icon: '💼', server_count: 0 },
		{ id: 'language', name: 'Language Learning', slug: 'language', icon: '🗣️', server_count: 0 },
	];

	$: allCategories = categories.length > 0
		? [
				{ id: 'all', name: 'All', slug: 'all', icon: '🏠', server_count: categories.reduce((sum, c) => sum + (c.server_count || 0), 0) },
				...categories.map(c => ({ ...c, id: c.slug, icon: c.icon || getDefaultIcon(c.slug) }))
		  ]
		: defaultCategories;

	function getDefaultIcon(slug: string): string {
		const found = defaultCategories.find(c => c.slug === slug);
		return found?.icon || '🏠';
	}

	function formatCount(count: number): string {
		if (count >= 1000000) {
			return (count / 1000000).toFixed(1) + 'M';
		}
		if (count >= 1000) {
			return (count / 1000).toFixed(1) + 'K';
		}
		return count.toString();
	}

	function selectCategory(categoryId: string) {
		selectedCategory = categoryId;
	}

	import { createEventDispatcher } from 'svelte';
	const dispatch = createEventDispatcher();

	function handleSelect(categoryId: string) {
		selectCategory(categoryId);
		dispatch('select', categoryId);
	}
</script>

<div class="category-filter">
	<div class="category-header">
		<h3>Categories</h3>
	</div>
	<nav class="category-list" aria-label="Server categories">
		{#each allCategories as category}
			<button
				class="category-item"
				class:active={selectedCategory === category.id}
				on:click={() => handleSelect(category.id || category.slug)}
				aria-pressed={selectedCategory === category.id}
			>
				<span class="category-icon">{category.icon || '🏠'}</span>
				<span class="category-name">{category.name}</span>
				{#if showCounts && category.server_count !== undefined}
					<span class="category-count">{formatCount(category.server_count)}</span>
				{/if}
			</button>
		{/each}
	</nav>
</div>

<style>
	.category-filter {
		display: flex;
		flex-direction: column;
	}

	.category-header {
		padding: 12px 16px;
		border-bottom: 1px solid #1e1f22;
	}

	.category-header h3 {
		margin: 0;
		font-size: 12px;
		font-weight: 600;
		color: #6d6f78;
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.category-list {
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 2px;
		overflow-y: auto;
		max-height: calc(100vh - 200px);
	}

	.category-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 12px;
		border: none;
		border-radius: 4px;
		background: transparent;
		color: #949ba4;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s, color 0.15s;
		text-align: left;
		width: 100%;
	}

	.category-item:hover {
		background: #35373c;
		color: #dbdee1;
	}

	.category-item.active {
		background: #404249;
		color: #f2f3f5;
	}

	.category-icon {
		font-size: 18px;
		width: 24px;
		text-align: center;
		flex-shrink: 0;
	}

	.category-name {
		flex: 1;
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.category-count {
		font-size: 12px;
		color: #6d6f78;
		background: #1e1f22;
		padding: 2px 6px;
		border-radius: 4px;
		flex-shrink: 0;
	}

	.category-item.active .category-count {
		background: #2b2d31;
		color: #949ba4;
	}

	/* Scrollbar styling */
	.category-list::-webkit-scrollbar {
		width: 6px;
	}

	.category-list::-webkit-scrollbar-track {
		background: transparent;
	}

	.category-list::-webkit-scrollbar-thumb {
		background: #1e1f22;
		border-radius: 3px;
	}

	.category-list::-webkit-scrollbar-thumb:hover {
		background: #3f4147;
	}
</style>
