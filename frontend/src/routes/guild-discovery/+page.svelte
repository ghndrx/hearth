<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { onMount, createEventDispatcher } from 'svelte';
	import { 
		ServerCard, 
		CategoryFilter, 
		SearchBar, 
		TrendingServers, 
		RecommendedServers 
	} from '$lib/components/discovery';

	interface Category {
		id?: string;
		name: string;
		slug: string;
		icon?: string;
		server_count?: number;
		total_members?: number;
		avg_member_count?: number;
	}

	interface Server {
		id: string;
		server_id?: string;
		name: string;
		description?: string;
		short_description?: string;
		icon_url?: string;
		banner_url?: string;
		member_count: number;
		category?: string;
		tags?: string[];
		is_featured?: boolean;
		featured_at?: string;
		is_verified?: boolean;
		reason?: string;
		mutual_member_count?: number;
		trend_score?: number;
		growth_rate?: number;
		invite_code?: string;
	}

	// Default categories
	const defaultCategories: Category[] = [
		{ id: 'all', name: 'All', slug: 'all', icon: '🏠' },
		{ id: 'gaming', name: 'Gaming', slug: 'gaming', icon: '🎮' },
		{ id: 'music', name: 'Music', slug: 'music', icon: '🎵' },
		{ id: 'technology', name: 'Technology', slug: 'technology', icon: '💻' },
		{ id: 'art', name: 'Art & Design', slug: 'art', icon: '🎨' },
		{ id: 'education', name: 'Education', slug: 'education', icon: '📚' },
		{ id: 'entertainment', name: 'Entertainment', slug: 'entertainment', icon: '🎬' },
		{ id: 'social', name: 'Social', slug: 'social', icon: '💬' },
		{ id: 'sports', name: 'Sports', slug: 'sports', icon: '⚽' },
		{ id: 'anime', name: 'Anime & Manga', slug: 'anime', icon: '🍜' },
	];

	// Mock data for demo
	const mockServers: Server[] = [
		{
			id: '1', name: 'Hearth Official',
			description: 'The official Hearth community server. Get help, share feedback, and connect with other users.',
			icon_url: undefined, banner_url: undefined, member_count: 12543, category: 'technology',
			tags: ['open-source', 'community', 'support'], is_featured: true
		},
		{
			id: '2', name: 'Pixel Warriors',
			description: 'A friendly gaming community for casual and competitive players alike.',
			icon_url: undefined, banner_url: undefined, member_count: 8932, category: 'gaming',
			tags: ['gaming', 'fun', 'events'], is_featured: true
		},
		{
			id: '3', name: 'Synthwave Lounge',
			description: 'Share and discover synthwave, vaporwave, and retro music.',
			icon_url: undefined, banner_url: undefined, member_count: 5621, category: 'music',
			tags: ['music', 'artists', 'discovery'], is_featured: true
		},
		{
			id: '4', name: 'Code Crafters',
			description: 'Programming discussions, code reviews, and learning resources.',
			icon_url: undefined, banner_url: undefined, member_count: 15234, category: 'technology',
			tags: ['programming', 'learning', 'help'], is_featured: true
		},
		{
			id: '5', name: 'Digital Artists Hub',
			description: 'A creative space for digital artists to share work and get feedback.',
			icon_url: undefined, banner_url: undefined, member_count: 3421, category: 'art',
			tags: ['art', 'design', 'feedback'], is_featured: false
		},
		{
			id: '6', name: 'Study Buddies',
			description: 'Join study sessions, share resources, and motivate each other.',
			icon_url: undefined, banner_url: undefined, member_count: 2156, category: 'education',
			tags: ['study', 'motivation', 'productivity'], is_featured: false
		},
		{
			id: '7', name: 'Space Explorers',
			description: 'Discuss astronomy, space missions, and the mysteries of the universe.',
			icon_url: undefined, banner_url: undefined, member_count: 8765, category: 'science',
			tags: ['space', 'science', 'discussion'], is_featured: true
		},
		{
			id: '8', name: 'Movie Night',
			description: 'Watch parties, movie discussions, and recommendations.',
			icon_url: undefined, banner_url: undefined, member_count: 4521, category: 'entertainment',
			tags: ['movies', 'tv', 'watch-party'], is_featured: false
		},
		{
			id: '9', name: 'Chill & Chat',
			description: 'A relaxed place to hang out and make new friends.',
			icon_url: undefined, banner_url: undefined, member_count: 9876, category: 'social',
			tags: ['social', 'friends', 'chill'], is_featured: false
		},
		{
			id: '10', name: 'Sports Central',
			description: 'Discuss your favorite sports, teams, and athletes.',
			icon_url: undefined, banner_url: undefined, member_count: 6789, category: 'sports',
			tags: ['sports', 'discussion', 'news'], is_featured: false
		},
	];

	// State
	let categories: Category[] = defaultCategories;
	let selectedCategory = 'all';
	let searchQuery = '';
	let servers: Server[] = [];
	let featuredServers: Server[] = [];
	let trendingServers: Server[] = [];
	let recommendedServers: Server[] = [];
	let suggestions: Array<{ type: string; value: string; count?: number }> = [];
	
	let loading = true;
	let loadingMore = false;
	let error: string | null = null;
	let joiningServerId: string | null = null;
	
	// Pagination
	let currentPage = 1;
	let totalServers = 0;
	const serversPerPage = 25;

	const dispatch = createEventDispatcher();

	onMount(async () => {
		await loadInitialData();
	});

	async function loadInitialData() {
		loading = true;
		error = null;

		try {
			// Fetch discovery home page data
			const homeData = await api.get<any>('/servers/discover/home').catch(() => null);

			if (homeData) {
				// Update categories
				if (homeData.categories && homeData.categories.length > 0) {
					categories = [
						{ id: 'all', name: 'All', slug: 'all', icon: '🏠', server_count: homeData.categories.reduce((sum: number, c: any) => sum + (c.server_count || 0), 0) },
						...homeData.categories.map((c: any) => ({
							id: c.slug,
							name: c.name,
							slug: c.slug,
							icon: c.icon,
							server_count: c.server_count,
							total_members: c.total_members,
							avg_member_count: c.avg_member_count
						}))
					];
				}

				// Update featured servers
				if (homeData.featured && homeData.featured.length > 0) {
					featuredServers = homeData.featured.map((s: any) => normalizeServer(s, true));
				}

				// Update trending servers
				if (homeData.trending && homeData.trending.length > 0) {
					trendingServers = homeData.trending.map((t: any) => normalizeServer(t.server || t, false));
				}

				// Update recommended servers
				if (homeData.recommended && homeData.recommended.length > 0) {
					recommendedServers = homeData.recommended.map((r: any) => normalizeServer(r, false));
				}

				// Update stats
				if (homeData.stats) {
					totalServers = homeData.stats.total_servers || 0;
				}
			}

			// Fetch all servers for browsing
			const serversResponse = await api.get<any>('/servers/discover').catch(() => null);
			
			if (serversResponse) {
				servers = serversResponse.servers?.map((s: any) => normalizeServer(s, false)) || [];
				totalServers = serversResponse.total || servers.length;
			} else {
				// Fallback to mock data
				servers = mockServers;
				featuredServers = mockServers.filter(s => s.is_featured);
				totalServers = mockServers.length;
			}

		} catch (err: any) {
			console.error('Failed to load discovery data:', err);
			error = err.message || 'Failed to load servers';
			// Fallback to mock data
			servers = mockServers;
			featuredServers = mockServers.filter(s => s.is_featured);
			totalServers = mockServers.length;
		} finally {
			loading = false;
		}
	}

	async function loadMoreServers() {
		if (loadingMore) return;
		
		loadingMore = true;
		currentPage++;

		try {
			const response = await api.get<any>(`/servers/discover?page=${currentPage}&limit=${serversPerPage}`).catch(() => null);
			
			if (response && response.servers) {
				const newServers = response.servers.map((s: any) => normalizeServer(s, false));
				servers = [...servers, ...newServers];
			}
		} catch (err) {
			console.error('Failed to load more servers:', err);
			currentPage--; // Revert on error
		} finally {
			loadingMore = false;
		}
	}

	async function searchServers(query: string) {
		if (!query.trim()) {
			// Reset to default
			await loadInitialData();
			return;
		}

		loading = true;
		try {
			const response = await api.get<any>(`/servers/discover/search?q=${encodeURIComponent(query)}`).catch(() => null);
			
			if (response) {
				servers = response.servers?.map((s: any) => normalizeServer(s, false)) || [];
				totalServers = response.total || servers.length;
			}

			// Fetch suggestions
			const suggestionsResponse = await api.get<any>(`/servers/discover/suggestions?q=${encodeURIComponent(query)}`).catch(() => null);
			if (suggestionsResponse && suggestionsResponse.suggestions) {
				suggestions = suggestionsResponse.suggestions;
			}
		} catch (err) {
			console.error('Search failed:', err);
		} finally {
			loading = false;
		}
	}

	async function filterByCategory(categoryId: string) {
		selectedCategory = categoryId;
		
		if (categoryId === 'all') {
			await loadInitialData();
			return;
		}

		loading = true;
		try {
			const response = await api.get<any>(`/servers/discover?category=${categoryId}`).catch(() => null);
			
			if (response) {
				servers = response.servers?.map((s: any) => normalizeServer(s, false)) || [];
				totalServers = response.total || servers.length;
			} else {
				// Filter mock data
				servers = mockServers.filter(s => s.category === categoryId);
				totalServers = servers.length;
			}
		} catch (err) {
			console.error('Category filter failed:', err);
		} finally {
			loading = false;
		}
	}

	async function handleSearch(event: CustomEvent<string>) {
		searchQuery = event.detail;
		if (searchQuery.trim()) {
			await searchServers(searchQuery);
		}
	}

	async function handleSuggestionSelect(event: CustomEvent<{ type: string; value: string }>) {
		const suggestion = event.detail;
		searchQuery = suggestion.value;
		await searchServers(searchQuery);
	}

	function handleCategorySelect(event: CustomEvent<string>) {
		filterByCategory(event.detail);
	}

	async function handleJoin(event: CustomEvent<string>) {
		const serverId = event.detail;
		joiningServerId = serverId;

		try {
			// Get server details to find invite code
			const serverDetail = await api.get<any>(`/servers/${serverId}`).catch(() => null);
			
			if (serverDetail?.invite_code) {
				await api.post(`/invites/${serverDetail.invite_code}`);
			} else {
				// Try direct join endpoint
				await api.post(`/servers/${serverId}/join`);
			}

			// Navigate to the server
			goto(`/channels/${serverId}/general`);
		} catch (err: any) {
			console.error('Failed to join server:', err);
			// Still navigate even if join fails
			goto(`/channels/${serverId}/general`);
		} finally {
			joiningServerId = null;
		}
	}

	function normalizeServer(data: any, isFeatured: boolean): Server {
		return {
			id: data.id || data.server_id || '',
			server_id: data.server_id,
			name: data.name,
			description: data.short_description || data.description || '',
			icon_url: data.icon_url,
			banner_url: data.banner_url,
			member_count: data.member_count || data.member_count_snapshot || 0,
			category: data.primary_category || data.category || 'other',
			tags: data.tags || [],
			is_featured: data.is_featured || isFeatured,
			featured_at: data.featured_at,
			is_verified: data.is_verified || false,
			reason: data.reason,
			mutual_member_count: data.mutual_member_count,
			trend_score: data.trend_score,
			growth_rate: data.growth_rate,
			invite_code: data.invite_code
		};
	}

	// Computed values
	$: filteredServers = servers.filter(server => {
		const matchesCategory = selectedCategory === 'all' || server.category === selectedCategory;
		const matchesSearch = !searchQuery || 
			server.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			(server.description || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
			(server.tags || []).some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()));
		return matchesCategory && matchesSearch;
	});

	$: displayFeatured = selectedCategory === 'all' && !searchQuery ? featuredServers : [];
	$: displayTrending = selectedCategory === 'all' && !searchQuery ? trendingServers : [];
	$: displayRecommended = selectedCategory === 'all' && !searchQuery ? recommendedServers : [];
</script>

<svelte:head>
	<title>Discover Servers | Hearth</title>
	<meta name="description" content="Discover and join public servers on Hearth" />
</svelte:head>

<div class="guild-discovery">
	<!-- Left Sidebar -->
	<aside class="sidebar">
		<div class="sidebar-header">
			<h2>Discover</h2>
		</div>
		<CategoryFilter 
			{categories}
			{selectedCategory}
			on:select={handleCategorySelect}
		/>
	</aside>

	<!-- Main Content -->
	<main class="main-content">
		<!-- Header -->
		<header class="content-header">
			<div class="header-content">
				<h1>
					{#if searchQuery}
						Search Results
					{:else if selectedCategory !== 'all'}
						{categories.find(c => c.id === selectedCategory)?.name || 'Servers'}
					{:else}
						Discover Servers
					{/if}
				</h1>
				<p class="header-subtitle">
					{#if searchQuery}
						{filteredServers.length} results for "{searchQuery}"
					{:else if selectedCategory === 'all'}
						Find communities that share your interests
					{:else}
						{filteredServers.length} servers in this category
					{/if}
				</p>
				
				<SearchBar
					bind:value={searchQuery}
					{suggestions}
					on:search={handleSearch}
					on:select={handleSuggestionSelect}
					placeholder="Search for servers..."
				/>
			</div>
		</header>

		<!-- Content -->
		<div class="content-body">
			{#if loading}
				<div class="loading-state">
					<div class="spinner"></div>
					<p>Loading servers...</p>
				</div>
			{:else if error && servers.length === 0}
				<div class="error-state">
					<p>{error}</p>
					<button class="retry-btn" on:click={loadInitialData}>Try Again</button>
				</div>
			{:else}
				<!-- Trending Section -->
				{#if displayTrending.length > 0}
					<TrendingServers
						servers={displayTrending}
						on:join={handleJoin}
						{joiningServerId}
					/>
				{/if}

				<!-- Recommended Section -->
				{#if displayRecommended.length > 0}
					<RecommendedServers
						servers={displayRecommended}
						on:join={handleJoin}
						{joiningServerId}
					/>
				{/if}

				<!-- Featured Section -->
				{#if displayFeatured.length > 0}
					<section class="servers-section">
						<div class="section-header">
							<h2>Featured Servers</h2>
							<span class="server-count">{displayFeatured.length} servers</span>
						</div>
						<div class="servers-grid featured">
							{#each displayFeatured as server (server.id)}
								<ServerCard
									{server}
									variant="featured"
									on:join={handleJoin}
									{joiningServerId}
								/>
							{/each}
						</div>
					</section>
				{/if}

				<!-- All Servers Section -->
				<section class="servers-section">
					<div class="section-header">
						<h2>
							{#if searchQuery}
								Search Results
							{:else if selectedCategory !== 'all'}
								{categories.find(c => c.id === selectedCategory)?.name || 'All'} Servers
							{:else}
								Browse All Servers
							{/if}
						</h2>
						<span class="server-count">{filteredServers.length} servers</span>
					</div>

					{#if filteredServers.length === 0}
						<div class="empty-state">
							<svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
							</svg>
							<h3>No servers found</h3>
							<p>Try adjusting your search or category filter</p>
						</div>
					{:else}
						<div class="servers-grid">
							{#each filteredServers as server (server.id)}
								<ServerCard
									{server}
									variant="default"
									on:join={handleJoin}
									{joiningServerId}
								/>
							{/each}
						</div>

						{#if filteredServers.length >= serversPerPage}
							<div class="load-more">
								<button 
									class="load-more-btn" 
									on:click={loadMoreServers}
									disabled={loadingMore}
								>
									{#if loadingMore}
										<span class="spinner small"></span>
										Loading...
									{:else}
										Load More
									{/if}
								</button>
							</div>
						{/if}
					{/if}
				</section>
			{/if}
		</div>
	</main>
</div>

<style>
	.guild-discovery {
		display: flex;
		width: 100%;
		height: 100%;
		background: #313338;
		overflow: hidden;
	}

	/* Sidebar */
	.sidebar {
		width: 240px;
		flex-shrink: 0;
		background: #2b2d31;
		display: flex;
		flex-direction: column;
		overflow-y: auto;
	}

	.sidebar-header {
		padding: 16px;
		border-bottom: 1px solid #1e1f22;
	}

	.sidebar-header h2 {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
		color: #f2f3f5;
	}

	/* Main Content */
	.main-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.content-header {
		padding: 24px 32px;
		border-bottom: 1px solid #3f4147;
		background: linear-gradient(180deg, #313338 0%, #2b2d31 100%);
	}

	.header-content {
		max-width: 1200px;
	}

	.content-header h1 {
		margin: 0 0 4px 0;
		font-size: 24px;
		font-weight: 700;
		color: #f2f3f5;
	}

	.header-subtitle {
		margin: 0 0 20px 0;
		color: #949ba4;
		font-size: 14px;
	}

	/* Content Body */
	.content-body {
		flex: 1;
		overflow-y: auto;
		padding: 24px 32px;
	}

	/* Sections */
	.servers-section {
		margin-bottom: 32px;
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 16px;
	}

	.section-header h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: #f2f3f5;
	}

	.server-count {
		font-size: 13px;
		color: #6d6f78;
	}

	/* Grids */
	.servers-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 16px;
	}

	.servers-grid.featured {
		grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
	}

	/* Loading & Error States */
	.loading-state,
	.error-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 64px 32px;
		color: #949ba4;
		text-align: center;
	}

	.spinner {
		width: 48px;
		height: 48px;
		border: 3px solid #3f4147;
		border-top-color: #5865f2;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	.spinner.small {
		width: 16px;
		height: 16px;
		border-width: 2px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.empty-icon {
		width: 64px;
		height: 64px;
		color: #6d6f78;
		margin-bottom: 16px;
	}

	.empty-state h3 {
		margin: 0 0 8px 0;
		font-size: 18px;
		color: #f2f3f5;
	}

	.empty-state p {
		margin: 0;
	}

	.retry-btn {
		margin-top: 16px;
		padding: 10px 20px;
		background: #5865f2;
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
	}

	.retry-btn:hover {
		background: #4752c4;
	}

	/* Load More */
	.load-more {
		display: flex;
		justify-content: center;
		margin-top: 24px;
	}

	.load-more-btn {
		padding: 12px 24px;
		background: #4f545c;
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 8px;
		transition: background-color 0.15s;
	}

	.load-more-btn:hover:not(:disabled) {
		background: #5d6269;
	}

	.load-more-btn:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	/* Scrollbar */
	::-webkit-scrollbar {
		width: 8px;
	}

	::-webkit-scrollbar-track {
		background: transparent;
	}

	::-webkit-scrollbar-thumb {
		background: #1e1f22;
		border-radius: 4px;
	}

	::-webkit-scrollbar-thumb:hover {
		background: #3f4147;
	}

	/* Responsive */
	@media (max-width: 1024px) {
		.sidebar {
			width: 200px;
		}
	}

	@media (max-width: 768px) {
		.sidebar {
			width: 60px;
		}

		.sidebar-header h2 {
			display: none;
		}

		.content-header {
			padding: 16px;
		}

		.content-body {
			padding: 16px;
		}

		.servers-grid,
		.servers-grid.featured {
			grid-template-columns: 1fr;
		}
	}
</style>
