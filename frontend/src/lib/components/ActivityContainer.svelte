<script lang="ts">
	import { voiceActivityState, leaveActivity, endActivity } from '$lib/stores/voiceActivity';
	import { user as authUser } from '$lib/stores/auth';
	import PokerGame from './games/PokerGame.svelte';
	import ChessGame from './games/ChessGame.svelte';
	import WatchTogether from './games/WatchTogether.svelte';

	$: activity = $voiceActivityState.currentActivity;
	$: gameState = $voiceActivityState.gameState;
	$: isCreator = activity?.creator_id === $authUser?.id;

	async function handleLeave() {
		if (activity) await leaveActivity(activity.id);
	}

	async function handleEnd() {
		if (activity) await endActivity(activity.id);
	}
</script>

{#if activity}
	<div class="activity-container">
		<div class="activity-header">
			<div class="activity-info">
				<span class="activity-type-badge">
					{#if activity.activity_type === 'poker'}🃏 Poker Night
					{:else if activity.activity_type === 'chess'}♟️ Chess
					{:else if activity.activity_type === 'watch_together'}📺 Watch Together
					{/if}
				</span>
				<span class="participant-count">
					{activity.participants?.length ?? 0}/{activity.max_participants} joined
				</span>
			</div>
			<div class="activity-controls">
				<button class="btn-leave" on:click={handleLeave}>Leave</button>
				{#if isCreator}
					<button class="btn-end" on:click={handleEnd}>End</button>
				{/if}
			</div>
		</div>

		<div class="activity-content">
			{#if activity.activity_type === 'poker'}
				<PokerGame activityId={activity.id} state={gameState} participants={activity.participants ?? []} />
			{:else if activity.activity_type === 'chess'}
				<ChessGame activityId={activity.id} state={gameState} participants={activity.participants ?? []} />
			{:else if activity.activity_type === 'watch_together'}
				<WatchTogether activityId={activity.id} state={gameState} participants={activity.participants ?? []} />
			{/if}
		</div>

		<div class="activity-participants">
			<h4>Participants</h4>
			<div class="participant-list">
				{#each activity.participants ?? [] as participant}
					<div class="participant">
						<div class="participant-avatar">
							{#if participant.avatar}
								<img src={participant.avatar} alt={participant.username} />
							{:else}
								<div class="avatar-placeholder">{participant.username[0]?.toUpperCase()}</div>
							{/if}
						</div>
						<span class="participant-name">{participant.display_name || participant.username}</span>
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}

<style>
	.activity-container {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--bg-primary, #313338);
		border-radius: 8px;
		overflow: hidden;
	}

	.activity-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 12px 16px;
		background: var(--bg-secondary, #2b2d31);
		border-bottom: 1px solid var(--border-subtle, #1e1f22);
	}

	.activity-info {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.activity-type-badge {
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.participant-count {
		font-size: 12px;
		color: var(--text-muted, #6d6f78);
	}

	.activity-controls {
		display: flex;
		gap: 8px;
	}

	.btn-leave, .btn-end {
		padding: 4px 12px;
		border: none;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.btn-leave {
		background: var(--bg-tertiary, #1e1f22);
		color: var(--text-primary, #f2f3f5);
	}

	.btn-leave:hover {
		background: var(--bg-modifier-hover, #36373d);
	}

	.btn-end {
		background: var(--status-danger, #ed4245);
		color: white;
	}

	.btn-end:hover {
		background: #c93b3e;
	}

	.activity-content {
		flex: 1;
		overflow: auto;
		padding: 16px;
	}

	.activity-participants {
		padding: 12px 16px;
		background: var(--bg-secondary, #2b2d31);
		border-top: 1px solid var(--border-subtle, #1e1f22);
	}

	.activity-participants h4 {
		font-size: 11px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-muted, #6d6f78);
		margin-bottom: 8px;
		letter-spacing: 0.02em;
	}

	.participant-list {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.participant {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.participant-avatar img,
	.avatar-placeholder {
		width: 24px;
		height: 24px;
		border-radius: 50%;
	}

	.participant-avatar img {
		object-fit: cover;
	}

	.avatar-placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--brand-primary, #5865f2);
		color: white;
		font-size: 11px;
		font-weight: 700;
	}

	.participant-name {
		font-size: 12px;
		color: var(--text-primary, #f2f3f5);
	}
</style>
