<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { serverChannels, type Channel } from '$lib/stores/channels';

	export let serverId: string;
	export let canManageWebhooks = false;

	interface Webhook {
		id: string;
		name: string;
		channel_id: string;
		guild_id: string;
		token?: string;
		avatar?: string | null;
		type: number;
	}

	let webhooks: Webhook[] = [];
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
				} catch {
					// Skip channels we can't access
				}
			}
			webhooks = allWebhooks;
		} catch (err) {
			error = 'Failed to load webhooks';
			console.error('Failed to load webhooks:', err);
		} finally {
			loading = false;
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

	function getChannelName(channelId: string): string {
		const channel = textChannels.find((c: Channel) => c.id === channelId);
		return channel ? `#${channel.name}` : 'Unknown';
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
							<div class="webhook-info">
								<div class="webhook-avatar">
									{webhook.name.charAt(0).toUpperCase()}
								</div>
								<div class="webhook-details">
									<div class="webhook-name">{webhook.name}</div>
									<div class="webhook-channel">{getChannelName(webhook.channel_id)}</div>
								</div>
							</div>
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
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		background: var(--bg-secondary, #2f3136);
		border-radius: 8px;
	}

	.webhook-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		min-width: 0;
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

	.action-btn:hover {
		background: var(--bg-modifier-hover, #36393f);
		color: var(--text-primary, #dcddde);
	}

	.action-btn.copy {
		color: var(--brand-color, #5865f2);
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

	.edit-form {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		width: 100%;
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
</style>
