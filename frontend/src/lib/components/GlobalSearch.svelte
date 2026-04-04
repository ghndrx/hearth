<script lang="ts">
	import { createEventDispatcher, onMount, tick } from 'svelte';
	import { api } from '$lib/api';
	import Avatar from './Avatar.svelte';
	import LoadingSpinner from './LoadingSpinner.svelte';
	import EmptyState from './EmptyState.svelte';

	// ==================== Types ====================

	export interface GlobalSearchResult {
		id: string;
		channel_id: string;
		guild_id?: string;
		server_name?: string;
		channel_name: string;
		is_dm: boolean;
		author: {
			id: string;
			username: string;
			display_name?: string | null;
			avatar?: string | null;
		} | null;
		content: string;
		timestamp: string;
		edited_timestamp?: string | null;
		attachments?: { id: string; filename: string; url: string; alt_text?: string }[];
		pinned: boolean;
	}

	interface GlobalSearchResponse {
		messages: GlobalSearchResult[];
		total_count: number;
		has_more: boolean;
	}

	interface FilterState {
		query: string;
		author_id?: string;
		before?: string;
		after?: string;
		has_attachments?: boolean;
		has_embeds?: boolean;
		has_links?: boolean;
		has_reactions?: boolean;
		pinned?: boolean;
		server_ids: string[];
		include_dms: boolean;
	}

	// ==================== Props & State ====================

	const dispatch = createEventDispatcher<{
		jumpToMessage: { channelId: string; messageId: string; serverId?: string };
		close: void;
	}>();

	let searchInput = '';
	let searchInputEl: HTMLInputElement;
	let resultsContainer: HTMLDivElement;
	let searchTimeout: ReturnType<typeof setTimeout>;
	let selectedIndex = -1;

	let results: GlobalSearchResult[] = [];
	let totalCount = 0;
	let hasMore = false;
	let loading = false;
	let error: string | null = null;
	let offset = 0;

	// Filter state
	let includeDMs = true;
	let activeFilters: FilterState = {
		query: '',
		server_ids: [],
		include_dms: true,
	};

	// Filter dropdown states
	let showServerFilter = false;
	let showDateFilter = false;
	let showAuthorFilter = false;

	// ==================== Helper Functions ====================

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const isToday = date.toDateString() === now.toDateString();
		const isYesterday = new Date(now.getTime() - 86400000).toDateString() === date.toDateString();

		if (isToday) {
			return 'Today at ' + date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
		}
		if (isYesterday) {
			return 'Yesterday at ' + date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
		}
		return date.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' }) +
			' at ' + date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
	}

	function truncateContent(content: string, maxLength = 150): string {
		if (!content) return '';
		if (content.length <= maxLength) return content;
		return content.slice(0, maxLength).trim() + '…';
	}

	function escapeHtml(str: string): string {
		const div = document.createElement('div');
		div.textContent = str;
		return div.innerHTML;
	}

	function highlightMatches(content: string, query: string): string {
		if (!content || !query) return escapeHtml(content);
		
		const escaped = escapeHtml(content);
		const terms = query.toLowerCase().split(/\s+/).filter(Boolean);
		const escapedTerms = terms.map(t => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
		const regex = new RegExp(`(${escapedTerms.join('|')})`, 'gi');
		return escaped.replace(regex, '<mark class="search-highlight">$1</mark>');
	}

	function getServerContext(result: GlobalSearchResult): string {
		if (result.is_dm) {
			return 'Direct Message';
		}
		return result.server_name || 'Unknown Server';
	}

	function getServerIcon(result: GlobalSearchResult): string {
		if (result.is_dm) {
			return '💬';
		}
		return '🏠';
	}

	// ==================== Search Functions ====================

	async function search(append = false) {
		const query = searchInput.trim();
		
		if (!query) {
			results = [];
			totalCount = 0;
			hasMore = false;
			error = null;
			return;
		}

		if (!append) {
			offset = 0;
		}

		loading = true;
		error = null;

		try {
			const params = new URLSearchParams();
			params.set('q', query);
			params.set('limit', '25');
			params.set('offset', String(offset));

			if (activeFilters.author_id) {
				params.set('author_id', activeFilters.author_id);
			}
			if (activeFilters.before) {
				params.set('before', activeFilters.before);
			}
			if (activeFilters.after) {
				params.set('after', activeFilters.after);
			}
			if (activeFilters.has_attachments) {
				params.set('has_attachments', 'true');
			}
			if (activeFilters.has_embeds) {
				params.set('has_embeds', 'true');
			}
			if (activeFilters.has_links) {
				params.set('has_links', 'true');
			}
			if (activeFilters.has_reactions) {
				params.set('has_reactions', 'true');
			}
			if (activeFilters.pinned !== undefined) {
				params.set('pinned', String(activeFilters.pinned));
			}
			if (activeFilters.server_ids.length > 0) {
				for (const sid of activeFilters.server_ids) {
					params.append('server_ids', sid);
				}
			}
			if (activeFilters.include_dms) {
				params.set('include_dms', 'true');
			}

			const response = await api.get<GlobalSearchResponse>(`/search/global/messages?${params}`);

			if (append) {
				results = [...results, ...response.messages];
			} else {
				results = response.messages;
			}
			totalCount = response.total_count;
			hasMore = response.has_more;
			offset = append ? offset + response.messages.length : response.messages.length;
		} catch (err) {
			console.error('Global search failed:', err);
			error = err instanceof Error ? err.message : 'Search failed';
		} finally {
			loading = false;
		}
	}

	function handleSearchInput() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			search();
		}, 300);
	}

	function loadMore() {
		if (hasMore && !loading) {
			search(true);
		}
	}

	function handleScroll() {
		if (!resultsContainer) return;
		const { scrollTop, scrollHeight, clientHeight } = resultsContainer;
		if (scrollHeight - scrollTop - clientHeight < 100) {
			loadMore();
		}
	}

	// ==================== Event Handlers ====================

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			close();
			return;
		}

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			if (results.length > 0) {
				selectedIndex = selectedIndex < results.length - 1 ? selectedIndex + 1 : 0;
				scrollToSelected();
			}
			return;
		}

		if (e.key === 'ArrowUp') {
			e.preventDefault();
			if (results.length > 0) {
				selectedIndex = selectedIndex > 0 ? selectedIndex - 1 : results.length - 1;
				scrollToSelected();
			}
			return;
		}

		if (e.key === 'Enter' && selectedIndex >= 0 && selectedIndex < results.length) {
			e.preventDefault();
			jumpToMessage(results[selectedIndex]);
			return;
		}

		if (e.key === 'Enter' && selectedIndex < 0) {
			e.preventDefault();
			clearTimeout(searchTimeout);
			search();
		}
	}

	function scrollToSelected() {
		tick().then(() => {
			const selected = resultsContainer?.querySelector(`[data-index="${selectedIndex}"]`);
			selected?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		});
	}

	function jumpToMessage(result: GlobalSearchResult) {
		dispatch('jumpToMessage', {
			channelId: result.channel_id,
			messageId: result.id,
			serverId: result.guild_id,
		});
	}

	function close() {
		dispatch('close');
	}

	function toggleDMs() {
		includeDMs = !includeDMs;
		activeFilters.include_dms = includeDMs;
		search();
	}

	function toggleFilter(filter: 'has_attachments' | 'has_embeds' | 'has_links' | 'has_reactions') {
		activeFilters[filter] = !activeFilters[filter];
		search();
	}

	function clearFilters() {
		activeFilters = {
			query: searchInput,
			server_ids: [],
			include_dms: includeDMs,
		};
		search();
	}

	onMount(() => {
		searchInputEl?.focus();
	});

	$: authorName = (result: GlobalSearchResult) => 
		result.author?.display_name || result.author?.username || 'Unknown User';
</script>

<aside class="global-search-panel" role="search" aria-label="Global cross-server search">
	<!-- Header -->
	<header class="search-header">
		<div class="search-header-left">
			<svg class="search-icon-header" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
				<path d="M21.707 20.293L16.314 14.9C17.403 13.504 18 11.799 18 10C18 5.589 14.411 2 10 2C5.589 2 2 5.589 2 10C2 14.411 5.589 18 10 18C11.799 18 13.504 17.403 14.9 16.314L20.293 21.707L21.707 20.293ZM10 16C6.691 16 4 13.309 4 10C4 6.691 6.691 4 10 4C13.309 4 16 6.691 16 10C16 13.309 13.309 16 10 16Z" />
			</svg>
			<span class="search-title">Global Search</span>
		</div>
		<button class="close-btn" on:click={close} title="Close search" aria-label="Close global search" type="button">
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</header>

	<!-- Search Input -->
	<div class="search-input-container">
		<div class="search-input-wrapper">
			<svg class="search-input-icon" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
				<path d="M21.707 20.293L16.314 14.9C17.403 13.504 18 11.799 18 10C18 5.589 14.411 2 10 2C5.589 2 2 5.589 2 10C2 14.411 5.589 18 10 18C11.799 18 13.504 17.403 14.9 16.314L20.293 21.707L21.707 20.293ZM10 16C6.691 16 4 13.309 4 10C4 6.691 6.691 4 10 4C13.309 4 16 6.691 16 10C16 13.309 13.309 16 10 16Z" />
			</svg>
			<input
				bind:this={searchInputEl}
				bind:value={searchInput}
				on:input={handleSearchInput}
				on:keydown={handleKeydown}
				type="text"
				class="search-input"
				placeholder="Search across all servers..."
				aria-label="Search across all servers"
			/>
			{#if searchInput}
				<button 
					class="clear-btn" 
					on:click={() => { searchInput = ''; results = []; totalCount = 0; }}
					title="Clear search"
					aria-label="Clear search"
					type="button"
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
						<path d="M18.4 4L12 10.4L5.6 4L4 5.6L10.4 12L4 18.4L5.6 20L12 13.6L18.4 20L20 18.4L13.6 12L20 5.6L18.4 4Z" />
					</svg>
				</button>
			{/if}
		</div>

		<!-- Quick Filters -->
		<div class="quick-filters">
			<label class="filter-toggle" title="Include Direct Messages">
				<input type="checkbox" bind:checked={includeDMs} on:change={toggleDMs} />
				<span>💬 DMs</span>
			</label>
			<button 
				class="filter-btn" 
				class:active={activeFilters.has_attachments}
				on:click={() => toggleFilter('has_attachments')}
				title="Has attachments"
				type="button"
			>
				📎 Attachments
			</button>
			<button 
				class="filter-btn" 
				class:active={activeFilters.has_links}
				on:click={() => toggleFilter('has_links')}
				title="Has links"
				type="button"
			>
				🔗 Links
			</button>
			<button 
				class="filter-btn" 
				class:active={activeFilters.has_reactions}
				on:click={() => toggleFilter('has_reactions')}
				title="Has reactions"
				type="button"
			>
				👍 Reactions
			</button>
			<button 
				class="filter-btn" 
				class:active={activeFilters.has_embeds}
				on:click={() => toggleFilter('has_embeds')}
				title="Has embeds"
				type="button"
			>
				📋 Embeds
			</button>
			{#if activeFilters.has_attachments || activeFilters.has_links || activeFilters.has_reactions || activeFilters.has_embeds}
				<button class="clear-filters-btn" on:click={clearFilters} type="button">
					Clear filters
				</button>
			{/if}
		</div>
	</div>

	<!-- Results Count -->
	{#if searchInput.trim() && !loading}
		<div class="results-count" role="status" aria-live="polite">
			{#if totalCount > 0}
				<span>🔍 {totalCount.toLocaleString()} result{totalCount !== 1 ? 's' : ''} across all servers</span>
			{:else if results.length === 0}
				<span>No results found</span>
			{/if}
		</div>
	{/if}

	<!-- Loading State -->
	{#if loading && results.length === 0}
		<div class="loading-container">
			<LoadingSpinner />
			<span>Searching...</span>
		</div>
	{/if}

	<!-- Error State -->
	{#if error}
		<div class="error-container">
			<span class="error-icon">⚠️</span>
			<span>{error}</span>
		</div>
	{/if}

	<!-- Results List -->
	<div 
		bind:this={resultsContainer}
		class="results-container"
		on:scroll={handleScroll}
		role="list"
		aria-label="Search results"
	>
		{#each results as result, i (result.id)}
			<article 
				class="search-result"
				class:selected={selectedIndex === i}
				data-index={i}
				on:click={() => jumpToMessage(result)}
				on:keydown={(e) => e.key === 'Enter' && jumpToMessage(result)}
				role="listitem"
				tabindex="0"
			>
				<!-- Server Context -->
				<div class="result-context">
					<span class="server-icon" title={getServerContext(result)}>
						{getServerIcon(result)}
					</span>
					<span class="server-name" title={result.server_name || 'Direct Message'}>
						{result.is_dm ? 'Direct Message' : result.server_name}
					</span>
					<span class="channel-separator">›</span>
					<span class="channel-name" title={result.channel_name}>
						# {result.channel_name}
					</span>
				</div>

				<!-- Message Preview -->
				<div class="result-content">
					<div class="result-header">
						<Avatar 
							user={result.author} 
							size={20} 
							showStatus={false}
						/>
						<span class="author-name">
							{authorName(result)}
						</span>
						<span class="message-time">
							{formatTime(result.timestamp)}
						</span>
					</div>
					<p class="message-preview">
						{@html highlightMatches(truncateContent(result.content), searchInput)}
					</p>
					{#if result.attachments && result.attachments.length > 0}
						<div class="attachment-indicator">
							📎 {result.attachments.length} attachment{result.attachments.length !== 1 ? 's' : ''}
						</div>
					{/if}
				</div>

				<!-- Jump Arrow -->
				<div class="result-arrow">
					<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
						<path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6-6-6z" />
					</svg>
				</div>
			</article>
		{/each}

		<!-- Load More Indicator -->
		{#if loading && results.length > 0}
			<div class="loading-more">
				<LoadingSpinner size={20} />
			</div>
		{/if}

		<!-- Empty State for No Results -->
		{#if !loading && searchInput.trim() && results.length === 0}
			<div class="empty-state">
				<EmptyState
					title="No results found"
					message="Try different keywords or remove some filters"
					icon="🔍"
				/>
			</div>
		{/if}

		<!-- Initial State -->
		{#if !searchInput.trim()}
			<div class="initial-state">
				<div class="initial-icon">🌐</div>
				<p>Search across all your servers and direct messages</p>
				<div class="search-tips">
					<h4>Search tips:</h4>
					<ul>
						<li>Use <code>from:@username</code> to filter by author</li>
						<li>Use <code>has:link</code> to find messages with links</li>
						<li>Use <code>before:2024-01-01</code> to filter by date</li>
					</ul>
				</div>
			</div>
		{/if}
	</div>
</aside>

<style>
	.global-search-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--bg-secondary, #2b2d31);
		border-left: 1px solid var(--border-color, #1e1f22);
	}

	.search-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid var(--border-color, #1e1f22);
	}

	.search-header-left {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.search-icon-header {
		color: var(--text-secondary, #b5bac1);
	}

	.search-title {
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-secondary, #b5bac1);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.close-btn:hover {
		background: var(--bg-hover, #35373c);
		color: var(--text-primary, #f2f3f5);
	}

	.search-input-container {
		padding: 12px 16px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.search-input-wrapper {
		display: flex;
		align-items: center;
		background: var(--bg-primary, #1e1f22);
		border-radius: 4px;
		padding: 0 8px;
	}

	.search-input-icon {
		color: var(--text-secondary, #b5bac1);
		flex-shrink: 0;
	}

	.search-input {
		flex: 1;
		background: none;
		border: none;
		outline: none;
		padding: 8px;
		color: var(--text-primary, #f2f3f5);
		font-size: 14px;
	}

	.search-input::placeholder {
		color: var(--text-secondary, #b5bac1);
	}

	.clear-btn {
		background: none;
		border: none;
		color: var(--text-secondary, #b5bac1);
		cursor: pointer;
		padding: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.clear-btn:hover {
		color: var(--text-primary, #f2f3f5);
	}

	.quick-filters {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.filter-toggle {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--text-secondary, #b5bac1);
		cursor: pointer;
		padding: 4px 8px;
		background: var(--bg-primary, #1e1f22);
		border-radius: 4px;
	}

	.filter-toggle input {
		cursor: pointer;
	}

	.filter-btn {
		font-size: 12px;
		color: var(--text-secondary, #b5bac1);
		background: var(--bg-primary, #1e1f22);
		border: none;
		border-radius: 4px;
		padding: 4px 8px;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.filter-btn:hover {
		background: var(--bg-hover, #35373c);
	}

	.filter-btn.active {
		background: var(--accent-color, #5865f2);
		color: white;
	}

	.clear-filters-btn {
		font-size: 12px;
		color: var(--accent-color, #5865f2);
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px 8px;
	}

	.clear-filters-btn:hover {
		text-decoration: underline;
	}

	.results-count {
		padding: 8px 16px;
		font-size: 12px;
		color: var(--text-secondary, #b5bac1);
		border-bottom: 1px solid var(--border-color, #1e1f22);
	}

	.loading-container,
	.error-container {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 32px;
		color: var(--text-secondary, #b5bac1);
	}

	.error-container {
		color: var(--error-color, #ed4245);
	}

	.error-icon {
		font-size: 20px;
	}

	.results-container {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
	}

	.search-result {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 12px;
		border-radius: 4px;
		cursor: pointer;
		position: relative;
	}

	.search-result:hover,
	.search-result.selected {
		background: var(--bg-hover, #35373c);
	}

	.result-context {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 11px;
		color: var(--text-secondary, #b5bac1);
	}

	.server-icon {
		font-size: 12px;
	}

	.server-name {
		max-width: 120px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.channel-separator {
		color: var(--text-tertiary, #6d6f78);
	}

	.channel-name {
		max-width: 120px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.result-content {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.result-header {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.author-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary, #f2f3f5);
	}

	.message-time {
		font-size: 11px;
		color: var(--text-secondary, #b5bac1);
	}

	.message-preview {
		font-size: 13px;
		color: var(--text-secondary, #b5bac1);
		line-height: 1.4;
		margin: 0;
		word-break: break-word;
	}

	.message-preview :global(mark.search-highlight) {
		background: var(--highlight-bg, #ffc107);
		color: var(--highlight-text, #000);
		padding: 0 2px;
		border-radius: 2px;
	}

	.attachment-indicator {
		font-size: 11px;
		color: var(--text-secondary, #b5bac1);
	}

	.result-arrow {
		position: absolute;
		right: 12px;
		top: 50%;
		transform: translateY(-50%);
		color: var(--text-tertiary, #6d6f78);
		opacity: 0;
		transition: opacity 0.15s;
	}

	.search-result:hover .result-arrow,
	.search-result.selected .result-arrow {
		opacity: 1;
	}

	.loading-more {
		display: flex;
		justify-content: center;
		padding: 16px;
	}

	.empty-state,
	.initial-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
		text-align: center;
		color: var(--text-secondary, #b5bac1);
	}

	.initial-icon {
		font-size: 48px;
		margin-bottom: 16px;
	}

	.initial-state p {
		margin: 0 0 24px 0;
	}

	.search-tips {
		text-align: left;
		font-size: 12px;
		background: var(--bg-primary, #1e1f22);
		padding: 12px;
		border-radius: 4px;
	}

	.search-tips h4 {
		margin: 0 0 8px 0;
		font-size: 12px;
	}

	.search-tips ul {
		margin: 0;
		padding-left: 16px;
	}

	.search-tips li {
		margin: 4px 0;
	}

	.search-tips code {
		background: var(--bg-hover, #35373c);
		padding: 1px 4px;
		border-radius: 3px;
		font-size: 11px;
	}
</style>
