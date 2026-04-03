<script lang="ts">
	import { onMount, createEventDispatcher } from 'svelte';
	import { api } from '$lib/api';
	import ForumTagBadge from './ForumTagBadge.svelte';
	import ForumTagPicker from './ForumTagPicker.svelte';
	import { type ForumTag } from '$lib/stores/forumTags';

	export let channelId: string;

	const dispatch = createEventDispatcher<{
		openPost: { threadId: string };
		createPost: void;
	}>();

	interface ForumPost {
		id: string;
		name: string;
		owner_id: string;
		owner?: {
			id: string;
			username: string;
			display_name?: string;
			avatar?: string;
		};
		message_count: number;
		applied_tags: string[];
		tags?: ForumTag[];
		created_at: string;
		is_pinned: boolean;
		archived: boolean;
	}

	interface ForumPostsResponse {
		threads: ForumPost[];
		total: number;
		has_more: boolean;
	}

	let posts: ForumPost[] = [];
	let tags: ForumTag[] = [];
	let loading = false;
	let error = '';
	let total = 0;
	let hasMore = false;
	let offset = 0;

	// Filter/sort state
	let selectedTagIds: string[] = [];
	let sortOrder: 0 | 1 | 2 = 0; // 0=latest_activity, 1=creation_date, 2=pin_weight
	let layout: 'list' | 'gallery' = 'list';
	let searchQuery = '';

	const LIMIT = 25;

	onMount(async () => {
		await Promise.all([loadTags(), loadPosts()]);
	});

	async function loadTags() {
		try {
			const response = await api.get<{ tags: ForumTag[] }>(`/channels/${channelId}/tags`);
			tags = response.tags || [];
		} catch (err) {
			console.error('[ForumChannelView] Failed to load tags:', err);
		}
	}

	async function loadPosts(reset = true) {
		if (loading) return;
		loading = true;
		error = '';

		if (reset) {
			offset = 0;
			posts = [];
		}

		try {
			const params = new URLSearchParams();
			params.set('limit', String(LIMIT));
			params.set('offset', String(offset));
			params.set('sort', String(sortOrder));
			if (selectedTagIds.length > 0) {
				params.set('tag_ids', selectedTagIds.join(','));
			}

			const response = await api.get<ForumPostsResponse>(
				`/channels/${channelId}/posts?${params.toString()}`
			);

			if (reset) {
				posts = response.threads || [];
			} else {
				posts = [...posts, ...(response.threads || [])];
			}
			total = response.total ?? 0;
			hasMore = response.has_more ?? false;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load posts';
			console.error('[ForumChannelView] Failed to load posts:', err);
		} finally {
			loading = false;
		}
	}

	function handleTagFilterChange(event: CustomEvent<string[]>) {
		selectedTagIds = event.detail;
		loadPosts(true);
	}

	function handleSortChange(newSort: 0 | 1 | 2) {
		sortOrder = newSort;
		loadPosts(true);
	}

	function toggleLayout() {
		layout = layout === 'list' ? 'gallery' : 'list';
	}

	function loadMore() {
		offset += LIMIT;
		loadPosts(false);
	}

	function formatTimeAgo(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);

		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		if (days < 7) return `${days}d ago`;
		return date.toLocaleDateString();
	}

	function getTagById(tagId: string): ForumTag | undefined {
		return tags.find(t => t.id === tagId);
	}

	$: sortedAndFilteredPosts = posts;
</script>

<div class="forum-channel-view">
	<!-- Toolbar -->
	<div class="forum-toolbar">
		<div class="toolbar-left">
			<div class="sort-options" role="group" aria-label="Sort posts">
				<button
					class="sort-btn"
					class:active={sortOrder === 0}
					on:click={() => handleSortChange(0)}
					title="Latest Activity"
				>
					🕐 Latest Activity
				</button>
				<button
					class="sort-btn"
					class:active={sortOrder === 1}
					on:click={() => handleSortChange(1)}
					title="Creation Date"
				>
					📅 Creation Date
				</button>
				<button
					class="sort-btn"
					class:active={sortOrder === 2}
					on:click={() => handleSortChange(2)}
					title="Pin Weight"
				>
					📌 Pinned
				</button>
			</div>
		</div>

		<div class="toolbar-right">
			<!-- Tag filter -->
			<div class="tag-filter">
				<ForumTagPicker
					{channelId}
					bind:selectedTagIds
					on:change={handleTagFilterChange}
				/>
			</div>

			<!-- Layout toggle -->
			<div class="layout-toggle" role="group" aria-label="Layout">
				<button
					class="layout-btn"
					class:active={layout === 'list'}
					on:click={() => (layout = 'list')}
					title="List view"
				>
					≡
				</button>
				<button
					class="layout-btn"
					class:active={layout === 'gallery'}
					on:click={() => (layout = 'gallery')}
					title="Gallery view"
				>
					⊞
				</button>
			</div>
		</div>
	</div>

	<!-- Posts count -->
	<div class="posts-header">
		<span class="posts-count">
			{total} {total === 1 ? 'post' : 'posts'}
			{#if selectedTagIds.length > 0}
				<span class="filter-note">filtered by {selectedTagIds.length} tag{selectedTagIds.length > 1 ? 's' : ''}</span>
			{/if}
		</span>
	</div>

	<!-- Loading state -->
	{#if loading && posts.length === 0}
		<div class="loading-state">
			<div class="spinner"></div>
			<span>Loading posts...</span>
		</div>
	{:else if error && posts.length === 0}
		<!-- Error state -->
		<div class="error-state">
			<span class="error-icon">⚠️</span>
			<span class="error-text">{error}</span>
			<button class="retry-btn" on:click={() => loadPosts(true)}>Retry</button>
		</div>
	{:else if posts.length === 0}
		<!-- Empty state -->
		<div class="empty-state">
			{#if selectedTagIds.length > 0}
				<span class="empty-icon">🏷️</span>
				<h3>No posts with these tags</h3>
				<p>Try selecting different tags or clear the filter.</p>
				<button class="clear-filter-btn" on:click={() => { selectedTagIds = []; loadPosts(true); }}>
					Clear Tag Filter
				</button>
			{:else}
				<span class="empty-icon">📝</span>
				<h3>No posts yet</h3>
				<p>Be the first to start a discussion!</p>
				<button class="create-btn" on:click={() => dispatch('createPost')}>
					Create Post
				</button>
			{/if}
		</div>
	{:else}
		<!-- Posts list/gallery -->
		<div class="posts-container" class:list-view={layout === 'list'} class:gallery-view={layout === 'gallery'}>
			{#each posts as post (post.id)}
				<button
					class="forum-post"
					class:pinned={post.is_pinned}
					on:click={() => dispatch('openPost', { threadId: post.id })}
				>
					{#if post.is_pinned}
						<span class="pin-indicator" title="Pinned">📌</span>
					{/if}

					<div class="post-main">
						<div class="post-header">
							<h3 class="post-title">{post.name}</h3>
							{#if post.archived}
								<span class="archived-badge">Archived</span>
							{/if}
						</div>

						<!-- Applied tags -->
						{#if post.applied_tags && post.applied_tags.length > 0}
							<div class="post-tags">
								{#each post.applied_tags as tagId}
									{@const tag = getTagById(tagId)}
									{#if tag}
										<ForumTagBadge {tag} size="sm" />
									{/if}
								{/each}
							</div>
						{/if}

						<div class="post-meta">
							<span class="post-author">
								{#if post.owner}
									<span class="author-avatar">
										{post.owner.avatar ? '🧑' : '👤'}
									</span>
									<span class="author-name">
										{post.owner.display_name || post.owner.username}
									</span>
								{/if}
							</span>
							<span class="post-stats">
								💬 {post.message_count}
							</span>
							<span class="post-time">
								{formatTimeAgo(post.created_at)}
							</span>
						</div>
					</div>
				</button>
			{/each}
		</div>

		<!-- Load more -->
		{#if hasMore}
			<div class="load-more">
				<button class="load-more-btn" on:click={loadMore} disabled={loading}>
					{loading ? 'Loading...' : 'Load More'}
				</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	.forum-channel-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.forum-toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 16px;
		border-bottom: 1px solid var(--border-color, #4f545c);
		background: var(--bg-secondary, #2b2d31);
		flex-shrink: 0;
	}

	.toolbar-left,
	.toolbar-right {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.sort-options {
		display: flex;
		gap: 4px;
	}

	.sort-btn {
		padding: 4px 10px;
		background: transparent;
		border: 1px solid var(--border-color, #4f545c);
		border-radius: var(--radius-sm, 4px);
		color: var(--text-secondary, #b5bac1);
		font-size: var(--font-size-xs, 11px);
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
		white-space: nowrap;
	}

	.sort-btn:hover {
		background: var(--bg-modifier-accent, #4e5058);
		color: var(--text-primary, #f2f3f5);
	}

	.sort-btn.active {
		background: var(--bg-modifier-selected, #4e5058);
		color: var(--text-primary, #f2f3f5);
		border-color: var(--text-muted, #b5bac1);
	}

	.tag-filter {
		display: flex;
		align-items: center;
	}

	.layout-toggle {
		display: flex;
		border: 1px solid var(--border-color, #4f545c);
		border-radius: var(--radius-sm, 4px);
		overflow: hidden;
	}

	.layout-btn {
		padding: 4px 8px;
		background: transparent;
		border: none;
		color: var(--text-secondary, #b5bac1);
		font-size: 14px;
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
	}

	.layout-btn:hover {
		background: var(--bg-modifier-accent, #4e5058);
	}

	.layout-btn.active {
		background: var(--bg-modifier-selected, #4e5058);
		color: var(--text-primary, #f2f3f5);
	}

	.posts-header {
		padding: 8px 16px;
		font-size: var(--font-size-xs, 11px);
		color: var(--text-muted, #b5bac1);
		flex-shrink: 0;
	}

	.posts-count {
		font-weight: 500;
	}

	.filter-note {
		margin-left: 8px;
		font-style: italic;
	}

	/* Loading state */
	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		flex: 1;
		color: var(--text-muted, #b5bac1);
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--border-color, #4f545c);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Error state */
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		flex: 1;
		color: var(--text-danger, #ed4245);
	}

	.retry-btn {
		padding: 6px 16px;
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: white;
		font-size: var(--font-size-sm, 13px);
		cursor: pointer;
	}

	/* Empty state */
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		flex: 1;
		color: var(--text-secondary, #b5bac1);
		text-align: center;
		padding: 24px;
	}

	.empty-icon {
		font-size: 48px;
		margin-bottom: 8px;
	}

	.empty-state h3 {
		margin: 0;
		font-size: var(--font-size-lg, 18px);
		color: var(--text-primary, #f2f3f5);
	}

	.empty-state p {
		margin: 0;
		font-size: var(--font-size-sm, 13px);
	}

	.clear-filter-btn,
	.create-btn {
		margin-top: 8px;
		padding: 8px 20px;
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: white;
		font-size: var(--font-size-sm, 13px);
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s;
	}

	.clear-filter-btn:hover,
	.create-btn:hover {
		background: #4752c4;
	}

	/* Posts list */
	.posts-container {
		flex: 1;
		overflow-y: auto;
		padding: 0 8px 8px;
	}

	.posts-container.list-view {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.posts-container.gallery-view {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 8px;
	}

	.forum-post {
		position: relative;
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 12px;
		background: transparent;
		border: none;
		border-radius: var(--radius-md, 8px);
		text-align: left;
		cursor: pointer;
		transition: background 0.1s;
		width: 100%;
	}

	.forum-post:hover {
		background: var(--bg-modifier-accent, #4e5058);
	}

	.forum-post.pinned {
		background: var(--bg-modifier-accent, rgba(88, 101, 242, 0.1));
	}

	.forum-post.pinned:hover {
		background: rgba(88, 101, 242, 0.2);
	}

	.pin-indicator {
		position: absolute;
		top: 8px;
		right: 8px;
		font-size: 12px;
	}

	.post-main {
		flex: 1;
		min-width: 0;
	}

	.post-header {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 4px;
	}

	.post-title {
		margin: 0;
		font-size: var(--font-size-md, 15px);
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.archived-badge {
		font-size: var(--font-size-xs, 10px);
		padding: 1px 6px;
		background: var(--bg-modifier-accent, #4e5058);
		border-radius: 3px;
		color: var(--text-muted, #b5bac1);
		text-transform: uppercase;
		flex-shrink: 0;
	}

	.post-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-bottom: 6px;
	}

	.post-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: var(--font-size-xs, 11px);
		color: var(--text-muted, #b5bac1);
	}

	.post-author {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.author-avatar {
		font-size: 14px;
	}

	.author-name {
		font-weight: 500;
	}

	/* Load more */
	.load-more {
		display: flex;
		justify-content: center;
		padding: 12px;
		flex-shrink: 0;
	}

	.load-more-btn {
		padding: 8px 24px;
		background: var(--bg-modifier-accent, #4e5058);
		border: none;
		border-radius: var(--radius-sm, 4px);
		color: var(--text-primary, #f2f3f5);
		font-size: var(--font-size-sm, 13px);
		cursor: pointer;
		transition: background 0.15s;
	}

	.load-more-btn:hover:not(:disabled) {
		background: var(--bg-modifier-selected, #5d6378);
	}

	.load-more-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
</style>
