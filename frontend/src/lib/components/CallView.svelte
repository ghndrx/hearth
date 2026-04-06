<script lang="ts">
<<<<<<< HEAD
	import { onMount, onDestroy } from 'svelte';
=======
	import { onDestroy } from 'svelte';
>>>>>>> 67bf1a2 (check/hearth build status (#147))
	import {
		videoCallStore,
		isInVideoCall,
		videoCallParticipants,
		videoCallState,
		isCameraOn,
		isMuted,
		isScreenSharing,
		videoCallActions,
		incomingVideoRing,
		type VideoCallState
	} from '$lib/stores/videoCall';
<<<<<<< HEAD
	import { getVideoCallManager } from '$lib/voice/VideoCallManager';
=======
>>>>>>> 67bf1a2 (check/hearth build status (#147))
	import Avatar from './Avatar.svelte';

	// Call duration tracking
	let durationInterval: ReturnType<typeof setInterval> | null = null;
	let durationText = '00:00';
	let callStartTime: Date | null = null;

	// Track call state changes to manage duration timer
	$: if ($videoCallState === 'connected' && !callStartTime) {
		callStartTime = new Date();
		durationInterval = setInterval(updateDuration, 1000);
	}

	$: if ($videoCallState === 'ended' || $videoCallState === 'idle') {
		if (durationInterval) {
			clearInterval(durationInterval);
			durationInterval = null;
		}
		callStartTime = null;
		durationText = '00:00';
	}

	function updateDuration() {
		if (!callStartTime) return;
		const elapsed = Math.floor((Date.now() - callStartTime.getTime()) / 1000);
		const minutes = Math.floor(elapsed / 60).toString().padStart(2, '0');
		const seconds = (elapsed % 60).toString().padStart(2, '0');
		durationText = `${minutes}:${seconds}`;
	}

	function getStateLabel(state: VideoCallState): string {
		switch (state) {
			case 'ringing_out': return 'Calling...';
			case 'ringing_in': return 'Incoming call';
			case 'connecting': return 'Connecting...';
			case 'connected': return '';
			case 'reconnecting': return 'Reconnecting...';
			case 'ended': return 'Call ended';
			default: return '';
		}
	}

	function getDisplayName(p: { username: string; display_name: string | null }): string {
		return p.display_name || p.username || 'Unknown';
	}

<<<<<<< HEAD
	// Attach remote video streams to video elements
	function attachVideoStreams() {
		const manager = getVideoCallManager();
		for (const participant of $videoCallParticipants) {
			const videoEl = document.getElementById(`video-${participant.id}`) as HTMLVideoElement;
			if (videoEl) {
				const remoteStream = manager.getRemoteStream(participant.id);
				if (remoteStream && videoEl.srcObject !== remoteStream) {
					videoEl.srcObject = remoteStream;
				}
			}
		}
	}

	// Poll for video stream updates when call is active
	let videoPollInterval: ReturnType<typeof setInterval> | null = null;

	$: if ($videoCallState === 'connected' && !videoPollInterval) {
		videoPollInterval = setInterval(attachVideoStreams, 500);
	}

	$: if (($videoCallState === 'ended' || $videoCallState === 'idle') && videoPollInterval) {
		clearInterval(videoPollInterval);
		videoPollInterval = null;
	}

=======
>>>>>>> 67bf1a2 (check/hearth build status (#147))
	onDestroy(() => {
		if (durationInterval) {
			clearInterval(durationInterval);
		}
<<<<<<< HEAD
		if (videoPollInterval) {
			clearInterval(videoPollInterval);
		}
=======
>>>>>>> 67bf1a2 (check/hearth build status (#147))
	});
</script>

<!-- Incoming call ring overlay -->
{#if $incomingVideoRing}
	<div class="call-ring-overlay">
		<div class="ring-card">
			<div class="ring-avatar">
				<Avatar username={$incomingVideoRing.from_username} size="xl" />
			</div>
			<h3>Incoming {$incomingVideoRing.call_type === 'direct' ? '' : 'group '}call</h3>
			<p class="ring-caller">{$incomingVideoRing.from_username}</p>
			<div class="ring-actions">
				<button class="btn-decline" on:click={videoCallActions.declineCall}>
					Decline
				</button>
				<button class="btn-accept" on:click={videoCallActions.acceptCall}>
					Accept
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Active call view -->
{#if $isInVideoCall}
	<div class="call-view">
		<!-- Header -->
		<div class="call-header">
			<div class="call-info">
				{#if $videoCallState === 'connected'}
					<span class="call-duration">{durationText}</span>
				{:else}
					<span class="call-status">{getStateLabel($videoCallState)}</span>
				{/if}
			</div>
			<span class="participant-count">
				{$videoCallParticipants.length} participant{$videoCallParticipants.length !== 1 ? 's' : ''}
			</span>
		</div>

		<!-- Video area -->
		<div class="call-video-area">
			{#if $videoCallParticipants.length === 0}
				<div class="call-waiting">
					<p>Waiting for others to join...</p>
				</div>
			{:else}
				<div class="participant-grid" class:grid-1={$videoCallParticipants.length === 1}
					class:grid-2={$videoCallParticipants.length === 2}
					class:grid-many={$videoCallParticipants.length > 2}>
					{#each $videoCallParticipants as participant (participant.id)}
<<<<<<< HEAD
						{@const videoElId = `video-${participant.id}`}
						{@const hasVideoEl = typeof document !== 'undefined' && !!document.getElementById(videoElId)}
						<div class="participant-tile">
							{#if participant.isCameraOn && hasVideoEl}
								<video
									id={videoElId}
									class="participant-video"
									autoplay
									playsinline
									muted={false}
								></video>
=======
						<div class="participant-tile">
							{#if participant.isCameraOn}
								<!-- TODO: Attach actual video track from WebRTC peer connection -->
								<video class="participant-video" autoplay playsinline muted={false}>
									<track kind="captions" />
								</video>
>>>>>>> 67bf1a2 (check/hearth build status (#147))
							{:else}
								<div class="participant-avatar">
									<Avatar username={participant.username} size="lg" />
								</div>
							{/if}
							<div class="participant-label">
								<span class="participant-name">{getDisplayName(participant)}</span>
								{#if participant.isMuted}
									<span class="muted-icon" title="Muted">M</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Controls -->
		<div class="call-controls">
			<button
				class="control-btn"
				class:active={!$isMuted}
				on:click={videoCallActions.toggleMute}
				title={$isMuted ? 'Unmute' : 'Mute'}
			>
				{$isMuted ? 'Unmute' : 'Mute'}
			</button>

			<button
				class="control-btn"
				class:active={$isCameraOn}
				on:click={videoCallActions.toggleCamera}
				title={$isCameraOn ? 'Turn off camera' : 'Turn on camera'}
			>
				{$isCameraOn ? 'Cam Off' : 'Cam On'}
			</button>

			<button
				class="control-btn"
				class:active={$isScreenSharing}
				on:click={videoCallActions.toggleScreenShare}
				title={$isScreenSharing ? 'Stop sharing' : 'Share screen'}
			>
				{$isScreenSharing ? 'Stop Share' : 'Share'}
			</button>

			<button
				class="control-btn end-call"
				on:click={videoCallActions.endCall}
				title="End call"
			>
				End
			</button>
		</div>
	</div>
{/if}

<style>
	.call-ring-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.ring-card {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 12px;
		padding: 2rem;
		text-align: center;
		min-width: 300px;
	}

	.ring-avatar {
		margin-bottom: 1rem;
	}

	.ring-card h3 {
		color: var(--text-primary, #fff);
		margin: 0 0 0.5rem;
		font-size: 1.1rem;
	}

	.ring-caller {
		color: var(--text-secondary, #b5bac1);
		margin: 0 0 1.5rem;
		font-size: 0.95rem;
	}

	.ring-actions {
		display: flex;
		gap: 1rem;
		justify-content: center;
	}

	.btn-decline {
		background: #ed4245;
		color: white;
		border: none;
		border-radius: 50px;
		padding: 0.75rem 2rem;
		font-size: 0.95rem;
		cursor: pointer;
	}

	.btn-accept {
		background: #3ba55d;
		color: white;
		border: none;
		border-radius: 50px;
		padding: 0.75rem 2rem;
		font-size: 0.95rem;
		cursor: pointer;
	}

	.call-view {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: var(--bg-tertiary, #1e1f22);
		display: flex;
		flex-direction: column;
		z-index: 999;
	}

	.call-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem 1.5rem;
		background: var(--bg-secondary, #2b2d31);
	}

	.call-info {
		color: var(--text-primary, #fff);
		font-weight: 600;
	}

	.call-duration {
		font-variant-numeric: tabular-nums;
	}

	.call-status {
		color: var(--text-secondary, #b5bac1);
	}

	.participant-count {
		color: var(--text-secondary, #b5bac1);
		font-size: 0.85rem;
	}

	.call-video-area {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
		overflow: hidden;
	}

	.call-waiting {
		color: var(--text-secondary, #b5bac1);
		font-size: 1.1rem;
	}

	.participant-grid {
		display: grid;
		gap: 0.5rem;
		width: 100%;
		height: 100%;
		max-width: 1200px;
	}

	.grid-1 { grid-template-columns: 1fr; }
	.grid-2 { grid-template-columns: 1fr 1fr; }
	.grid-many { grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); }

	.participant-tile {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		min-height: 200px;
	}

	.participant-video {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.participant-avatar {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.participant-label {
		position: absolute;
		bottom: 0.5rem;
		left: 0.5rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(0, 0, 0, 0.6);
		border-radius: 4px;
		padding: 0.25rem 0.5rem;
	}

	.participant-name {
		color: #fff;
		font-size: 0.85rem;
	}

	.muted-icon {
		color: #ed4245;
		font-size: 0.75rem;
		font-weight: bold;
	}

	.call-controls {
		display: flex;
		justify-content: center;
		gap: 1rem;
		padding: 1.5rem;
		background: var(--bg-secondary, #2b2d31);
	}

	.control-btn {
		background: var(--bg-primary, #313338);
		color: var(--text-primary, #fff);
		border: none;
		border-radius: 50px;
		padding: 0.75rem 1.5rem;
		font-size: 0.9rem;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.control-btn:hover {
		background: var(--bg-modifier-hover, #3a3c42);
	}

	.control-btn.active {
		background: var(--brand-color, #5865f2);
	}

	.control-btn.end-call {
		background: #ed4245;
	}

	.control-btn.end-call:hover {
		background: #c03537;
	}
</style>
