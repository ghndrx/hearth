<script lang="ts">
	export let soundId: string;
	export let soundName: string;
	export let emojiName: string = '🔊';
	export let audioUrl: string;
	export let volume: number = 1.0;
	export let durationMs: number = 1000;
	export let userId: string;
	export let displayName: string;
	export let avatarUrl: string = '';
	export let timestamp: string;

	let isPlaying = false;
	let audioElement: HTMLAudioElement | null = null;

	function playSound() {
		if (isPlaying) {
			stopSound();
			return;
		}

		audioElement = new Audio(audioUrl);
		audioElement.volume = volume;
		audioElement.play().catch(err => {
			console.error('Failed to play sound:', err);
		});

		isPlaying = true;

		// Auto-stop after duration
		setTimeout(() => {
			stopSound();
		}, durationMs);
	}

	function stopSound() {
		if (audioElement) {
			audioElement.pause();
			audioElement = null;
		}
		isPlaying = false;
	}

	function formatTime(ms: number): string {
		const seconds = Math.floor(ms / 1000);
		const hundredths = Math.floor((ms % 1000) / 10);
		return `${seconds}.${hundredths.toString().padStart(2, '0')}s`;
	}
</script>

<div class="soundboard-message">
	<div class="avatar">
		<img src={avatarUrl || '/default-avatar.png'} alt={displayName} />
	</div>
	<div class="content">
		<div class="header">
			<span class="username">{displayName}</span>
			<span class="timestamp">{timestamp}</span>
		</div>
		<div class="sound-card" class:playing={isPlaying}>
			<button 
				class="play-button" 
				on:click={playSound}
				aria-label={isPlaying ? 'Stop sound' : `Play ${soundName}`}
			>
				{#if isPlaying}
					<svg viewBox="0 0 24 24" width="32" height="32" aria-hidden="true">
						<rect x="6" y="4" width="4" height="16" fill="currentColor"/>
						<rect x="14" y="4" width="4" height="16" fill="currentColor"/>
					</svg>
				{:else}
					<svg viewBox="0 0 24 24" width="32" height="32" aria-hidden="true">
						<path fill="currentColor" d="M8 5v14l11-7z"/>
					</svg>
				{/if}
			</button>
			<div class="sound-info">
				<span class="sound-emoji" aria-hidden="true">{emojiName}</span>
				<span class="sound-name">{soundName}</span>
				{#if isPlaying}
					<div class="playing-indicator">
						<span class="bar"></span>
						<span class="bar"></span>
						<span class="bar"></span>
					</div>
				{:else}
					<span class="duration">{formatTime(durationMs)}</span>
				{/if}
			</div>
		</div>
	</div>
</div>

<style>
	.soundboard-message {
		display: flex;
		gap: 16px;
		padding: 16px 0;
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

	.timestamp {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.sound-card {
		display: flex;
		align-items: center;
		gap: 12px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		padding: 12px;
		transition: background-color 0.15s;
	}

	.sound-card.playing {
		background: var(--bg-modifier-selected, rgba(79, 84, 92, 0.24));
	}

	.play-button {
		width: 48px;
		height: 48px;
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

	.sound-info {
		display: flex;
		align-items: center;
		gap: 8px;
		flex: 1;
		min-width: 0;
	}

	.sound-emoji {
		font-size: 20px;
	}

	.sound-name {
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		font-size: 15px;
	}

	.duration {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
		margin-left: auto;
	}

	.playing-indicator {
		display: flex;
		gap: 2px;
		align-items: flex-end;
		height: 16px;
		margin-left: auto;
	}

	.playing-indicator .bar {
		width: 3px;
		background: var(--brand-primary, #5865f2);
		border-radius: 1px;
		animation: soundbar 0.5s ease-in-out infinite alternate;
	}

	.playing-indicator .bar:nth-child(1) {
		height: 8px;
		animation-delay: 0s;
	}

	.playing-indicator .bar:nth-child(2) {
		height: 16px;
		animation-delay: 0.15s;
	}

	.playing-indicator .bar:nth-child(3) {
		height: 10px;
		animation-delay: 0.3s;
	}

	@keyframes soundbar {
		from { transform: scaleY(0.3); }
		to { transform: scaleY(1); }
	}
</style>
