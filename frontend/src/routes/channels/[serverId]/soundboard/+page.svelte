<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { currentServer } from '$lib/stores/servers';
	import SoundboardPicker from '$lib/components/soundboard/SoundboardPicker.svelte';
	import SoundUploadModal from '$lib/components/soundboard/SoundUploadModal.svelte';
	import SoundboardManager from '$lib/components/soundboard/SoundboardManager.svelte';

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
		guild_id?: string;
	}

	let defaultSounds: Sound[] = [];
	let serverSounds: Sound[] = [];
	let loading = false;
	let error = '';
	let showUploadModal = false;
	let showManager = false;
	let playingSoundId: string | null = null;
	let audioPreview: HTMLAudioElement | null = null;

	$: serverId = $page.params.serverId;

	async function fetchSounds() {
		loading = true;
		error = '';

		try {
			// Get default/global sounds
			const defaultResponse = await api.get<Sound[]>('/soundboard/defaults');
			defaultSounds = defaultResponse || [];

			// Get server sounds
			const serverResponse = await api.get<Sound[]>(`/servers/${serverId}/soundboard`);
			serverSounds = serverResponse || [];
		} catch (err) {
			console.error('Failed to fetch sounds:', err);
			error = 'Failed to load sounds';
		} finally {
			loading = false;
		}
	}

	function playSound(sound: Sound) {
		// Stop any existing preview
		stopPreview();

		// Create preview
		audioPreview = new Audio(sound.audio_url);
		audioPreview.volume = sound.volume;
		audioPreview.play().catch(err => {
			console.error('Failed to play sound:', err);
		});

		playingSoundId = sound.id;

		// Auto-stop after duration
		setTimeout(() => {
			if (playingSoundId === sound.id) {
				stopPreview();
			}
		}, sound.duration_ms);
	}

	function stopPreview() {
		if (audioPreview) {
			audioPreview.pause();
			audioPreview = null;
		}
		playingSoundId = null;
	}

	function formatDuration(ms: number): string {
		const seconds = Math.floor(ms / 1000);
		return `${seconds}s`;
	}

	function handleKeydown(event: KeyboardEvent) {
		// Number keys 1-9 to play sounds
		const num = parseInt(event.key);
		if (num >= 1 && num <= 9) {
			const allSounds = [...defaultSounds, ...serverSounds];
			if (allSounds[num - 1]) {
				playSound(allSounds[num - 1]);
			}
		} else if (event.key === 'Escape') {
			stopPreview();
		}
	}

	onMount(() => {
		fetchSounds();
		document.addEventListener('keydown', handleKeydown);
		return () => {
			document.removeEventListener('keydown', handleKeydown);
			stopPreview();
		};
	});

	$: allSounds = [...defaultSounds, ...serverSounds];
</script>

<svelte:head>
	<title>Soundboard | {$currentServer?.name || 'Hearth'}</title>
</svelte:head>

<div class="soundboard-page">
	<div class="header">
		<div class="header-info">
			<h1>Soundboard</h1>
			<p class="subtitle">Play sounds or create your own for this server</p>
		</div>
		<div class="header-actions">
			<button class="manage-btn" on:click={() => showManager = !showManager}>
				{showManager ? 'Hide' : 'Manage'} Sounds
			</button>
			<button class="upload-btn" on:click={() => showUploadModal = true}>
				Upload Sound
			</button>
		</div>
	</div>

	{#if loading}
		<div class="loading">
			<div class="spinner"></div>
			<span>Loading sounds...</span>
		</div>
	{:else if error}
		<div class="error">
			<span>{error}</span>
			<button class="retry-btn" on:click={fetchSounds}>Retry</button>
		</div>
	{:else}
		<div class="content">
			{#if allSounds.length === 0}
				<div class="empty">
					<svg viewBox="0 0 24 24" width="64" height="64" class="empty-icon">
						<path fill="currentColor" d="M12 3c-4.97 0-9 4.03-9 9v7c0 1.1.9 2 2 2h4v-8H5v-1c0-3.87 3.13-7 7-7s7 3.13 7 7v1h-4v8h4c1.1 0 2-.9 2-2v-7c0-4.97-4.03-9-9-9z"/>
					</svg>
					<h2>No sounds yet</h2>
					<p>Upload a sound to get started!</p>
					<button class="upload-btn" on:click={() => showUploadModal = true}>
						Upload Your First Sound
					</button>
				</div>
			{:else}
				<!-- Sound Grid -->
				<div class="sound-section">
					<h2>All Sounds ({allSounds.length})</h2>
					<p class="section-hint">Press 1-9 to play sounds</p>
					<div class="sound-grid">
						{#each allSounds as sound, index}
							<button
								class="sound-card"
								class:playing={playingSoundId === sound.id}
								on:click={() => playSound(sound)}
								disabled={!sound.available}
							>
								{#if index < 9}
									<span class="hotkey-badge">{index + 1}</span>
								{/if}
								<span class="sound-emoji" aria-hidden="true">
									{sound.emoji_name || '🔊'}
								</span>
								<span class="sound-name">{sound.name}</span>
								<span class="sound-duration">{formatDuration(sound.duration_ms)}</span>
								{#if sound.guild_id}
									<span class="sound-badge server">Server</span>
								{:else}
									<span class="sound-badge default">Default</span>
								{/if}
								{#if playingSoundId === sound.id}
									<span class="playing-indicator">
										<span class="bar"></span>
										<span class="bar"></span>
										<span class="bar"></span>
									</span>
								{/if}
							</button>
						{/each}
					</div>
				</div>

				{#if serverSounds.length > 0}
					<div class="sound-section">
						<h2>Server Sounds ({serverSounds.length})</h2>
						<div class="sound-grid">
							{#each serverSounds as sound, index}
								<button
									class="sound-card"
									class:playing={playingSoundId === sound.id}
									on:click={() => playSound(sound)}
									disabled={!sound.available}
								>
									{#if index < 9}
										<span class="hotkey-badge">{index + 1}</span>
									{/if}
									<span class="sound-emoji" aria-hidden="true">
										{sound.emoji_name || '🔊'}
									</span>
									<span class="sound-name">{sound.name}</span>
									<span class="sound-duration">{formatDuration(sound.duration_ms)}</span>
									{#if playingSoundId === sound.id}
										<span class="playing-indicator">
											<span class="bar"></span>
											<span class="bar"></span>
											<span class="bar"></span>
										</span>
									{/if}
								</button>
							{/each}
						</div>
					</div>
				{/if}
			{/if}
		</div>
	{/if}

	<!-- Upload Modal -->
	<SoundUploadModal
		show={showUploadModal}
		on:close={() => showUploadModal = false}
		on:uploaded={fetchSounds}
	/>

	<!-- Manager Modal -->
	{#if showManager}
		<SoundboardManager show={true} on:close={() => showManager = false} />
	{/if}
</div>

<style>
	.soundboard-page {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--bg-primary, #313338);
	}

	.header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 24px;
		background: var(--bg-secondary, #2b2d31);
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}

	.header-info h1 {
		margin: 0;
		font-size: 20px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.subtitle {
		margin: 4px 0 0;
		font-size: 13px;
		color: var(--text-secondary, #b5bac1);
	}

	.header-actions {
		display: flex;
		gap: 8px;
	}

	.manage-btn {
		background: var(--bg-tertiary, #1e1f22);
		border: none;
		border-radius: 4px;
		padding: 8px 12px;
		color: var(--text-secondary, #b5bac1);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s, color 0.15s;
	}

	.manage-btn:hover {
		color: var(--text-primary, #f2f3f5);
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
	}

	.upload-btn {
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: 4px;
		padding: 8px 12px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.upload-btn:hover {
		background: var(--brand-hover, #4752c4);
	}

	.loading,
	.error {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		color: var(--text-muted, #949ba4);
		gap: 12px;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--bg-tertiary, #1e1f22);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.retry-btn {
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: 4px;
		padding: 8px 16px;
		color: white;
		font-size: 14px;
		cursor: pointer;
	}

	.content {
		flex: 1;
		overflow-y: auto;
		padding: 24px;
	}

	.empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 64px 24px;
		text-align: center;
	}

	.empty-icon {
		color: var(--text-muted, #949ba4);
		margin-bottom: 16px;
	}

	.empty h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.empty p {
		margin: 8px 0 24px;
		font-size: 14px;
		color: var(--text-secondary, #b5bac1);
	}

	.sound-section {
		margin-bottom: 32px;
	}

	.sound-section h2 {
		margin: 0 0 8px;
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.section-hint {
		margin: 0 0 16px;
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.sound-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
		gap: 8px;
	}

	.sound-card {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 16px 8px;
		background: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: background-color 0.15s, transform 0.1s;
		gap: 8px;
		min-height: 120px;
	}

	.sound-card:hover:not(:disabled) {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
		transform: translateY(-2px);
	}

	.sound-card:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.sound-card.playing {
		background: var(--bg-modifier-selected, rgba(79, 84, 92, 0.24));
	}

	.hotkey-badge {
		position: absolute;
		top: 4px;
		left: 4px;
		font-size: 10px;
		font-weight: 600;
		color: var(--text-muted, #949ba4);
		background: var(--bg-tertiary, #1e1f22);
		padding: 2px 5px;
		border-radius: 3px;
		line-height: 1.2;
	}

	.sound-emoji {
		font-size: 32px;
		line-height: 1;
	}

	.sound-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary, #f2f3f5);
		text-align: center;
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.sound-duration {
		font-size: 11px;
		color: var(--text-muted, #949ba4);
	}

	.sound-badge {
		position: absolute;
		top: 4px;
		right: 4px;
		font-size: 9px;
		font-weight: 600;
		padding: 2px 5px;
		border-radius: 3px;
		line-height: 1.2;
	}

	.sound-badge.default {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.sound-badge.server {
		background: var(--success, #23a559);
		color: white;
	}

	.playing-indicator {
		position: absolute;
		bottom: 8px;
		display: flex;
		gap: 2px;
		align-items: flex-end;
		height: 12px;
	}

	.playing-indicator .bar {
		width: 3px;
		background: var(--brand-primary, #5865f2);
		animation: soundbar 0.5s ease-in-out infinite alternate;
	}

	.playing-indicator .bar:nth-child(1) {
		height: 6px;
		animation-delay: 0s;
	}

	.playing-indicator .bar:nth-child(2) {
		height: 12px;
		animation-delay: 0.15s;
	}

	.playing-indicator .bar:nth-child(3) {
		height: 8px;
		animation-delay: 0.3s;
	}

	@keyframes soundbar {
		from { transform: scaleY(0.3); }
		to { transform: scaleY(1); }
	}
</style>
