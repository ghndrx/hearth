<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import Modal from './Modal.svelte';
	import { api } from '$lib/api';

	export let open = false;

	const dispatch = createEventDispatcher<{
		select: { code: string; name: string };
		close: void;
	}>();

	interface Template {
		id: string;
		code: string;
		name: string;
		description: string;
		usage_count: number;
		creator_id: string;
		created_at: string;
	}

	let loading = false;
	let templates: Template[] = [];
	let error = '';
	let searchQuery = '';

	async function loadTemplates() {
		loading = true;
		error = '';
		try {
			const response = await api.get<{ templates: Template[]; next_cursor?: string }>('/templates?limit=20');
			templates = response?.templates || [];
		} catch (e) {
			error = 'Failed to load templates';
			templates = [];
		} finally {
			loading = false;
		}
	}

	function selectTemplate(template: Template) {
		dispatch('select', { code: template.code, name: template.name });
	}

	function handleClose() {
		dispatch('close');
	}

	$: filteredTemplates = searchQuery
		? templates.filter(t =>
			t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			t.description?.toLowerCase().includes(searchQuery.toLowerCase())
		)
		: templates;

	onMount(() => {
		if (open) {
			loadTemplates();
		}
	});

	$: if (open) {
		loadTemplates();
	}
</script>

<Modal {open} title="Server Templates" subtitle="Start with a template to quickly set up your server" size="small" on:close={handleClose}>
	{#if error}
		<div class="error-banner">
			<span>{error}</span>
			<button on:click={loadTemplates}>Retry</button>
		</div>
	{/if}

	<!-- Search -->
	<div class="search-container">
		<input
			type="text"
			bind:value={searchQuery}
			placeholder="Search templates..."
			class="search-input"
		/>
	</div>

	<!-- Templates Grid -->
	<div class="templates-list">
		{#if loading}
			<div class="loading">
				<div class="spinner"></div>
				<span>Loading templates...</span>
			</div>
		{:else if filteredTemplates.length === 0}
			<div class="empty">
				{#if searchQuery}
					<span>No templates match "{searchQuery}"</span>
				{:else}
					<span>No public templates available</span>
				{/if}
			</div>
		{:else}
			{#each filteredTemplates as template (template.id)}
				<button
					class="template-card"
					on:click={() => selectTemplate(template)}
					type="button"
				>
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
						<span class="template-meta">Used {template.usage_count} times</span>
					</div>
					<span class="arrow" aria-hidden="true">
						<svg viewBox="0 0 20 20" width="20" height="20" fill="none">
							<path d="M7.5 4.5L13 10L7.5 15.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
						</svg>
					</span>
				</button>
			{/each}
		{/if}
	</div>
</Modal>

<style>
	.error-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		background: rgba(218, 55, 60, 0.1);
		color: var(--status-danger, #da373c);
		font-size: 13px;
	}

	.error-banner button {
		background: none;
		border: none;
		color: var(--status-danger, #da373c);
		text-decoration: underline;
		cursor: pointer;
	}

	.search-container {
		padding: 12px 16px;
	}

	.search-input {
		width: 100%;
		padding: 8px 12px;
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: 4px;
		color: var(--text-normal, #f2f3f5);
		font-size: 14px;
	}

	.search-input::placeholder {
		color: var(--text-muted, #949ba4);
	}

	.search-input:focus {
		outline: none;
		box-shadow: 0 0 0 2px var(--blurple, #5865f2);
	}

	.templates-list {
		max-height: 400px;
		overflow-y: auto;
		padding: 0 8px 8px;
	}

	.loading,
	.empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 32px;
		color: var(--text-muted, #949ba4);
		font-size: 14px;
		gap: 8px;
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--bg-tertiary, #1e1f22);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.template-card {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 12px;
		background: var(--bg-secondary, #2b2d31);
		border: 1px solid transparent;
		border-radius: 8px;
		cursor: pointer;
		text-align: left;
		transition: background-color 0.1s ease, border-color 0.1s ease;
		margin-bottom: 4px;
	}

	.template-card:hover {
		background: var(--bg-modifier-hover, #35373c);
		border-color: var(--bg-modifier-accent, #404249);
	}

	.template-card:focus-visible {
		outline: 2px solid var(--blurple, #5865f2);
		outline-offset: 2px;
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
		color: var(--text-muted, #949ba4);
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
		font-size: 11px;
		color: var(--text-faint, #6d6f78);
	}

	.arrow {
		flex-shrink: 0;
		color: var(--text-muted, #b5bac1);
	}
</style>
