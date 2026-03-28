<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { api } from '$lib/api';
	import { currentServer } from '$lib/stores/servers';

	export let show = false;

	const dispatch = createEventDispatcher<{ close: void }>();

	interface Sound {
		id: string;
		name: string;
		emoji_name?: string;
		volume: number;
		audio_url: string;
		duration_ms: number;
		available: boolean;
		creator_id: string;
		created_at: string;
	}

	let sounds: Sound[] = [];
	let loading = false;
	let error = '';
	let uploading = false;
	let uploadError = '';

	// Upload form
	let uploadName = '';
	let uploadEmoji = '';
	let uploadVolume = 1.0;
	let uploadFile: File | null = null;

	// Edit state
	let editingSound: Sound | null = null;
	let editName = '';
	let editEmoji = '';
	let editVolume = 1.0;
	let editAvailable = true;

	async function fetchSounds() {
		if (!$currentServer) return;
		
		loading = true;
		error = '';

		try {
			const response = await api.get<Sound[]>(`/servers/${$currentServer.id}/soundboard`);
			sounds = response || [];
		} catch (err) {
			console.error('Failed to fetch sounds:', err);
			error = 'Failed to load sounds';
		} finally {
			loading = false;
		}
	}

	function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files && input.files.length > 0) {
			const file = input.files[0];
			
			// Validate file size (500KB max)
			if (file.size > 500 * 1024) {
				uploadError = 'File must be under 500KB';
				uploadFile = null;
				return;
			}

			// Validate file type
			const allowedTypes = ['audio/mpeg', 'audio/ogg', 'audio/wav', 'audio/x-wav', 'audio/opus'];
			if (!allowedTypes.includes(file.type)) {
				uploadError = 'Invalid file type. Use MP3, OGG, WAV, or OPUS';
				uploadFile = null;
				return;
			}

			uploadFile = file;
			uploadError = '';

			// Auto-fill name from filename if empty
			if (!uploadName) {
				uploadName = file.name.replace(/\.[^/.]+$/, '').replace(/[-_]/g, ' ');
			}
		}
	}

	async function handleUpload() {
		if (!$currentServer || !uploadFile || !uploadName) return;

		uploading = true;
		uploadError = '';

		try {
			const formData = new FormData();
			formData.append('name', uploadName);
			formData.append('audio', uploadFile);
			if (uploadEmoji) {
				formData.append('emoji_name', uploadEmoji);
			}
			formData.append('volume', uploadVolume.toString());

			const response = await api.post<Sound>(`/servers/${$currentServer.id}/soundboard`, formData);

			sounds = [...sounds, response];
			
			// Reset form
			uploadName = '';
			uploadEmoji = '';
			uploadVolume = 1.0;
			uploadFile = null;
			
			// Clear file input
			const fileInput = document.getElementById('sound-file-input') as HTMLInputElement;
			if (fileInput) fileInput.value = '';
		} catch (err) {
			console.error('Failed to upload sound:', err);
			uploadError = 'Failed to upload sound. Please try again.';
		} finally {
			uploading = false;
		}
	}

	function startEdit(sound: Sound) {
		editingSound = sound;
		editName = sound.name;
		editEmoji = sound.emoji_name || '';
		editVolume = sound.volume;
		editAvailable = sound.available;
	}

	async function saveEdit() {
		if (!editingSound || !$currentServer) return;

		try {
			const response = await api.patch<Sound>(
				`/servers/${$currentServer.id}/soundboard/${editingSound.id}`,
				{
					name: editName,
					emoji_name: editEmoji,
					volume: editVolume,
					available: editAvailable
				}
			);

			sounds = sounds.map(s => s.id === response.id ? response : s);
			cancelEdit();
		} catch (err) {
			console.error('Failed to update sound:', err);
		}
	}

	function cancelEdit() {
		editingSound = null;
		editName = '';
		editEmoji = '';
		editVolume = 1.0;
		editAvailable = true;
	}

	async function deleteSound(sound: Sound) {
		if (!$currentServer) return;
		
		if (!confirm(`Delete "${sound.name}"? This cannot be undone.`)) return;

		try {
			await api.delete(`/servers/${$currentServer.id}/soundboard/${sound.id}`);
			sounds = sounds.filter(s => s.id !== sound.id);
		} catch (err) {
			console.error('Failed to delete sound:', err);
		}
	}

	function formatDuration(ms: number): string {
		const seconds = Math.floor(ms / 1000);
		return `${seconds}s`;
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes}B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
	}

	$: if (show && $currentServer) {
		fetchSounds();
	}
</script>

{#if show}
	<div class="modal-backdrop" on:click={() => dispatch('close')} role="presentation">
		<div class="modal" on:click|stopPropagation role="dialog" aria-label="Soundboard Manager">
			<div class="modal-header">
				<h2>Soundboard</h2>
				<button class="close-btn" on:click={() => dispatch('close')} aria-label="Close">
					<svg viewBox="0 0 24 24" width="20" height="20">
						<path fill="currentColor" d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
					</svg>
				</button>
			</div>

			<div class="modal-content">
				<!-- Upload Section -->
				<div class="upload-section">
					<h3>Add Sound</h3>
					<div class="upload-form">
						<div class="form-row">
							<label for="sound-name">Name</label>
							<input
								id="sound-name"
								type="text"
								bind:value={uploadName}
								placeholder="Sound name"
								maxlength="100"
							/>
						</div>
						<div class="form-row">
							<label for="sound-emoji">Emoji</label>
							<input
								id="sound-emoji"
								type="text"
								bind:value={uploadEmoji}
								placeholder="🔊"
								maxlength="100"
							/>
						</div>
						<div class="form-row">
							<label for="sound-volume">Volume: {uploadVolume.toFixed(1)}</label>
							<input
								id="sound-volume"
								type="range"
								bind:value={uploadVolume}
								min="0.1"
								max="1.0"
								step="0.1"
							/>
						</div>
						<div class="form-row">
							<label for="sound-file-input">Audio File</label>
							<input
								id="sound-file-input"
								type="file"
								accept="audio/mpeg,audio/ogg,audio/wav,audio/x-wav,audio/opus"
								on:change={handleFileSelect}
							/>
							{#if uploadFile}
								<span class="file-info">{uploadFile.name} ({formatFileSize(uploadFile.size)})</span>
							{/if}
						</div>
						{#if uploadError}
							<div class="error-message">{uploadError}</div>
						{/if}
						<button 
							class="upload-btn" 
							on:click={handleUpload}
							disabled={uploading || !uploadFile || !uploadName}
						>
							{uploading ? 'Uploading...' : 'Upload Sound'}
						</button>
					</div>
				</div>

				<!-- Sounds List -->
				<div class="sounds-section">
					<h3>Server Sounds ({sounds.length})</h3>
					{#if loading}
						<div class="loading">Loading sounds...</div>
					{:else if error}
						<div class="error-message">{error}</div>
					{:else if sounds.length === 0}
						<div class="empty">No sounds yet. Upload one above!</div>
					{:else}
						<div class="sounds-list">
							{#each sounds as sound}
								<div class="sound-item" class:unavailable={!sound.available}>
									{#if editingSound?.id === sound.id}
										<!-- Edit Mode -->
										<div class="edit-form">
											<input
												type="text"
												bind:value={editName}
												placeholder="Name"
												maxlength="100"
											/>
											<input
												type="text"
												bind:value={editEmoji}
												placeholder="Emoji"
												maxlength="100"
											/>
											<div class="volume-edit">
												<span>Vol:</span>
												<input
													type="range"
													bind:value={editVolume}
													min="0.1"
													max="1.0"
													step="0.1"
												/>
												<span>{editVolume.toFixed(1)}</span>
											</div>
											<label class="available-toggle">
												<input type="checkbox" bind:checked={editAvailable} />
												Available
											</label>
											<div class="edit-actions">
												<button class="save-btn" on:click={saveEdit}>Save</button>
												<button class="cancel-btn" on:click={cancelEdit}>Cancel</button>
											</div>
										</div>
									{:else}
										<!-- View Mode -->
										<div class="sound-info">
											<span class="sound-emoji">{sound.emoji_name || '🔊'}</span>
											<div class="sound-details">
												<span class="sound-name">{sound.name}</span>
												<span class="sound-meta">
													{formatDuration(sound.duration_ms)} • Vol {sound.volume.toFixed(1)}
													{#if !sound.available}
														<span class="unavailable-badge">Disabled</span>
													{/if}
												</span>
											</div>
										</div>
										<div class="sound-actions">
											<button class="edit-btn" on:click={() => startEdit(sound)} aria-label="Edit">
												<svg viewBox="0 0 24 24" width="16" height="16">
													<path fill="currentColor" d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
												</svg>
											</button>
											<button class="delete-btn" on:click={() => deleteSound(sound)} aria-label="Delete">
												<svg viewBox="0 0 24 24" width="16" height="16">
													<path fill="currentColor" d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
												</svg>
											</button>
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		width: 90%;
		max-width: 600px;
		max-height: 80vh;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}

	.modal-header h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-secondary, #b5bac1);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
	}

	.close-btn:hover {
		color: var(--text-primary, #f2f3f5);
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.modal-content {
		flex: 1;
		overflow-y: auto;
		padding: 20px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	h3 {
		margin: 0 0 12px;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.upload-section {
		background: var(--bg-tertiary, #1e1f22);
		border-radius: 8px;
		padding: 16px;
	}

	.upload-form {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.form-row {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.form-row label {
		font-size: 13px;
		color: var(--text-secondary, #b5bac1);
	}

	.form-row input[type="text"],
	.form-row input[type="file"] {
		padding: 8px 12px;
		background: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 4px;
		color: var(--text-primary, #f2f3f5);
		font-size: 14px;
	}

	.form-row input[type="file"] {
		font-size: 13px;
	}

	.form-row input[type="range"] {
		width: 100%;
	}

	.file-info {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.error-message {
		color: var(--danger, #ed4245);
		font-size: 13px;
	}

	.upload-btn {
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: 4px;
		padding: 10px 16px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: opacity 0.15s;
	}

	.upload-btn:hover:not(:disabled) {
		opacity: 0.9;
	}

	.upload-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.sounds-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.sound-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		background: var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		transition: background-color 0.15s;
	}

	.sound-item:hover {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.sound-item.unavailable {
		opacity: 0.6;
	}

	.sound-info {
		display: flex;
		align-items: center;
		gap: 12px;
		flex: 1;
		min-width: 0;
	}

	.sound-emoji {
		font-size: 24px;
	}

	.sound-details {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.sound-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--text-primary, #f2f3f5);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.sound-meta {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.unavailable-badge {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
		padding: 1px 6px;
		border-radius: 3px;
		font-size: 10px;
		margin-left: 4px;
	}

	.sound-actions {
		display: flex;
		gap: 4px;
	}

	.edit-btn,
	.delete-btn {
		background: none;
		border: none;
		padding: 6px;
		border-radius: 4px;
		cursor: pointer;
		color: var(--text-secondary, #b5bac1);
		transition: color 0.15s, background-color 0.15s;
	}

	.edit-btn:hover {
		color: var(--text-primary, #f2f3f5);
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.delete-btn:hover {
		color: var(--danger, #ed4245);
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.edit-form {
		display: flex;
		align-items: center;
		gap: 8px;
		flex: 1;
		flex-wrap: wrap;
	}

	.edit-form input[type="text"] {
		padding: 4px 8px;
		background: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 4px;
		color: var(--text-primary, #f2f3f5);
		font-size: 13px;
	}

	.volume-edit {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--text-secondary, #b5bac1);
	}

	.volume-edit input[type="range"] {
		width: 60px;
	}

	.available-toggle {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--text-secondary, #b5bac1);
		cursor: pointer;
	}

	.edit-actions {
		display: flex;
		gap: 4px;
		margin-left: auto;
	}

	.save-btn,
	.cancel-btn {
		padding: 4px 8px;
		border: none;
		border-radius: 4px;
		font-size: 12px;
		cursor: pointer;
	}

	.save-btn {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.cancel-btn {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
		color: var(--text-secondary, #b5bac1);
	}

	.loading,
	.empty {
		padding: 24px;
		text-align: center;
		color: var(--text-muted, #949ba4);
		font-size: 13px;
	}
</style>
