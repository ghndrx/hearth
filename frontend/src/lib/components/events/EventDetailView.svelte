<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { 
		type Event, 
		type EventRSVP,
		rsvpToEvent, 
		removeRsvp, 
		getEventUsers,
		deleteEvent,
		startEvent,
		EVENT_TYPE_LABELS,
		EVENT_STATUS_LABELS
	} from '$lib/stores/events';
	import { currentServer } from '$lib/stores/servers';
	import { user as userStore } from '$lib/stores/auth';
	import { hasPermission, getServerRoles } from '$lib/stores/roles';
	import { members } from '$lib/stores/members';

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
	$: isServerOwner = $currentServer && $userStore?.id === $currentServer.owner_id;

	$: currentServerRoles = $currentServer ? getServerRoles($currentServer.id) : null;
	$: userMember = $members.find(m => m.user_id === $userStore?.id);
	$: userPermissions = userMember && $currentServerRoles ?
		$currentServerRoles
			.filter(role => userMember.roles.includes(role.id))
			.flatMap(role => role.permissions || []) : [];

	$: hasManageEventsPermission = userPermissions.length > 0 && hasPermission(userPermissions, 'MANAGE_EVENTS');
	$: canManage = isCreator || isServerOwner || hasManageEventsPermission;

	const entityTypeIcons: Record<number, string> = {
		1: '🎤',
		2: '🔊',
		3: '📍'
	};

	const statusColors: Record<number, string> = {
		1: '#5865f2',
		2: '#3ba55d',
		3: '#6d6f78',
		4: '#da373c'
	};

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

	function formatFullDate(dateStr: string): string {
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

	function getRelativeTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = date.getTime() - now.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

		if (diffDays < 0) return 'Started';
		if (diffDays === 0) return 'Today';
		if (diffDays === 1) return 'Tomorrow';
		if (diffDays < 7) return `In ${diffDays} days`;
		if (diffDays < 30) return `In ${Math.floor(diffDays / 7)} weeks`;
		return `In ${Math.floor(diffDays / 30)} months`;
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="detail-overlay" on:click={handleOverlayClick} role="dialog" aria-modal="true">
	<div class="detail-view">
		<button class="close-btn" on:click={handleClose} aria-label="Close">
			<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
				<path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
			</svg>
		</button>

		{#if event.image_url}
			<div class="cover-image" style="background-image: url({event.image_url})">
				<div class="cover-overlay"></div>
			</div>
		{/if}

		<div class="detail-content">
			<div class="detail-header">
				<div class="header-badges">
					<span class="type-badge type-{event.entity_type}">
						{entityTypeIcons[event.entity_type]} {EVENT_TYPE_LABELS[event.entity_type]}
					</span>
					<span class="status-badge" style="color: {statusColors[event.status]}">
						{EVENT_STATUS_LABELS[event.status]}
					</span>
				</div>
				<h1 class="event-title">{event.name}</h1>
				{#if event.status === 1}
					<span class="upcoming-badge">{getRelativeTime(event.scheduled_start)}</span>
				{/if}
			</div>

			{#if event.description}
				<div class="description-section">
					<h3>About</h3>
					<p class="description-text">{event.description}</p>
				</div>
			{/if}

			<div class="datetime-section">
				<div class="datetime-card">
					<div class="datetime-icon">
						<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
							<path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2zm-8 4H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"/>
						</svg>
					</div>
					<div class="datetime-info">
						<span class="datetime-label">Date & Time</span>
						<span class="datetime-value">{formatFullDate(event.scheduled_start)}</span>
						<span class="datetime-time">{formatTime(event.scheduled_start)}</span>
						{#if event.scheduled_end}
							<span class="datetime-end">to {formatTime(event.scheduled_end)}</span>
						{/if}
					</div>
				</div>

				{#if event.channel_id}
					<div class="location-card">
						<div class="location-icon">
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 15c-1.66 0-3-1.34-3-3V8c0-1.66 1.34-3 3-3s3 1.34 3 3v6c0 1.66-1.34 3-3 3z"/>
							</svg>
						</div>
						<div class="location-info">
							<span class="location-label">Voice Channel</span>
							<span class="location-value">Join voice to attend</span>
						</div>
					</div>
				{:else if event.location}
					<div class="location-card">
						<div class="location-icon">
							<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
								<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>
							</svg>
						</div>
						<div class="location-info">
							<span class="location-label">Location</span>
							<span class="location-value">{event.location}</span>
						</div>
					</div>
				{/if}
			</div>

			<div class="attendees-section">
				<div class="attendees-header">
					<h3>Interested ({rsvps.length})</h3>
				</div>

				{#if rsvps.length > 0}
					<div class="attendees-grid">
						{#each rsvps as rsvp (rsvp.user_id)}
							<div class="attendee-item">
								<div class="attendee-avatar">
									{rsvp.user?.username?.[0]?.toUpperCase() || '?'}
								</div>
								<span class="attendee-name">{rsvp.user?.username || 'Unknown'}</span>
							</div>
						{/each}
					</div>
				{:else}
					<div class="no-attendees">
						<p>No one has RSVPed yet. Be the first!</p>
					</div>
				{/if}
			</div>

			{#if error}
				<div class="error-message">{error}</div>
			{/if}

			<div class="action-bar">
				{#if event.status === 1}
					{#if userHasRsvpd}
						<button 
							class="btn secondary" 
							on:click={handleRemoveRsvp}
							disabled={loading}
						>
							<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
								<path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
							</svg>
							Remove RSVP
						</button>
					{:else}
						<button 
							class="btn primary" 
							on:click={handleRsvp}
							disabled={loading}
						>
							<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
								<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
							</svg>
							Interested
						</button>
					{/if}

					{#if canManage}
						<button 
							class="btn secondary" 
							on:click={handleStartEvent}
							disabled={loading}
						>
							Start Event
						</button>
						<button 
							class="btn danger" 
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
</div>

<style>
	.detail-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.85);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 24px;
	}

	.detail-view {
		position: relative;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 12px;
		width: 100%;
		max-width: 600px;
		max-height: calc(100vh - 48px);
		overflow-y: auto;
		box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
	}

	.close-btn {
		position: absolute;
		top: 16px;
		right: 16px;
		z-index: 10;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		background: rgba(0, 0, 0, 0.5);
		border: none;
		border-radius: 50%;
		color: white;
		cursor: pointer;
		transition: background 0.15s;
	}

	.close-btn:hover {
		background: rgba(0, 0, 0, 0.7);
	}

	.cover-image {
		width: 100%;
		height: 180px;
		background-size: cover;
		background-position: center;
		border-radius: 12px 12px 0 0;
		position: relative;
	}

	.cover-overlay {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 60px;
		background: linear-gradient(transparent, rgba(0, 0, 0, 0.5));
	}

	.detail-content {
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.detail-header {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.header-badges {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.type-badge {
		padding: 4px 10px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 600;
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

	.status-badge {
		padding: 4px 10px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
	}

	.event-title {
		margin: 0;
		font-size: 24px;
		font-weight: 700;
		color: var(--text-primary, #f2f3f5);
		line-height: 1.2;
	}

	.upcoming-badge {
		display: inline-block;
		padding: 4px 10px;
		background: rgba(88, 101, 242, 0.2);
		color: var(--brand-primary, #5865f2);
		border-radius: 4px;
		font-size: 12px;
		font-weight: 500;
		width: fit-content;
	}

	.description-section h3,
	.attendees-header h3 {
		margin: 0 0 8px 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.description-text {
		margin: 0;
		font-size: 14px;
		color: var(--text-secondary, #b5bac1);
		line-height: 1.6;
	}

	.datetime-section {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.datetime-card,
	.location-card {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px;
		background: var(--bg-primary, #1e1f22);
		border-radius: 8px;
	}

	.datetime-icon,
	.location-icon {
		flex-shrink: 0;
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(88, 101, 242, 0.15);
		border-radius: 8px;
		color: var(--brand-primary, #5865f2);
	}

	.location-icon {
		background: rgba(250, 166, 26, 0.15);
		color: #faa61a;
	}

	.datetime-info,
	.location-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.datetime-label,
	.location-label {
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted, #b5bac1);
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.datetime-value {
		font-size: 14px;
		font-weight: 500;
		color: var(--text-primary, #f2f3f5);
	}

	.datetime-time,
	.datetime-end,
	.location-value {
		font-size: 13px;
		color: var(--text-secondary, #b5bac1);
	}

	.attendees-section {
		padding-top: 4px;
	}

	.attendees-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.attendee-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 12px 6px 6px;
		background: var(--bg-primary, #1e1f22);
		border-radius: 20px;
	}

	.attendee-avatar {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		background: var(--brand-primary, #5865f2);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 12px;
		font-weight: 600;
		color: white;
	}

	.attendee-name {
		font-size: 13px;
		color: var(--text-primary, #f2f3f5);
	}

	.no-attendees {
		padding: 24px;
		text-align: center;
		background: var(--bg-primary, #1e1f22);
		border-radius: 8px;
	}

	.no-attendees p {
		margin: 0;
		color: var(--text-muted, #b5bac1);
		font-size: 13px;
	}

	.error-message {
		padding: 12px;
		background: rgba(242, 63, 67, 0.15);
		border: 1px solid rgba(242, 63, 67, 0.3);
		border-radius: 6px;
		color: #f23f43;
		font-size: 13px;
	}

	.action-bar {
		display: flex;
		gap: 12px;
		padding-top: 8px;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
	}

	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 10px 20px;
		border-radius: 6px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s, opacity 0.15s;
		border: none;
	}

	.btn.primary {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.btn.primary:hover:not(:disabled) {
		background: var(--brand-hover, #4752c4);
	}

	.btn.secondary {
		background: transparent;
		border: 1px solid rgba(79, 84, 92, 0.48);
		color: var(--text-primary, #f2f3f5);
	}

	.btn.secondary:hover:not(:disabled) {
		background: rgba(79, 84, 92, 0.24);
	}

	.btn.danger {
		background: transparent;
		border: 1px solid rgba(242, 63, 67, 0.5);
		color: #f23f43;
	}

	.btn.danger:hover:not(:disabled) {
		background: rgba(242, 63, 67, 0.15);
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
