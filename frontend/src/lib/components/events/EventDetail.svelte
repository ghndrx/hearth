<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { 
		type Event, 
		type EventRSVP,
		rsvpToEvent, 
		removeRsvp, 
		getEventUsers,
		deleteEvent,
		updateEvent,
		startEvent,
		formatEventTime,
		EVENT_TYPE_LABELS,
		EVENT_STATUS_LABELS
	} from '$lib/stores/events';
	import { currentServer } from '$lib/stores/servers';
	import { user as userStore } from '$lib/stores/auth';

	export let event: Event;

	const dispatch = createEventDispatcher<{
		close: void;
		updated: void;
		deleted: void;
	}>();

	let rsvps: EventRSVP[] = [];
	let loading = false;
	let error: string | null = null;
	let userHasRsvpd = false;

	$: isCreator = $userStore?.id === event.creator_id;
	$: canManage = isCreator || ($currentServer && false); // TODO: Check MANAGE_EVENTS permission

	onMount(async () => {
		await loadRsvps();
	});

	async function loadRsvps() {
		try {
			rsvps = await getEventUsers(event.id);
			const currentUserId = $userStore?.id;
			userHasRsvpd = rsvps.some(r => r.user_id === currentUserId);
		} catch (e) {
			console.error('Failed to load RSVPs:', e);
		}
	}

	async function handleRsvp() {
		loading = true;
		error = null;
		try {
			await rsvpToEvent(event.id, 1);
			userHasRsvpd = true;
			await loadRsvps();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to RSVP';
		} finally {
			loading = false;
		}
	}

	async function handleRemoveRsvp() {
		loading = true;
		error = null;
		try {
			await removeRsvp(event.id);
			userHasRsvpd = false;
			await loadRsvps();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to remove RSVP';
		} finally {
			loading = false;
		}
	}

	async function handleStartEvent() {
		loading = true;
		error = null;
		try {
			await startEvent(event.id);
			dispatch('updated');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to start event';
		} finally {
			loading = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Are you sure you want to delete this event?')) {
			return;
		}
		loading = true;
		error = null;
		try {
			await deleteEvent(event.id);
			dispatch('deleted');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete event';
		} finally {
			loading = false;
		}
	}

	function handleClose() {
		dispatch('close');
	}

	function handleOverlayClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			handleClose();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			handleClose();
		}
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString([], {
			weekday: 'long',
			year: 'numeric',
			month: 'long',
			day: 'numeric',
		});
	}

	function formatTime(dateStr: string): string {
		return new Date(dateStr).toLocaleTimeString([], {
			hour: '2-digit',
			minute: '2-digit',
		});
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="modal-overlay" on:click={handleOverlayClick}>
	<div class="modal-content">
		<div class="modal-header">
			<h2>Event Details</h2>
			<button class="close-btn" on:click={handleClose}>
				<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
					<path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
				</svg>
			</button>
		</div>

		<div class="modal-body">
			{#if event.image_url}
				<div class="event-cover" style="background-image: url({event.image_url})"></div>
			{/if}

			<div class="event-header-info">
				<span class="event-type-badge type-{event.entity_type}">
					{EVENT_TYPE_LABELS[event.entity_type]}
				</span>
				<span class="event-status status-{event.status}">
					{EVENT_STATUS_LABELS[event.status]}
				</span>
			</div>

			<h1 class="event-name">{event.name}</h1>

			{#if event.description}
				<p class="event-description">{event.description}</p>
			{/if}

			<div class="event-meta">
				<div class="meta-item">
					<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
						<path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2zm-8 4H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"/>
					</svg>
					<span>{formatDate(event.scheduled_start)}</span>
				</div>
				<div class="meta-item">
					<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
						<path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
					</svg>
					<span>{formatTime(event.scheduled_start)}</span>
				</div>
				{#if event.channel_id}
					<div class="meta-item">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 15c-1.66 0-3-1.34-3-3V8c0-1.66 1.34-3 3-3s3 1.34 3 3v6c0 1.66-1.34 3-3 3z"/>
						</svg>
						<span>Voice Channel</span>
					</div>
				{:else if event.location}
					<div class="meta-item">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>
						</svg>
						<span>{event.location}</span>
					</div>
				{/if}
			</div>

			<div class="rsvp-section">
				<div class="rsvp-header">
					<h3>Interested ({rsvps.length})</h3>
				</div>

				{#if rsvps.length > 0}
					<div class="rsvp-list">
						{#each rsvps as rsvp (rsvp.user_id)}
							<div class="rsvp-user">
								<div class="avatar-placeholder">
									{rsvp.user?.username?.[0]?.toUpperCase() || '?'}
								</div>
								<span class="username">{rsvp.user?.username || 'Unknown'}</span>
							</div>
						{/each}
					</div>
				{:else}
					<p class="no-rsvps">No one has RSVPed yet. Be the first!</p>
				{/if}
			</div>

			{#if error}
				<div class="error-message">{error}</div>
			{/if}
		</div>

		<div class="modal-footer">
			{#if event.status === 1}
				{#if userHasRsvpd}
					<button 
						class="btn-secondary" 
						on:click={handleRemoveRsvp}
						disabled={loading}
					>
						Remove RSVP
					</button>
				{:else}
					<button 
						class="btn-primary" 
						on:click={handleRsvp}
						disabled={loading}
					>
						Interested
					</button>
				{/if}

				{#if canManage}
					<button 
						class="btn-secondary" 
						on:click={handleStartEvent}
						disabled={loading}
					>
						Start Event
					</button>
					<button 
						class="btn-danger" 
						on:click={handleDelete}
						disabled={loading}
					>
						Delete
					</button>
				{/if}
			{/if}
		</div>
	</div>
</div>

<style>
	.modal-overlay {
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

	.modal-content {
		background: var(--bg-secondary);
		border-radius: 8px;
		width: 100%;
		max-width: 560px;
		max-height: 90vh;
		overflow-y: auto;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.24);
	}

	.modal-header h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.close-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 4px;
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		border-radius: 4px;
		transition: background 0.15s ease;
	}

	.close-btn:hover {
		background: rgba(79, 84, 92, 0.32);
		color: var(--text-primary);
	}

	.modal-body {
		padding: 20px;
	}

	.event-cover {
		width: calc(100% + 40px);
		margin: -20px -20px 20px -20px;
		height: 160px;
		background-size: cover;
		background-position: center;
		border-radius: 8px 8px 0 0;
	}

	.event-header-info {
		display: flex;
		gap: 8px;
		margin-bottom: 12px;
	}

	.event-type-badge,
	.event-status {
		padding: 4px 8px;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
	}

	.type-1 {
		background: rgba(88, 101, 242, 0.2);
		color: #7289da;
	}

	.type-2 {
		background: rgba(82, 196, 130, 0.2);
		color: #52c482;
	}

	.type-3 {
		background: rgba(250, 166, 26, 0.2);
		color: #faa61a;
	}

	.status-1 {
		background: rgba(88, 101, 242, 0.2);
		color: #7289da;
	}

	.status-2 {
		background: rgba(82, 196, 130, 0.2);
		color: #52c482;
	}

	.status-3 {
		background: rgba(142, 148, 161, 0.2);
		color: #8e94a1;
	}

	.status-4 {
		background: rgba(242, 63, 67, 0.2);
		color: #f23f43;
	}

	.event-name {
		margin: 0 0 12px 0;
		font-size: 22px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.event-description {
		margin: 0 0 20px 0;
		font-size: 14px;
		color: var(--text-secondary);
		line-height: 1.5;
	}

	.event-meta {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;
		background: var(--bg-primary);
		border-radius: 8px;
		margin-bottom: 20px;
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 12px;
		color: var(--text-secondary);
		font-size: 14px;
	}

	.meta-item svg {
		color: var(--text-muted);
		flex-shrink: 0;
	}

	.rsvp-section {
		margin-top: 20px;
	}

	.rsvp-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 12px;
	}

	.rsvp-header h3 {
		margin: 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.rsvp-list {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.rsvp-user {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 10px;
		background: var(--bg-primary);
		border-radius: 16px;
	}

	.avatar-placeholder {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: var(--brand-primary);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 12px;
		font-weight: 600;
		color: white;
	}

	.username {
		font-size: 13px;
		color: var(--text-primary);
	}

	.no-rsvps {
		margin: 0;
		padding: 20px;
		text-align: center;
		color: var(--text-muted);
		font-size: 14px;
		background: var(--bg-primary);
		border-radius: 8px;
	}

	.error-message {
		margin-top: 16px;
		background: rgba(242, 63, 67, 0.15);
		border: 1px solid rgba(242, 63, 67, 0.3);
		color: #f23f43;
		padding: 10px 12px;
		border-radius: 4px;
		font-size: 13px;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		padding: 16px 20px;
		border-top: 1px solid rgba(0, 0, 0, 0.24);
	}

	.btn-secondary,
	.btn-primary,
	.btn-danger {
		padding: 10px 20px;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.btn-secondary {
		background: transparent;
		border: 1px solid rgba(79, 84, 92, 0.48);
		color: var(--text-primary);
	}

	.btn-secondary:hover:not(:disabled) {
		background: rgba(79, 84, 92, 0.24);
	}

	.btn-primary {
		background: var(--brand-primary);
		border: none;
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background: var(--brand-hover);
	}

	.btn-danger {
		background: transparent;
		border: 1px solid rgba(242, 63, 67, 0.5);
		color: #f23f43;
	}

	.btn-danger:hover:not(:disabled) {
		background: rgba(242, 63, 67, 0.15);
	}

	button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
</style>
