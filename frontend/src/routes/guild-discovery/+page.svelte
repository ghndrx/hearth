<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { onMount } from 'svelte';

	interface Category {
		id: string;
		name: string;
		slug: string;
		icon: string;
		server_count?: number;
	}

	// Default categories (will be replaced by API data)
	let categories: (Category & { id: string })[] = [
		{ id: 'all', name: 'All', slug: 'all', icon: '🏠' },
		{ id: 'gaming', name: 'Gaming', slug: 'gaming', icon: '🎮' },
		{ id: 'music', name: 'Music', slug: 'music', icon: '🎵' },
		{ id: 'technology', name: 'Technology', slug: 'technology', icon: '💻' },
		{ id: 'art', name: 'Art & Design', slug: 'art', icon: '🎨' },
		{ id: 'education', name: 'Education', slug: 'education', icon: '📚' },
		{ id: 'science', name: 'Science', slug: 'science', icon: '🔬' },
		{ id: 'entertainment', name: 'Entertainment', slug: 'entertainment', icon: '🎬' },
		{ id: 'social', name: 'Social', slug: 'social', icon: '💬' },
		{ id: 'sports', name: 'Sports', slug: 'sports', icon: '⚽' },
		{ id: 'anime', name: 'Anime & Manga', slug: 'anime', icon: '🍜' }
	];

	interface Guild {
		id: string;
		name: string;
		description: string;
		icon: string | null;
		banner: string | null;
		member_count: number;
		category: string;
		tags: string[];
		is_featured?: boolean;
	}

	let selectedCategory = 'all';
	let searchQuery = '';
	let guilds: Guild[] = [];
	let loading = true;
	let error: string | null = null;
	let joiningGuildId: string | null = null;

	// Mock data for demonstration (will be replaced by API call)
	const mockGuilds: Guild[] = [
		{
			id: '1',
			name: 'Hearth Official',
			description: 'The official Hearth community server. Get help, share feedback, and connect with other users.',
			icon: null,
			banner: null,
			member_count: 12543,
			category: 'tech',
			tags: ['open-source', 'community', 'support'],
			is_featured: true
		},
		{
			id: '2',
			name: 'Pixel Warriors',
			description: 'A friendly gaming community for casual and competitive players alike.',
			icon: null,
			banner: null,
			member_count: 8932,
			category: 'gaming',
			tags: ['gaming', 'fun', 'events'],
			is_featured: true
		},
		{
			id: '3',
			name: 'Synthwave Lounge',
			description: 'Share and discover synthwave, vaporwave, and retro music.',
			icon: null,
			banner: null,
			member_count: 5621,
			category: 'music',
			tags: ['music', 'artists', 'discovery'],
			is_featured: true
		},
		{
			id: '4',
			name: 'Code Crafters',
			description: 'Programming discussions, code reviews, and learning resources.',
			icon: null,
			banner: null,
			member_count: 15234,
			category: 'tech',
			tags: ['programming', 'learning', 'help'],
			is_featured: true
		},
		{
			id: '5',
			name: 'Digital Artists Hub',
			description: 'A creative space for digital artists to share work and get feedback.',
			icon: null,
			banner: null,
			member_count: 3421,
			category: 'art',
			tags: ['art', 'design', 'feedback'],
			is_featured: false
		},
		{
			id: '6',
			name: 'Study Buddies',
			description: 'Join study sessions, share resources, and motivate each other.',
			icon: null,
			banner: null,
			member_count: 2156,
			category: 'education',
			tags: ['study', 'motivation', 'productivity'],
			is_featured: false
		},
		{
			id: '7',
			name: 'Space Explorers',
			description: 'Discuss astronomy, space missions, and the mysteries of the universe.',
			icon: null,
			banner: null,
			member_count: 8765,
			category: 'science',
			tags: ['space', 'science', 'discussion'],
			is_featured: true
		},
		{
			id: '8',
			name: 'Movie Night',
			description: 'Watch parties, movie discussions, and recommendations.',
			icon: null,
			banner: null,
			member_count: 4521,
			category: 'entertainment',
			tags: ['movies', 'tv', 'watch-party'],
			is_featured: false
		},
		{
			id: '9',
			name: 'Chill & Chat',
			description: 'A relaxed place to hang out and make new friends.',
			icon: null,
			banner: null,
			member_count: 9876,
			category: 'social',
			tags: ['social', 'friends', 'chill'],
			is_featured: false
		},
		{
			id: '10',
			name: 'Sports Central',
			description: 'Discuss your favorite sports, teams, and athletes.',
			icon: null,
			banner: null,
			member_count: 6789,
			category: 'sports',
			tags: ['sports', 'discussion', 'news'],
			is_featured: false
		},
		{
			id: '11',
			name: 'Anime Universe',
			description: 'Discuss anime, manga, and Japanese culture.',
			icon: null,
			banner: null,
			member_count: 22345,
			category: 'anime',
			tags: ['anime', 'manga', 'culture'],
			is_featured: true
		},
		{
			id: '12',
			name: 'Indie Game Dev',
			description: 'Support and collaborate with independent game developers.',
			icon: null,
			banner: null,
			member_count: 1234,
			category: 'gaming',
			tags: ['gamedev', 'indie', 'collaboration'],
			is_featured: false
		}
	];

	onMount(async () => {
		await loadGuilds();
	});

	async function loadGuilds() {
		loading = true;
		error = null;
		
		try {
			// Fetch categories, featured servers, and search results in parallel
			const [categoriesRes, featuredRes, searchRes] = await Promise.all([
				api.get<any[]>('/discovery/categories').catch(() => null),
				api.get<any[]>('/discovery/featured?limit=10').catch(() => []),
				api.get<any>('/discovery/search?limit=50').catch(() => ({ servers: [] }))
			]);
			
			// Update categories from API if available
			if (categoriesRes && Array.isArray(categoriesRes)) {
				const apiCategories = categoriesRes.map((c: any) => ({
					id: c.slug,
					name: c.name,
					slug: c.slug,
					icon: c.icon,
					server_count: c.server_count
				}));
				// Merge with default categories, API data takes precedence
				const defaultIds = new Set(categories.map(c => c.slug));
				apiCategories.forEach((c: any) => defaultIds.delete(c.slug));
				categories = [
					{ id: 'all', name: 'All', slug: 'all', icon: '🏠' },
					...apiCategories,
					...categories.filter(c => defaultIds.has(c.slug))
				];
			}
			
			// Merge featured and regular servers
			const featuredServers = featuredRes || [];
			const searchServers = searchRes?.servers || [];
			
			// Mark featured servers and merge
			featuredServers.forEach((g: any) => g.is_featured = true);
			const allServers = [...featuredServers, ...searchServers.filter((s: any) => 
				!featuredServers.some((f: any) => f.server_id === s.server_id)
			)];
			
			guilds = allServers.map((g: any) => ({
				id: g.server_id || g.id,
				name: g.name,
				description: g.short_description || g.description || '',
				icon: g.icon_url,
				banner: g.banner_url,
				member_count: g.member_count || g.member_count_snapshot || 0,
				category: g.primary_category || g.category || 'social',
				tags: g.tags || [],
				is_featured: g.is_featured || false
			}));
		} catch (err: any) {
			console.error('Failed to load guilds:', err);
			error = err.message || 'Failed to load servers';
			// Fallback to mock data on error
			guilds = mockGuilds;
		} finally {
			loading = false;
		}
	}

	async function joinGuild(guildId: string) {
		joiningGuildId = guildId;
		
		try {
			// First get the server listing to get invite code
			const listing = await api.get<any>(`/discovery/servers/${guildId}`);
			
			if (listing?.invite_code) {
				// Accept the invite
				await api.post(`/invites/${listing.invite_code}`);
			}
			
			// Navigate to the joined server
			goto(`/channels/${guildId}/general`);
		} catch (err: any) {
			console.error('Failed to join guild:', err);
			// Still navigate to the server even if invite acceptance fails
			goto(`/channels/${guildId}/general`);
		} finally {
			joiningGuildId = null;
		}
	}

	function selectCategory(categoryId: string) {
		selectedCategory = categoryId;
	}

	function formatMemberCount(count: number): string {
		if (count >= 1000000) {
			return (count / 1000000).toFixed(1) + 'M';
		}
		if (count >= 1000) {
			return (count / 1000).toFixed(1) + 'K';
		}
		return count.toString();
	}

	function getInitials(name: string): string {
		return name
			.split(' ')
			.map(word => word[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function getRandomColor(id: string): string {
		const colors = [
			'#5865f2', '#eb459e', '#3ba55d', '#f23f43', '#faa61a',
			'#2d7d46', '#91a6e6', '#f37b68', '#4f5d7e', '#72767d'
		];
		let hash = 0;
		for (let i = 0; i < id.length; i++) {
			hash = id.charCodeAt(i) + ((hash << 5) - hash);
		}
		return colors[Math.abs(hash) % colors.length];
	}

	$: filteredGuilds = guilds.filter(guild => {
		const matchesCategory = selectedCategory === 'all' || guild.category === selectedCategory;
		const matchesSearch = searchQuery === '' || 
			guild.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			guild.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
			guild.tags.some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()));
		return matchesCategory && matchesSearch;
	});

	$: featuredGuilds = filteredGuilds.filter(g => g.is_featured);
	$: regularGuilds = filteredGuilds.filter(g => !g.is_featured);
</script>

<svelte:head>
	<title>Discover Servers | Hearth</title>
	<meta name="description" content="Discover and join public servers on Hearth" />
</svelte:head>

<div class="guild-discovery">
	<!-- Left Sidebar - Categories -->
	<aside class="sidebar">
		<div class="sidebar-header">
			<h2>Discover</h2>
		</div>
		
		<nav class="category-list" aria-label="Server categories">
			{#each categories as category}
				<button
					class="category-item"
					class:active={selectedCategory === category.id}
					on:click={() => selectCategory(category.id)}
					aria-pressed={selectedCategory === category.id}
				>
					<span class="category-icon">{category.icon}</span>
					<span class="category-name">{category.name}</span>
				</button>
			{/each}
		</nav>
	</aside>

	<!-- Main Content -->
	<main class="main-content">
		<!-- Header with Search -->
		<header class="content-header">
			<div class="header-content">
				<h1>
					{#if selectedCategory === 'all'}
						Discover Servers
					{:else}
						{categories.find(c => c.id === selectedCategory)?.name} Servers
					{/if}
				</h1>
				<p class="header-subtitle">Find communities that share your interests</p>
				
				<div class="search-container">
					<svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
					<input
						type="text"
						placeholder="Search for servers..."
						bind:value={searchQuery}
						class="search-input"
						aria-label="Search for servers"
					/>
					{#if searchQuery}
						<button
							class="clear-search"
							on:click={() => searchQuery = ''}
							aria-label="Clear search"
						>
							<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>
			</div>
		</header>

		<!-- Guilds Grid -->
		<div class="guilds-container">
			{#if loading}
				<div class="loading-state">
					<div class="spinner"></div>
					<p>Loading servers...</p>
				</div>
			{:else if error && guilds.length === 0}
				<div class="error-state">
					<p>{error}</p>
					<button class="retry-btn" on:click={loadGuilds}>Try Again</button>
				</div>
			{:else if filteredGuilds.length === 0}
				<div class="empty-state">
					<svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
					<h3>No servers found</h3>
					<p>Try adjusting your search or category filter</p>
				</div>
			{:else}
				<!-- Featured Guilds Section -->
				{#if featuredGuilds.length > 0 && searchQuery === '' && selectedCategory === 'all'}
					<section class="guild-section" aria-labelledby="featured-heading">
						<h2 id="featured-heading" class="section-title">Featured Servers</h2>
						<div class="guilds-grid featured">
							{#each featuredGuilds as guild (guild.id)}
								<article class="guild-card featured">
									<div class="guild-banner" style="background: linear-gradient(135deg, {getRandomColor(guild.id)}40, {getRandomColor(guild.id)}20);">
										{#if guild.banner}
											<img src={guild.banner} alt="" loading="lazy" />
										{/if}
									</div>
									<div class="guild-content">
										<div class="guild-icon-large" style="background-color: {getRandomColor(guild.id)};">
											{#if guild.icon}
												<img src={guild.icon} alt="" loading="lazy" />
											{:else}
												<span>{getInitials(guild.name)}</span>
											{/if}
										</div>
										<h3 class="guild-name">{guild.name}</h3>
										<p class="guild-description">{guild.description}</p>
										<div class="guild-meta">
											<span class="member-count">
												<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
												</svg>
												{formatMemberCount(guild.member_count)} members
											</span>
										</div>
										<div class="guild-tags">
											{#each guild.tags.slice(0, 3) as tag}
												<span class="tag">{tag}</span>
											{/each}
										</div>
										<button
											class="join-btn"
											on:click={() => joinGuild(guild.id)}
											disabled={joiningGuildId === guild.id}
										>
											{#if joiningGuildId === guild.id}
												<span class="btn-spinner"></span>
											{:else}
												Join Server
											{/if}
										</button>
									</div>
								</article>
							{/each}
						</div>
					</section>
				{/if}

				<!-- All Guilds Section -->
				<section class="guild-section" aria-labelledby="all-heading">
					<h2 id="all-heading" class="section-title">
						{#if searchQuery}
							Search Results
						{:else if selectedCategory !== 'all'}
							{categories.find(c => c.id === selectedCategory)?.name} Servers
						{:else}
							All Servers
						{/if}
					</h2>
					<div class="guilds-grid">
						{#each regularGuilds.length > 0 || searchQuery !== '' || selectedCategory !== 'all' ? filteredGuilds : [] as guild (guild.id)}
							<article class="guild-card">
								<div class="guild-icon" style="background-color: {getRandomColor(guild.id)};">
									{#if guild.icon}
										<img src={guild.icon} alt="" loading="lazy" />
									{:else}
										<span>{getInitials(guild.name)}</span>
									{/if}
								</div>
								<div class="guild-info">
									<h3 class="guild-name">{guild.name}</h3>
									<p class="guild-description">{guild.description}</p>
									<div class="guild-meta">
										<span class="member-count">
											<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
											</svg>
											{formatMemberCount(guild.member_count)}
										</span>
									</div>
								</div>
								<button
									class="join-btn"
									on:click={() => joinGuild(guild.id)}
									disabled={joiningGuildId === guild.id}
									aria-label="Join {guild.name}"
								>
									{#if joiningGuildId === guild.id}
										<span class="btn-spinner"></span>
									{:else}
										Join
									{/if}
								</button>
							</article>
						{/each}
					</div>
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

	.category-list {
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 2px;
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
		font-size: 20px;
		width: 24px;
		text-align: center;
	}

	.category-name {
		flex: 1;
	}

	/* Main Content */
	.main-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		background: #313338;
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

	.search-container {
		position: relative;
		display: flex;
		align-items: center;
		max-width: 600px;
	}

	.search-icon {
		position: absolute;
		left: 12px;
		width: 20px;
		height: 20px;
		color: #949ba4;
		pointer-events: none;
	}

	.search-input {
		width: 100%;
		padding: 12px 40px;
		background: #1e1f22;
		border: 1px solid transparent;
		border-radius: 8px;
		color: #f2f3f5;
		font-size: 14px;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.search-input::placeholder {
		color: #6d6f78;
	}

	.search-input:focus {
		outline: none;
		border-color: #5865f2;
		background: #2b2d31;
	}

	.clear-search {
		position: absolute;
		right: 12px;
		width: 20px;
		height: 20px;
		padding: 0;
		border: none;
		background: transparent;
		color: #949ba4;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 4px;
		transition: color 0.15s, background-color 0.15s;
	}

	.clear-search:hover {
		color: #f2f3f5;
		background: #404249;
	}

	/* Guilds Container */
	.guilds-container {
		flex: 1;
		overflow-y: auto;
		padding: 24px 32px;
	}

	.guild-section {
		margin-bottom: 32px;
	}

	.section-title {
		margin: 0 0 16px 0;
		font-size: 18px;
		font-weight: 600;
		color: #f2f3f5;
	}

	/* Loading & Empty States */
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

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.btn-spinner {
		display: inline-block;
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
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
		color: #949ba4;
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
		transition: background-color 0.15s;
	}

	.retry-btn:hover {
		background: #4752c4;
	}

	/* Guilds Grid */
	.guilds-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 16px;
	}

	.guilds-grid.featured {
		grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
	}

	/* Guild Cards */
	.guild-card {
		background: #2b2d31;
		border-radius: 12px;
		overflow: hidden;
		transition: transform 0.15s, box-shadow 0.15s;
		border: 1px solid #1e1f22;
	}

	.guild-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
	}

	.guild-card.featured {
		background: #2b2d31;
	}

	.guild-banner {
		height: 100px;
		background: linear-gradient(135deg, #5865f240, #eb459e20);
		position: relative;
	}

	.guild-banner img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.guild-content {
		padding: 0 16px 16px;
		position: relative;
	}

	.guild-icon-large {
		width: 64px;
		height: 64px;
		border-radius: 12px;
		background: #5865f2;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 24px;
		font-weight: 700;
		color: white;
		margin-top: -32px;
		border: 4px solid #2b2d31;
		overflow: hidden;
	}

	.guild-icon-large img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.guild-icon {
		width: 48px;
		height: 48px;
		border-radius: 12px;
		background: #5865f2;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 16px;
		font-weight: 600;
		color: white;
		flex-shrink: 0;
		overflow: hidden;
	}

	.guild-icon img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.guild-info {
		flex: 1;
		min-width: 0;
	}

	.guild-card:not(.featured) {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 16px;
	}

	.guild-name {
		margin: 8px 0 4px 0;
		font-size: 16px;
		font-weight: 600;
		color: #f2f3f5;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.guild-card:not(.featured) .guild-name {
		margin: 0 0 4px 0;
	}

	.guild-description {
		margin: 0 0 12px 0;
		font-size: 13px;
		color: #949ba4;
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.guild-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 12px;
	}

	.member-count {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: #949ba4;
	}

	.member-count svg {
		width: 14px;
		height: 14px;
	}

	.guild-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-bottom: 12px;
	}

	.tag {
		padding: 2px 8px;
		background: #1e1f22;
		border-radius: 4px;
		font-size: 11px;
		color: #949ba4;
		font-weight: 500;
	}

	.join-btn {
		width: 100%;
		padding: 10px;
		background: #23a559;
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 36px;
	}

	.join-btn:hover:not(:disabled) {
		background: #1a8f4a;
	}

	.join-btn:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.guild-card:not(.featured) .join-btn {
		width: auto;
		padding: 8px 16px;
		min-height: 32px;
	}

	/* Scrollbar Styling */
	.sidebar::-webkit-scrollbar,
	.guilds-container::-webkit-scrollbar {
		width: 8px;
	}

	.sidebar::-webkit-scrollbar-track,
	.guilds-container::-webkit-scrollbar-track {
		background: transparent;
	}

	.sidebar::-webkit-scrollbar-thumb,
	.guilds-container::-webkit-scrollbar-thumb {
		background: #1e1f22;
		border-radius: 4px;
	}

	.sidebar::-webkit-scrollbar-thumb:hover,
	.guilds-container::-webkit-scrollbar-thumb:hover {
		background: #3f4147;
	}

	/* Focus styles for accessibility */
	.category-item:focus-visible,
	.search-input:focus-visible,
	.join-btn:focus-visible,
	.clear-search:focus-visible,
	.retry-btn:focus-visible {
		outline: 2px solid #5865f2;
		outline-offset: 2px;
	}

	/* Responsive adjustments */
	@media (max-width: 1024px) {
		.sidebar {
			width: 200px;
		}
		
		.guilds-grid {
			grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		}
	}

	@media (max-width: 768px) {
		.sidebar {
			width: 60px;
		}
		
		.category-name {
			display: none;
		}
		
		.category-item {
			justify-content: center;
			padding: 12px;
		}
		
		.content-header {
			padding: 16px;
		}
		
		.guilds-container {
			padding: 16px;
		}
		
		.guilds-grid,
		.guilds-grid.featured {
			grid-template-columns: 1fr;
		}
	}
</style>
