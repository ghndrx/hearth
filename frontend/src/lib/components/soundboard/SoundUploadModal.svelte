<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { api } from '$lib/api';
	import { currentServer } from '$lib/stores/servers';
	import { invalidateAll } from '$app/navigation';

	export let show = false;

	const dispatch = createEventDispatcher<{ close: void; uploaded: void }>();

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

	let uploadName = '';
	let uploadEmoji = '';
	let uploadVolume = 1.0;
	let uploadFile: File | null = null;
	let uploadError = '';
	let uploading = false;
	let audioPreview: HTMLAudioElement | null = null;
	let isPlayingPreview = false;

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
			const allowedTypes = ['audio/mpeg', 'audio/ogg', 'audio/wav', 'audio/x-wav', 'audio/opus', 'audio/webm'];
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

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		const files = event.dataTransfer?.files;
		if (files && files.length > 0) {
			const file = files[0];
			// Create a fake event to reuse the handler
			const fakeInput = { target: { files: [file] } } as unknown as Event;
			handleFileSelect(fakeInput);
		}
	}

	function handleDragOver(event: DragEvent) {
		event.preventDefault();
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

			await api.post<Sound>(`/servers/${$currentServer.id}/soundboard`, formData);

			// Reset form
			uploadName = '';
			uploadEmoji = '';
			uploadVolume = 1.0;
			uploadFile = null;

			// Clear file input
			const fileInput = document.getElementById('sound-upload-file-input') as HTMLInputElement;
			if (fileInput) fileInput.value = '';

			dispatch('uploaded');
			dispatch('close');
		} catch (err) {
			console.error('Failed to upload sound:', err);
			uploadError = 'Failed to upload sound. Please try again.';
		} finally {
			uploading = false;
		}
	}

	function previewSound() {
		if (!uploadFile) return;

		// Stop any existing preview
		stopPreview();

		// Create preview
		const url = URL.createObjectURL(uploadFile);
		audioPreview = new Audio(url);
		audioPreview.volume = uploadVolume;
		audioPreview.play().then(() => {
			isPlayingPreview = true;
		}).catch(err => {
			console.error('Failed to play preview:', err);
			uploadError = 'Failed to preview audio';
		});

		audioPreview.onended = () => {
			isPlayingPreview = false;
		};

		audioPreview.onerror = () => {
			isPlayingPreview = false;
			uploadError = 'Failed to load audio preview';
		};
	}

	function stopPreview() {
		if (audioPreview) {
			audioPreview.pause();
			audioPreview.currentTime = 0;
			URL.revokeObjectURL(audioPreview.src);
			audioPreview = null;
			isPlayingPreview = false;
		}
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes}B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			stopPreview();
			dispatch('close');
		}
	}

	$: if (!show) {
		stopPreview();
		uploadError = '';
	}
</script>

{#if show}
	<div class="modal-backdrop" on:click={() => dispatch('close')} on:keydown={handleKeydown} role="presentation">
		<div class="modal" on:click|stopPropagation role="dialog" aria-label="Upload sound" aria-modal="true">
			<div class="modal-header">
				<h2>Upload Sound</h2>
				<button class="close-btn" on:click={() => dispatch('close')} aria-label="Close">
					<svg viewBox="0 0 24 24" width="20" height="20">
						<path fill="currentColor" d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
					</svg>
				</button>
			</div>

			<div class="modal-content">
				<!-- Upload Form -->
				<div class="upload-form">
					<div class="form-row">
						<label for="sound-upload-name">Name</label>
						<input
							id="sound-upload-name"
							type="text"
							bind:value={uploadName}
							placeholder="Sound name"
							maxlength="100"
						/>
					</div>

					<div class="form-row">
						<label for="sound-upload-emoji">Emoji (optional)</label>
						<input
							id="sound-upload-emoji"
							type="text"
							bind:value={uploadEmoji}
							placeholder="🔊"
							maxlength="100"
							class="emoji-input"
						/>
					</div>

					<div class="form-row">
						<label for="sound-upload-volume">Volume: {uploadVolume.toFixed(1)}</label>
						<input
							id="sound-upload-volume"
							type="range"
							bind:value={uploadVolume}
							min="0.1"
							max="1.0"
							step="0.1"
						/>
					</div>

					<div class="form-row drop-zone"
						class:has-file={uploadFile}
						on:drop={handleDrop}
						on:dragover={handleDragOver}
						role="button"
						tabindex="0"
					>
						<input
							id="sound-upload-file-input"
							type="file"
							accept="audio/mpeg,audio/ogg,audio/wav,audio/x-wav,audio/opus,audio/webm"
							on:change={handleFileSelect}
						/>
						{#if uploadFile}
							<div class="file-info">
								<svg viewBox="0 0 24 24" width="24" height="24" class="file-icon">
									<path fill="currentColor" d="M12 3c-4.97 0-9 4.03-9 9v7c0 1.1.9 2 2 2h4v-8H5v-1c0-3.87 3.13-7 7-7s7 3.13 7 7v1h-4v8h4c1.1 0 2-.9 2-2v-7c0-4.97-4.03-9-9-9z"/>
								</svg>
								<span class="file-name">{uploadFile.name}</span>
								<span class="file-size">({formatFileSize(uploadFile.size)})</span>
							</div>
						{:else}
							<div class="drop-hint">
								<svg viewBox="0 0 24 24" width="32" height="32" class="upload-icon">
									<path fill="currentColor" d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z"/>
								</svg>
								<span>Drop audio file here or click to browse</span>
								<span class="file-constraints">MP3, OGG, WAV, OPUS • Max 500KB</span>
							</div>
						{/if}
					</div>

					{#if uploadError}
						<div class="error-message">{uploadError}</div>
					{/if}

					<div class="button-row">
						{#if uploadFile && !isPlayingPreview}
							<button class="preview-btn" on:click={previewSound} type="button">
								Preview
							</button>
						{:else if isPlayingPreview}
							<button class="preview-btn playing" on:click={stopPreview} type="button">
								Stop Preview
							</button>
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
		max-width: 480px;
		max-height: 90vh;
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
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.close-btn:hover {
		color: var(--text-primary, #f2f3f5);
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.modal-content {
		padding: 20px;
		overflow-y: auto;
	}

	.upload-form {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.form-row {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.form-row label {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-secondary, #b5bac1);
	}

	.form-row input[type="text"] {
		padding: 10px 12px;
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: 4px;
		color: var(--text-primary, #f2f3f5);
		font-size: 14px;
		outline: none;
	}

	.form-row input[type="text"]:focus {
		box-shadow: 0 0 0 2px var(--brand-primary, #5865f2);
	}

	.form-row input[type="text"]::placeholder {
		color: var(--text-muted, #949ba4);
	}

	.emoji-input {
		width: 80px;
		text-align: center;
		font-size: 18px;
	}

	.form-row input[type="range"] {
		width: 100%;
		cursor: pointer;
	}

	.drop-zone {
		position: relative;
		border: 2px dashed var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
		border-radius: 8px;
		padding: 24px;
		text-align: center;
		cursor: pointer;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.drop-zone:hover {
		border-color: var(--brand-primary, #5865f2);
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.16));
	}

	.drop-zone.has-file {
		border-color: var(--success, #23a559);
		background: rgba(35, 165, 89, 0.1);
	}

	.drop-zone input[type="file"] {
		position: absolute;
		inset: 0;
		opacity: 0;
		cursor: pointer;
	}

	.drop-hint {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		color: var(--text-muted, #949ba4);
		font-size: 14px;
		pointer-events: none;
	}

	.upload-icon {
		color: var(--text-muted, #949ba4);
	}

	.file-constraints {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.file-info {
		display: flex;
		align-items: center;
		gap: 12px;
		color: var(--text-primary, #f2f3f5);
		pointer-events: none;
	}

	.file-icon {
		color: var(--success, #23a559);
		flex-shrink: 0;
	}

	.file-name {
		font-size: 14px;
		font-weight: 500;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.file-size {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.error-message {
		color: var(--danger, #ed4245);
		font-size: 13px;
		padding: 8px 12px;
		background: rgba(237, 66, 69, 0.1);
		border-radius: 4px;
	}

	.button-row {
		display: flex;
		gap: 8px;
		justify-content: flex-end;
	}

	.preview-btn {
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: 4px;
		padding: 10px 16px;
		color: var(--text-primary, #f2f3f5);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.preview-btn:hover {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
	}

	.preview-btn.playing {
		background: var(--brand-primary, #5865f2);
	}

	.preview-btn.playing:hover {
		background: var(--brand-hover, #4752c4);
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
		transition: background-color 0.15s;
	}

	.upload-btn:hover:not(:disabled) {
		background: var(--brand-hover, #4752c4);
	}

	.upload-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
