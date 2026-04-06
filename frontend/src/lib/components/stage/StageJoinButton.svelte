<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import HandRaiseButton from './HandRaiseButton.svelte';

	export let isInStage: boolean = false;
	export let isSpeaker: boolean = false;
	export let hasPendingRequest: boolean = false;
	export let stageRequestToSpeak: boolean = true;
	export let stageModeratorOnly: boolean = false;
	export let disabled: boolean = false;
	export let loading: boolean = false;

	const dispatch = createEventDispatcher();

	function handleJoin() {
		if (disabled || loading || isInStage) return;
		dispatch('join');
	}

	function handleLeave() {
		if (disabled || loading || !isInStage) return;
		dispatch('leave');
	}

	function handleRequestToSpeak() {
		if (disabled || loading) return;
		dispatch('requestToSpeak');
	}

	function handleCancelRequest() {
		if (disabled || loading) return;
		dispatch('cancelRequest');
	}

	function handleToggleHand(raised: boolean) {
		if (raised) {
			dispatch('raiseHand');
		} else {
			dispatch('lowerHand');
		}
	}

	$: canRequestToSpeak = isInStage && !isSpeaker && stageRequestToSpeak && !stageModeratorOnly;
	$: canRaiseHand = isInStage && isSpeaker && !isSpeaker; // Audience can raise hand
</script>

<div class="stage-join-container">
	{#if !isInStage}
		<!-- Not in stage - show Join button -->
		<button
			class="join-btn"
			disabled={disabled || loading}
			on:click={handleJoin}
		>
			{#if loading}
				<svg class="spinner" viewBox="0 0 24 24" width="20" height="20">
					<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" fill="none" stroke-dasharray="30 70" />
				</svg>
			{:else}
				<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
					<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
				</svg>
			{/if}
			<span>Join Stage</span>
		</button>
	{:else}
		<!-- In stage - show appropriate controls -->
		<div class="stage-controls">
			{#if isSpeaker}
				<!-- Is a speaker - show muted indicator and leave -->
				<div class="speaker-controls">
					<button
						class="control-btn"
						disabled={disabled || loading}
						on:click={handleLeave}
						title="Leave stage"
					>
						<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
							<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4 11h-8c-.55 0-1-.45-1-1s.45-1 1-1h8c.55 0 1 .45 1 1s-.45 1-1 1z"/>
						</svg>
						<span>Leave</span>
					</button>
				</div>
			{:else}
				<!-- Is audience -->
				<div class="audience-controls">
					{#if hasPendingRequest}
						<!-- Has pending request to speak -->
						<button
							class="request-pending-btn"
							disabled={disabled || loading}
							on:click={handleCancelRequest}
						>
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4 11h-8c-.55 0-1-.45-1-1s.45-1 1-1h8c.55 0 1 .45 1 1s-.45 1-1 1z"/>
							</svg>
							<span>Cancel Request</span>
						</button>
					{:else if canRequestToSpeak}
						<!-- Can request to speak -->
						<button
							class="request-speak-btn"
							disabled={disabled || loading}
							on:click={handleRequestToSpeak}
						>
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M12.09 13.91c.27.27.7.27.97 0l2.47-2.47c.39-.39.11-1.06-.44-1.06h-1.77l1.89-5.66c.11-.33.05-.71-.17-.97-.22-.26-.55-.36-.87-.25L10.5 5.3c-.11-.05-.23-.08-.36-.08H7.5C6.67 5.22 6 5.89 6 6.71V13c0 .55.45 1 1 1h2.44c.55 0 1-.45 1-1v-.09H12l-.91.91z"/>
							</svg>
							<span>Request to Speak</span>
						</button>
					{/if}

					<!-- Raise hand button for audience -->
					<HandRaiseButton
						hasRaisedHand={false}
						{loading}
						on:toggle={(e) => handleToggleHand(e.detail.raised)}
					/>

					<!-- Leave button for audience -->
					<button
						class="leave-btn"
						disabled={disabled || loading}
						on:click={handleLeave}
					>
						<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
							<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4 11h-8c-.55 0-1-.45-1-1s.45-1 1-1h8c.55 0 1 .45 1 1s-.45 1-1 1z"/>
						</svg>
						<span>Leave</span>
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.stage-join-container {
		display: flex;
		align-items: center;
	}

	.join-btn,
	.request-speak-btn,
	.request-pending-btn,
	.leave-btn,
	.control-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s ease, color 0.15s ease;
	}

	.join-btn {
		background-color: var(--brand-primary, #5865f2);
		color: white;
	}

	.join-btn:hover:not(:disabled) {
		background-color: var(--brand-hover, #4752c4);
	}

	.request-speak-btn {
		background-color: var(--green, #23a559);
		color: white;
	}

	.request-speak-btn:hover:not(:disabled) {
		background-color: var(--green-hover, #1f8a47);
	}

	.request-pending-btn {
		background-color: var(--yellow, #f0b232);
		color: #000;
	}

	.request-pending-btn:hover:not(:disabled) {
		background-color: var(--yellow-hover, #dfa02e);
	}

	.leave-btn,
	.control-btn {
		background-color: var(--bg-secondary, #2b2d31);
		color: var(--text-muted, #b5bac1);
	}

	.leave-btn:hover:not(:disabled),
	.control-btn:hover:not(:disabled) {
		background-color: var(--red, #da373c);
		color: white;
	}

	button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.stage-controls {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.speaker-controls,
	.audience-controls {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.spinner {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}
</style>
