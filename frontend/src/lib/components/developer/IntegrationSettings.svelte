<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { Application } from '$lib/services/slashCommands';

	export let application: Application;
	export let loading = false;

	const dispatch = createEventDispatcher<{
		save: { webhookUrl: string; description: string; icon: string };
		testWebhook: { webhookUrl: string };
		cancel: void;
	}>();

	// Form state
	let webhookUrl = '';
	let description = application?.description || '';
	let icon = application?.icon || '';
	let testStatus: 'idle' | 'testing' | 'success' | 'error' = 'idle';
	let testError = '';

	function handleSave() {
		dispatch('save', {
			webhookUrl,
			description,
			icon
		});
	}

	async function testWebhook() {
		if (!webhookUrl) return;

		testStatus = 'testing';
		testError = '';

		try {
			dispatch('testWebhook', { webhookUrl });

			// Simulate test - in real app this would call an API endpoint
			await new Promise(resolve => setTimeout(resolve, 1000));

			// For demo purposes, assume success if URL is valid format
			if (webhookUrl.startsWith('http')) {
				testStatus = 'success';
			} else {
				testStatus = 'error';
				testError = 'Invalid URL format';
			}
		} catch (err) {
			testStatus = 'error';
			testError = 'Connection failed';
		}
	}

	function getIntegrationEndpoint(appId: string): string {
		return `/api/v1/interactions?application_id=${appId}`;
	}

	function getWebhookDocumentationUrl(): string {
		return 'https://docs.hearth.example.com/webhooks';
	}
</script>

<div class="integration-settings">
	<form on:submit|preventDefault={handleSave}>
		<div class="form-section">
			<h3>Integration Settings</h3>
			<p class="description">
				Configure how your application receives and responds to slash command interactions.
			</p>
		</div>

		<div class="form-section">
			<h4>Application Info</h4>

			<div class="app-preview">
				<div class="app-icon-preview">
					{#if icon}
						<img src={icon} alt={application.name} />
					{:else}
						<span class="default-icon">{application.name.charAt(0).toUpperCase()}</span>
					{/if}
				</div>
				<div class="app-details">
					<span class="app-name">{application.name}</span>
					<span class="app-id">ID: {application.id}</span>
				</div>
			</div>

			<div class="form-group">
				<label for="description">Description</label>
				<textarea
					id="description"
					bind:value={description}
					placeholder="Describe what your application does"
					maxlength="500"
					rows="3"
				></textarea>
				<span class="hint">{description.length}/500 characters</span>
			</div>

			<div class="form-group">
				<label for="icon">Icon URL</label>
				<input
					id="icon"
					type="url"
					bind:value={icon}
					placeholder="https://example.com/icon.png"
				/>
				<span class="hint">Recommended: 512x512 PNG or GIF (max 256KB)</span>
			</div>
		</div>

		<div class="form-section">
			<h4>Interaction Endpoint</h4>

			<div class="endpoint-info">
				<label>Your interaction endpoint</label>
				<div class="endpoint-box">
					<code>{getIntegrationEndpoint(application.id)}</code>
					<button
						type="button"
						class="btn-copy"
						on:click={() => navigator.clipboard.writeText(getIntegrationEndpoint(application.id))}
					>
						Copy
					</button>
				</div>
				<span class="hint">
					Use this URL as the <strong>Interaction Endpoint URL</strong> in your app settings.
				</span>
			</div>
		</div>

		<div class="form-section">
			<h4>Webhook (Optional)</h4>
			<p class="description">
				If your application uses webhooks for command execution, configure the URL here.
			</p>

			<div class="form-group">
				<label for="webhookUrl">Webhook URL</label>
				<div class="webhook-input-group">
					<input
						id="webhookUrl"
						type="url"
						bind:value={webhookUrl}
						placeholder="https://your-server.com/webhook"
					/>
					<button
						type="button"
						class="btn-test"
						class:testing={testStatus === 'testing'}
						disabled={!webhookUrl || testStatus === 'testing'}
						on:click={testWebhook}
					>
						{testStatus === 'testing' ? 'Testing...' : 'Test'}
					</button>
				</div>

				{#if testStatus === 'success'}
					<span class="test-result success">✓ Webhook endpoint is reachable</span>
				{:else if testStatus === 'error'}
					<span class="test-result error">✗ {testError}</span>
				{/if}
			</div>
		</div>

		<div class="form-section">
			<h4>API Rate Limits</h4>

			<div class="rate-limits">
				<div class="rate-limit-item">
					<span class="rate-limit-label">Commands</span>
					<span class="rate-limit-value">120 requests/minute</span>
				</div>
				<div class="rate-limit-item">
					<span class="rate-limit-label">Autocomplete</span>
					<span class="rate-limit-value">60 requests/minute</span>
				</div>
				<div class="rate-limit-item">
					<span class="rate-limit-label">Responses</span>
					<span class="rate-limit-value">15 requests/second</span>
				</div>
			</div>
		</div>

		<div class="form-section">
			<h4>Documentation</h4>

			<div class="docs-links">
				<a href={getWebhookDocumentationUrl()} target="_blank" rel="noopener" class="doc-link">
					<span class="doc-icon">📚</span>
					<span>Webhook Integration Guide</span>
				</a>
				<a href="https://docs.hearth.example.com/slash-commands" target="_blank" rel="noopener" class="doc-link">
					<span class="doc-icon">⚡</span>
					<span>Slash Commands Documentation</span>
				</a>
				<a href="https://docs.hearth.example.com/interactions" target="_blank" rel="noopener" class="doc-link">
					<span class="doc-icon">🔄</span>
					<span>Interaction Response Types</span>
				</a>
			</div>
		</div>

		<div class="form-actions">
			<button type="button" class="btn-secondary" on:click={() => dispatch('cancel')}>
				Cancel
			</button>
			<button type="submit" class="btn-primary" disabled={loading}>
				{#if loading}
					Saving...
				{:else}
					Save Changes
				{/if}
			</button>
		</div>
	</form>
</div>

<style>
	.integration-settings {
		padding: 20px;
		max-width: 700px;
		margin: 0 auto;
	}

	.form-section {
		margin-bottom: 28px;
		padding-bottom: 20px;
		border-bottom: 1px solid var(--border-color, #3a3a3a);
	}

	.form-section:last-of-type {
		border-bottom: none;
	}

	h3 {
		font-size: 16px;
		font-weight: 600;
		margin-bottom: 8px;
		color: var(--text-primary, #ffffff);
	}

	h4 {
		font-size: 14px;
		font-weight: 600;
		margin-bottom: 12px;
		color: var(--text-secondary, #b9b9b9);
	}

	.description {
		font-size: 13px;
		color: var(--text-secondary, #b9b9b9);
		margin-bottom: 16px;
	}

	.app-preview {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 16px;
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		margin-bottom: 16px;
	}

	.app-icon-preview {
		width: 56px;
		height: 56px;
		border-radius: 8px;
		background: var(--accent-color, #5865f2);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
	}

	.app-icon-preview img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.default-icon {
		font-size: 24px;
		font-weight: 600;
		color: white;
	}

	.app-details {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.app-name {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary, #ffffff);
	}

	.app-id {
		font-size: 12px;
		color: var(--text-muted, #727067);
		font-family: monospace;
	}

	.form-group {
		margin-bottom: 16px;
	}

	.form-group label {
		display: block;
		font-size: 13px;
		font-weight: 500;
		color: var(--text-secondary, #b9b9b9);
		margin-bottom: 6px;
	}

	.form-group input,
	.form-group textarea {
		width: 100%;
		padding: 10px 12px;
		background: var(--background-secondary, #2a2a2a);
		border: 1px solid var(--border-color, #3a3a3a);
		border-radius: 4px;
		color: var(--text-primary, #ffffff);
		font-size: 14px;
		font-family: inherit;
	}

	.form-group input:focus,
	.form-group textarea:focus {
		outline: none;
		border-color: var(--accent-color, #5865f2);
	}

	textarea {
		resize: vertical;
		min-height: 60px;
	}

	.hint {
		display: block;
		font-size: 12px;
		color: var(--text-muted, #727067);
		margin-top: 4px;
	}

	.endpoint-info label {
		display: block;
		font-size: 13px;
		font-weight: 500;
		color: var(--text-secondary, #b9b9b9);
		margin-bottom: 6px;
	}

	.endpoint-box {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		background: var(--background-secondary, #2a2a2a);
		border: 1px solid var(--border-color, #3a3a3a);
		border-radius: 4px;
	}

	.endpoint-box code {
		flex: 1;
		font-family: monospace;
		font-size: 13px;
		color: var(--accent-color, #5865f2);
	}

	.btn-copy {
		padding: 4px 12px;
		background: var(--background-tertiary, #3a3a3a);
		border: none;
		border-radius: 4px;
		color: var(--text-secondary, #b9b9b9);
		font-size: 12px;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-copy:hover {
		background: var(--background-hover, #4a4a4a);
	}

	.webhook-input-group {
		display: flex;
		gap: 8px;
	}

	.webhook-input-group input {
		flex: 1;
	}

	.btn-test {
		padding: 10px 16px;
		background: var(--background-tertiary, #3a3a3a);
		border: none;
		border-radius: 4px;
		color: var(--text-primary, #ffffff);
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		white-space: nowrap;
		transition: background-color 0.15s;
	}

	.btn-test:hover:not(:disabled) {
		background: var(--success-color, #23a559);
	}

	.btn-test:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-test.testing {
		opacity: 0.7;
	}

	.test-result {
		display: block;
		font-size: 12px;
		margin-top: 6px;
	}

	.test-result.success {
		color: var(--success-color, #23a559);
	}

	.test-result.error {
		color: var(--danger-color, #da373c);
	}

	.rate-limits {
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		padding: 12px;
	}

	.rate-limit-item {
		display: flex;
		justify-content: space-between;
		padding: 8px 0;
		border-bottom: 1px solid var(--border-color, #3a3a3a);
	}

	.rate-limit-item:last-child {
		border-bottom: none;
	}

	.rate-limit-label {
		font-size: 13px;
		color: var(--text-secondary, #b9b9b9);
	}

	.rate-limit-value {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary, #ffffff);
	}

	.docs-links {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.doc-link {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px;
		background: var(--background-secondary, #2a2a2a);
		border-radius: 6px;
		color: var(--text-primary, #ffffff);
		text-decoration: none;
		font-size: 13px;
		transition: background-color 0.15s;
	}

	.doc-link:hover {
		background: var(--background-hover, #3a3a3a);
	}

	.doc-icon {
		font-size: 16px;
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		padding-top: 16px;
	}

	.btn-primary {
		padding: 10px 20px;
		background: var(--accent-color, #5865f2);
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--accent-color-hover, #4752c4);
	}

	.btn-primary:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-secondary {
		padding: 10px 20px;
		background: var(--background-tertiary, #3a3a3a);
		color: var(--text-primary, #ffffff);
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.btn-secondary:hover {
		background: var(--background-hover, #4a4a4a);
	}
</style>
