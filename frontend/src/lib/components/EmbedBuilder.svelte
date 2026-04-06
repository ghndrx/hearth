<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import EmbedRenderer from './EmbedRenderer.svelte';
	import type { EmbedData } from './EmbedRenderer.svelte';

	export let show = false;

	const dispatch = createEventDispatcher<{
		close: void;
		send: { embed: EmbedData };
		saveTemplate: { embed: EmbedData; name: string };
		loadTemplate: { template: EmbedData };
	}>();

	// Embed form state
	let embed: EmbedData = {
		title: '',
		description: '',
		url: '',
		color: undefined,
		author: { name: '', url: '', icon: '' },
		footer: { text: '', icon: '' },
		image_url: '',
		thumbnail_url: '',
		timestamp: '',
		type: 'rich',
		fields: []
	};

	// UI state
	let activeTab: 'content' | 'author' | 'footer' | 'fields' | 'advanced' = 'content';
	let colorInput = '#5865f2';
	let fields: Array<{ name: string; value: string; inline: boolean }> = [];
	let isFetchingPreview = false;
	let previewUrl = '';
	let previewData: EmbedData | null = null;

	// Predefined colors
	const presetColors = [
		'#000000', '#7289da', '#5865f2', '#3ba55c', '#f04747',
		'#faa61a', '#f04747', '#eb459e', '#1abc9c', '#3498db'
	];

	onMount(() => {
		if (show) {
			resetForm();
		}
	});

	$: if (show) {
		resetForm();
	}

	function resetForm() {
		embed = {
			title: '',
			description: '',
			url: '',
			color: undefined,
			author: { name: '', url: '', icon: '' },
			footer: { text: '', icon: '' },
			image_url: '',
			thumbnail_url: '',
			timestamp: '',
			type: 'rich',
			fields: []
		};
		fields = [];
		colorInput = '#5865f2';
		previewData = null;
		previewUrl = '';
	}

	function handleClose() {
		dispatch('close');
	}

	function handleSend() {
		// Clean up empty fields
		const cleanEmbed: EmbedData = {
			...embed,
			author: embed.author?.name || embed.author?.icon || embed.author?.url
				? embed.author
				: undefined,
			footer: embed.footer?.text || embed.footer?.icon
				? embed.footer
				: undefined
		};
		dispatch('send', { embed: cleanEmbed });
		handleClose();
	}

	function addField() {
		fields = [...fields, { name: '', value: '', inline: false }];
	}

	function removeField(index: number) {
		fields = fields.filter((_, i) => i !== index);
	}

	function handleColorChange(e: Event) {
		const target = e.target as HTMLInputElement;
		colorInput = target.value;
		embed.color = parseInt(colorInput.replace('#', ''), 16);
	}

	function handlePresetColor(color: string) {
		colorInput = color;
		embed.color = parseInt(color.replace('#', ''), 16);
	}

	async function fetchPreview() {
		if (!previewUrl) return;

		isFetchingPreview = true;
		try {
			const response = await fetch(`/api/v1/embeds/fetch?url=${encodeURIComponent(previewUrl)}`);
			if (response.ok) {
				const data = await response.json();
				previewData = data;
			}
		} catch (err) {
			console.error('Failed to fetch preview:', err);
		} finally {
			isFetchingPreview = false;
		}
	}

	// Build current embed from form
	$: currentEmbed = {
		...embed,
		fields: fields.filter(f => f.name && f.value)
	};

	// Check if embed has content
	$: hasContent = currentEmbed.title || currentEmbed.description || currentEmbed.url ||
		currentEmbed.image_url || currentEmbed.thumbnail_url || hasFields;

	$: hasFields = fields.some(f => f.name && f.value);
</script>

{#if show}
	<div class="modal-backdrop" on:click={handleClose} on:keydown={(e) => e.key === 'Escape' && handleClose()} role="dialog" aria-modal="true" aria-label="Embed Builder">
		<div class="modal-content" on:click|stopPropagation role="document">
			<!-- Header -->
			<div class="modal-header">
				<h2>Embed Builder</h2>
				<button class="close-btn" on:click={handleClose} aria-label="Close">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="18" y1="6" x2="6" y2="18" />
						<line x1="6" y1="6" x2="18" y2="18" />
					</svg>
				</button>
			</div>

			<!-- Tabs -->
			<div class="tabs">
				<button class="tab" class:active={activeTab === 'content'} on:click={() => activeTab = 'content'}>
					Content
				</button>
				<button class="tab" class:active={activeTab === 'author'} on:click={() => activeTab = 'author'}>
					Author
				</button>
				<button class="tab" class:active={activeTab === 'footer'} on:click={() => activeTab = 'footer'}>
					Footer
				</button>
				<button class="tab" class:active={activeTab === 'fields'} on:click={() => activeTab = 'fields'}>
					Fields
				</button>
				<button class="tab" class:active={activeTab === 'advanced'} on:click={() => activeTab = 'advanced'}>
					Advanced
				</button>
			</div>

			<!-- Tab Content -->
			<div class="tab-content">
				{#if activeTab === 'content'}
					<div class="form-group">
						<label for="embed-title">Title</label>
						<input
							id="embed-title"
							type="text"
							bind:value={embed.title}
							placeholder="Embed title"
							maxlength="256"
						/>
					</div>

					<div class="form-group">
						<label for="embed-description">Description</label>
						<textarea
							id="embed-description"
							bind:value={embed.description}
							placeholder="Embed description"
							rows="4"
						></textarea>
					</div>

					<div class="form-group">
						<label for="embed-url">URL</label>
						<input
							id="embed-url"
							type="url"
							bind:value={embed.url}
							placeholder="https://example.com"
						/>
					</div>

					<div class="form-row">
						<div class="form-group">
							<label for="embed-image">Image URL</label>
							<input
								id="embed-image"
								type="url"
								bind:value={embed.image_url}
								placeholder="https://example.com/image.png"
							/>
						</div>
						<div class="form-group">
							<label for="embed-thumbnail">Thumbnail URL</label>
							<input
								id="embed-thumbnail"
								type="url"
								bind:value={embed.thumbnail_url}
								placeholder="https://example.com/thumb.png"
							/>
						</div>
					</div>
				{/if}

				{#if activeTab === 'author'}
					<div class="form-group">
						<label for="embed-author-name">Author Name</label>
						<input
							id="embed-author-name"
							type="text"
							bind:value={embed.author.name}
							placeholder="Author name"
							maxlength="256"
						/>
					</div>

					<div class="form-group">
						<label for="embed-author-url">Author URL</label>
						<input
							id="embed-author-url"
							type="url"
							bind:value={embed.author.url}
							placeholder="https://example.com"
						/>
					</div>

					<div class="form-group">
						<label for="embed-author-icon">Author Icon URL</label>
						<input
							id="embed-author-icon"
							type="url"
							bind:value={embed.author.icon}
							placeholder="https://example.com/icon.png"
						/>
					</div>
				{/if}

				{#if activeTab === 'footer'}
					<div class="form-group">
						<label for="embed-footer-text">Footer Text</label>
						<input
							id="embed-footer-text"
							type="text"
							bind:value={embed.footer.text}
							placeholder="Footer text"
							maxlength="2048"
						/>
					</div>

					<div class="form-group">
						<label for="embed-footer-icon">Footer Icon URL</label>
						<input
							id="embed-footer-icon"
							type="url"
							bind:value={embed.footer.icon}
							placeholder="https://example.com/icon.png"
						/>
					</div>
				{/if}

				{#if activeTab === 'fields'}
					<div class="fields-section">
						{#each fields as field, index}
							<div class="field-row">
								<div class="field-inputs">
									<input
										type="text"
										bind:value={field.name}
										placeholder="Field name"
										maxlength="256"
									/>
									<input
										type="text"
										bind:value={field.value}
										placeholder="Field value"
									/>
								</div>
								<label class="inline-checkbox">
									<input type="checkbox" bind:checked={field.inline} />
									<span>Inline</span>
								</label>
								<button class="remove-field-btn" on:click={() => removeField(index)} aria-label="Remove field">
									<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<line x1="18" y1="6" x2="6" y2="18" />
										<line x1="6" y1="6" x2="18" y2="18" />
									</svg>
								</button>
							</div>
						{/each}

						<button class="add-field-btn" on:click={addField}>
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<line x1="12" y1="5" x2="12" y2="19" />
								<line x1="5" y1="12" x2="19" y2="12" />
							</svg>
							Add Field
						</button>
					</div>
				{/if}

				{#if activeTab === 'advanced'}
					<div class="form-group">
						<label>Color</label>
						<div class="color-input-row">
							<input
								type="color"
								value={colorInput}
								on:input={handleColorChange}
								class="color-picker"
							/>
							<input
								type="text"
								value={colorInput}
								on:change={(e) => handlePresetColor((e.target as HTMLInputElement).value)}
								placeholder="#5865f2"
								maxlength="7"
							/>
						</div>
						<div class="preset-colors">
							{#each presetColors as color}
								<button
									class="preset-color"
									style="background-color: {color}"
									on:click={() => handlePresetColor(color)}
									aria-label="Color {color}"
								></button>
							{/each}
						</div>
					</div>

					<div class="form-group">
						<label for="embed-timestamp">Timestamp</label>
						<input
							id="embed-timestamp"
							type="datetime-local"
							bind:value={embed.timestamp}
						/>
					</div>

					<div class="form-group">
						<label>URL Preview Fetcher</label>
						<div class="url-preview-row">
							<input
								type="url"
								bind:value={previewUrl}
								placeholder="Enter URL to fetch metadata"
							/>
							<button class="fetch-btn" on:click={fetchPreview} disabled={!previewUrl || isFetchingPreview}>
								{isFetchingPreview ? 'Fetching...' : 'Fetch'}
							</button>
						</div>
						{#if previewData}
							<div class="preview-notice">
								Loaded preview from URL
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Preview Section -->
			<div class="preview-section">
				<h3>Preview</h3>
				<div class="preview-container">
					{#if hasContent}
						<EmbedRenderer embed={currentEmbed} />
					{:else}
						<div class="preview-empty">
							<p>Add content to see preview</p>
						</div>
					{/if}
				</div>
			</div>

			<!-- Actions -->
			<div class="modal-actions">
				<button class="btn btn-secondary" on:click={handleClose}>Cancel</button>
				<button class="btn btn-primary" on:click={handleSend} disabled={!hasContent}>
					Send Embed
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.85);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal-content {
		background: #36393f;
		border-radius: 8px;
		width: 90%;
		max-width: 900px;
		max-height: 90vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid rgba(79, 84, 92, 0.4);
	}

	.modal-header h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: #fff;
	}

	.close-btn {
		background: none;
		border: none;
		color: #b9bbbe;
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.close-btn:hover {
		color: #fff;
		background: rgba(79, 84, 92, 0.4);
	}

	.tabs {
		display: flex;
		padding: 0 20px;
		border-bottom: 1px solid rgba(79, 84, 92, 0.4);
	}

	.tab {
		background: none;
		border: none;
		color: #b9bbbe;
		padding: 12px 16px;
		font-size: 14px;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		margin-bottom: -1px;
	}

	.tab:hover {
		color: #fff;
	}

	.tab.active {
		color: #fff;
		border-bottom-color: #5865f2;
	}

	.tab-content {
		padding: 20px;
		overflow-y: auto;
		flex: 1;
		min-height: 200px;
	}

	.form-group {
		margin-bottom: 16px;
	}

	.form-group label {
		display: block;
		font-size: 12px;
		font-weight: 600;
		color: #b9bbbe;
		margin-bottom: 6px;
		text-transform: uppercase;
	}

	.form-group input,
	.form-group textarea {
		width: 100%;
		background: #40444b;
		border: none;
		border-radius: 4px;
		padding: 10px 12px;
		font-size: 14px;
		color: #fff;
		box-sizing: border-box;
	}

	.form-group input::placeholder,
	.form-group textarea::placeholder {
		color: #72767d;
	}

	.form-group input:focus,
	.form-group textarea:focus {
		outline: none;
		box-shadow: 0 0 0 2px #5865f2;
	}

	.form-row {
		display: flex;
		gap: 16px;
	}

	.form-row .form-group {
		flex: 1;
	}

	textarea {
		resize: vertical;
		min-height: 80px;
	}

	/* Color picker */
	.color-input-row {
		display: flex;
		gap: 8px;
		align-items: center;
	}

	.color-picker {
		width: 40px !important;
		height: 40px;
		padding: 0;
		border: none;
		cursor: pointer;
	}

	.color-picker::-webkit-color-swatch-wrapper {
		padding: 0;
	}

	.color-picker::-webkit-color-swatch {
		border-radius: 4px;
		border: 2px solid #4f545c;
	}

	.preset-colors {
		display: flex;
		gap: 6px;
		margin-top: 8px;
		flex-wrap: wrap;
	}

	.preset-color {
		width: 24px;
		height: 24px;
		border-radius: 4px;
		border: 2px solid transparent;
		cursor: pointer;
	}

	.preset-color:hover {
		border-color: #fff;
	}

	/* Fields section */
	.fields-section {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.field-row {
		display: flex;
		gap: 8px;
		align-items: flex-start;
		padding: 12px;
		background: #40444b;
		border-radius: 4px;
	}

	.field-inputs {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.field-inputs input {
		background: #36393f;
		border: none;
		border-radius: 4px;
		padding: 8px 10px;
		font-size: 13px;
		color: #fff;
	}

	.field-inputs input::placeholder {
		color: #72767d;
	}

	.inline-checkbox {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: #b9bbbe;
		white-space: nowrap;
		cursor: pointer;
	}

	.inline-checkbox input {
		cursor: pointer;
	}

	.remove-field-btn {
		background: none;
		border: none;
		color: #72767d;
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.remove-field-btn:hover {
		color: #f04747;
		background: rgba(240, 71, 71, 0.1);
	}

	.add-field-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		background: rgba(79, 84, 92, 0.3);
		border: none;
		border-radius: 4px;
		padding: 10px;
		color: #b9bbbe;
		font-size: 13px;
		cursor: pointer;
	}

	.add-field-btn:hover {
		background: rgba(79, 84, 92, 0.5);
		color: #fff;
	}

	/* URL Preview */
	.url-preview-row {
		display: flex;
		gap: 8px;
	}

	.url-preview-row input {
		flex: 1;
	}

	.fetch-btn {
		background: #5865f2;
		border: none;
		border-radius: 4px;
		padding: 8px 16px;
		color: #fff;
		font-size: 13px;
		cursor: pointer;
		white-space: nowrap;
	}

	.fetch-btn:hover:not(:disabled) {
		background: #4752c4;
	}

	.fetch-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.preview-notice {
		margin-top: 8px;
		font-size: 12px;
		color: #3ba55c;
	}

	/* Preview section */
	.preview-section {
		border-top: 1px solid rgba(79, 84, 92, 0.4);
		padding: 16px 20px;
		background: #2f3136;
	}

	.preview-section h3 {
		margin: 0 0 12px 0;
		font-size: 13px;
		font-weight: 600;
		color: #b9bbbe;
		text-transform: uppercase;
	}

	.preview-container {
		min-height: 80px;
	}

	.preview-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 80px;
		color: #72767d;
		font-size: 13px;
	}

	/* Actions */
	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		padding: 16px 20px;
		border-top: 1px solid rgba(79, 84, 92, 0.4);
	}

	.btn {
		padding: 10px 20px;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		border: none;
	}

	.btn-secondary {
		background: #4f545c;
		color: #fff;
	}

	.btn-secondary:hover {
		background: #5d6269;
	}

	.btn-primary {
		background: #5865f2;
		color: #fff;
	}

	.btn-primary:hover:not(:disabled) {
		background: #4752c4;
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
