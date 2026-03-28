<script lang="ts">
	import { api } from '$lib/api';

	export let serverId: string;
	export let isOwner = false;

	interface Template {
		id: string;
		code: string;
		name: string;
		description: string;
		usage_count: number;
		is_public: boolean;
		created_at: string;
	}

	let myTemplates: Template[] = [];
	let loading = false;
	let creating = false;
	let deleting: string | null = null;
	let error = '';

	// Create form state
	let showCreateForm = false;
	let newTemplateName = '';
	let newTemplateDescription = '';
	let newTemplatePublic = false;
	let createError = '';

	async function loadMyTemplates() {
		loading = true;
		error = '';
		try {
			myTemplates = await api.get<Template[]>('/users/me/templates');
		} catch (e) {
			error = 'Failed to load templates';
		} finally {
			loading = false;
		}
	}

	async function handleCreate() {
		if (!newTemplateName.trim() || creating) return;

		creating = true;
		createError = '';

		try {
			const template = await api.post<Template>(`/servers/${serverId}/templates`, {
				name: newTemplateName.trim(),
				description: newTemplateDescription.trim(),
				is_public: newTemplatePublic
			});

			myTemplates = [template, ...myTemplates];
			showCreateForm = false;
			newTemplateName = '';
			newTemplateDescription = '';
			newTemplatePublic = false;
		} catch (e: unknown) {
			createError = e instanceof Error ? e.message : 'Failed to create template';
		} finally {
			creating = false;
		}
	}

	async function handleDelete(templateId: string) {
		if (deleting) return;

		deleting = templateId;
		try {
			await api.delete(`/templates/${templateId}`);
			myTemplates = myTemplates.filter(t => t.id !== templateId);
		} catch (e) {
			console.error('Failed to delete template:', e);
		} finally {
			deleting = null;
		}
	}

	function copyTemplateLink(code: string) {
		const url = `${window.location.origin}/template/${code}`;
		navigator.clipboard.writeText(url).catch(() => {
			// Fallback for older browsers
			const textarea = document.createElement('textarea');
			textarea.value = url;
			document.body.appendChild(textarea);
			textarea.select();
			document.execCommand('copy');
			document.body.removeChild(textarea);
		});
	}

	$: if (serverId && isOwner) {
		loadMyTemplates();
	}
</script>

<div class="template-creator">
	<!-- Create Template Button -->
	{#if isOwner && !showCreateForm}
		<div class="create-section">
			<h3>Create Template</h3>
			<p class="section-hint">
				Save this server's channel structure and roles as a template. 
				Share it with others so they can create servers with the same setup.
			</p>
			<button class="btn primary" on:click={() => (showCreateForm = true)} type="button">
				Create Template
			</button>
		</div>
	{/if}

	<!-- Create Form -->
	{#if showCreateForm}
		<div class="create-form">
			<h3>Create Template from this server</h3>
			
			{#if createError}
				<div class="error-banner">{createError}</div>
			{/if}

			<div class="form-group">
				<label for="template-name">TEMPLATE NAME</label>
				<input
					id="template-name"
					type="text"
					bind:value={newTemplateName}
					placeholder="Gaming Community, Study Group, etc."
					maxlength="100"
				/>
			</div>

			<div class="form-group">
				<label for="template-desc">DESCRIPTION (optional)</label>
				<textarea
					id="template-desc"
					bind:value={newTemplateDescription}
					placeholder="A template for gaming servers with voice and text channels..."
					rows="3"
				></textarea>
			</div>

			<div class="form-group checkbox">
				<input
					id="template-public"
					type="checkbox"
					bind:checked={newTemplatePublic}
				/>
				<label for="template-public">Make this template public</label>
				<p class="checkbox-hint">Public templates appear in the template gallery for everyone to use.</p>
			</div>

			<div class="form-actions">
				<button 
					class="btn secondary" 
					on:click={() => { showCreateForm = false; createError = ''; }}
					disabled={creating}
					type="button"
				>
					Cancel
				</button>
				<button 
					class="btn primary" 
					on:click={handleCreate}
					disabled={creating || !newTemplateName.trim()}
					type="button"
				>
					{creating ? 'Creating...' : 'Create Template'}
				</button>
			</div>
		</div>
	{/if}

	<!-- Error Loading -->
	{#if error}
		<div class="error-banner">
			<span>{error}</span>
			<button on:click={loadMyTemplates}>Retry</button>
		</div>
	{/if}

	<!-- Loading -->
	{#if loading}
		<div class="loading">
			<div class="spinner"></div>
			<span>Loading templates...</span>
		</div>
	{:else if myTemplates.length === 0 && !showCreateForm}
		<div class="empty">
			<div class="empty-icon">📋</div>
			<h3>No templates yet</h3>
			<p>Create a template to share this server's structure with others.</p>
		</div>
	{:else}
		<!-- Template List -->
		<div class="templates-list">
			<h3>My Templates</h3>
			{#each myTemplates as template (template.id)}
				<div class="template-card">
					<div class="template-icon" aria-hidden="true">
						<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
							<path d="M3 5v14h18V5H3zm16 12H5V7h14v10z"/>
							<path d="M7 9h10v2H7V9zm0 4h7v2H7v-2z"/>
						</svg>
					</div>
					<div class="template-info">
						<span class="template-name">{template.name}</span>
						{#if template.description}
							<span class="template-desc">{template.description}</span>
						{/if}
						<div class="template-meta">
							<span class="usage-count">Used {template.usage_count} times</span>
							{#if template.is_public}
								<span class="public-badge">Public</span>
							{/if}
						</div>
					</div>
					<div class="template-actions">
						<button 
							class="action-btn" 
							on:click={() => copyTemplateLink(template.code)}
							title="Copy template link"
							type="button"
						>
							<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
								<path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
							</svg>
						</button>
						{#if isOwner}
							<button 
								class="action-btn danger" 
								on:click={() => handleDelete(template.id)}
								disabled={deleting === template.id}
								title="Delete template"
								type="button"
							>
								{#if deleting === template.id}
									<span class="spinner-sm"></span>
								{:else}
									<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
										<path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
									</svg>
								{/if}
							</button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.template-creator {
		padding: 16px;
	}

	.create-section {
		padding: 16px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		margin-bottom: 16px;
	}

	.create-section h3 {
		margin: 0 0 8px 0;
		font-size: 16px;
		font-weight: 600;
	}

	.section-hint {
		margin: 0 0 16px 0;
		font-size: 14px;
		color: var(--text-muted, #b5bac1);
		line-height: 1.5;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		min-width: 96px;
		min-height: 38px;
		padding: 8px 16px;
		border-radius: 3px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		border: none;
		transition: background-color 0.1s ease;
	}

	.btn.primary {
		background: var(--blurple, #5865f2);
		color: white;
	}

	.btn.primary:hover:not(:disabled) {
		background: var(--blurple-hover, #4752c4);
	}

	.btn.secondary {
		background: var(--bg-secondary, #2b2d31);
		color: var(--text-normal, #f2f3f5);
	}

	.btn.secondary:hover:not(:disabled) {
		background: var(--bg-modifier-hover, #35373c);
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.create-form {
		padding: 16px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		margin-bottom: 16px;
	}

	.create-form h3 {
		margin: 0 0 16px 0;
		font-size: 16px;
		font-weight: 600;
	}

	.error-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 12px;
		background: rgba(218, 55, 60, 0.1);
		border: 1px solid rgba(218, 55, 60, 0.3);
		border-radius: 4px;
		color: var(--status-danger, #da373c);
		font-size: 13px;
		margin-bottom: 16px;
	}

	.error-banner button {
		background: none;
		border: none;
		color: var(--status-danger, #da373c);
		text-decoration: underline;
		cursor: pointer;
	}

	.form-group {
		margin-bottom: 16px;
	}

	.form-group label {
		display: block;
		margin-bottom: 8px;
		font-size: 12px;
		font-weight: 700;
		color: var(--text-muted, #b5bac1);
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.form-group.checkbox label {
		display: inline;
		font-size: 14px;
		font-weight: 500;
		text-transform: none;
		letter-spacing: normal;
		color: var(--text-normal, #f2f3f5);
	}

	.checkbox-hint {
		margin: 4px 0 0 0;
		font-size: 12px;
		color: var(--text-muted, #b5bac1);
	}

	input[type="text"],
	textarea {
		width: 100%;
		padding: 10px;
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: 4px;
		color: var(--text-normal, #f2f3f5);
		font-size: 14px;
		font-family: inherit;
		box-sizing: border-box;
	}

	input[type="text"]::placeholder,
	textarea::placeholder {
		color: var(--text-faint, #6d6f78);
	}

	input[type="text"]:focus,
	textarea:focus {
		outline: none;
		box-shadow: 0 0 0 2px var(--blurple, #5865f2);
	}

	textarea {
		resize: vertical;
		min-height: 60px;
	}

	input[type="checkbox"] {
		width: 18px;
		height: 18px;
		margin-right: 8px;
		vertical-align: middle;
		cursor: pointer;
	}

	.form-actions {
		display: flex;
		gap: 8px;
		justify-content: flex-end;
	}

	.loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		color: var(--text-muted, #b5bac1);
		gap: 12px;
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--bg-tertiary, #1e1f22);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.spinner-sm {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		text-align: center;
	}

	.empty-icon {
		font-size: 48px;
		margin-bottom: 16px;
	}

	.empty h3 {
		margin: 0 0 8px 0;
		font-size: 18px;
		font-weight: 600;
	}

	.empty p {
		margin: 0;
		color: var(--text-muted, #b5bac1);
		font-size: 14px;
	}

	.templates-list {
		margin-top: 16px;
	}

	.templates-list h3 {
		margin: 0 0 12px 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-muted, #b5bac1);
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.template-card {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		margin-bottom: 8px;
	}

	.template-icon {
		flex-shrink: 0;
		width: 48px;
		height: 48px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--bg-tertiary, #1e1f22);
		border-radius: 8px;
		color: var(--text-muted, #b5bac1);
	}

	.template-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.template-name {
		font-size: 14px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.template-desc {
		font-size: 12px;
		color: var(--text-muted, #b5bac1);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.template-meta {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 2px;
	}

	.usage-count {
		font-size: 11px;
		color: var(--text-faint, #6d6f78);
	}

	.public-badge {
		font-size: 10px;
		padding: 2px 6px;
		background: var(--brand-primary, #5865f2);
		color: white;
		border-radius: 4px;
		font-weight: 500;
	}

	.template-actions {
		display: flex;
		gap: 4px;
	}

	.action-btn {
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		transition: background-color 0.15s, color 0.15s;
	}

	.action-btn:hover {
		background: var(--bg-modifier-hover, #35373c);
		color: var(--text-normal, #f2f3f5);
	}

	.action-btn.danger:hover {
		background: rgba(218, 55, 60, 0.1);
		color: var(--status-danger, #da373c);
	}

	.action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	h1 {
		margin: 0 0 8px 0;
		font-size: 20px;
		font-weight: 600;
	}

	.section-desc {
		margin: 0 0 16px 0;
		font-size: 14px;
		color: var(--text-muted, #b5bac1);
		line-height: 1.5;
	}
</style>
