<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { onMount } from 'svelte';
	import type { App, AppCategory, AppReview, CreateReviewRequest } from '$lib/types';
	import { APP_CATEGORIES, CATEGORY_ICONS } from '$lib/types';

	let apps: App[] = [];
	let loading = true;
	let error: string | null = null;
	let selectedCategory: AppCategory | 'all' = 'all';
	let searchQuery = '';
	let selectedApp: App | null = null;
	let showAppDetail = false;
	let userReview: { rating: number; review_text?: string } | null = null;
	let reviewText = '';
	let reviewRating = 5;
	let submittingReview = false;

	const categoryLabels: Record<string, string> = {
		all: 'All Apps',
		moderation: 'Moderation',
		music: 'Music',
		gaming: 'Gaming',
		utility: 'Utility',
		fun: 'Fun',
		education: 'Education',
		roleplay: 'Roleplay',
		economy: 'Economy'
	};

	async function loadApps() {
		loading = true;
		error = null;
		try {
			const params = new URLSearchParams();
			if (selectedCategory !== 'all') {
				params.set('category', selectedCategory);
			}
			if (searchQuery) {
				params.set('query', searchQuery);
			}
			params.set('limit', '50');

			const response = await api.get<{ apps: App[] }>(`/apps?${params}`);
			apps = response.apps || [];
		} catch (e) {
			console.error('Failed to load apps:', e);
			error = 'Failed to load apps. Please try again.';
			// Use mock data for demo
			apps = getMockApps();
		} finally {
			loading = false;
		}
	}

	function getMockApps(): App[] {
		return [
			{
				id: '1',
				developer_id: 'dev1',
				name: 'ModBot',
				description: 'Advanced moderation bot with auto-moderation, warnings, and logging',
				category: 'moderation',
				tags: ['moderation', 'logging', 'automod'],
				icon_url: undefined,
				screenshots: [],
				install_count: 15420,
				rating: 4.8,
				review_count: 342,
				status: 'approved',
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: '2',
				developer_id: 'dev1',
				name: 'MusicMaster',
				description: 'Play music from YouTube, Spotify, and more with high quality audio',
				category: 'music',
				tags: ['music', 'audio', 'youtube'],
				icon_url: undefined,
				screenshots: [],
				install_count: 28350,
				rating: 4.6,
				review_count: 521,
				status: 'approved',
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: '3',
				developer_id: 'dev2',
				name: 'GameStats',
				description: 'Track game stats, leaderboards, and tournament brackets',
				category: 'gaming',
				tags: ['gaming', 'stats', 'leaderboards'],
				icon_url: undefined,
				screenshots: [],
				install_count: 8930,
				rating: 4.3,
				review_count: 187,
				status: 'approved',
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: '4',
				developer_id: 'dev2',
				name: 'UtilityPro',
				description: 'Collection of useful utilities including reminders, polls, and timers',
				category: 'utility',
				tags: ['utility', 'reminders', 'polls'],
				icon_url: undefined,
				screenshots: [],
				install_count: 12400,
				rating: 4.5,
				review_count: 278,
				status: 'approved',
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: '5',
				developer_id: 'dev3',
				name: 'FunGames',
				description: 'Play games like trivia, word games, and more with your server',
				category: 'fun',
				tags: ['fun', 'games', 'trivia'],
				icon_url: undefined,
				screenshots: [],
				install_count: 19800,
				rating: 4.7,
				review_count: 456,
				status: 'approved',
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString()
			}
		];
	}

	async function openAppDetail(app: App) {
		selectedApp = app;
		showAppDetail = true;
		reviewText = '';
		reviewRating = 5;

		// Load user's review if any
		try {
			const review = await api.get<AppReview>(`/apps/${app.id}/reviews/@me`);
			userReview = { rating: review.rating, review_text: review.review_text };
			reviewText = review.review_text || '';
			reviewRating = review.rating;
		} catch (err) {
			console.error('[AppDirectory] Failed to load user review:', err);
			userReview = null;
		}
	}

	function closeAppDetail() {
		showAppDetail = false;
		selectedApp = null;
		userReview = null;
	}

	async function installApp(app: App) {
		try {
			// For now, just show a message since we need server selection
			alert(`Install ${app.name} - Server selection would appear here`);
		} catch (e) {
			console.error('Failed to install app:', e);
			alert('Failed to install app. Please try again.');
		}
	}

	async function submitReview() {
		if (!selectedApp) return;

		submittingReview = true;
		try {
			const body: CreateReviewRequest = {
				rating: reviewRating,
				review_text: reviewText || undefined
			};

			if (userReview) {
				// Update existing review
				await api.patch(`/apps/${selectedApp.id}/reviews/${(userReview as any).id}`, body);
			} else {
				// Create new review
				await api.post(`/apps/${selectedApp.id}/reviews`, body);
			}

			// Reload app detail to show updated review
			await openAppDetail(selectedApp);
		} catch (e) {
			console.error('Failed to submit review:', e);
			alert('Failed to submit review. Please try again.');
		} finally {
			submittingReview = false;
		}
	}

	function handleSearch() {
		loadApps();
	}

	function handleCategoryChange(category: AppCategory | 'all') {
		selectedCategory = category;
		loadApps();
	}

	onMount(() => {
		loadApps();
	});
</script>

<div class="app-directory">
	<div class="header">
		<h1>App Directory</h1>
		<p>Discover bots and apps to enhance your server</p>
	</div>

	<div class="search-bar">
		<input
			type="text"
			placeholder="Search apps..."
			bind:value={searchQuery}
			on:input={handleSearch}
		/>
	</div>

	<div class="categories">
		<button
			class="category-btn"
			class:active={selectedCategory === 'all'}
			on:click={() => handleCategoryChange('all')}
		>
			<span>🏠</span> All
		</button>
		{#each APP_CATEGORIES as category}
			<button
				class="category-btn"
				class:active={selectedCategory === category}
				on:click={() => handleCategoryChange(category)}
			>
				<span>{CATEGORY_ICONS[category]}</span> {categoryLabels[category]}
			</button>
		{/each}
	</div>

	{#if loading}
		<div class="loading">
			<div class="spinner"></div>
			<p>Loading apps...</p>
		</div>
	{:else if error}
		<div class="error">
			<p>{error}</p>
			<button on:click={loadApps}>Retry</button>
		</div>
	{:else}
		<div class="apps-grid">
			{#each apps as app (app.id)}
				<button class="app-card" on:click={() => openAppDetail(app)}>
					<div class="app-icon">
						{#if app.icon_url}
							<img src={app.icon_url} alt={app.name} />
						{:else}
							<div class="placeholder-icon">{CATEGORY_ICONS[app.category] || '🤖'}</div>
						{/if}
					</div>
					<div class="app-info">
						<h3>{app.name}</h3>
						<p class="description">{app.description}</p>
						<div class="app-meta">
							<span class="rating">⭐ {app.rating.toFixed(1)}</span>
							<span class="installs">📥 {app.install_count.toLocaleString()}</span>
						</div>
					</div>
				</button>
			{/each}
		</div>

		{#if apps.length === 0}
			<div class="empty">
				<p>No apps found in this category.</p>
			</div>
		{/if}
	{/if}
</div>

{#if showAppDetail && selectedApp}
	<div class="modal-overlay" on:click={closeAppDetail}>
		<div class="modal-content" on:click|stopPropagation>
			<button class="close-btn" on:click={closeAppDetail}>×</button>

			<div class="app-detail">
				<div class="detail-header">
					<div class="detail-icon">
						{#if selectedApp.icon_url}
							<img src={selectedApp.icon_url} alt={selectedApp.name} />
						{:else}
							<div class="placeholder-icon large">{CATEGORY_ICONS[selectedApp.category] || '🤖'}</div>
						{/if}
					</div>
					<div class="detail-info">
						<h2>{selectedApp.name}</h2>
						<p class="detail-description">{selectedApp.description}</p>
						<div class="detail-meta">
							<span class="rating">⭐ {selectedApp.rating.toFixed(1)} ({selectedApp.review_count} reviews)</span>
							<span class="installs">📥 {selectedApp.install_count.toLocaleString()} installs</span>
							<span class="category">{CATEGORY_ICONS[selectedApp.category]} {selectedApp.category}</span>
						</div>
						{#if selectedApp.tags.length > 0}
							<div class="tags">
								{#each selectedApp.tags as tag}
									<span class="tag">{tag}</span>
								{/each}
							</div>
						{/if}
					</div>
				</div>

				{#if selectedApp.long_description}
					<div class="long-description">
						<h3>About</h3>
						<p>{selectedApp.long_description}</p>
					</div>
				{/if}

				<div class="actions">
					<button class="install-btn" on:click={() => installApp(selectedApp!)}>
						Install App
					</button>
					{#if selectedApp.privacy_policy_url}
						<a href={selectedApp.privacy_policy_url} target="_blank" class="policy-link">
							Privacy Policy
						</a>
					{/if}
				</div>

				<div class="review-section">
					<h3>Write a Review</h3>
					<div class="review-form">
						<div class="rating-input">
							<label>Rating:</label>
							<select bind:value={reviewRating}>
								<option value={5}>⭐⭐⭐⭐⭐ (5)</option>
								<option value={4}>⭐⭐⭐⭐ (4)</option>
								<option value={3}>⭐⭐⭐ (3)</option>
								<option value={2}>⭐⭐ (2)</option>
								<option value={1}>⭐ (1)</option>
							</select>
						</div>
						<div class="review-text-input">
							<label>Review (optional):</label>
							<textarea
								bind:value={reviewText}
								placeholder="Share your experience with this app..."
								rows="3"
							></textarea>
						</div>
						<button
							class="submit-review-btn"
							on:click={submitReview}
							disabled={submittingReview}
						>
							{submittingReview ? 'Submitting...' : userReview ? 'Update Review' : 'Submit Review'}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.app-directory {
		max-width: 1200px;
		margin: 0 auto;
		padding: 20px;
	}

	.header {
		text-align: center;
		margin-bottom: 24px;
	}

	.header h1 {
		font-size: 28px;
		font-weight: 600;
		margin: 0 0 8px;
	}

	.header p {
		color: #72767d;
		margin: 0;
	}

	.search-bar {
		margin-bottom: 20px;
	}

	.search-bar input {
		width: 100%;
		padding: 12px 16px;
		border: 1px solid #40444b;
		border-radius: 4px;
		background: #36393f;
		color: #fff;
		font-size: 14px;
	}

	.search-bar input:focus {
		outline: none;
		border-color: #5865f2;
	}

	.categories {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
		margin-bottom: 24px;
	}

	.category-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		border: 1px solid #40444b;
		border-radius: 20px;
		background: #36393f;
		color: #b9bbbe;
		cursor: pointer;
		font-size: 14px;
		transition: all 0.2s;
	}

	.category-btn:hover {
		background: #4f545c;
		color: #fff;
	}

	.category-btn.active {
		background: #5865f2;
		border-color: #5865f2;
		color: #fff;
	}

	.loading, .error, .empty {
		text-align: center;
		padding: 40px;
		color: #72767d;
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 3px solid #40444b;
		border-top-color: #5865f2;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin: 0 auto 16px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.error button {
		margin-top: 12px;
		padding: 8px 16px;
		background: #5865f2;
		color: #fff;
		border: none;
		border-radius: 4px;
		cursor: pointer;
	}

	.apps-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 16px;
	}

	.app-card {
		display: flex;
		gap: 12px;
		padding: 16px;
		background: #36393f;
		border: 1px solid #40444b;
		border-radius: 8px;
		cursor: pointer;
		text-align: left;
		transition: all 0.2s;
		width: 100%;
	}

	.app-card:hover {
		background: #40444b;
		border-color: #5865f2;
	}

	.app-icon {
		width: 48px;
		height: 48px;
		border-radius: 8px;
		overflow: hidden;
		flex-shrink: 0;
	}

	.app-icon img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.placeholder-icon {
		width: 100%;
		height: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #5865f2;
		font-size: 24px;
	}

	.placeholder-icon.large {
		font-size: 48px;
	}

	.app-info {
		flex: 1;
		min-width: 0;
	}

	.app-info h3 {
		font-size: 16px;
		font-weight: 600;
		margin: 0 0 4px;
		color: #fff;
	}

	.description {
		font-size: 13px;
		color: #b9bbbe;
		margin: 0 0 8px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.app-meta {
		display: flex;
		gap: 12px;
		font-size: 12px;
		color: #72767d;
	}

	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal-content {
		background: #36393f;
		border-radius: 8px;
		max-width: 600px;
		width: 90%;
		max-height: 90vh;
		overflow-y: auto;
		position: relative;
	}

	.close-btn {
		position: absolute;
		top: 12px;
		right: 12px;
		width: 32px;
		height: 32px;
		border: none;
		background: transparent;
		color: #b9bbbe;
		font-size: 24px;
		cursor: pointer;
		border-radius: 4px;
	}

	.close-btn:hover {
		background: #40444b;
		color: #fff;
	}

	.app-detail {
		padding: 24px;
	}

	.detail-header {
		display: flex;
		gap: 16px;
		margin-bottom: 20px;
	}

	.detail-icon {
		width: 80px;
		height: 80px;
		border-radius: 12px;
		overflow: hidden;
		flex-shrink: 0;
	}

	.detail-icon img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.detail-info h2 {
		font-size: 22px;
		font-weight: 600;
		margin: 0 0 8px;
	}

	.detail-description {
		color: #b9bbbe;
		margin: 0 0 12px;
	}

	.detail-meta {
		display: flex;
		gap: 16px;
		font-size: 13px;
		color: #72767d;
		flex-wrap: wrap;
	}

	.tags {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
		margin-top: 12px;
	}

	.tag {
		padding: 4px 8px;
		background: #4f545c;
		border-radius: 4px;
		font-size: 12px;
		color: #b9bbbe;
	}

	.long-description {
		margin-bottom: 20px;
	}

	.long-description h3 {
		font-size: 16px;
		margin: 0 0 8px;
	}

	.long-description p {
		color: #b9bbbe;
		margin: 0;
		line-height: 1.5;
	}

	.actions {
		display: flex;
		gap: 12px;
		align-items: center;
		margin-bottom: 24px;
	}

	.install-btn {
		padding: 12px 24px;
		background: #5865f2;
		color: #fff;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.2s;
	}

	.install-btn:hover {
		background: #4752c4;
	}

	.policy-link {
		color: #00aff4;
		font-size: 14px;
	}

	.review-section {
		border-top: 1px solid #40444b;
		padding-top: 20px;
	}

	.review-section h3 {
		font-size: 16px;
		margin: 0 0 12px;
	}

	.review-form {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.rating-input, .review-text-input {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.rating-input label, .review-text-input label {
		font-size: 13px;
		color: #b9bbbe;
	}

	.rating-input select {
		padding: 8px;
		background: #40444b;
		border: 1px solid #4f545c;
		border-radius: 4px;
		color: #fff;
	}

	.review-text-input textarea {
		padding: 8px;
		background: #40444b;
		border: 1px solid #4f545c;
		border-radius: 4px;
		color: #fff;
		resize: vertical;
	}

	.review-text-input textarea:focus {
		outline: none;
		border-color: #5865f2;
	}

	.submit-review-btn {
		padding: 10px 20px;
		background: #4f545c;
		color: #fff;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		cursor: pointer;
		transition: background 0.2s;
		align-self: flex-start;
	}

	.submit-review-btn:hover:not(:disabled) {
		background: #5d6269;
	}

	.submit-review-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
</style>
