<script lang="ts">
	import { sendGameMove } from '$lib/stores/voiceActivity';
	import { user as authUser } from '$lib/stores/auth';
	import type { VoiceActivityParticipant, WatchTogetherState } from '$lib/stores/voiceActivity';

	export let activityId: string;
	export let state: unknown;
	export let participants: VoiceActivityParticipant[];

	$: watchState = state as WatchTogetherState | null;
	$: hasVideo = !!watchState?.video_url;

	let videoUrlInput = '';
	let videoTitleInput = '';
	let queueUrlInput = '';
	let queueTitleInput = '';

	function getPlayerName(userId: string | undefined): string {
		if (!userId) return 'Unknown';
		const p = participants.find(p => p.user_id === userId);
		return p?.display_name || p?.username || 'Unknown';
	}

	async function setVideo() {
		if (!videoUrlInput) return;
		await sendGameMove(activityId, 'set_video', {
			url: videoUrlInput,
			title: videoTitleInput || videoUrlInput
		});
		videoUrlInput = '';
		videoTitleInput = '';
	}

	async function togglePlayback() {
		if (!watchState) return;
		await sendGameMove(activityId, watchState.is_playing ? 'pause' : 'play', {});
	}

	async function seek(time: number) {
		await sendGameMove(activityId, 'seek', { time });
	}

	async function addToQueue() {
		if (!queueUrlInput) return;
		await sendGameMove(activityId, 'queue_add', {
			url: queueUrlInput,
			title: queueTitleInput || queueUrlInput
		});
		queueUrlInput = '';
		queueTitleInput = '';
	}

	async function setPlaybackRate(rate: number) {
		await sendGameMove(activityId, 'set_rate', { rate });
	}

	function formatTime(seconds: number): string {
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}

	// Extract YouTube video ID for embed
	function getYouTubeEmbedUrl(url: string): string | null {
		const match = url.match(/(?:youtube\.com\/(?:watch\?v=|embed\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})/);
		if (match) return `https://www.youtube-nocookie.com/embed/${match[1]}?autoplay=0&enablejsapi=1`;
		return null;
	}
</script>

<div class="watch-together">
	{#if watchState}
		<!-- Video Player Area -->
		<div class="video-area">
			{#if hasVideo}
				{@const embedUrl = getYouTubeEmbedUrl(watchState.video_url)}
				{#if embedUrl}
					<div class="video-embed">
						<iframe
							src={embedUrl}
							title={watchState.video_title || 'Watch Together'}
							frameborder="0"
							allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
							allowfullscreen
							sandbox="allow-scripts allow-same-origin allow-popups"
						></iframe>
					</div>
				{:else}
					<div class="video-link">
						<p>Playing: <a href={watchState.video_url} target="_blank" rel="noopener">{watchState.video_title || watchState.video_url}</a></p>
					</div>
				{/if}

				<!-- Playback Controls -->
				<div class="playback-controls">
					<button class="btn-playback" on:click={togglePlayback}>
						{watchState.is_playing ? '⏸ Pause' : '▶ Play'}
					</button>
					<span class="time-display">{formatTime(watchState.current_time)}</span>
					<input
						type="range"
						class="seek-bar"
						min="0"
						max="3600"
						value={watchState.current_time}
						on:change={(e) => seek(Number(e.currentTarget.value))}
					/>
					<div class="rate-controls">
						{#each [0.5, 1, 1.5, 2] as rate}
							<button
								class="btn-rate"
								class:active={watchState.playback_rate === rate}
								on:click={() => setPlaybackRate(rate)}
							>{rate}x</button>
						{/each}
					</div>
				</div>

				{#if watchState.updated_by}
					<div class="sync-info">
						Last synced by {getPlayerName(watchState.updated_by)}
					</div>
				{/if}
			{:else}
				<div class="no-video">
					<span class="no-video-icon">📺</span>
					<p>No video selected. Add a URL to start watching!</p>
				</div>
			{/if}
		</div>

		<!-- Set Video -->
		<div class="video-input-section">
			<h4>Set Video</h4>
			<div class="input-row">
				<input
					type="text"
					bind:value={videoUrlInput}
					placeholder="Paste a YouTube URL..."
					class="text-input url-input"
				/>
				<input
					type="text"
					bind:value={videoTitleInput}
					placeholder="Title (optional)"
					class="text-input title-input"
				/>
				<button class="btn-set" on:click={setVideo} disabled={!videoUrlInput}>Set</button>
			</div>
		</div>

		<!-- Queue -->
		<div class="queue-section">
			<h4>Queue ({watchState.queue?.length ?? 0})</h4>
			{#if watchState.queue && watchState.queue.length > 0}
				<div class="queue-list">
					{#each watchState.queue as item, i}
						<div class="queue-item">
							<span class="queue-index">{i + 1}.</span>
							<span class="queue-title">{item.title || item.url}</span>
							<span class="queue-added">Added by {getPlayerName(item.added_by)}</span>
						</div>
					{/each}
				</div>
			{/if}
			<div class="input-row">
				<input
					type="text"
					bind:value={queueUrlInput}
					placeholder="Add to queue..."
					class="text-input url-input"
				/>
				<input
					type="text"
					bind:value={queueTitleInput}
					placeholder="Title"
					class="text-input title-input"
				/>
				<button class="btn-add" on:click={addToQueue} disabled={!queueUrlInput}>Add</button>
			</div>
		</div>
	{:else}
		<div class="loading-state">Loading Watch Together...</div>
	{/if}
</div>

<style>
	.watch-together {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.video-area {
		background: #000;
		border-radius: 8px;
		overflow: hidden;
	}

	.video-embed {
		position: relative;
		padding-bottom: 56.25%;
		height: 0;
	}

	.video-embed iframe {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
	}

	.video-link {
		padding: 16px;
		text-align: center;
	}

	.video-link a {
		color: #00aff4;
		text-decoration: none;
	}

	.video-link a:hover { text-decoration: underline; }

	.no-video {
		padding: 48px 16px;
		text-align: center;
		color: var(--text-muted, #6d6f78);
	}

	.no-video-icon { font-size: 48px; display: block; margin-bottom: 8px; }

	.playback-controls {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		background: rgba(0, 0, 0, 0.8);
	}

	.btn-playback {
		padding: 6px 16px;
		background: var(--brand-primary, #5865f2);
		color: white;
		border: none;
		border-radius: 4px;
		font-weight: 600;
		cursor: pointer;
		font-size: 13px;
		white-space: nowrap;
	}

	.btn-playback:hover { background: #4752c4; }

	.time-display {
		color: white;
		font-size: 12px;
		font-variant-numeric: tabular-nums;
		min-width: 40px;
	}

	.seek-bar {
		flex: 1;
		height: 4px;
		accent-color: var(--brand-primary, #5865f2);
	}

	.rate-controls {
		display: flex;
		gap: 2px;
	}

	.btn-rate {
		padding: 2px 8px;
		background: rgba(255, 255, 255, 0.1);
		border: none;
		border-radius: 3px;
		color: rgba(255, 255, 255, 0.7);
		font-size: 11px;
		cursor: pointer;
	}

	.btn-rate.active {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.sync-info {
		padding: 4px 16px 8px;
		font-size: 11px;
		color: rgba(255, 255, 255, 0.4);
		background: rgba(0, 0, 0, 0.8);
	}

	.video-input-section, .queue-section {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		padding: 12px 16px;
	}

	h4 {
		font-size: 11px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-muted, #6d6f78);
		margin-bottom: 8px;
		letter-spacing: 0.02em;
	}

	.input-row {
		display: flex;
		gap: 8px;
	}

	.text-input {
		padding: 8px 12px;
		background: var(--bg-tertiary, #1e1f22);
		border: 1px solid var(--border-subtle, #1e1f22);
		border-radius: 4px;
		color: var(--text-primary, #f2f3f5);
		font-size: 13px;
	}

	.text-input:focus {
		border-color: var(--brand-primary, #5865f2);
		outline: none;
	}

	.url-input { flex: 2; }
	.title-input { flex: 1; }

	.btn-set, .btn-add {
		padding: 8px 16px;
		border: none;
		border-radius: 4px;
		font-weight: 600;
		cursor: pointer;
		font-size: 13px;
		white-space: nowrap;
	}

	.btn-set {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.btn-add {
		background: #43b581;
		color: white;
	}

	.btn-set:disabled, .btn-add:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.queue-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
		margin-bottom: 8px;
	}

	.queue-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
		background: var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
	}

	.queue-index { color: var(--text-muted, #6d6f78); font-size: 12px; min-width: 20px; }
	.queue-title { flex: 1; font-size: 13px; color: var(--text-primary, #f2f3f5); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.queue-added { font-size: 11px; color: var(--text-muted, #6d6f78); white-space: nowrap; }

	.loading-state {
		text-align: center;
		color: var(--text-muted, #6d6f78);
		padding: 40px;
	}
</style>
