<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { serverChannels, type Channel } from '$lib/stores/channels';

	export let serverId: string;
	export let canManageWebhooks = false;

	interface Webhook {
		id: string;
		token: string;
		name: string;
		channel_id: string;
		guild_id: string;
		avatar?: string | null;
		type: number;
	}

	interface WebhookDeliveryStats {
		total_deliveries: number;
		successful_count: number;
		failed_count: number;
		success_rate: number;
		avg_duration_ms: number;
		last_delivery_at?: string;
		last_failure_at?: string;
	}

	interface WebhookDelivery {
		id: string;
		webhook_id: string;
		status_code?: number;
		error?: string;
		attempt_number: number;
		created_at: string;
	}

	let webhooks: Webhook[] = [];
	let webhookStats: Record<string, WebhookDeliveryStats> = {};
	let recentFailures: Record<string, WebhookDelivery[]> = {};
	let loading = true;
	let error = '';
	let creating = false;
	let newWebhookName = '';
	let newWebhookChannel = '';
	let editingWebhook: Webhook | null = null;
	let editName = '';
	let editChannel = '';
	let deleteConfirmId: string | null = null;
	let copiedId: string | null = null;
	let testingId: string | null = null;
	let expandedStatsId: string | null = null;

	$: textChannels = ($serverChannels || []).filter(
		(c: Channel) => c.type === 0 && c.server_id === serverId
	);

	onMount(() => {
		loadWebhooks();
	});

	async function loadWebhooks() {
		loading = true;
		error = '';
		try {
			// Load webhooks from all text channels
			const allWebhooks: Webhook[] = [];
			for (const channel of textChannels) {
				try {
					const channelWebhooks = await api.get<Webhook[]>(`/channels/${channel.id}/webhooks`);
					allWebhooks.push(...channelWebhooks);
				} catch (err) {
					console.error('[WebhookManager] Failed to load webhooks for channel:', channel.id, err);
				}
			}
			webhooks = allWebhooks;
			
			// Load stats for each webhook
			for (const webhook of allWebhooks) {
				loadWebhookStats(webhook.id);
			}
		} catch (err) {
			error = 'Failed to load webhooks';
			console.error('Failed to load webhooks:', err);
		} finally {
			loading = false;
		}
	}

	async function loadWebhookStats(webhookId: string) {
		try {
			const stats = await api.get<WebhookDeliveryStats>(`/webhooks/${webhookId}/stats`);
			webhookStats[webhookId] = stats;
			webhookStats = webhookStats; // Trigger reactivity
		} catch (err) {
			console.error('Failed to load webhook stats:', err);
		}
	}

	async function loadRecentFailures(webhookId: string) {
		try {
			const deliveries = await api.get<WebhookDelivery[]>(`/webhooks/${webhookId}/deliveries?limit=5`);
			recentFailures[webhookId] = deliveries.filter(d => d.status_code && (d.status_code < 200 || d.status_code >= 300));
			recentFailures = recentFailures; // Trigger reactivity
		} catch (err) {
			console.error('Failed to load recent failures:', err);
		}
	}

	function toggleStats(webhookId: string) {
		if (expandedStatsId === webhookId) {
			expandedStatsId = null;
		} else {
			expandedStatsId = webhookId;
			loadRecentFailures(webhookId);
		}
	}

	async function createWebhook() {
		if (!newWebhookName.trim() || !newWebhookChannel) return;
		creating = true;
		error = '';
		try {
			const webhook = await api.post<Webhook>(`/channels/${newWebhookChannel}/webhooks`, {
				name: newWebhookName.trim()
			});
			webhooks = [...webhooks, webhook];
			newWebhookName = '';
			newWebhookChannel = '';
			// Load stats for new webhook
			loadWebhookStats(webhook.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create webhook';
		} finally {
			creating = false;
		}
	}

	function startEdit(webhook: Webhook) {
		editingWebhook = webhook;
		editName = webhook.name;
		editChannel = webhook.channel_id;
	}

	function cancelEdit() {
		editingWebhook = null;
		editName = '';
		editChannel = '';
	}

	async function saveEdit() {
		if (!editingWebhook || !editName.trim()) return;
		error = '';
		try {
			const updates: Record<string, string> = {};
			if (editName.trim() !== editingWebhook.name) {
				updates.name = editName.trim();
			}
			if (editChannel !== editingWebhook.channel_id) {
				updates.channel_id = editChannel;
			}
			if (Object.keys(updates).length > 0) {
				const updated = await api.patch<Webhook>(`/webhooks/${editingWebhook.id}`, updates);
				webhooks = webhooks.map(w => w.id === updated.id ? updated : w);
			}
			editingWebhook = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update webhook';
		}
	}

	async function confirmDelete(webhookId: string) {
		error = '';
		try {
			await api.delete(`/webhooks/${webhookId}`);
			webhooks = webhooks.filter(w => w.id !== webhookId);
			delete webhookStats[webhookId];
			delete recentFailures[webhookId];
			deleteConfirmId = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete webhook';
		}
	}

	function copyWebhookUrl(webhook: Webhook) {
		if (!webhook.token) return;
		const baseUrl = window.location.origin;
		const url = `${baseUrl}/api/v1/webhooks/${webhook.id}/${webhook.token}`;
		navigator.clipboard.writeText(url);
		copiedId = webhook.id;
		setTimeout(() => {
			if (copiedId === webhook.id) copiedId = null;
		}, 2000);
	}

	async function testWebhook(webhook: Webhook) {
		testingId = webhook.id;
		error = '';
		try {
			await api.post(`/webhooks/${webhook.id}/test`);
			// Reload stats after test
			await loadWebhookStats(webhook.id);
			// Show success message briefly
			setTimeout(() => {
				if (testingId === webhook.id) testingId = null;
			}, 2000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to test webhook';
			testingId = null;
		}
	}

	function getChannelName(channelId: string): string {
		const channel = textChannels.find((c: Channel) => c.id === channelId);
		return channel ? `#${channel.name}` : 'Unknown';
	}

	function formatSuccessRate(rate: number): string {
		return `${rate.toFixed(1)}%`;
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return 'Never';
		const date = new Date(dateStr);
		return date.toLocaleString();
	}

	function getStatusColor(stats: WebhookDeliveryStats | undefined): string {
		if (!stats || stats.total_deliveries === 0) return 'var(--text-muted, #72767d)';
		if (stats.success_rate >= 95) return '#3ba55d';
		if (stats.success_rate >= 80) return '#faa61a';
		return '#ed4245';
	}
</script>

<div class="webhook-manager">
	{#if error}
		<div class="error-banner">
			<span>{error}</span>
			<button class="dismiss" on:click={() => error = ''}>Dismiss</button>
		</div>
	{/if}

	{#if loading}
		<div class="loading">Loading webhooks...</div>
	{:else}
		{#if canManageWebhooks}
			<div class="create-section">
				<h3>Create Webhook</h3>
				<div class="create-form">
					<input
						type="text"
						bind:value={newWebhookName}
						placeholder="Webhook name"
						maxlength="80"
						disabled={creating}
					/>
					<select bind:value={newWebhookChannel} disabled={creating}>
						<option value="">Select channel</option>
						{#each textChannels as channel}
							<option value={channel.id}>#{channel.name}</option>
						{/each}
					</select>
					<button
						class="create-btn"
						on:click={createWebhook}
						disabled={creating || !newWebhookName.trim() || !newWebhookChannel}
					>
						{creating ? 'Creating...' : 'Create'}
					</button>
				</div>
			</div>
		{/if}

		{#if webhooks.length === 0}
			<div class="empty-state">
				<div class="empty-icon">🔗</div>
				<h3>No Webhooks</h3>
				<p>Webhooks allow external services to send messages to your channels.</p>
			</div>
		{:else}
			<div class="webhook-list">
				{#each webhooks as webhook (webhook.id)}
					<div class="webhook-card">
						{#if editingWebhook?.id === webhook.id}
							<div class="edit-form">
								<input
									type="text"
									bind:value={editName}
									placeholder="Webhook name"
									maxlength="80"
								/>
								<select bind:value={editChannel}>
									{#each textChannels as channel}
										<option value={channel.id}>#{channel.name}</option>
									{/each}
								</select>
								<div class="edit-actions">
									<button class="save-btn" on:click={saveEdit}>Save</button>
									<button class="cancel-btn" on:click={cancelEdit}>Cancel</button>
								</div>
							</div>
						{:else}
							<div class="webhook-main">
								<div class="webhook-info">
									<div class="webhook-avatar">
										{webhook.name.charAt(0).toUpperCase()}
									</div>
									<div class="webhook-details">
										<div class="webhook-name">{webhook.name}</div>
										<div class="webhook-channel">{getChannelName(webhook.channel_id)}</div>
									</div>
								</div>
								
								{#if webhookStats[webhook.id]}
									{@const stats = webhookStats[webhook.id]}
									<div class="webhook-stats-summary">
										<span class="stat-badge" style="color: {getStatusColor(stats)}">
											{formatSuccessRate(stats.success_rate)} success
										</span>
										{#if stats.failed_count > 0}
											<span class="stat-badge error">{stats.failed_count} failed</span>
										{/if}
									</div>
								{/if}
								
								{#if canManageWebhooks}
									<div class="webhook-actions">
										{#if webhook.token}
											<button
												class="action-btn copy"
												on:click={() => copyWebhookUrl(webhook)}
												title="Copy webhook URL"
											>
												{copiedId === webhook.id ? 'Copied!' : 'Copy URL'}
											</button>
										{/if}
										<button
											class="action-btn test"
											on:click={() => testWebhook(webhook)}
											disabled={testingId === webhook.id}
											title="Test webhook"
										>
											{testingId === webhook.id ? 'Testing...' : 'Test'}
										</button>
										<button
											class="action-btn stats"
											on:click={() => toggleStats(webhook.id)}
											title="View stats"
										>
											{expandedStatsId === webhook.id ? 'Hide Stats' : 'Stats'}
										</button>
										<button
											class="action-btn edit"
											on:click={() => startEdit(webhook)}
											title="Edit webhook"
										>
											Edit
										</button>
										{#if deleteConfirmId === webhook.id}
											<button
												class="action-btn delete confirm"
												on:click={() => confirmDelete(webhook.id)}
											>
												Confirm
											</button>
											<button
												class="action-btn cancel"
												on:click={() => deleteConfirmId = null}
											>
												Cancel
											</button>
										{:else}
											<button
												class="action-btn delete"
												on:click={() => deleteConfirmId = webhook.id}
												title="Delete webhook"
											>
												Delete
											</button>
										{/if}
									</div>
								{/if}
							</div>
							
							{#if expandedStatsId === webhook.id && webhookStats[webhook.id]}
								{@const stats = webhookStats[webhook.id]}
								<div class="webhook-stats-expanded">
									<div class="stats-grid">
										<div class="stat-item">
											<span class="stat-label">Total Deliveries</span>
											<span class="stat-value">{stats.total_deliveries}</span>
										</div>
										<div class="stat-item">
											<span class="stat-label">Successful</span>
											<span class="stat-value success">{stats.successful_count}</span>
										</div>
										<div class="stat-item">
											<span class="stat-label">Failed</span>
											<span class="stat-value error">{stats.failed_count}</span>
										</div>
										<div class="stat-item">
											<span class="stat-label">Success Rate</span>
											<span class="stat-value" style="color: {getStatusColor(stats)}">
												{formatSuccessRate(stats.success_rate)}
											</span>
										</div>
										<div class="stat-item">
											<span class="stat-label">Avg Duration</span>
											<span class="stat-value">{stats.avg_duration_ms.toFixed(0)}ms</span>
										</div>
										<div class="stat-item">
											<span class="stat-label">Last Delivery</span>
											<span class="stat-value">{formatDate(stats.last_delivery_at)}</span>
										</div>
									</div>
									
									{#if recentFailures[webhook.id]?.length > 0}
										<div class="recent-failures">
											<h4>Recent Failures</h4>
											{#each recentFailures[webhook.id] as failure}
												<div class="failure-item">
													<span class="failure-status">HTTP {failure.status_code}</span>
													<span class="failure-error" title={failure.error}>
														{failure.error || 'Unknown error'}
													</span>
													<span class="failure-time">
														{new Date(failure.created_at).toLocaleString()}
													</span>
												</div>
											{/each}
										</div>
									{/if}
								</div>
							{/if}
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	.webhook-manager {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.error-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		background: rgba(237, 66, 69, 0.15);
		border: 1px solid rgba(237, 66, 69, 0.3);
		border-radius: 8px;
		color: #ed4245;
		font-size: 0.875rem;
	}

	.dismiss {
		background: none;
		border: none;
		color: #ed4245;
		cursor: pointer;
		text-decoration: underline;
		font-size: 0.8rem;
	}

	.loading {
		text-align: center;
		padding: 2rem;
		color: var(--text-muted, #72767d);
	}

	.create-section {
		background: var(--bg-secondary, #2f3136);
		border-radius: 8px;
		padding: 1rem;
	}

	.create-section h3 {
		margin: 0 0 0.75rem 0;
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--text-secondary, #b9bbbe);
		text-transform: uppercase;
	}

	.create-form {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.create-form input,
	.create-form select {
		flex: 1;
		padding: 0.5rem 0.75rem;
		background: var(--bg-tertiary, #202225);
		border: 1px solid var(--border-color, #40444b);
		border-radius: 4px;
		color: var(--text-primary, #dcddde);
		font-size: 0.875rem;
	}

	.create-btn {
		padding: 0.5rem 1rem;
		background: var(--brand-color, #5865f2);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 0.875rem;
		cursor: pointer;
		white-space: nowrap;
	}

	.create-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.create-btn:hover:not(:disabled) {
		background: var(--brand-color-hover, #4752c4);
	}

	.empty-state {
		text-align: center;
		padding: 3rem 1rem;
		color: var(--text-muted, #72767d);
	}

	.empty-icon {
		font-size: 3rem;
		margin-bottom: 0.5rem;
	}

	.empty-state h3 {
		margin: 0 0 0.5rem 0;
		color: var(--text-primary, #dcddde);
	}

	.empty-state p {
		margin: 0;
		font-size: 0.875rem;
	}

	.webhook-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.webhook-card {
		display: flex;
		flex-direction: column;
		background: var(--bg-secondary, #2f3136);
		border-radius: 8px;
		overflow: hidden;
	}

	.webhook-main {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		gap: 0.5rem;
	}

	.webhook-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		min-width: 0;
		flex: 1;
	}

	.webhook-avatar {
		width: 40px;
		height: 40px;
		border-radius: 50%;
		background: var(--brand-color, #5865f2);
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-weight: 600;
		font-size: 1.125rem;
		flex-shrink: 0;
	}

	.webhook-details {
		min-width: 0;
	}

	.webhook-name {
		font-weight: 600;
		color: var(--text-primary, #dcddde);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.webhook-channel {
		font-size: 0.8rem;
		color: var(--text-muted, #72767d);
	}

	.webhook-stats-summary {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		flex-shrink: 0;
	}

	.stat-badge {
		font-size: 0.75rem;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		background: var(--bg-tertiary, #202225);
		white-space: nowrap;
	}

	.stat-badge.error {
		background: rgba(237, 66, 69, 0.2);
		color: #ed4245;
	}

	.webhook-actions {
		display: flex;
		gap: 0.375rem;
		flex-shrink: 0;
	}

	.action-btn {
		padding: 0.375rem 0.625rem;
		border: none;
		border-radius: 4px;
		font-size: 0.8rem;
		cursor: pointer;
		background: var(--bg-tertiary, #202225);
		color: var(--text-secondary, #b9bbbe);
	}

	.action-btn:hover:not(:disabled) {
		background: var(--bg-modifier-hover, #36393f);
		color: var(--text-primary, #dcddde);
	}

	.action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.action-btn.copy {
		color: var(--brand-color, #5865f2);
	}

	.action-btn.test {
		color: #3ba55d;
	}

	.action-btn.stats {
		color: #faa61a;
	}

	.action-btn.delete {
		color: #ed4245;
	}

	.action-btn.delete:hover {
		background: rgba(237, 66, 69, 0.15);
	}

	.action-btn.delete.confirm {
		background: #ed4245;
		color: white;
	}

	.webhook-stats-expanded {
		padding: 1rem;
		border-top: 1px solid var(--border-color, #40444b);
		background: var(--bg-tertiary, #202225);
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
		gap: 1rem;
	}

	.stat-item {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.stat-label {
		font-size: 0.75rem;
		color: var(--text-muted, #72767d);
		text-transform: uppercase;
	}

	.stat-value {
		font-size: 1rem;
		font-weight: 600;
		color: var(--text-primary, #dcddde);
	}

	.stat-value.success {
		color: #3ba55d;
	}

	.stat-value.error {
		color: #ed4245;
	}

	.recent-failures {
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid var(--border-color, #40444b);
	}

	.recent-failures h4 {
		margin: 0 0 0.5rem 0;
		font-size: 0.875rem;
		color: #ed4245;
	}

	.failure-item {
		display: grid;
		grid-template-columns: auto 1fr auto;
		gap: 0.75rem;
		align-items: center;
		padding: 0.5rem;
		background: rgba(237, 66, 69, 0.1);
		border-radius: 4px;
		margin-bottom: 0.5rem;
		font-size: 0.8rem;
	}

	.failure-status {
		color: #ed4245;
		font-weight: 600;
		white-space: nowrap;
	}

	.failure-error {
		color: var(--text-secondary, #b9bbbe);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.failure-time {
		color: var(--text-muted, #72767d);
		white-space: nowrap;
		font-size: 0.75rem;
	}

	.edit-form {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		width: 100%;
		padding: 0.75rem 1rem;
	}

	.edit-form input,
	.edit-form select {
		flex: 1;
		padding: 0.5rem 0.75rem;
		background: var(--bg-tertiary, #202225);
		border: 1px solid var(--border-color, #40444b);
		border-radius: 4px;
		color: var(--text-primary, #dcddde);
		font-size: 0.875rem;
	}

	.edit-actions {
		display: flex;
		gap: 0.375rem;
	}

	.save-btn {
		padding: 0.375rem 0.75rem;
		background: var(--brand-color, #5865f2);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 0.8rem;
		cursor: pointer;
	}

	.cancel-btn {
		padding: 0.375rem 0.75rem;
		background: var(--bg-tertiary, #202225);
		color: var(--text-secondary, #b9bbbe);
		border: none;
		border-radius: 4px;
		font-size: 0.8rem;
		cursor: pointer;
	}

	@media (max-width: 768px) {
		.create-form {
			flex-direction: column;
			align-items: stretch;
		}

		.webhook-main {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.75rem;
		}

		.webhook-actions {
			width: 100%;
			flex-wrap: wrap;
		}

		.stats-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.failure-item {
			grid-template-columns: 1fr;
			gap: 0.25rem;
		}
	}
</style>
