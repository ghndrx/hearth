<script lang="ts">
	/**
	 * ContentSafetySettings Component
	 * 
	 * Server content safety configuration with:
	 * - NSFW content filtering
	 * - Age verification requirements
	 * - Violence/hate speech filters
	 * - Custom keyword filtering
	 * - Server and channel-level settings
	 */
	
	import { createEventDispatcher, onMount } from 'svelte';
	import { api, ApiError } from '$lib/api';
	import Button from './Button.svelte';
	import LoadingSpinner from './LoadingSpinner.svelte';

	export let serverId: string;
	export let isOwner = false;

	const dispatch = createEventDispatcher<{
		saved: void;
	}>();

	interface ContentFilter {
		id?: string;
		server_id?: string;
		channel_id?: string;
		type: number;
		name: string;
		enabled: boolean;
		threshold: number;
		action: number;
		filter_data?: {
			keywords?: string[];
			regex_patterns?: string[];
			whitelist?: string[];
			threshold_value?: number;
			alert_channel_id?: string;
		};
		exempt_roles?: string[];
	}

	interface AgeVerification {
		id?: string;
		server_id?: string;
		channel_id?: string;
		enabled: boolean;
		required_age: number;
		verification_type: string;
	}

	interface ContentSafetySettings {
		server_id: string;
		filters: ContentFilter[];
		age_verification?: AgeVerification;
		server_default_threshold: number;
	}

	type FilterType = 1 | 2 | 3 | 4 | 5 | 6;
	type FilterAction = 0 | 1 | 2 | 3 | 4 | 5 | 6;

	const filterTypeLabels: Record<number, { label: string; icon: string; description: string }> = {
		1: { label: 'NSFW Filter', icon: '🔞', description: 'Block adult/explicit content' },
		2: { label: 'Violence Filter', icon: '⚠️', description: 'Block violent content' },
		3: { label: 'Hate Speech', icon: '🚫', description: 'Block hate speech and slurs' },
		4: { label: 'Harassment', icon: '😠', description: 'Block harassing content' },
		5: { label: 'Spam Filter', icon: '📧', description: 'Block spam and repetitive content' },
		6: { label: 'Custom Keywords', icon: '🔍', description: 'Block custom keywords/phrases' },
	};

	const actionLabels: Record<number, string> = {
		0: 'Allow (Log Only)',
		1: 'Warn User',
		2: 'Block Message',
		3: 'Delete & Warn',
		4: 'Timeout User',
		5: 'Kick User',
		6: 'Ban User',
	};

	const thresholdLabels: Record<number, string> = {
		0: 'No Filtering',
		1: 'Low (Explicit Only)',
		2: 'Medium',
		3: 'High (All Questionable)',
	};

	let settings: ContentSafetySettings | null = null;
	let loading = false;
	let saving = false;
	let error: string | null = null;
	let activeFilter: ContentFilter | null = null;
	let showAddFilter = false;
	let newFilterType: FilterType = 1;
	let testContent = '';
	let testResult: { passed: boolean; flags?: any[] } | null = null;
	let testing = false;

	onMount(async () => {
		await loadSettings();
	});

	async function loadSettings() {
		loading = true;
		error = null;

		try {
			const data = await api.get<ContentSafetySettings>(`/servers/${serverId}/content-safety`);
			settings = data;
		} catch (err) {
			console.error('Failed to load content safety settings:', err);
			if (err instanceof ApiError && err.status !== 404) {
				error = err.message;
			}
			settings = {
				server_id: serverId,
				filters: [],
				age_verification: undefined,
				server_default_threshold: 0,
			};
		} finally {
			loading = false;
		}
	}

	function createFilter(type: FilterType) {
		const filter: ContentFilter = {
			type,
			name: filterTypeLabels[type].label,
			enabled: true,
			threshold: 1,
			action: 2,
			filter_data: {
				keywords: [],
				regex_patterns: [],
				whitelist: [],
			},
			exempt_roles: [],
		};
		activeFilter = filter;
		showAddFilter = false;
	}

	async function saveFilter() {
		if (!activeFilter || saving) return;

		saving = true;
		error = null;

		try {
			if (activeFilter.id) {
				// Update existing filter
				await api.patch(`/content-filters/${activeFilter.id}`, activeFilter);
			} else {
				// Create new filter
				const created = await api.post<ContentFilter>(`/servers/${serverId}/content-filters`, activeFilter);
				activeFilter = created;
			}

			await loadSettings();
			activeFilter = null;
			dispatch('saved');
		} catch (err) {
			console.error('Failed to save filter:', err);
			if (err instanceof ApiError) {
				error = err.message;
			} else {
				error = 'Failed to save filter';
			}
		} finally {
			saving = false;
		}
	}

	async function deleteFilter(filter: ContentFilter) {
		if (!filter.id || !confirm(`Delete the "${filter.name}" filter?`)) return;

		try {
			await api.delete(`/content-filters/${filter.id}`);
			await loadSettings();
			if (activeFilter?.id === filter.id) {
				activeFilter = null;
			}
		} catch (err) {
			console.error('Failed to delete filter:', err);
		}
	}

	async function toggleFilter(filter: ContentFilter) {
		if (!filter.id) return;

		try {
			await api.patch(`/content-filters/${filter.id}`, {
				enabled: !filter.enabled
			});
			await loadSettings();
		} catch (err) {
			console.error('Failed to toggle filter:', err);
		}
	}

	function editFilter(filter: ContentFilter) {
		activeFilter = JSON.parse(JSON.stringify(filter));
	}

	function cancelEdit() {
		activeFilter = null;
		error = null;
	}

	// Age Verification
	let ageVerification: AgeVerification | null = null;
	let editingAgeVerification = false;
	let ageVerificationForm: Partial<AgeVerification> = {};

	$: if (settings?.age_verification) {
		ageVerification = settings.age_verification;
	}

	function startEditAgeVerification() {
		ageVerificationForm = ageVerification ? { ...ageVerification } : {
			enabled: false,
			required_age: 18,
			verification_type: 'manual'
		};
		editingAgeVerification = true;
	}

	async function saveAgeVerification() {
		if (!ageVerificationForm) return;
		saving = true;
		error = null;

		try {
			if (ageVerification) {
				await api.patch(`/servers/${serverId}/age-verification`, ageVerificationForm);
			} else {
				await api.put(`/servers/${serverId}/age-verification`, ageVerificationForm);
			}
			await loadSettings();
			editingAgeVerification = false;
			dispatch('saved');
		} catch (err) {
			console.error('Failed to save age verification:', err);
			if (err instanceof ApiError) {
				error = err.message;
			} else {
				error = 'Failed to save age verification';
			}
		} finally {
			saving = false;
		}
	}

	// Content testing
	async function testContentFilter() {
		if (!testContent.trim()) return;
		testing = true;
		testResult = null;

		try {
			const result = await api.post<{ passed: boolean; flags?: any[] }>(
				`/servers/${serverId}/content-safety/test`,
				{ content: testContent }
			);
			testResult = result;
		} catch (err) {
			console.error('Failed to test content:', err);
		} finally {
			testing = false;
		}
	}

	// Keyword management
	let newKeyword = '';
	
	function addKeyword() {
		if (!newKeyword.trim() || !activeFilter) return;
		if (!activeFilter.filter_data) {
			activeFilter.filter_data = { keywords: [] };
		}
		if (!activeFilter.filter_data.keywords) {
			activeFilter.filter_data.keywords = [];
		}
		if (!activeFilter.filter_data.keywords.includes(newKeyword.trim())) {
			activeFilter.filter_data.keywords = [...activeFilter.filter_data.keywords, newKeyword.trim()];
		}
		newKeyword = '';
	}

	function removeKeyword(keyword: string) {
		if (!activeFilter || !activeFilter.filter_data?.keywords) return;
		activeFilter.filter_data.keywords = activeFilter.filter_data.keywords.filter(k => k !== keyword);
	}
</script>

<div class="content-safety-settings">
	<div class="settings-header">
		<div class="header-info">
			<h2>Content Safety</h2>
			<p class="header-description">
				Configure NSFW filtering, age verification, and content moderation.
			</p>
		</div>
		{#if isOwner && !activeFilter && !editingAgeVerification}
			<Button variant="primary" on:click={() => showAddFilter = true}>
				Add Filter
			</Button>
		{/if}
	</div>

	{#if loading}
		<div class="loading-state">
			<LoadingSpinner />
			<span>Loading settings...</span>
		</div>
	{:else if activeFilter}
		<!-- Filter Editor -->
		<div class="filter-editor">
			<div class="editor-header">
				<h3>{activeFilter.id ? 'Edit Filter' : 'Create Filter'}</h3>
				<span class="filter-type-badge">
					{filterTypeLabels[activeFilter.type].icon}
					{filterTypeLabels[activeFilter.type].label}
				</span>
			</div>

			{#if error}
				<div class="error-message" role="alert">{error}</div>
			{/if}

			<div class="editor-form">
				<!-- Filter Name -->
				<div class="form-group">
					<label for="filter-name">Filter Name</label>
					<input
						type="text"
						id="filter-name"
						bind:value={activeFilter.name}
						placeholder="My Filter"
						maxlength={100}
						disabled={!isOwner}
					/>
				</div>

				<!-- Enabled Toggle -->
				<div class="toggle-row">
					<div class="toggle-info">
						<span class="toggle-label">Enabled</span>
						<span class="toggle-description">This filter is currently active</span>
					</div>
					<label class="toggle">
						<input type="checkbox" bind:checked={activeFilter.enabled} disabled={!isOwner} />
						<span class="toggle-slider"></span>
					</label>
				</div>

				<!-- Threshold (for NSFW filter) -->
				{#if activeFilter.type === 1}
					<div class="form-group">
						<label for="filter-threshold">Detection Threshold</label>
						<select id="filter-threshold" bind:value={activeFilter.threshold} disabled={!isOwner}>
							{#each Object.entries(thresholdLabels) as [value, label]}
								<option {value}>{label}</option>
							{/each}
						</select>
					</div>
				{/if}

				<!-- Action -->
				<div class="form-group">
					<label for="filter-action">Action When Flagged</label>
					<select id="filter-action" bind:value={activeFilter.action} disabled={!isOwner}>
						{#each Object.entries(actionLabels) as [value, label]}
							<option {value}>{label}</option>
						{/each}
					</select>
				</div>

				<!-- Keywords (for custom keyword filter and others) -->
				{#if activeFilter.type === 6 || (activeFilter.filter_data?.keywords)}
					<div class="form-group">
						<label>Blocked Keywords</label>
						<div class="keyword-input">
							<input
								type="text"
								bind:value={newKeyword}
								placeholder="Add a keyword..."
								on:keydown={(e) => e.key === 'Enter' && addKeyword()}
								disabled={!isOwner}
							/>
							<Button variant="secondary" size="sm" on:click={addKeyword} disabled={!isOwner}>
								Add
							</Button>
						</div>
						{#if activeFilter.filter_data?.keywords && activeFilter.filter_data.keywords.length > 0}
							<div class="keyword-list">
								{#each activeFilter.filter_data.keywords as keyword}
									<span class="keyword-tag">
										{keyword}
										{#if isOwner}
											<button class="remove-btn" on:click={() => removeKeyword(keyword)}>×</button>
										{/if}
									</span>
								{/each}
							</div>
						{:else}
							<p class="no-keywords">No blocked keywords added</p>
						{/if}
					</div>
				{/if}
			</div>

			<div class="editor-actions">
				<Button variant="ghost" on:click={cancelEdit}>Cancel</Button>
				{#if activeFilter.id && isOwner}
					<Button variant="danger" on:click={() => activeFilter && deleteFilter(activeFilter)}>Delete</Button>
				{/if}
				{#if isOwner}
					<Button variant="primary" on:click={saveFilter} disabled={saving}>
						{saving ? 'Saving...' : 'Save Filter'}
					</Button>
				{/if}
			</div>
		</div>
	{:else if editingAgeVerification}
		<!-- Age Verification Editor -->
		<div class="filter-editor">
			<div class="editor-header">
				<h3>Age Verification Settings</h3>
			</div>

			{#if error}
				<div class="error-message" role="alert">{error}</div>
			{/if}

			<div class="editor-form">
				<div class="toggle-row">
					<div class="toggle-info">
						<span class="toggle-label">Enable Age Verification</span>
						<span class="toggle-description">Require users to verify their age before accessing this server</span>
					</div>
					<label class="toggle">
						<input type="checkbox" bind:checked={ageVerificationForm.enabled} disabled={!isOwner} />
						<span class="toggle-slider"></span>
					</label>
				</div>

				<div class="form-group">
					<label for="required-age">Required Age</label>
					<select id="required-age" bind:value={ageVerificationForm.required_age} disabled={!isOwner}>
						<option value={13}>13+</option>
						<option value={16}>16+</option>
						<option value={18}>18+</option>
						<option value={21}>21+</option>
					</select>
				</div>

				<div class="form-group">
					<label for="verification-type">Verification Type</label>
					<select id="verification-type" bind:value={ageVerificationForm.verification_type} disabled={!isOwner}>
						<option value="manual">Manual Review</option>
						<option value="automatic">Automatic (Age Gate)</option>
						<option value="id_verification">ID Verification</option>
					</select>
				</div>
			</div>

			<div class="editor-actions">
				<Button variant="ghost" on:click={() => editingAgeVerification = false}>Cancel</Button>
				{#if isOwner}
					<Button variant="primary" on:click={saveAgeVerification} disabled={saving}>
						{saving ? 'Saving...' : 'Save'}
					</Button>
				{/if}
			</div>
		</div>
	{:else if showAddFilter}
		<!-- Filter Type Selection -->
		<div class="filter-type-selection">
			<h3>Choose Filter Type</h3>
			<div class="filter-types-grid">
				{#each Object.entries(filterTypeLabels) as [type, info]}
					<button
						class="filter-type-card"
						on:click={() => createFilter(parseInt(type) as FilterType)}
					>
						<span class="type-icon">{info.icon}</span>
						<span class="type-label">{info.label}</span>
						<span class="type-description">{info.description}</span>
					</button>
				{/each}
			</div>
			<div class="selection-actions">
				<Button variant="ghost" on:click={() => showAddFilter = false}>Cancel</Button>
			</div>
		</div>
	{:else}
		<!-- Settings List -->
		<div class="settings-section">
			<div class="section-header">
				<h3>Content Filters</h3>
			</div>

			{#if !settings?.filters || settings.filters.length === 0}
				<div class="empty-state">
					<span class="empty-icon">🛡️</span>
					<h3>No Content Filters</h3>
					<p>Create your first filter to start moderating content.</p>
				</div>
			{:else}
				<div class="filters-list">
					{#each settings.filters as filter}
						<div class="filter-item" class:disabled={!filter.enabled}>
							<div class="filter-status">
								<label class="toggle small">
									<input
										type="checkbox"
										checked={filter.enabled}
										on:change={() => toggleFilter(filter)}
										disabled={!isOwner}
									/>
									<span class="toggle-slider"></span>
								</label>
							</div>
							<div class="filter-icon">
								{filterTypeLabels[filter.type]?.icon || '🔍'}
							</div>
							<div class="filter-info">
								<span class="filter-name">{filter.name}</span>
								<span class="filter-meta">
									{thresholdLabels[filter.threshold] || 'No threshold'} • {actionLabels[filter.action] || 'Allow'}
								</span>
							</div>
							<div class="filter-actions">
								{#if isOwner}
									<Button variant="ghost" size="sm" on:click={() => editFilter(filter)}>
										Edit
									</Button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Age Verification Section -->
		<div class="settings-section">
			<div class="section-header">
				<h3>Age Verification</h3>
				{#if isOwner && ageVerification}
					<Button variant="ghost" size="sm" on:click={startEditAgeVerification}>
						Edit
					</Button>
				{/if}
			</div>

			{#if ageVerification?.enabled}
				<div class="age-verification-info">
					<div class="info-row">
						<span class="info-label">Required Age:</span>
						<span class="info-value">{ageVerification.required_age}+</span>
					</div>
					<div class="info-row">
						<span class="info-label">Verification Type:</span>
						<span class="info-value">{ageVerification.verification_type}</span>
					</div>
					<div class="info-row">
						<span class="info-label">Status:</span>
						<span class="info-value status-enabled">Enabled</span>
					</div>
				</div>
			{:else}
				<div class="empty-state small">
					<p>Age verification is not enabled.</p>
					{#if isOwner}
						<Button variant="secondary" size="sm" on:click={startEditAgeVerification}>
							Enable Age Verification
						</Button>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Content Test Section -->
		<div class="settings-section">
			<div class="section-header">
				<h3>Test Content Filter</h3>
			</div>
			<div class="test-section">
				<div class="test-input">
					<textarea
						bind:value={testContent}
						placeholder="Enter content to test against filters..."
						rows={3}
					></textarea>
					<Button variant="secondary" on:click={testContentFilter} disabled={testing || !testContent.trim()}>
						{testing ? 'Testing...' : 'Test'}
					</Button>
				</div>
				{#if testResult}
					<div class="test-result" class:passed={testResult.passed} class:failed={!testResult.passed}>
						{#if testResult.passed}
							<span class="result-icon">✅</span>
							<span>Content passed all filters</span>
						{:else}
							<span class="result-icon">⚠️</span>
							<span>Content flagged: {testResult.flags?.length || 0} issue(s) detected</span>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<style>
	.content-safety-settings {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-lg, 24px);
	}

	.settings-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--spacing-md, 16px);
	}

	.header-info h2 {
		font-size: var(--font-size-xl, 20px);
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		margin: 0;
	}

	.header-description {
		font-size: var(--font-size-sm, 14px);
		color: var(--text-muted, #b5bac1);
		margin: var(--spacing-xs, 4px) 0 0;
	}

	/* Loading state */
	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--spacing-md, 16px);
		padding: var(--spacing-xl, 40px);
		color: var(--text-muted, #b5bac1);
	}

	/* Settings Section */
	.settings-section {
		background: var(--bg-secondary, #2b2d31);
		border-radius: var(--radius-md, 4px);
		padding: var(--spacing-lg, 24px);
	}

	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: var(--spacing-md, 16px);
	}

	.section-header h3 {
		font-size: var(--font-size-md, 16px);
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		margin: 0;
	}

	/* Filters List */
	.filters-list {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-xs, 4px);
	}

	.filter-item {
		display: flex;
		align-items: center;
		gap: var(--spacing-md, 16px);
		padding: var(--spacing-md, 16px);
		background: var(--bg-tertiary, #1e1f22);
		border-radius: var(--radius-sm, 4px);
		transition: background-color 0.1s ease;
	}

	.filter-item:hover {
		background: var(--bg-modifier-hover, #35373c);
	}

	.filter-item.disabled {
		opacity: 0.6;
	}

	.filter-status {
		flex-shrink: 0;
	}

	.filter-icon {
		font-size: 24px;
		width: 32px;
		text-align: center;
	}

	.filter-info {
		flex: 1;
		min-width: 0;
	}

	.filter-name {
		display: block;
		font-size: var(--font-size-md, 16px);
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
	}

	.filter-meta {
		display: block;
		font-size: var(--font-size-sm, 14px);
		color: var(--text-muted, #b5bac1);
	}

	.filter-actions {
		flex-shrink: 0;
	}

	/* Empty state */
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		padding: var(--spacing-lg, 24px);
		gap: var(--spacing-sm, 12px);
	}

	.empty-state.small {
		padding: var(--spacing-md, 16px);
	}

	.empty-icon {
		font-size: 48px;
		opacity: 0.5;
	}

	.empty-state h3 {
		font-size: var(--font-size-md, 16px);
		color: var(--text-normal, #f2f3f5);
		margin: 0;
	}

	.empty-state p {
		color: var(--text-muted, #b5bac1);
		margin: 0;
	}

	/* Age Verification Info */
	.age-verification-info {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-sm, 8px);
	}

	.info-row {
		display: flex;
		justify-content: space-between;
		padding: var(--spacing-xs, 0);
	}

	.info-label {
		font-size: var(--font-size-sm, 14px);
		color: var(--text-muted, #b5bac1);
	}

	.info-value {
		font-size: var(--font-size-sm, 14px);
		color: var(--text-normal, #f2f3f5);
		font-weight: 500;
	}

	.info-value.status-enabled {
		color: var(--green, #23a559);
	}

	/* Filter Type Selection */
	.filter-type-selection h3 {
		font-size: var(--font-size-md, 16px);
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		margin: 0 0 var(--spacing-md, 16px);
	}

	.filter-types-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
		gap: var(--spacing-sm, 12px);
	}

	.filter-type-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--spacing-xs, 8px);
		padding: var(--spacing-lg, 24px) var(--spacing-md, 16px);
		background: var(--bg-tertiary, #1e1f22);
		border: 2px solid transparent;
		border-radius: var(--radius-md, 4px);
		cursor: pointer;
		text-align: center;
		transition: all 0.15s ease;
	}

	.filter-type-card:hover {
		border-color: var(--blurple, #5865f2);
		background: rgba(88, 101, 242, 0.1);
	}

	.type-icon {
		font-size: 32px;
	}

	.type-label {
		font-size: var(--font-size-md, 16px);
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
	}

	.type-description {
		font-size: var(--font-size-xs, 12px);
		color: var(--text-muted, #b5bac1);
	}

	.selection-actions {
		margin-top: var(--spacing-md, 16px);
		display: flex;
		justify-content: flex-end;
	}

	/* Filter Editor */
	.filter-editor {
		background: var(--bg-secondary, #2b2d31);
		border-radius: var(--radius-md, 4px);
		padding: var(--spacing-lg, 24px);
	}

	.editor-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: var(--spacing-lg, 24px);
	}

	.editor-header h3 {
		font-size: var(--font-size-lg, 18px);
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		margin: 0;
	}

	.filter-type-badge {
		display: flex;
		align-items: center;
		gap: var(--spacing-xs, 6px);
		padding: var(--spacing-xs, 6px) var(--spacing-sm, 12px);
		background: var(--bg-tertiary, #1e1f22);
		border-radius: var(--radius-sm, 3px);
		font-size: var(--font-size-sm, 14px);
		color: var(--text-muted, #b5bac1);
	}

	.error-message {
		padding: var(--spacing-sm, 10px) var(--spacing-md, 16px);
		background: rgba(237, 66, 69, 0.1);
		border: 1px solid var(--red, #ed4245);
		border-radius: var(--radius-md, 4px);
		color: var(--red, #ed4245);
		font-size: var(--font-size-sm, 14px);
		margin-bottom: var(--spacing-md, 16px);
	}

	.editor-form {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-lg, 20px);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-xs, 8px);
	}

	.form-group label {
		font-size: var(--font-size-xs, 12px);
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-muted, #b5bac1);
	}

	.form-group input[type="text"],
	.form-group textarea,
	.form-group select {
		padding: var(--spacing-sm, 10px);
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: var(--radius-sm, 3px);
		color: var(--text-normal, #f2f3f5);
		font-size: var(--font-size-md, 16px);
		font-family: inherit;
	}

	.form-group textarea {
		resize: vertical;
		min-height: 80px;
	}

	.form-group input:focus,
	.form-group textarea:focus,
	.form-group select:focus {
		outline: 2px solid var(--blurple, #5865f2);
	}

	.form-group input:disabled,
	.form-group textarea:disabled,
	.form-group select:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Keywords */
	.keyword-input {
		display: flex;
		gap: var(--spacing-sm, 8px);
	}

	.keyword-input input {
		flex: 1;
	}

	.keyword-list {
		display: flex;
		flex-wrap: wrap;
		gap: var(--spacing-xs, 6px);
		margin-top: var(--spacing-sm, 8px);
	}

	.keyword-tag {
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-xs, 4px);
		padding: var(--spacing-xs, 4px) var(--spacing-sm, 10px);
		background: var(--bg-modifier-accent, #4e5058);
		border-radius: var(--radius-sm, 3px);
		font-size: var(--font-size-sm, 14px);
		color: var(--text-normal, #f2f3f5);
	}

	.remove-btn {
		background: none;
		border: none;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		padding: 0;
		font-size: 16px;
		line-height: 1;
	}

	.remove-btn:hover {
		color: var(--red, #da373c);
	}

	.no-keywords {
		font-size: var(--font-size-sm, 14px);
		color: var(--text-muted, #b5bac1);
		font-style: italic;
		margin: var(--spacing-xs, 8px) 0 0;
	}

	/* Toggle */
	.toggle-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--spacing-md, 16px) 0;
		border-bottom: 1px solid var(--bg-modifier-accent, #4e505899);
	}

	.toggle-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.toggle-label {
		font-size: var(--font-size-md, 16px);
		color: var(--text-normal, #f2f3f5);
	}

	.toggle-description {
		font-size: var(--font-size-sm, 14px);
		color: var(--text-muted, #b5bac1);
	}

	.toggle {
		position: relative;
		display: inline-block;
		width: 40px;
		height: 24px;
		flex-shrink: 0;
	}

	.toggle.small {
		width: 32px;
		height: 18px;
	}

	.toggle input {
		opacity: 0;
		width: 0;
		height: 0;
	}

	.toggle-slider {
		position: absolute;
		cursor: pointer;
		inset: 0;
		background: var(--bg-modifier-accent, #4e5058);
		border-radius: 24px;
		transition: background 0.2s;
	}

	.toggle-slider::before {
		content: '';
		position: absolute;
		height: 18px;
		width: 18px;
		left: 3px;
		bottom: 3px;
		background: white;
		border-radius: 50%;
		transition: transform 0.2s;
	}

	.toggle.small .toggle-slider::before {
		height: 12px;
		width: 12px;
	}

	.toggle input:checked + .toggle-slider {
		background: var(--green, #23a559);
	}

	.toggle input:checked + .toggle-slider::before {
		transform: translateX(16px);
	}

	.toggle.small input:checked + .toggle-slider::before {
		transform: translateX(14px);
	}

	.toggle input:disabled + .toggle-slider {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Editor actions */
	.editor-actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--spacing-sm, 8px);
		margin-top: var(--spacing-lg, 24px);
		padding-top: var(--spacing-md, 16px);
		border-top: 1px solid var(--bg-modifier-accent, #4e505899);
	}

	/* Test Section */
	.test-section {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-md, 16px);
	}

	.test-input {
		display: flex;
		gap: var(--spacing-sm, 8px);
	}

	.test-input textarea {
		flex: 1;
	}

	.test-result {
		display: flex;
		align-items: center;
		gap: var(--spacing-sm, 8px);
		padding: var(--spacing-md, 16px);
		border-radius: var(--radius-md, 4px);
		font-size: var(--font-size-sm, 14px);
	}

	.test-result.passed {
		background: rgba(35, 165, 89, 0.1);
		border: 1px solid var(--green, #23a559);
		color: var(--green, #23a559);
	}

	.test-result.failed {
		background: rgba(237, 66, 69, 0.1);
		border: 1px solid var(--red, #ed4245);
		color: var(--red, #ed4245);
	}

	.result-icon {
		font-size: 20px;
	}
</style>
