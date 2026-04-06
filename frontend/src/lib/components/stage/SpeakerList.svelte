<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Avatar from '../Avatar.svelte';
	import type { StageParticipant } from '$lib/stores/stage';
	import { StageRole } from '$lib/stores/stage';

	export let speakers: StageParticipant[] = [];
	export let currentUserId: string = '';
	export let canManage: boolean = false;
	export let isMuted: boolean = false;

	const dispatch = createEventDispatcher();

	function handleMute(userId: string) {
		dispatch('mute', { userId });
	}

	function handleUnmute(userId: string) {
		dispatch('unmute', { userId });
	}

	function handleRemoveSpeaker(userId: string) {
		dispatch('removeSpeaker', { userId });
	}

	function getRoleLabel(role: number): string {
		switch (role) {
			case StageRole.HOST:
				return 'Host';
			case StageRole.MODERATOR:
				return 'Moderator';
			case StageRole.SPEAKER:
				return 'Speaker';
			default:
				return '';
		}
	}
</script>

<div class="speaker-list">
	<div class="speaker-list-header">
		<span class="speaker-count">{speakers.length} Speaking</span>
	</div>

	{#if speakers.length === 0}
		<div class="empty-speakers">
			<span>No one is speaking</span>
		</div>
	{:else}
		<div class="speakers">
			{#each speakers as speaker (speaker.user_id)}
				<div class="speaker-item" class:is-self={speaker.user_id === currentUserId}>
					<div class="speaker-avatar">
						<Avatar userId={speaker.user_id} size="md" />
						<div class="speaker-indicator" class:muted={speaker.is_muted}>
							{#if speaker.is_muted}
								<svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
									<path d="M7.72 21.78c.39.39 1.02.39 1.41 0L12 18.91l2.87 2.87c.39.39 1.02.39 1.41 0 .39-.39.39-1.02 0-1.41L13.41 17.5l2.87-2.87c.39-.39.39-1.02 0-1.41-.39-.39-1.02-.39-1.41 0L12 16.09l-2.87-2.87c-.39-.39-1.02-.39-1.41 0-.39.39-.39 1.02 0 1.41l2.87 2.87-2.87 2.87c-.39.39-.39 1.02 0 1.41z"/>
								</svg>
							{:else}
								<svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
									<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
								</svg>
							{/if}
						</div>
					</div>

					<div class="speaker-info">
						<span class="speaker-name">
							{#if speaker.user_id === currentUserId}
								You
							{:else}
								{speaker.user_id}
							{/if}
						</span>
						{#if getRoleLabel(speaker.role)}
							<span class="speaker-role">{getRoleLabel(speaker.role)}</span>
						{/if}
					</div>

					{#if canManage && speaker.user_id !== currentUserId}
						<div class="speaker-actions">
							{#if speaker.is_muted}
								<button
									class="action-btn"
									on:click={() => handleUnmute(speaker.user_id)}
									title="Unmute"
								>
									<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
										<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
									</svg>
								</button>
							{:else}
								<button
									class="action-btn muted"
									on:click={() => handleMute(speaker.user_id)}
									title="Mute"
								>
									<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
										<path d="M7.72 21.78c.39.39 1.02.39 1.41 0L12 18.91l2.87 2.87c.39.39 1.02.39 1.41 0 .39-.39.39-1.02 0-1.41L13.41 17.5l2.87-2.87c.39-.39.39-1.02 0-1.41-.39-.39-1.02-.39-1.41 0L12 16.09l-2.87-2.87c-.39-.39-1.02-.39-1.41 0-.39.39-.39 1.02 0 1.41l2.87 2.87-2.87 2.87c-.39.39-.39 1.02 0 1.41z"/>
									</svg>
								</button>
							{/if}

							<button
								class="action-btn danger"
								on:click={() => handleRemoveSpeaker(speaker.user_id)}
								title="Remove from stage"
							>
								<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
									<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4 11h-8c-.55 0-1-.45-1-1s.45-1 1-1h8c.55 0 1 .45 1 1s-.45 1-1 1z"/>
								</svg>
							</button>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.speaker-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.speaker-list-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 4px;
	}

	.speaker-count {
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted, #949ba4);
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.empty-speakers {
		padding: 16px;
		text-align: center;
		color: var(--text-muted, #949ba4);
		font-size: 13px;
	}

	.speakers {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.speaker-item {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 6px 8px;
		border-radius: 4px;
		transition: background-color 0.15s ease;
	}

	.speaker-item:hover {
		background-color: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
	}

	.speaker-item.is-self {
		background-color: var(--bg-modifier-selected, rgba(79, 84, 92, 0.48));
	}

	.speaker-avatar {
		position: relative;
		flex-shrink: 0;
	}

	.speaker-indicator {
		position: absolute;
		bottom: -2px;
		right: -2px;
		width: 18px;
		height: 18px;
		display: flex;
		align-items: center;
		justify-content: center;
		background-color: var(--green, #23a559);
		border-radius: 50%;
		border: 2px solid var(--bg-primary, #313338);
		color: white;
	}

	.speaker-indicator.muted {
		background-color: var(--red, #da373c);
	}

	.speaker-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.speaker-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--text-normal, #f2f3f5);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.speaker-role {
		font-size: 11px;
		color: var(--text-muted, #949ba4);
	}

	.speaker-actions {
		display: flex;
		gap: 4px;
		opacity: 0;
		transition: opacity 0.15s ease;
	}

	.speaker-item:hover .speaker-actions {
		opacity: 1;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		padding: 0;
		background: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		transition: background-color 0.15s ease, color 0.15s ease;
	}

	.action-btn:hover {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
		color: var(--text-normal, #f2f3f5);
	}

	.action-btn.muted {
		color: var(--red, #da373c);
	}

	.action-btn.muted:hover {
		background: var(--red, #da373c);
		color: white;
	}

	.action-btn.danger:hover {
		background: var(--red, #da373c);
		color: white;
	}
</style>
