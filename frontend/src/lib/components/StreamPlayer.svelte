<script lang="ts">
	import { onMount, onDestroy, createEventDispatcher } from 'svelte';
	import { streamState, streamActions, activeStreamInChannel, isViewing, type Stream, type StreamQuality } from '$lib/stores/stream';
	import { user as authUser } from '$lib/stores/auth';
	import { getLiveKitManager } from '$lib/voice/livekit';

	const dispatch = createEventDispatcher<{
		close: void;
	}>();

	export let stream: Stream;
	export let channelId: string;

	let videoElement: HTMLVideoElement | null = null;
	let isFullscreen = false;
	let isMinimized = false;
	let selectedQuality: StreamQuality | 'auto' = 'auto';
	let showControls = true;
	let controlsTimeout: ReturnType<typeof setTimeout> | null = null;
	let streamStartTime: Date;
	let elapsedTime = '00:00';

	type QualityOption = {
		value: StreamQuality | 'auto';
		label: string;
		bitrate?: string;
	};

	const qualityOptions: QualityOption[] = [
		{ value: 'auto', label: 'Auto' },
		{ value: 1, label: '480p', bitrate: '1.5 Mbps' },
		{ value: 2, label: '720p', bitrate: '3 Mbps' },
		{ value: 3, label: '1080p', bitrate: '6 Mbps' }
	];

	$: streamerName = stream.streamer?.display_name || stream.streamer?.username || 'Unknown';
	$: streamerAvatar = stream.streamer?.avatar;
	$: streamTypeLabel = getStreamTypeLabel(stream.type);
	$: qualityLabel = qualityOptions.find(q => q.value === selectedQuality)?.label || 'Auto';
	$: isOwnStream = stream.streamer_id === $authUser?.id;
	$: canWatch = !isOwnStream && stream.status === 1;

	function getStreamTypeLabel(type: number): string {
		switch (type) {
			case 1: return 'Screen';
			case 2: return 'Application';
			case 3: return 'Camera';
			default: return 'Stream';
		}
	}

	function formatElapsedTime(startTime: Date): string {
		const diff = Math.floor((Date.now() - startTime.getTime()) / 1000);
		const hours = Math.floor(diff / 3600);
		const minutes = Math.floor((diff % 3600) / 60);
		const seconds = diff % 60;

		if (hours > 0) {
			return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
		}
		return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
	}

	function startTimer() {
		streamStartTime = new Date(stream.started_at);
		updateTimer();
	}

	function updateTimer() {
		elapsedTime = formatElapsedTime(streamStartTime);
		setTimeout(updateTimer, 1000);
	}

	async function handleJoinStream() {
		try {
			await streamActions.joinStream(stream.id);
		} catch (error) {
			console.error('Failed to join stream:', error);
		}
	}

	async function handleLeaveStream() {
		try {
			await streamActions.leaveStream(stream.id);
		} catch (error) {
			console.error('Failed to leave stream:', error);
		}
	}

	function toggleFullscreen() {
		if (!videoElement) return;

		if (!isFullscreen) {
			if (videoElement.requestFullscreen) {
				videoElement.requestFullscreen();
			}
		} else {
			if (document.exitFullscreen) {
				document.exitFullscreen();
			}
		}
		isFullscreen = !isFullscreen;
	}

	function toggleMinimize() {
		isMinimized = !isMinimized;
	}

	function handleMouseMove() {
		showControls = true;
		if (controlsTimeout) {
			clearTimeout(controlsTimeout);
		}
		controlsTimeout = setTimeout(() => {
			showControls = false;
		}, 3000);
	}

	function handleQualityChange(quality: StreamQuality | 'auto') {
		selectedQuality = quality;
		// In a real implementation, this would signal to the LiveKit SFU
		// to switch quality tiers for this subscriber
	}

	async function handleClose() {
		if ($isViewing) {
			await handleLeaveStream();
		}
		dispatch('close');
	}

	onMount(() => {
		if (canWatch) {
			handleJoinStream();
		}
		startTimer();
	});

	onDestroy(() => {
		if (controlsTimeout) {
			clearTimeout(controlsTimeout);
		}
		if ($isViewing) {
			streamActions.leaveStream(stream.id);
		}
	});
</script>

<div
	class="stream-player"
	class:minimized={isMinimized}
	on:mousemove={handleMouseMove}
	role="region"
	aria-label="Stream player"
>
	<!-- Stream header -->
	<div class="stream-header" class:hidden={!showControls && isMinimized}>
		<div class="streamer-info">
			{#if streamerAvatar}
				<img src={streamerAvatar} alt="" class="streamer-avatar" />
			{:else}
				<div class="streamer-avatar-placeholder">
					{streamerName.charAt(0).toUpperCase()}
				</div>
			{/if}
			<div class="streamer-details">
				<span class="streamer-name">{streamerName}</span>
				<span class="stream-meta">
					<span class="stream-type">{streamTypeLabel}</span>
					<span class="stream-quality">{qualityLabel}</span>
				</span>
			</div>
		</div>

		<div class="stream-info">
			<div class="viewer-count">
				<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
					<path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
				</svg>
				<span>{stream.viewer_count}</span>
			</div>
			<div class="stream-time">
				<div class="live-indicator"></div>
				<span>{elapsedTime}</span>
			</div>
		</div>

		<div class="header-actions">
			<button
				class="action-btn"
				on:click={toggleMinimize}
				aria-label={isMinimized ? 'Maximize' : 'Minimize'}
				title={isMinimized ? 'Maximize' : 'Minimize'}
			>
				{#if isMinimized}
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
						<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
					</svg>
				{:else}
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
						<path d="M19 13H5v-2h14v2z"/>
					</svg>
				{/if}
			</button>

			<button
				class="action-btn"
				on:click={toggleFullscreen}
				aria-label={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
				title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
			>
				{#if isFullscreen}
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
						<path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"/>
					</svg>
				{:else}
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
						<path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/>
					</svg>
				{/if}
			</button>

			<button
				class="action-btn close-btn"
				on:click={handleClose}
				aria-label="Close"
				title="Close"
			>
				<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
					<path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
				</svg>
			</button>
		</div>
	</div>

	<!-- Video area -->
	{#if isMinimized}
		<!-- Minimized view - just show small preview -->
		<div class="minimized-preview" on:click={toggleMinimize} role="button" tabindex="0" on:keypress={(e) => e.key === 'Enter' && toggleMinimize()}>
			<div class="preview-placeholder">
				<svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor">
					<path d="M21 3H3c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H3V5h18v14zM9 8c0 .55-.45 1-1 1s-1-.45-1-1 .45-1 1-1 1 .45 1 1zm4 4.5c0 .28-.22.5-.5.5h-5c-.28 0-.5-.22-.5-.5v-5c0-.28.22-.5.5-.5h5c.28 0 .5.22.5.5v5zm2.5 2c0 .28-.22.5-.5.5h-7c-.28 0-.5-.22-.5-.5v-7c0-.28.22-.5.5-.5h7c.28 0 .5.22.5.5v7z"/>
				</svg>
			</div>
			<div class="preview-info">
				<span class="preview-name">{streamerName}</span>
				<span class="preview-live">LIVE</span>
			</div>
		</div>
	{:else}
		<!-- Full video player -->
		<div class="video-container">
			<!-- In a real implementation, this would be connected to LiveKit or WebRTC -->
			<video
				bind:this={videoElement}
				class="stream-video"
				autoplay
				playsinline
			>
				<track kind="captions" />
			</video>

			<!-- Placeholder when no video track -->
			{#if !stream.type}
				<div class="video-placeholder">
					<svg viewBox="0 0 24 24" width="64" height="64" fill="currentColor">
						<path d="M21 3H3c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H3V5h18v14zM9 8c0 .55-.45 1-1 1s-1-.45-1-1 .45-1 1-1 1 .45 1 1zm4 4.5c0 .28-.22.5-.5.5h-5c-.28 0-.5-.22-.5-.5v-5c0-.28.22-.5.5-.5h5c.28 0 .5.22.5.5v5zm2.5 2c0 .28-.22.5-.5.5h-7c-.28 0-.5-.22-.5-.5v-7c0-.28.22-.5.5-.5h7c.28 0 .5.22.5.5v7z"/>
					</svg>
					<span>{streamTypeLabel} stream</span>
				</div>
			{/if}

			<!-- Controls overlay -->
			<div class="video-controls" class:hidden={!showControls}>
				<!-- Quality selector -->
				<div class="quality-selector">
					<select
						bind:value={selectedQuality}
						on:change={(e) => handleQualityChange(parseInt(e.currentTarget.value) as StreamQuality)}
						aria-label="Video quality"
					>
						{#each qualityOptions as option}
							<option value={option.value}>
								{option.label}{option.bitrate ? ` (${option.bitrate})` : ''}
							</option>
						{/each}
					</select>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.stream-player {
		position: relative;
		display: flex;
		flex-direction: column;
		background: #18191c;
		border-radius: 8px;
		overflow: hidden;
		max-width: 640px;
		max-height: 480px;
	}

	.stream-player.minimized {
		max-width: 280px;
		max-height: 80px;
	}

	.stream-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		background: linear-gradient(to bottom, rgba(0, 0, 0, 0.8), transparent);
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		z-index: 10;
		transition: opacity 0.2s ease;
	}

	.stream-header.hidden {
		opacity: 0;
		pointer-events: none;
	}

	.streamer-info {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.streamer-avatar,
	.streamer-avatar-placeholder {
		width: 32px;
		height: 32px;
		border-radius: 50%;
	}

	.streamer-avatar-placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		background: #5865f2;
		color: white;
		font-size: 14px;
		font-weight: 600;
	}

	.streamer-details {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.streamer-name {
		font-size: 14px;
		font-weight: 600;
		color: white;
	}

	.stream-meta {
		display: flex;
		gap: 8px;
		font-size: 12px;
		color: rgba(255, 255, 255, 0.7);
	}

	.stream-type {
		text-transform: capitalize;
	}

	.stream-info {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.viewer-count {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		color: white;
	}

	.stream-time {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		color: white;
	}

	.live-indicator {
		width: 8px;
		height: 8px;
		background: #da373c;
		border-radius: 50%;
		animation: live-pulse 1.5s ease-in-out infinite;
	}

	@keyframes live-pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	.header-actions {
		display: flex;
		gap: 4px;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		padding: 0;
		background: rgba(0, 0, 0, 0.5);
		border: none;
		border-radius: 4px;
		color: white;
		cursor: pointer;
		transition: background-color 0.15s ease;
	}

	.action-btn:hover {
		background: rgba(0, 0, 0, 0.7);
	}

	.close-btn:hover {
		background: #da373c;
	}

	.video-container {
		position: relative;
		flex: 1;
		min-height: 200px;
		background: #0e0e0e;
	}

	.stream-player.minimized .video-container {
		display: none;
	}

	.stream-video {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}

	.video-placeholder {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		color: rgba(255, 255, 255, 0.5);
	}

	.video-controls {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		display: flex;
		justify-content: flex-end;
		padding: 12px;
		background: linear-gradient(to top, rgba(0, 0, 0, 0.8), transparent);
		transition: opacity 0.2s ease;
	}

	.video-controls.hidden {
		opacity: 0;
		pointer-events: none;
	}

	.quality-selector select {
		padding: 6px 10px;
		background: rgba(0, 0, 0, 0.7);
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 12px;
		cursor: pointer;
	}

	.quality-selector select:hover {
		background: rgba(0, 0, 0, 0.9);
	}

	.minimized-preview {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		cursor: pointer;
	}

	.preview-placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 48px;
		height: 36px;
		background: #2b2d31;
		border-radius: 4px;
		color: rgba(255, 255, 255, 0.5);
	}

	.preview-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.preview-name {
		font-size: 13px;
		font-weight: 600;
		color: white;
	}

	.preview-live {
		font-size: 10px;
		font-weight: 700;
		color: #da373c;
	}
</style>
