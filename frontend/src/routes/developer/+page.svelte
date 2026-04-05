<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { onMount } from 'svelte';
	import type { App, AppCategory, CreateAppRequest, UpdateAppRequest, AppDeveloperAnalytics } from '$lib/types';
	import { APP_CATEGORIES, CATEGORY_ICONS } from '$lib/types';

	let apps: App[] = [];
	let analytics: AppDeveloperAnalytics | null = null;
	let loading = true;
	let error: string | null = null;
	let showCreateModal = false;
	let showEditModal = false;
	let selectedApp: App | null = null;

	// Form state
	let formName = '';
	let formDescription = '';
	let formLongDescription = '';
	let formCategory: AppCategory = 'utility';
	let formTags = '';
	let formIconUrl = '';
	let formPrivacyPolicyUrl = '';
	let formTermsOfServiceUrl = '';
	let formSupportServerId = '';

	let submitting = false;

	async function loadDeveloperApps() {
		loading = true;
		error = null;
		try {
			const response = await api.get<{ apps: App[] }>('/developer/apps');
			apps = response.apps || [];
		} catch (e) {
			console.error('Failed to load apps:', e);
			error = 'Failed to load your apps. Please try again.';
			// Use mock data for demo
			apps = getMockApps();
		} finally {
			loading = false;
		}
	}

	async function loadAnalytics() {
		try {
			analytics = await api.get<AppDeveloperAnalytics>('/developer/analytics');
		} catch (e) {
			console.error('Failed to load analytics:', e);
			// Use mock data for demo
			analytics = {
				total_apps: 3,
				total_installs: 45230,
				total_reviews: 892,
				average_rating: 4.5,
				apps_by_status: { approved: 2, pending: 1 },
				install_trend: [120, 145, 132, 178, 201, 189, 215],
				review_trend: [12, 18, 15, 22, 19, 24, 28]
			};
		}
	}

	function getMockApps(): App[] {
		return [
			{
				id: '1',
				developer_id: 'dev1',
				name: 'ModBot Pro',
				description: 'Advanced server moderation with auto-moderation, warnings, and detailed logging',
				long_description: 'ModBot Pro is a comprehensive moderation bot that helps keep your server safe. Features include:\n\n- Auto-moderation with customizable filters\n- Warning system with escalating punishments\n- Detailed audit logs\n- Role-based permissions\n- And much more!',
				category: 'moderation',
				tags: ['moderation', 'logging', 'automod', 'safety'],
				icon_url: undefined,
				screenshots: [],
				install_count: 15420,
				rating: 4.8,
				review_count: 342,
				status: 'approved',
				privacy_policy_url: 'https://example.com/privacy',
				terms_of_service_url: 'https://example.com/terms',
				created_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: '2',
				developer_id: 'dev1',
				name: 'WelcomeBot',
				description: 'Custom welcome messages, goodbye messages, and server onboarding',
				category: 'utility',
				tags: ['welcome', 'onboarding', 'messages'],
				icon_url: undefined,
				screenshots: [],
				install_count: 8750,
				rating: 4.6,
				review_count: 156,
				status: 'approved',
				created_at: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: '3',
				developer_id: 'dev1',
				name: 'NewApp',
				description: 'A new app awaiting approval',
				category: 'fun',
				tags: ['new', 'testing'],
				icon_url: undefined,
				screenshots: [],
				install_count: 0,
				rating: 0,
				review_count: 0,
				status: 'pending',
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString()
			}
		];
	}

	function openCreateModal() {
		formName = '';
		formDescription = '';
		formLongDescription = '';
		formCategory = 'utility';
		formTags = '';
		formIconUrl = '';
		formPrivacyPolicyUrl = '';
		formTermsOfServiceUrl = '';
		formSupportServerId = '';
		showCreateModal = true;
	}

	function openEditModal(app: App) {
		selectedApp = app;
		formName = app.name;
		formDescription = app.description;
		formLongDescription = app.long_description || '';
		formCategory = app.category;
		formTags = app.tags.join(', ');
		formIconUrl = app.icon_url || '';
		formPrivacyPolicyUrl = app.privacy_policy_url || '';
		formTermsOfServiceUrl = app.terms_of_service_url || '';
		formSupportServerId = app.support_server_id || '';
		showEditModal = true;
	}

	function closeModals() {
		showCreateModal = false;
		showEditModal = false;
		selectedApp = null;
	}

	async function createApp() {
		if (!formName || !formDescription) {
			alert('Name and description are required');
			return;
		}

		submitting = true;
		try {
			const body: CreateAppRequest = {
				name: formName,
				description: formDescription,
				long_description: formLongDescription || undefined,
				category: formCategory,
				tags: formTags ? formTags.split(',').map(t => t.trim()).filter(t => t) : [],
				icon_url: formIconUrl || undefined,
				privacy_policy_url: formPrivacyPolicyUrl || undefined,
				terms_of_service_url: formTermsOfServiceUrl || undefined,
				support_server_id: formSupportServerId || undefined
			};

			await api.post('/apps', body);
			closeModals();
			await loadDeveloperApps();
			await loadAnalytics();
		} catch (e) {
			console.error('Failed to create app:', e);
			alert('Failed to create app. Please try again.');
		} finally {
			submitting = false;
		}
	}

	async function updateApp() {
		if (!selectedApp || !formName || !formDescription) {
			alert('Name and description are required');
			return;
		}

		submitting = true;
		try {
			const body: UpdateAppRequest = {
				name: formName,
				description: formDescription,
				long_description: formLongDescription || undefined,
				category: formCategory,
				tags: formTags ? formTags.split(',').map(t => t.trim()).filter(t => t) : [],
				icon_url: formIconUrl || undefined,
				privacy_policy_url: formPrivacyPolicyUrl || undefined,
				terms_of_service_url: formTermsOfServiceUrl || undefined,
				support_server_id: formSupportServerId || undefined
			};

			await api.patch(`/apps/${selectedApp.id}`, body);
			closeModals();
			await loadDeveloperApps();
		} catch (e) {
			console.error('Failed to update app:', e);
			alert('Failed to update app. Please try again.');
		} finally {
			submitting = false;
		}
	}

	async function deleteApp(app: App) {
		if (!confirm(`Are you sure you want to delete "${app.name}"? This cannot be undone.`)) {
			return;
		}

		try {
			await api.delete(`/apps/${app.id}`);
			await loadDeveloperApps();
			await loadAnalytics();
		} catch (e) {
			console.error('Failed to delete app:', e);
			alert('Failed to delete app. Please try again.');
		}
	}

	function formatNumber(num: number): string {
		if (num >= 1000000) {
			return (num / 1000000).toFixed(1) + 'M';
		}
		if (num >= 1000) {
			return (num / 1000).toFixed(1) + 'K';
		}
		return num.toString();
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'approved': return '#3ba55c';
			case 'pending': return '#faa61a';
			case 'rejected': return '#ed4245';
			case 'suspended': return '#982929';
			default: return '#72767d';
		}
	}

	onMount(() => {
		loadDeveloperApps();
		loadAnalytics();
	});
</script>

<div class="developer-dashboard">
	<div class="header">
		<div class="header-content">
			<h1>Developer Dashboard</h1>
			<p>Manage your apps and track their performance</p>
		</div>
		<button class="create-btn" on:click={openCreateModal}>
			+ Create New App
		</button>
	</div>

	{#if analytics}
		<div class="analytics-grid">
			<div class="analytics-card">
				<h3>Total Apps</h3>
				<p class="stat">{analytics.total_apps}</p>
			</div>
			<div class="analytics-card">
				<h3>Total Installs</h3>
				<p class="stat">{formatNumber(analytics.total_installs)}</p>
			</div>
			<div class="analytics-card">
				<h3>Total Reviews</h3>
				<p class="stat">{formatNumber(analytics.total_reviews)}</p>
			</div>
			<div class="analytics-card">
				<h3>Average Rating</h3>
				<p class="stat">⭐ {analytics.average_rating.toFixed(1)}</p>
			</div>
		</div>

		{#if analytics.install_trend.length > 0}
			<div class="trend-chart">
				<h3>Install Trend (Last 7 Days)</h3>
				<div class="chart">
					{#each analytics.install_trend as count, i}
						<div class="bar-container">
							<div class="bar" style="height: {Math.max(10, (count / Math.max(...analytics.install_trend)) * 100)}%"></div>
							<span class="bar-label">Day {i + 1}</span>
							<span class="bar-value">{count}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}

	<div class="apps-section">
		<h2>Your Apps</h2>

		{#if loading}
			<div class="loading">
				<div class="spinner"></div>
				<p>Loading apps...</p>
			</div>
		{:else if error}
			<div class="error">
				<p>{error}</p>
				<button on:click={loadDeveloperApps}>Retry</button>
			</div>
		{:else if apps.length === 0}
			<div class="empty">
				<p>You haven't created any apps yet.</p>
				<button class="create-btn" on:click={openCreateModal}>Create Your First App</button>
			</div>
		{:else}
			<div class="apps-list">
				{#each apps as app (app.id)}
					<div class="app-item">
						<div class="app-icon">
							{#if app.icon_url}
								<img src={app.icon_url} alt={app.name} />
							{:else}
								<div class="placeholder-icon">{CATEGORY_ICONS[app.category] || '🤖'}</div>
							{/if}
						</div>
						<div class="app-info">
							<div class="app-header">
								<h3>{app.name}</h3>
								<span class="status-badge" style="background: {getStatusColor(app.status)}">
									{app.status}
								</span>
							</div>
							<p class="description">{app.description}</p>
							<div class="app-meta">
								<span>⭐ {app.rating > 0 ? app.rating.toFixed(1) : 'N/A'}</span>
								<span>📥 {formatNumber(app.install_count)} installs</span>
								<span>💬 {app.review_count} reviews</span>
							</div>
						</div>
						<div class="app-actions">
							<button class="edit-btn" on:click={() => openEditModal(app)}>Edit</button>
							{#if app.status === 'pending'}
								<button class="delete-btn" on:click={() => deleteApp(app)}>Delete</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

{#if showCreateModal || showEditModal}
	<div class="modal-overlay" on:click={closeModals} on:keydown={(e) => e.key === 'Escape' && closeModals()} role="presentation">
		<div class="modal-content" on:click|stopPropagation role="document">
			<button class="close-btn" on:click={closeModals}>×</button>

			<h2>{showEditModal ? 'Edit App' : 'Create New App'}</h2>

			<form on:submit|preventDefault={showEditModal ? updateApp : createApp}>
				<div class="form-group">
					<label for="name">App Name *</label>
					<input
						id="name"
						type="text"
						bind:value={formName}
						placeholder="My Awesome App"
						required
					/>
				</div>

				<div class="form-group">
					<label for="description">Short Description *</label>
					<input
						id="description"
						type="text"
						bind:value={formDescription}
						placeholder="A brief description of your app (10-200 characters)"
						required
					/>
				</div>

				<div class="form-group">
					<label for="longDescription">Long Description</label>
					<textarea
						id="longDescription"
						bind:value={formLongDescription}
						placeholder="Detailed description of your app's features and functionality"
						rows="4"
					></textarea>
				</div>

				<div class="form-group">
					<label for="category">Category *</label>
					<select id="category" bind:value={formCategory}>
						{#each APP_CATEGORIES as cat}
							<option value={cat}>{CATEGORY_ICONS[cat]} {cat}</option>
						{/each}
					</select>
				</div>

				<div class="form-group">
					<label for="tags">Tags (comma-separated)</label>
					<input
						id="tags"
						type="text"
						bind:value={formTags}
						placeholder="moderation, logging, automod"
					/>
				</div>

				<div class="form-group">
					<label for="iconUrl">Icon URL</label>
					<input
						id="iconUrl"
						type="url"
						bind:value={formIconUrl}
						placeholder="https://example.com/icon.png"
					/>
				</div>

				<div class="form-group">
					<label for="privacyPolicy">Privacy Policy URL</label>
					<input
						id="privacyPolicy"
						type="url"
						bind:value={formPrivacyPolicyUrl}
						placeholder="https://example.com/privacy"
					/>
				</div>

				<div class="form-group">
					<label for="termsOfService">Terms of Service URL</label>
					<input
						id="termsOfService"
						type="url"
						bind:value={formTermsOfServiceUrl}
						placeholder="https://example.com/terms"
					/>
				</div>

				<div class="form-group">
					<label for="supportServer">Support Server ID</label>
					<input
						id="supportServer"
						type="text"
						bind:value={formSupportServerId}
						placeholder="Server ID for support server"
					/>
				</div>

				<div class="form-actions">
					<button type="button" class="cancel-btn" on:click={closeModals}>Cancel</button>
					<button type="submit" class="submit-btn" disabled={submitting}>
						{submitting ? 'Saving...' : (showEditModal ? 'Save Changes' : 'Create App')}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.developer-dashboard {
		max-width: 1000px;
		margin: 0 auto;
		padding: 20px;
	}

	.header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 24px;
	}

	.header-content h1 {
		font-size: 28px;
		font-weight: 600;
		margin: 0 0 8px;
	}

	.header-content p {
		color: #72767d;
		margin: 0;
	}

	.create-btn {
		padding: 12px 20px;
		background: #5865f2;
		color: #fff;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.2s;
	}

	.create-btn:hover {
		background: #4752c4;
	}

	.analytics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 16px;
		margin-bottom: 24px;
	}

	.analytics-card {
		background: #36393f;
		border: 1px solid #40444b;
		border-radius: 8px;
		padding: 16px;
	}

	.analytics-card h3 {
		font-size: 13px;
		color: #72767d;
		margin: 0 0 8px;
		text-transform: uppercase;
	}

	.stat {
		font-size: 28px;
		font-weight: 600;
		margin: 0;
	}

	.trend-chart {
		background: #36393f;
		border: 1px solid #40444b;
		border-radius: 8px;
		padding: 16px;
		margin-bottom: 24px;
	}

	.trend-chart h3 {
		font-size: 14px;
		margin: 0 0 16px;
	}

	.chart {
		display: flex;
		gap: 8px;
		align-items: flex-end;
		height: 120px;
	}

	.bar-container {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		height: 100%;
	}

	.bar {
		width: 100%;
		background: #5865f2;
		border-radius: 4px 4px 0 0;
		transition: height 0.3s;
	}

	.bar-label {
		font-size: 10px;
		color: #72767d;
		margin-top: 4px;
	}

	.bar-value {
		font-size: 11px;
		color: #b9bbbe;
	}

	.apps-section h2 {
		font-size: 18px;
		margin: 0 0 16px;
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

	.error button, .empty .create-btn {
		margin-top: 12px;
		padding: 8px 16px;
		background: #5865f2;
		color: #fff;
		border: none;
		border-radius: 4px;
		cursor: pointer;
	}

	.apps-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.app-item {
		display: flex;
		gap: 16px;
		padding: 16px;
		background: #36393f;
		border: 1px solid #40444b;
		border-radius: 8px;
	}

	.app-icon {
		width: 64px;
		height: 64px;
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
		font-size: 32px;
	}

	.app-info {
		flex: 1;
		min-width: 0;
	}

	.app-header {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 4px;
	}

	.app-header h3 {
		font-size: 16px;
		font-weight: 600;
		margin: 0;
	}

	.status-badge {
		padding: 2px 8px;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
		text-transform: uppercase;
		color: #fff;
	}

	.description {
		font-size: 13px;
		color: #b9bbbe;
		margin: 0 0 8px;
	}

	.app-meta {
		display: flex;
		gap: 16px;
		font-size: 12px;
		color: #72767d;
	}

	.app-actions {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.edit-btn, .delete-btn {
		padding: 6px 12px;
		border: none;
		border-radius: 4px;
		font-size: 13px;
		cursor: pointer;
	}

	.edit-btn {
		background: #4f545c;
		color: #fff;
	}

	.edit-btn:hover {
		background: #5d6269;
	}

	.delete-btn {
		background: #ed4245;
		color: #fff;
	}

	.delete-btn:hover {
		background: #c93b3e;
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
		max-width: 500px;
		width: 90%;
		max-height: 90vh;
		overflow-y: auto;
		position: relative;
		padding: 24px;
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

	.modal-content h2 {
		font-size: 20px;
		margin: 0 0 20px;
	}

	.form-group {
		margin-bottom: 16px;
	}

	.form-group label {
		display: block;
		font-size: 13px;
		color: #b9bbbe;
		margin-bottom: 4px;
	}

	.form-group input, .form-group select, .form-group textarea {
		width: 100%;
		padding: 10px 12px;
		background: #40444b;
		border: 1px solid #4f545c;
		border-radius: 4px;
		color: #fff;
		font-size: 14px;
		box-sizing: border-box;
	}

	.form-group input:focus, .form-group select:focus, .form-group textarea:focus {
		outline: none;
		border-color: #5865f2;
	}

	.form-group textarea {
		resize: vertical;
	}

	.form-actions {
		display: flex;
		gap: 12px;
		justify-content: flex-end;
		margin-top: 24px;
	}

	.cancel-btn {
		padding: 10px 20px;
		background: #4f545c;
		color: #fff;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		cursor: pointer;
	}

	.cancel-btn:hover {
		background: #5d6269;
	}

	.submit-btn {
		padding: 10px 20px;
		background: #5865f2;
		color: #fff;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		cursor: pointer;
	}

	.submit-btn:hover:not(:disabled) {
		background: #4752c4;
	}

	.submit-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
</style>
