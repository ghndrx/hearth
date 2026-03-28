<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { api } from '$lib/api';

	export let serverId: string;
	export let canManageStickers = false;

	const dispatch = createEventDispatcher<{
		close: void;
		stickerAdded: { sticker: Sticker };
		stickerDeleted: { stickerId: string };
	}>();

	interface Sticker {
		id: string;
		name: string;
		tags: string[];
		url: string;
		format: 'PNG' | 'APNG' | 'GIF';
		guild_id?: string;
		creator_id?: string;
		created_at: string;
	}

	let stickers: Sticker[] = [];
	let loading = true;
	let error = '';
	let uploading = false;
	let uploadProgress = 0;
	let uploadError = '';
	let newStickerName = '';
	let newStickerTags = '';
	let selectedFile: File | null = null;
	let previewUrl: string | null = null;
	let fileInput: HTMLInputElement;
	let searchQuery = '';
	let deleteConfirmId: string | null = null;

	const MAX_STICKER_SIZE = 512 * 1024; // 512KB
	const ALLOWED_TYPES = ['image/png', 'image/apng', 'image/gif'];

	async function loadStickers() {
		loading = true;
		error = '';
		
		try {
			stickers = await api.get<Sticker[]>(`/servers/${serverId}/stickers`);
		} catch (err) {
			error = 'Failed to load stickers';
			console.error('Failed to load stickers:', err);
		} finally {
			loading = false;
		}
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		
		if (!file) return;
		
		uploadError = '';
		
		// Validate file type
		if (!ALLOWED_TYPES.includes(file.type)) {
			uploadError = 'Invalid file type. Use PNG, APNG, or GIF.';
			return;
		}
		
		// Validate file size
		if (file.size > MAX_STICKER_SIZE) {
			uploadError = 'File too large. Maximum size is 512KB.';
			return;
		}
		
		selectedFile = file;
		
		// Generate preview
		const reader = new FileReader();
		reader.onload = (e) => {
			previewUrl = e.target?.result as string;
		};
		reader.readAsDataURL(file);
		
		// Auto-generate name from filename
		if (!newStickerName) {
			const baseName = file.name.replace(/\.[^.]+$/, ''); // Remove extension
			newStickerName = baseName
				.replace(/[^a-zA-Z0-9_\s]/g, '') // Remove invalid chars
				.replace(/_+/g, ' ') // Replace underscores with spaces
				.replace(/\s+/g, ' ') // Collapse spaces
				.trim()
				.slice(0, 30); // Max length
		}
	}

	function clearSelection() {
		selectedFile = null;
		previewUrl = null;
		newStickerName = '';
		newStickerTags = '';
		uploadError = '';
		if (fileInput) {
			fileInput.value = '';
		}
	}

	async function handleUpload() {
		if (!selectedFile || !newStickerName.trim()) return;
		
		uploading = true;
		uploadError = '';
		uploadProgress = 0;
		
		try {
			const formData = new FormData();
			formData.append('image', selectedFile);
			formData.append('name', newStickerName.trim());
			formData.append('tags', newStickerTags.trim());
			
			const sticker = await api.upload<Sticker>(`/servers/${serverId}/stickers`, formData);
			
			stickers = [sticker, ...stickers];
			dispatch('stickerAdded', { sticker });
			
			clearSelection();
		} catch (err) {
			uploadError = err instanceof Error ? err.message : 'Failed to upload sticker';
			console.error('Failed to upload sticker:', err);
		} finally {
			uploading = false;
			uploadProgress = 0;
		}
	}

	async function handleDelete(stickerId: string) {
		try {
			await api.delete(`/servers/${serverId}/stickers/${stickerId}`);
			stickers = stickers.filter(s => s.id !== stickerId);
			dispatch('stickerDeleted', { stickerId });
			deleteConfirmId = null;
		} catch (err) {
			console.error('Failed to delete sticker:', err);
			error = 'Failed to delete sticker';
		}
	}

	function confirmDelete(stickerId: string) {
		deleteConfirmId = stickerId;
	}

	function cancelDelete() {
		deleteConfirmId = null;
	}

	$: filteredStickers = searchQuery
		? stickers.filter(s => 
			s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			s.tags?.some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()))
		)
		: stickers;

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return bytes + ' B';
		if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
		return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
	}

	onMount(() => {
		loadStickers();
	});
</script>

<div class="sticker-manager">
	<!-- Header -->
	<div class="header">
		<h2>Server Stickers</h2>
		<button class="close-btn" on:click={() => dispatch('close')} aria-label="Close">
			<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
				<path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
			</svg>
		</button>
	</div>

	<!-- Upload Section -->
	{#if canManageStickers}
		<div class="upload-section">
			<h3>Upload Sticker</h3>
			
			{#if selectedFile}
				<!-- Preview and Form -->
				<div class="preview-container">
					<img src={previewUrl} alt="Sticker preview" class="preview-image" />
					<button class="clear-btn" on:click={clearSelection} aria-label="Clear selection">
						<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
							<path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
						</svg>
					</button>
				</div>
				
				<div class="form-group">
					<label for="sticker-name">Name</label>
					<input
						id="sticker-name"
						type="text"
						bind:value={newStickerName}
						placeholder="Sticker name"
						maxlength="30"
						class="input"
					/>
					<span class="char-count">{newStickerName.length}/30</span>
				</div>
				
				<div class="form-group">
					<label for="sticker-tags">Tags (comma separated)</label>
					<input
						id="sticker-tags"
						type="text"
						bind:value={newStickerTags}
						placeholder="happy, sad, cat"
						class="input"
					/>
				</div>
				
				{#if uploadError}
					<div class="error-message">{uploadError}</div>
				{/if}
				
				<div class="upload-actions">
					<button 
						class="btn btn-secondary" 
						on:click={clearSelection}
						disabled={uploading}
					>
						Cancel
					</button>
					<button 
						class="btn btn-primary" 
						on:click={handleUpload}
						disabled={uploading || !newStickerName.trim()}
					>
						{uploading ? 'Uploading...' : 'Upload'}
					</button>
				</div>
			{:else}
				<!-- File Selection -->
				<div 
					class="drop-zone"
					role="button"
					tabindex="0"
					on:click={() => fileInput?.click()}
					on:keydown={(e) => e.key === 'Enter' && fileInput?.click()}
				>
					<input
						bind:this={fileInput}
						type="file"
						accept="image/png,image/apng,image/gif"
						on:change={handleFileSelect}
						class="hidden"
					/>
					<svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor" class="upload-icon">
						<path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM14 13v4h-4v-4H7l5-5 5 5h-3z"/>
					</svg>
					<p>Click to upload a sticker</p>
					<p class="hint">PNG, APNG, or GIF • Max 512KB</p>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Search -->
	{#if stickers.length > 0}
		<div class="search-container">
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search stickers..."
				class="search-input"
			/>
		</div>
	{/if}

	<!-- Error Message -->
	{#if error}
		<div class="error-banner">
			<span>{error}</span>
			<button on:click={loadStickers}>Retry</button>
		</div>
	{/if}

	<!-- Loading State -->
	{#if loading}
		<div class="loading">
			<div class="spinner"></div>
			<span>Loading stickers...</span>
		</div>
	{:else if stickers.length === 0}
		<div class="empty-state">
			<div class="empty-icon">📦</div>
			<h3>No stickers yet</h3>
			<p>
				{#if canManageStickers}
					Upload your first sticker to get started!
				{:else}
					This server doesn't have any stickers yet.
				{/if}
			</p>
		</div>
	{:else if filteredStickers.length === 0}
		<div class="empty-state">
			<div class="empty-icon">🔍</div>
			<h3>No results</h3>
			<p>No stickers match "{searchQuery}"</p>
		</div>
	{:else}
		<!-- Stickers Grid -->
		<div class="stickers-grid">
			{#each filteredStickers as sticker (sticker.id)}
				<div class="sticker-card">
					<div class="sticker-preview">
						<img src={sticker.url} alt={sticker.name} />
					</div>
					<div class="sticker-info">
						<span class="sticker-name" title={sticker.name}>{sticker.name}</span>
						<span class="sticker-format">{sticker.format}</span>
					</div>
					{#if canManageStickers}
						{#if deleteConfirmId === sticker.id}
							<div class="delete-confirm">
								<span>Delete?</span>
								<button class="btn-icon danger" on:click={() => handleDelete(sticker.id)}>Yes</button>
								<button class="btn-icon" on:click={cancelDelete}>No</button>
							</div>
						{:else}
							<button 
								class="delete-btn" 
								on:click={() => confirmDelete(sticker.id)}
								aria-label="Delete {sticker.name}"
							>
								<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
									<path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
								</svg>
							</button>
						{/if}
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.sticker-manager {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--bg-secondary);
	}

	.header h2 {
		font-size: 18px;
		font-weight: 600;
		margin: 0;
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		transition: background-color 0.15s;
	}

	.close-btn:hover {
		background: var(--bg-modifier-hover);
		color: var(--text-primary);
	}

	.upload-section {
		padding: 16px;
		border-bottom: 1px solid var(--bg-secondary);
	}

	.upload-section h3 {
		font-size: 14px;
		font-weight: 600;
		margin: 0 0 12px 0;
	}

	.drop-zone {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 32px;
		border: 2px dashed var(--bg-modifier-accent);
		border-radius: 8px;
		cursor: pointer;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.drop-zone:hover {
		border-color: var(--brand-primary);
		background: rgba(88, 101, 242, 0.05);
	}

	.upload-icon {
		color: var(--text-muted);
		margin-bottom: 8px;
	}

	.drop-zone p {
		margin: 0;
		color: var(--text-secondary);
		font-size: 14px;
	}

	.drop-zone .hint {
		font-size: 12px;
		color: var(--text-muted);
		margin-top: 4px;
	}

	.hidden {
		display: none;
	}

	.preview-container {
		position: relative;
		display: inline-block;
		margin-bottom: 12px;
	}

	.preview-image {
		max-width: 100px;
		max-height: 100px;
		object-fit: contain;
		border-radius: 8px;
		border: 1px solid var(--bg-modifier-accent);
	}

	.clear-btn {
		position: absolute;
		top: -8px;
		right: -8px;
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: var(--status-danger);
		border: none;
		color: white;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.form-group {
		margin-bottom: 12px;
		position: relative;
	}

	.form-group label {
		display: block;
		font-size: 12px;
		font-weight: 500;
		color: var(--text-secondary);
		margin-bottom: 4px;
	}

	.form-group .input {
		width: 100%;
		padding: 8px 12px;
		background: var(--bg-secondary);
		border: 1px solid var(--bg-modifier-accent);
		border-radius: 4px;
		color: var(--text-primary);
		font-size: 14px;
	}

	.form-group .input:focus {
		outline: none;
		border-color: var(--brand-primary);
	}

	.char-count {
		position: absolute;
		right: 8px;
		bottom: 10px;
		font-size: 11px;
		color: var(--text-muted);
	}

	.error-message {
		color: var(--status-danger);
		font-size: 13px;
		margin-bottom: 12px;
	}

	.upload-actions {
		display: flex;
		gap: 8px;
		justify-content: flex-end;
	}

	.btn {
		padding: 8px 16px;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
		border: none;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-primary {
		background: var(--brand-primary);
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--brand-primary-dark);
	}

	.btn-secondary {
		background: var(--bg-secondary);
		color: var(--text-primary);
	}

	.btn-secondary:hover:not(:disabled) {
		background: var(--bg-modifier-hover);
	}

	.search-container {
		padding: 12px 16px;
	}

	.search-input {
		width: 100%;
		padding: 8px 12px;
		background: var(--bg-secondary);
		border: 1px solid var(--bg-modifier-accent);
		border-radius: 4px;
		color: var(--text-primary);
		font-size: 14px;
	}

	.search-input:focus {
		outline: none;
		border-color: var(--brand-primary);
	}

	.error-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		background: rgba(242, 63, 67, 0.1);
		color: var(--status-danger);
		font-size: 13px;
	}

	.error-banner button {
		background: none;
		border: none;
		color: var(--status-danger);
		text-decoration: underline;
		cursor: pointer;
	}

	.loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		color: var(--text-muted);
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--bg-modifier-accent);
		border-top-color: var(--brand-primary);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
		margin-bottom: 8px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.empty-state {
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

	.empty-state h3 {
		font-size: 16px;
		font-weight: 600;
		margin: 0 0 8px 0;
	}

	.empty-state p {
		color: var(--text-muted);
		font-size: 14px;
		margin: 0;
	}

	.stickers-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
		gap: 12px;
		padding: 16px;
		overflow-y: auto;
	}

	.sticker-card {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 8px;
		background: var(--bg-secondary);
		border-radius: 8px;
		transition: background-color 0.15s;
	}

	.sticker-card:hover {
		background: var(--bg-modifier-hover);
	}

	.sticker-preview {
		width: 64px;
		height: 64px;
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 4px;
	}

	.sticker-preview img {
		max-width: 100%;
		max-height: 100%;
		object-fit: contain;
	}

	.sticker-info {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		width: 100%;
	}

	.sticker-name {
		font-size: 11px;
		font-weight: 500;
		color: var(--text-primary);
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		text-align: center;
	}

	.sticker-format {
		font-size: 9px;
		color: var(--text-muted);
		text-transform: uppercase;
	}

	.delete-btn {
		position: absolute;
		top: 4px;
		right: 4px;
		width: 20px;
		height: 20px;
		border-radius: 4px;
		background: transparent;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		opacity: 0;
		transition: opacity 0.15s, background-color 0.15s;
	}

	.sticker-card:hover .delete-btn {
		opacity: 1;
	}

	.delete-btn:hover {
		background: var(--status-danger);
		color: white;
	}

	.delete-confirm {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 10px;
	}

	.delete-confirm span {
		color: var(--text-muted);
	}

	.btn-icon {
		padding: 2px 6px;
		border-radius: 4px;
		font-size: 10px;
		background: var(--bg-secondary);
		border: none;
		color: var(--text-secondary);
		cursor: pointer;
	}

	.btn-icon:hover {
		background: var(--bg-modifier-hover);
	}

	.btn-icon.danger {
		background: var(--status-danger);
		color: white;
	}

	.btn-icon.danger:hover {
		background: #d73438;
	}
</style>
