<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { voiceActivityState, startActivity, type VoiceActivityType } from '$lib/stores/voiceActivity';
	import { voiceState } from '$lib/stores/voice';

	const dispatch = createEventDispatcher<{
		started: { activityId: string };
	}>();

	let loading = false;

	const activities: { type: VoiceActivityType; name: string; description: string; icon: string; maxPlayers: string }[] = [
		{ type: 'poker', name: 'Poker Night', description: 'Texas Hold\'em with friends', icon: '🃏', maxPlayers: '2-8 players' },
		{ type: 'chess', name: 'Chess', description: '1v1 chess with spectators', icon: '♟️', maxPlayers: '2 players' },
		{ type: 'watch_together', name: 'Watch Together', description: 'Sync video playback', icon: '📺', maxPlayers: 'Up to 50' },
	];

	async function launchActivity(type: VoiceActivityType) {
		if (!$voiceState.channelId || loading) return;
		loading = true;
		const result = await startActivity($voiceState.channelId, type);
		loading = false;
		if (result) {
			dispatch('started', { activityId: result.id });
		}
	}
</script>

<div class="activity-launcher">
	<h3 class="launcher-title">Start an Activity</h3>
	<div class="activity-grid">
		{#each activities as activity}
			<button
				class="activity-card"
				on:click={() => launchActivity(activity.type)}
				disabled={loading || !$voiceState.channelId}
			>
				<span class="activity-icon">{activity.icon}</span>
				<span class="activity-name">{activity.name}</span>
				<span class="activity-desc">{activity.description}</span>
				<span class="activity-players">{activity.maxPlayers}</span>
			</button>
		{/each}
	</div>
	{#if $voiceActivityState.error}
		<div class="activity-error">{$voiceActivityState.error}</div>
	{/if}
</div>

<style>
	.activity-launcher {
		padding: 16px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
	}

	.launcher-title {
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		margin-bottom: 12px;
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.activity-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
		gap: 8px;
	}

	.activity-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 16px 12px;
		background: var(--bg-tertiary, #1e1f22);
		border: 1px solid transparent;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		gap: 4px;
	}

	.activity-card:hover:not(:disabled) {
		background: var(--bg-modifier-hover, #36373d);
		border-color: var(--brand-primary, #5865f2);
	}

	.activity-card:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.activity-icon {
		font-size: 32px;
		margin-bottom: 4px;
	}

	.activity-name {
		font-size: 13px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.activity-desc {
		font-size: 11px;
		color: var(--text-secondary, #b5bac1);
		text-align: center;
	}

	.activity-players {
		font-size: 10px;
		color: var(--text-muted, #6d6f78);
		margin-top: 2px;
	}

	.activity-error {
		margin-top: 8px;
		padding: 8px;
		background: var(--status-danger-bg, rgba(237, 66, 69, 0.1));
		color: var(--status-danger, #ed4245);
		border-radius: 4px;
		font-size: 12px;
	}
</style>
