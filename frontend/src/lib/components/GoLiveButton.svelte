<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { streamState, streamActions, isStreaming, type StreamType, type StreamQuality } from '$lib/stores/stream';
	import { voiceState } from '$lib/stores/voice';
	import { user as authUser } from '$lib/stores/auth';

	const dispatch = createEventDispatcher<{
		streamStart: { streamType: StreamType; quality: StreamQuality };
		streamStop: void;
	}>();

	type QualityOption = {
		value: StreamQuality;
		label: string;
	};

	type StreamTypeOption = {
		value: StreamType;
		label: string;
		icon: string;
	};

	const qualityOptions: QualityOption[] = [
		{ value: 1, label: '480p' },
		{ value: 2, label: '720p' },
		{ value: 3, label: '1080p' }
	];

	const streamTypeOptions: StreamTypeOption[] = [
		{ value: 1, label: 'Screen', icon: 'M21 3H3c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H3V5h18v14z' },
		{ value: 2, label: 'Application', icon: 'M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 14H4V6h16v12zM4 0h16v2H4zM4 22h16v2H4z' },
		{ value: 3, label: 'Camera', icon: 'M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z' }
	];

	let selectedStreamType: StreamType = 1;
	let selectedQuality: StreamQuality = 2;
	let showMenu = false;
	let menuPosition: 'top' | 'bottom' = 'bottom';

	$: isInVoiceChannel = $voiceState.isConnected;
	$: isInSameChannel = $voiceState.channelId !== null;
	$: currentChannelId = $voiceState.channelId;
	$: canStream = isInVoiceChannel && isInSameChannel && !$isStreaming;
	$: streamerUsername = $streamState.currentStream?.streamer?.username;

	function toggleMenu() {
		if (!canStream && !$isStreaming) return;
		showMenu = !showMenu;

		// Position menu above if not enough space below
		if (showMenu && typeof window !== 'undefined') {
			const button = document.querySelector('.go-live-btn');
			if (button) {
				const rect = button.getBoundingClientRect();
				const spaceBelow = window.innerHeight - rect.bottom;
				menuPosition = spaceBelow < 200 ? 'top' : 'bottom';
			}
		}
	}

	function closeMenu() {
		showMenu = false;
	}

	async function handleStartStream() {
		closeMenu();

		try {
			await streamActions.startStream(currentChannelId!, selectedStreamType, selectedQuality);
			dispatch('streamStart', { streamType: selectedStreamType, quality: selectedQuality });
		} catch (error) {
			console.error('Failed to start stream:', error);
		}
	}

	async function handleStopStream() {
		closeMenu();

		try {
			await streamActions.stopStream(currentChannelId!);
			dispatch('streamStop');
		} catch (error) {
			console.error('Failed to stop stream:', error);
		}
	}

	function selectStreamType(type: StreamType) {
		selectedStreamType = type;
	}

	function selectQuality(quality: StreamQuality) {
		selectedQuality = quality;
	}

	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.go-live-menu-container')) {
			closeMenu();
		}
	}
</script>

<svelte:window on:click={handleClickOutside} />

<div class="go-live-menu-container">
	<button
		class="go-live-btn"
		class:streaming={$isStreaming}
		class:disabled={!canStream && !$isStreaming}
		on:click|stopPropagation={toggleMenu}
		disabled={!canStream && !$isStreaming}
		aria-label={$isStreaming ? `Stop streaming ${streamerUsername}` : 'Go Live'}
		title={$isStreaming ? `Streaming as ${streamerUsername}` : canStream ? 'Go Live' : 'Join a voice channel to stream'}
	>
		{#if $isStreaming}
			<!-- Streaming indicator -->
			<div class="streaming-indicator">
				<div class="pulse"></div>
				<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
					<circle cx="12" cy="12" r="8"/>
				</svg>
			</div>
			<span class="btn-text">Streaming</span>
		{:else}
			<!-- Go Live icon -->
			<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
				<path d="M21 3H3c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H3V5h18v14zM9 8c0 .55-.45 1-1 1s-1-.45-1-1 .45-1 1-1 1 .45 1 1zm4 4.5c0 .28-.22.5-.5.5h-5c-.28 0-.5-.22-.5-.5v-5c0-.28.22-.5.5-.5h5c.28 0 .5.22.5.5v5zm2.5 2c0 .28-.22.5-.5.5h-7c-.28 0-.5-.22-.5-.5v-7c0-.28.22-.5.5-.5h7c.28 0 .5.22.5.5v7z"/>
			</svg>
			<span class="btn-text">Go Live</span>
		{/if}
	</button>

	{#if showMenu}
		<div class="go-live-menu menu-{menuPosition}">
			{#if $isStreaming}
				<!-- Stop streaming option -->
				<div class="menu-section">
					<div class="menu-header">
						<span class="streaming-label">You are streaming</span>
					</div>
					<button class="menu-item stop" on:click={handleStopStream}>
						<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
							<path d="M6 6h12v12H6z"/>
						</svg>
						<span>Stop Stream</span>
					</button>
				</div>
			{:else}
				<!-- Stream type selection -->
				<div class="menu-section">
					<div class="menu-header">
						<span>Stream Type</span>
					</div>
					<div class="stream-types">
						{#each streamTypeOptions as option}
							<button
								class="stream-type-btn"
								class:selected={selectedStreamType === option.value}
								on:click={() => selectStreamType(option.value)}
								aria-label={option.label}
								title={option.label}
							>
								<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
									<path d={option.icon}/>
								</svg>
								<span class="type-label">{option.label}</span>
							</button>
						{/each}
					</div>
				</div>

				<!-- Quality selection -->
				<div class="menu-section">
					<div class="menu-header">
						<span>Quality</span>
					</div>
					<div class="quality-options">
						{#each qualityOptions as option}
							<button
								class="quality-btn"
								class:selected={selectedQuality === option.value}
								on:click={() => selectQuality(option.value)}
							>
								{option.label}
							</button>
						{/each}
					</div>
				</div>

				<!-- Start button -->
				<div class="menu-section">
					<button class="start-btn" on:click={handleStartStream}>
						<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
							<path d="M8 5v14l11-7z"/>
						</svg>
						<span>Go Live</span>
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.go-live-menu-container {
		position: relative;
		display: inline-block;
	}

	.go-live-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		background: #5865f2;
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		transition: background-color 0.15s ease;
	}

	.go-live-btn:hover:not(.disabled) {
		background: #4752c4;
	}

	.go-live-btn.disabled {
		background: #4e5058;
		color: #949ba4;
		cursor: not-allowed;
	}

	.go-live-btn.streaming {
		background: #da373c;
	}

	.go-live-btn.streaming:hover {
		background: #c8333b;
	}

	.streaming-indicator {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
	}

	.pulse {
		position: absolute;
		width: 16px;
		height: 16px;
		background: rgba(255, 255, 255, 0.3);
		border-radius: 50%;
		animation: pulse 1.5s ease-in-out infinite;
	}

	@keyframes pulse {
		0% {
			transform: scale(1);
			opacity: 0.5;
		}
		50% {
			transform: scale(1.5);
			opacity: 0;
		}
		100% {
			transform: scale(1);
			opacity: 0;
		}
	}

	.btn-text {
		line-height: 1;
	}

	.go-live-menu {
		position: absolute;
		right: 0;
		z-index: 100;
		min-width: 220px;
		background: #2b2d31;
		border: 1px solid #1e1f22;
		border-radius: 8px;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
		overflow: hidden;
	}

	.menu-bottom {
		top: calc(100% + 4px);
	}

	.menu-top {
		bottom: calc(100% + 4px);
	}

	.menu-section {
		padding: 8px;
		border-bottom: 1px solid #1e1f22;
	}

	.menu-section:last-child {
		border-bottom: none;
	}

	.menu-header {
		padding: 4px 8px;
		font-size: 11px;
		font-weight: 700;
		text-transform: uppercase;
		color: #6d6f78;
		letter-spacing: 0.02em;
	}

	.streaming-label {
		font-size: 12px;
		font-weight: 600;
		color: #da373c;
		text-transform: none;
	}

	.menu-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 8px;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: #dbdee1;
		font-size: 14px;
		cursor: pointer;
		transition: background-color 0.1s ease;
	}

	.menu-item:hover {
		background: rgba(79, 84, 92, 0.32);
	}

	.menu-item.stop {
		color: #f23f42;
	}

	.menu-item.stop:hover {
		background: rgba(240, 71, 71, 0.1);
	}

	.stream-types {
		display: flex;
		gap: 4px;
		padding: 4px;
	}

	.stream-type-btn {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		padding: 8px 4px;
		background: #2b2d31;
		border: 2px solid transparent;
		border-radius: 4px;
		color: #949ba4;
		font-size: 11px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.stream-type-btn:hover {
		background: #36383f;
		color: #dbdee1;
	}

	.stream-type-btn.selected {
		background: rgba(88, 101, 242, 0.15);
		border-color: #5865f2;
		color: #dbdee1;
	}

	.type-label {
		line-height: 1;
	}

	.quality-options {
		display: flex;
		gap: 4px;
		padding: 4px;
	}

	.quality-btn {
		flex: 1;
		padding: 6px 8px;
		background: #2b2d31;
		border: 2px solid transparent;
		border-radius: 4px;
		color: #949ba4;
		font-size: 12px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.quality-btn:hover {
		background: #36383f;
		color: #dbdee1;
	}

	.quality-btn.selected {
		background: rgba(88, 101, 242, 0.15);
		border-color: #5865f2;
		color: #dbdee1;
	}

	.start-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		width: 100%;
		padding: 10px;
		background: #5865f2;
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: background-color 0.15s ease;
	}

	.start-btn:hover {
		background: #4752c4;
	}
</style>
