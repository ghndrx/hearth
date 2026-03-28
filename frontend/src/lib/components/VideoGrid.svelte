<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Track, LocalParticipant, RemoteParticipant } from 'livekit-client';
	import Avatar from './Avatar.svelte';

	export let participants: Array<{
		id: string;
		username: string;
		display_name?: string | null;
		avatar?: string | null;
		isVideoEnabled: boolean;
		isScreenSharing: boolean;
		isSpeaking: boolean;
		isMuted: boolean;
	}> = [];
	
	export let localVideoTrack: HTMLVideoElement | null = null;
	export let localScreenShareTrack: HTMLVideoElement | null = null;
	export let onVideoElementClick: ((participantId: string, isLocal: boolean) => void) | null = null;

	// Track refs for remote participants
	let videoElements: Map<string, HTMLVideoElement> = new Map();

	function getDisplayName(p: typeof participants[0]): string {
		return p.display_name || p.username || 'Unknown';
	}

	function handleVideoClick(participantId: string, isLocal: boolean) {
		if (onVideoElementClick) {
			onVideoElementClick(participantId, isLocal);
		}
	}

	// Attach remote video tracks when participants change
	export function attachTrack(participantId: string, track: MediaStreamTrack, isScreenShare: boolean): HTMLVideoElement | null {
		let container = videoElements.get(participantId + (isScreenShare ? '-screen' : '-camera'));
		
		if (!container) {
			// Container doesn't exist yet, will be attached when component re-renders
			return null;
		}

		// Check if we already have a stream attached
		const existingSrc = container.srcObject as MediaStream;
		if (existingSrc && Array.from(existingSrc.getTracks()).some(t => t.id === track.id)) {
			return container;
		}

		const stream = new MediaStream([track]);
		container.srcObject = stream;
		container.autoplay = true;
		container.playsInline = true;
		
		return container;
	}

	// Detach track from element
	export function detachTrack(participantId: string, isScreenShare: boolean) {
		const key = participantId + (isScreenShare ? '-screen' : '-camera');
		const container = videoElements.get(key);
		if (container) {
			container.srcObject = null;
		}
	}

	// Get element for a specific participant's video
	export function getVideoElement(participantId: string, isScreenShare: boolean): HTMLVideoElement | null {
		return videoElements.get(participantId + (isScreenShare ? '-screen' : '-camera')) || null;
	}

	// Svelte action for setting video element refs
	function setVideoRef(node: HTMLVideoElement, params: { participantId: string; isScreenShare: boolean }) {
		const key = params.participantId + (params.isScreenShare ? '-screen' : '-camera');
		videoElements.set(key, node);
		return {
			destroy() {
				videoElements.delete(key);
			}
		};
	}

	$: {
		// Force reactivity on participants for re-render
		participants;
	}
</script>

<div class="video-grid">
	{#if localVideoTrack || localScreenShareTrack}
		<!-- Local video feeds -->
		{#if localScreenShareTrack}
			<div 
				class="video-tile screen-share"
				on:click={() => handleVideoClick('local', true)}
				role="button"
				tabindex="0"
				on:keypress={(e) => e.key === 'Enter' && handleVideoClick('local', true)}
			>
				<div class="video-container">
					<video
						bind:this={localScreenShareTrack}
						class="video-element"
						autoplay
						playsinline
						muted
					></video>
					<div class="screen-share-indicator">
						<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
							<path d="M20 18c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
						</svg>
						<span>Screen Share</span>
					</div>
				</div>
				<div class="participant-name local">You (Screen)</div>
			</div>
		{/if}

		{#if localVideoTrack}
			<div 
				class="video-tile"
				class:speaking={participants.some(p => p.isSpeaking)}
				on:click={() => handleVideoClick('local', false)}
				role="button"
				tabindex="0"
				on:keypress={(e) => e.key === 'Enter' && handleVideoClick('local', false)}
			>
				<div class="video-container">
					<video
						bind:this={localVideoTrack}
						class="video-element"
						autoplay
						playsinline
						muted
					></video>
					<div class="camera-indicator">
						<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
							<path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
						</svg>
					</div>
				</div>
				<div class="participant-name local">You (Camera)</div>
			</div>
		{/if}
	{/if}

	<!-- Remote participants -->
	{#each participants as participant (participant.id)}
		{@const hasVideo = participant.isVideoEnabled}
		{@const hasScreenShare = participant.isScreenSharing}
		
		<!-- Camera video tile -->
		{#if hasVideo}
			<div 
				class="video-tile"
				class:speaking={participant.isSpeaking}
				on:click={() => handleVideoClick(participant.id, false)}
				role="button"
				tabindex="0"
				on:keypress={(e) => e.key === 'Enter' && handleVideoClick(participant.id, false)}
			>
				<div class="video-container">
					<video
						use:setVideoRef={{ participantId: participant.id, isScreenShare: false }}
						class="video-element"
						id="video-{participant.id}-camera"
						autoplay
						playsinline
					></video>
					<div class="camera-indicator">
						<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
							<path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
						</svg>
					</div>
				</div>
				<div class="participant-name">
					{getDisplayName(participant)}
				</div>
			</div>
		{:else}
			<!-- Avatar fallback when no video -->
			<div 
				class="video-tile avatar-only"
				on:click={() => handleVideoClick(participant.id, false)}
				role="button"
				tabindex="0"
				on:keypress={(e) => e.key === 'Enter' && handleVideoClick(participant.id, false)}
			>
				<div class="avatar-container" class:speaking={participant.isSpeaking}>
					<Avatar
						src={participant.avatar}
						username={participant.username}
						size="xl"
					/>
					{#if participant.isSpeaking}
						<div class="speaking-ring"></div>
					{/if}
					{#if participant.isMuted}
						<div class="mute-overlay">
							<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
								<path d="M19 11c0 1.19-.34 2.3-.9 3.28l-1.23-1.23c.27-.62.43-1.3.43-2.05H19zm-4-2V5c0-1.66-1.34-3-3-3S9 3.34 9 5v.17l6 6V9zM4.27 3L3 4.27l6.01 6.01V11c0 1.66 1.33 3 2.99 3 .22 0 .44-.03.65-.08l1.66 1.66c-.71.33-1.5.52-2.31.52-2.76 0-5.3-2.1-5.3-5.1H5c0 3.41 2.72 6.23 6 6.72V21h2v-3.28c.91-.13 1.77-.45 2.54-.9L19.73 21 21 19.73 4.27 3z"/>
							</svg>
						</div>
					{/if}
				</div>
				<div class="participant-name">
					{getDisplayName(participant)}
				</div>
			</div>
		{/if}

		<!-- Screen share video tile -->
		{#if hasScreenShare}
			<div 
				class="video-tile screen-share"
				on:click={() => handleVideoClick(participant.id, true)}
				role="button"
				tabindex="0"
				on:keypress={(e) => e.key === 'Enter' && handleVideoClick(participant.id, true)}
			>
				<div class="video-container">
					<video
						use:setVideoRef={{ participantId: participant.id, isScreenShare: true }}
						class="video-element"
						id="video-{participant.id}-screen"
						autoplay
						playsinline
					></video>
					<div class="screen-share-indicator">
						<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
							<path d="M20 18c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
						</svg>
						<span>{getDisplayName(participant)}'s Screen</span>
					</div>
				</div>
				<div class="participant-name">
					{getDisplayName(participant)} (Screen)
				</div>
			</div>
		{/if}
	{:else}
		{#if !localVideoTrack && !localScreenShareTrack}
			<div class="empty-state">
				<svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor">
					<path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
				</svg>
				<p>No video feeds active</p>
				<span>Start your camera or share your screen</span>
			</div>
		{/if}
	{/each}
</div>

<style>
	.video-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 12px;
		padding: 12px;
		max-height: 400px;
		overflow-y: auto;
	}

	.video-tile {
		position: relative;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		overflow: hidden;
		cursor: pointer;
		transition: transform 0.15s ease, box-shadow 0.15s ease;
	}

	.video-tile:hover {
		transform: scale(1.02);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
	}

	.video-tile.speaking {
		box-shadow: 0 0 0 2px var(--status-online, #23a559);
	}

	.video-tile.screen-share {
		grid-column: span 2;
	}

	.video-container {
		position: relative;
		width: 100%;
		padding-top: 56.25%; /* 16:9 aspect ratio */
		background: var(--bg-primary, #1e1f22);
	}

	.video-element {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.avatar-only {
		padding: 16px;
		padding-top: 24px;
	}

	.avatar-container {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		aspect-ratio: 1;
		max-width: 160px;
		margin: 0 auto;
		border-radius: 50%;
	}

	.avatar-container.speaking {
		box-shadow: 0 0 0 3px var(--status-online, #23a559);
	}

	.speaking-ring {
		position: absolute;
		inset: -4px;
		border: 3px solid var(--status-online, #23a559);
		border-radius: 50%;
		animation: pulse 1s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	.mute-overlay {
		position: absolute;
		bottom: 8px;
		right: 8px;
		width: 32px;
		height: 32px;
		background: rgba(242, 63, 67, 0.9);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.participant-name {
		padding: 8px 12px;
		font-size: 13px;
		font-weight: 500;
		color: var(--text-primary, #f2f3f5);
		background: var(--bg-secondary, #2b2d31);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.participant-name.local {
		color: var(--brand-primary, #5865f2);
	}

	.camera-indicator {
		position: absolute;
		top: 8px;
		right: 8px;
		width: 28px;
		height: 28px;
		background: rgba(0, 0, 0, 0.6);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--status-online, #23a559);
	}

	.screen-share-indicator {
		position: absolute;
		top: 8px;
		left: 8px;
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 4px 8px;
		background: rgba(0, 0, 0, 0.7);
		border-radius: 4px;
		font-size: 12px;
		color: white;
	}

	.empty-state {
		grid-column: 1 / -1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		color: var(--text-muted, #949ba4);
		text-align: center;
	}

	.empty-state svg {
		margin-bottom: 12px;
		opacity: 0.5;
	}

	.empty-state p {
		margin: 0 0 4px;
		font-size: 15px;
		font-weight: 500;
		color: var(--text-secondary, #b5bac1);
	}

	.empty-state span {
		font-size: 13px;
		color: var(--text-muted, #949ba4);
	}

	/* Scrollbar styling */
	.video-grid::-webkit-scrollbar {
		width: 8px;
	}

	.video-grid::-webkit-scrollbar-track {
		background: transparent;
	}

	.video-grid::-webkit-scrollbar-thumb {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.4));
		border-radius: 4px;
	}

	.video-grid::-webkit-scrollbar-thumb:hover {
		background: var(--bg-modifier-active, rgba(79, 84, 92, 0.6));
	}
</style>
