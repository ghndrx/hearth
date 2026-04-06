<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import StageJoinButton from './StageJoinButton.svelte';
	import SpeakerList from './SpeakerList.svelte';
	import Avatar from '../Avatar.svelte';
	import LoadingSpinner from '../LoadingSpinner.svelte';
	import type { Stage, StageParticipant } from '$lib/stores/stage';
	import { 
		StageRole, 
		getStatusLabel,
		getRoleLabel,
		isHostOrModerator,
		isHost
	} from '$lib/stores/stage';

	export let stage: Stage;
	export let participants: StageParticipant[] = [];
	export let currentUserId: string = '';
	export let loading: boolean = false;
	export let error: string | null = null;

	const dispatch = createEventDispatcher();

	$: speakers = participants.filter(p => 
		p.role === StageRole.SPEAKER || 
		p.role === StageRole.MODERATOR || 
		p.role === StageRole.HOST
	);
	
	$: audience = participants.filter(p => p.role === StageRole.AUDIENCE);
	
	$: currentParticipant = participants.find(p => p.user_id === currentUserId);
	
	$: isInStage = !!currentParticipant;
	
	$: isSpeaker = currentParticipant ? 
		(currentParticipant.role === StageRole.SPEAKER || 
		 currentParticipant.role === StageRole.MODERATOR || 
		 currentParticipant.role === StageRole.HOST) : false;
	
	$: hasPendingRequest = currentParticipant?.has_pending_request ?? false;
	
	$: canManage = isHostOrModerator(currentParticipant);
	
	$: isStageHost = isHost(currentParticipant);

	// Actions
	function handleJoin() {
		dispatch('join');
	}

	function handleLeave() {
		dispatch('leave');
	}

	function handleRequestToSpeak() {
		dispatch('requestToSpeak');
	}

	function handleCancelRequest() {
		dispatch('cancelRequest');
	}

	function handleRaiseHand() {
		dispatch('raiseHand');
	}

	function handleLowerHand() {
		dispatch('lowerHand');
	}

	function handleMute(e: CustomEvent<{ userId: string }>) {
		dispatch('mute', e.detail);
	}

	function handleUnmute(e: CustomEvent<{ userId: string }>) {
		dispatch('unmute', e.detail);
	}

	function handleRemoveSpeaker(e: CustomEvent<{ userId: string }>) {
		dispatch('removeSpeaker', e.detail);
	}

	function handleEndStage() {
		dispatch('endStage');
	}

	function handlePauseStage() {
		dispatch('pauseStage');
	}

	function handleResumeStage() {
		dispatch('resumeStage');
	}

	function handleUpdateTopic() {
		dispatch('updateTopic');
	}
</script>

<div class="stage-channel-view">
	<!-- Stage Header -->
	<div class="stage-header">
		<div class="stage-info">
			<div class="stage-badge" class:live={stage.status === 2} class:paused={stage.status === 3}>
				{#if stage.status === 2}
					<span class="live-indicator"></span>
					LIVE
				{:else if stage.status === 3}
					PAUSED
				{:else if stage.status === 1}
					SCHEDULED
				{:else}
					ENDED
				{/if}
			</div>
			<h2 class="stage-topic">{stage.topic || 'Untitled Stage'}</h2>
			{#if stage.description}
				<p class="stage-description">{stage.description}</p>
			{/if}
		</div>

		<div class="stage-actions">
			{#if canManage}
				{#if stage.status === 2}
					<button class="action-btn" on:click={handlePauseStage} title="Pause stage">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
						</svg>
					</button>
				{:else if stage.status === 3}
					<button class="action-btn" on:click={handleResumeStage} title="Resume stage">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M8 5v14l11-7z"/>
						</svg>
					</button>
				{/if}
				
				<button class="action-btn" on:click={handleUpdateTopic} title="Edit stage">
					<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
						<path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
					</svg>
				</button>

				{#if isStageHost}
					<button class="action-btn danger" on:click={handleEndStage} title="End stage">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M6 6h12v12H6z"/>
						</svg>
					</button>
				{/if}
			{/if}
		</div>
	</div>

	<!-- Stage Stats -->
	<div class="stage-stats">
		<div class="stat">
			<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
				<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
			</svg>
			<span>{stage.speaker_count} Speaking</span>
		</div>
		<div class="stat">
			<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
				<path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
			</svg>
			<span>{stage.audience_count} Listening</span>
		</div>
		{#if stage.pending_request_count > 0}
			<div class="stat highlight">
				<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
					<path d="M12.09 13.91c.27.27.7.27.97 0l2.47-2.47c.39-.39.11-1.06-.44-1.06h-1.77l1.89-5.66c.11-.33.05-.71-.17-.97-.22-.26-.55-.36-.87-.25L10.5 5.3c-.11-.05-.23-.08-.36-.08H7.5C6.67 5.22 6 5.89 6 6.71V13c0 .55.45 1 1 1h2.44c.55 0 1-.45 1-1v-.09H12l-.91.91z"/>
				</svg>
				<span>{stage.pending_request_count} Pending</span>
			</div>
		{/if}
	</div>

	<!-- Speakers Section -->
	<div class="stage-section speakers-section">
		<SpeakerList 
			speakers={speakers}
			{currentUserId}
			{canManage}
			on:mute={handleMute}
			on:unmute={handleUnmute}
			on:removeSpeaker={handleRemoveSpeaker}
		/>
	</div>

	<!-- Audience Section (collapsed by default) -->
	{#if audience.length > 0}
		<details class="audience-section">
			<summary class="audience-header">
				<span class="audience-count">{audience.length} in Audience</span>
				<svg class="chevron" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
					<path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z"/>
				</svg>
			</summary>
			<div class="audience-list">
				{#each audience as participant (participant.user_id)}
					<div class="audience-item">
						<Avatar userId={participant.user_id} size="sm" />
						<span class="audience-name">
							{#if participant.user_id === currentUserId}
								You
							{:else}
								{participant.user_id}
							{/if}
						</span>
						{#if canManage && stage.request_to_speak}
							<div class="audience-actions">
								<button
									class="small-btn"
									on:click={() => dispatch('promote', { userId: participant.user_id })}
									title="Promote to speaker"
								>
									<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
										<path d="M4 12l1.41 1.41L11 7.83V20h2V7.83l5.58 5.59L20 12l-8-8-8 8z"/>
									</svg>
								</button>
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</details>
	{/if}

	<!-- Join/Leave Controls -->
	<div class="stage-footer">
		<StageJoinButton
			{isInStage}
			{isSpeaker}
			{hasPendingRequest}
			stageRequestToSpeak={stage.request_to_speak}
			stageModeratorOnly={stage.moderator_only}
			{loading}
			on:join={handleJoin}
			on:leave={handleLeave}
			on:requestToSpeak={handleRequestToSpeak}
			on:cancelRequest={handleCancelRequest}
			on:raiseHand={handleRaiseHand}
			on:lowerHand={handleLowerHand}
		/>
	</div>

	<!-- Error display -->
	{#if error}
		<div class="stage-error">
			{error}
		</div>
	{/if}
</div>

<style>
	.stage-channel-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		background-color: var(--bg-primary, #313338);
	}

	.stage-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}

	.stage-info {
		display: flex;
		flex-direction: column;
		gap: 4px;
		flex: 1;
		min-width: 0;
	}

	.stage-badge {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 4px 8px;
		background-color: var(--bg-secondary, #2b2d31);
		border-radius: 4px;
		font-size: 11px;
		font-weight: 700;
		color: var(--text-muted, #949ba4);
		text-transform: uppercase;
		letter-spacing: 0.02em;
		width: fit-content;
	}

	.stage-badge.live {
		background-color: var(--red, #da373c);
		color: white;
	}

	.stage-badge.paused {
		background-color: var(--yellow, #f0b232);
		color: #000;
	}

	.live-indicator {
		width: 6px;
		height: 6px;
		background-color: white;
		border-radius: 50%;
		animation: pulse 1.5s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	.stage-topic {
		font-size: 20px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		margin: 0;
		line-height: 1.3;
	}

	.stage-description {
		font-size: 13px;
		color: var(--text-muted, #949ba4);
		margin: 0;
		line-height: 1.4;
	}

	.stage-actions {
		display: flex;
		gap: 4px;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		padding: 0;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		transition: background-color 0.15s ease, color 0.15s ease;
	}

	.action-btn:hover {
		background-color: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
		color: var(--text-normal, #f2f3f5);
	}

	.action-btn.danger:hover {
		background-color: var(--red, #da373c);
		color: white;
	}

	.stage-stats {
		display: flex;
		gap: 16px;
		padding: 12px 16px;
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}

	.stat {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		color: var(--text-muted, #949ba4);
	}

	.stat.highlight {
		color: var(--yellow, #f0b232);
	}

	.stage-section {
		padding: 16px;
		flex: 1;
		overflow-y: auto;
	}

	.speakers-section {
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}

	.audience-section {
		border-bottom: 1px solid var(--bg-tertiary, #1e1f22);
	}

	.audience-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		cursor: pointer;
		user-select: none;
	}

	.audience-header:hover {
		background-color: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
	}

	.audience-count {
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted, #949ba4);
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.chevron {
		color: var(--text-muted, #949ba4);
		transition: transform 0.2s ease;
	}

	details[open] .chevron {
		transform: rotate(180deg);
	}

	.audience-list {
		padding: 0 16px 16px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.audience-item {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 6px 8px;
		border-radius: 4px;
		transition: background-color 0.15s ease;
	}

	.audience-item:hover {
		background-color: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
	}

	.audience-name {
		flex: 1;
		font-size: 14px;
		color: var(--text-normal, #f2f3f5);
	}

	.audience-actions {
		display: flex;
		gap: 4px;
		opacity: 0;
		transition: opacity 0.15s ease;
	}

	.audience-item:hover .audience-actions {
		opacity: 1;
	}

	.small-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		padding: 0;
		background: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		transition: background-color 0.15s ease, color 0.15s ease;
	}

	.small-btn:hover {
		background: var(--green, #23a559);
		color: white;
	}

	.stage-footer {
		padding: 16px;
		border-top: 1px solid var(--bg-tertiary, #1e1f22);
		display: flex;
		justify-content: center;
	}

	.stage-error {
		margin: 0 16px 16px;
		padding: 12px;
		background-color: var(--red, #da373c);
		border-radius: 4px;
		color: white;
		font-size: 13px;
	}
</style>
