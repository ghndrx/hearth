<script lang="ts">
	export let voiceMessageId: string;
	export let fileUrl: string;
	export let durationMs: number;
	export let waveformData: number[] = [];
	export let username: string = '';
	export let timestamp: string = '';
	export let avatarUrl: string = '';
	export let isOwnMessage: boolean = false;

	let isPlaying = false;
	let isPaused = false;
	let currentTime = 0;
	let duration = durationMs / 1000;
	let audioElement: HTMLAudioElement | null = null;
	let progress = 0;

	function playAudio() {
		if (!audioElement) {
			audioElement = new Audio(fileUrl);
			audioElement.addEventListener('timeupdate', updateProgress);
			audioElement.addEventListener('ended', handleEnded);
			audioElement.addEventListener('loadedmetadata', () => {
				duration = audioElement?.duration || durationMs / 1000;
			});
		}

		if (isPaused) {
			audioElement.play();
			isPlaying = true;
			isPaused = false;
			return;
		}

		if (isPlaying) {
			audioElement.pause();
			isPlaying = false;
			isPaused = true;
			return;
		}

		audioElement.play().catch(err => {
			console.error('Failed to play audio:', err);
		});
		isPlaying = true;
	}

	function updateProgress() {
		if (audioElement) {
			currentTime = audioElement.currentTime;
			progress = (currentTime / duration) * 100;
		}
	}

	function handleEnded() {
		isPlaying = false;
		isPaused = false;
		currentTime = 0;
		progress = 0;
	}

	function handleProgressClick(event: MouseEvent) {
		if (!audioElement) return;
		const target = event.currentTarget as HTMLElement;
		const rect = target.getBoundingClientRect();
		const clickX = event.clientX - rect.left;
		const percentage = clickX / rect.width;
		audioElement.currentTime = percentage * duration;
	}

	function formatTime(seconds: number): string {
		const mins = Math.floor(seconds / 60);
		const secs = Math.floor(seconds % 60);
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	function formatDuration(ms: number): string {
		const seconds = Math.floor(ms / 1000);
		return formatTime(seconds);
	}
</script>

<div class="voice-message-bubble" class:own={isOwnMessage}>
	<div class="avatar">
		<img src={avatarUrl || '/default-avatar.png'} alt={username} />
	</div>
	<div class="content">
		<div class="header">
			<span class="username">{username}</span>
			<span class="timestamp">{timestamp}</span>
		</div>
		<div class="audio-player">
			<button 
				class="play-button" 
				on:click={playAudio}
				aria-label={isPlaying ? 'Pause' : 'Play'}
			>
				{#if isPlaying}
					<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
						<rect x="6" y="4" width="4" height="16" fill="currentColor"/>
						<rect x="14" y="4" width="4" height="16" fill="currentColor"/>
					</svg>
				{:else}
					<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
						<path fill="currentColor" d="M8 5v14l11-7z"/>
					</svg>
				{/if}
			</button>
			
			<!-- Waveform visualization -->
			<div class="waveform-container">
				<div 
					class="waveform"
					class:playing={isPlaying}
					on:click={handleProgressClick}
					role="slider"
					aria-label="Audio progress"
					aria-valuemin="0"
					aria-valuemax="100"
					aria-valuenow={Math.round(progress)}
					tabindex="0"
				>
					{#each waveformData as amplitude, i}
						<div 
							class="waveform-bar" 
							style="height: {Math.max(4, amplitude * 32)}px"
							class:active={progress > (i / waveformData.length) * 100}
						></div>
					{/each}
				</div>
				<!-- Progress overlay -->
				<div class="progress-overlay" style="width: {progress}%"></div>
			</div>
			
			<div class="time-display">
				<span class="current-time">{formatTime(currentTime)}</span>
				<span class="separator">/</span>
				<span class="total-time">{formatDuration(durationMs)}</span>
			</div>
		</div>
	</div>
</div>

<style>
	.voice-message-bubble {
		display: flex;
		gap: 16px;
		padding: 8px 0;
		max-width: 400px;
	}

	.voice-message-bubble.own {
		flex-direction: row-reverse;
	}

	.avatar img {
		width: 40px;
		height: 40px;
		border-radius: 50%;
	}

	.content {
		flex: 1;
		min-width: 0;
	}

	.header {
		display: flex;
		align-items: baseline;
		gap: 8px;
		margin-bottom: 4px;
	}

	.username {
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		font-size: 15px;
	}

	.voice-message-bubble.own .username {
		color: var(--text-primary);
	}

	.timestamp {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.audio-player {
		display: flex;
		align-items: center;
		gap: 12px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 12px;
		padding: 10px 12px;
	}

	.voice-message-bubble.own .audio-player {
		background: var(--bg-modifier-selected, rgba(79, 84, 92, 0.24));
	}

	.play-button {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		border: none;
		background: var(--brand-primary, #5865f2);
		color: white;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: transform 0.1s, background-color 0.15s;
	}

	.play-button:hover {
		transform: scale(1.05);
		background: var(--brand-hover, #4752c4);
	}

	.play-button:active {
		transform: scale(0.95);
	}

	.waveform-container {
		flex: 1;
		position: relative;
		height: 32px;
		display: flex;
		align-items: center;
	}

	.waveform {
		display: flex;
		align-items: center;
		gap: 2px;
		height: 100%;
		width: 100%;
		cursor: pointer;
		position: relative;
		z-index: 2;
	}

	.waveform-bar {
		width: 3px;
		background: var(--text-muted, #949ba4);
		border-radius: 1px;
		transition: background-color 0.1s, height 0.1s;
	}

	.waveform-bar.active {
		background: var(--brand-primary, #5865f2);
	}

	.progress-overlay {
		position: absolute;
		top: 0;
		left: 0;
		height: 100%;
		background: rgba(88, 101, 242, 0.15);
		pointer-events: none;
		z-index: 1;
	}

	.time-display {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
		white-space: nowrap;
		display: flex;
		gap: 2px;
	}

	.separator {
		opacity: 0.5;
	}

	.current-time {
		min-width: 32px;
	}
</style>
