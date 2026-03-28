<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { api } from '$lib/api';
	import Modal from '../Modal.svelte';

	export let event: {
		id: string;
		server_id: string;
		channel_id: string | null;
		creator_id: string;
		name: string;
		description: string;
		image_url: string | null;
		scheduled_start: string;
		scheduled_end: string | null;
		entity_type: number;
		location: string;
		status: number;
		user_count: number;
		created_at: string;
	};

	export let canManageEvents = false;

	const dispatch = createEventDispatcher<{
		close: void;
		updated: typeof event;
		deleted: string;
		rsvpUpdated: void;
	}>();

	interface EventUser {
		event_id: string;
		user_id: string;
		status: number;
		user?: {
			id: string;
			username: string;
			avatar: string | null;
		};
	}

	// Entity types: 1=stage, 2=voice, 3=external
	const entityTypeLabels: Record<number, string> = {
		1: 'Stage Event',
		2: 'Voice Event',
		3: 'External Event'
	};

	const entityTypeIcons: Record<number, string> = {
		1: '🎤',
		2: '🔊',
		3: '📍'
	};

	// Status: 1=scheduled, 2=active, 3=completed, 4=cancelled
	const statusLabels: Record<number, string> = {
		1: 'Scheduled',
		2: 'Active',
		3: 'Completed',
		4: 'Cancelled'
	};

	const statusColors: Record<number, string> = {
		1: '#5865f2',
		2: '#3ba55d',
		3: '#6d6f78',
		4: '#da373c'
	};

	// RSVP statuses: 1=interested, 2=going
	const rsvpLabels: Record<number, string> = {
		1: 'Interested',
		2: 'Going'
	};

	let rsvpUsers: EventUser[] = [];
	let loadingUsers = false;
	let currentRsvpStatus: number | null = null;
	let isRsvping = false;
	let isStarting = false;

	async function loadEventUsers() {
		loadingUsers = true;
		try {
			rsvpUsers = await api.get<EventUser[]>(`/events/${event.id}/users`);
		} catch (err) {
			console.error('Failed to load event users:', err);
		} finally {
			loadingUsers = false;
		}
	}

	async function handleRsvp(status: number) {
		isRsvping = true;
		try {
			await api.post(`/events/${event.id}/rsvp`, { status });
			currentRsvpStatus = status;
			dispatch('rsvpUpdated');
			await loadEventUsers();
		} catch (err) {
			console.error('Failed to RSVP:', err);
		} finally {
			isRsvping = false;
		}
	}

	async function handleRemoveRsvp() {
		isRsvping = true;
		try {
			await api.delete(`/events/${event.id}/rsvp`);
			currentRsvpStatus = null;
			dispatch('rsvpUpdated');
			await loadEventUsers();
		} catch (err) {
			console.error('Failed to remove RSVP:', err);
		} finally {
			isRsvping = false;
		}
	}

	async function handleStartEvent() {
		isStarting = true;
		try {
			await api.post(`/events/${event.id}/start`);
			event = { ...event, status: 2 };
			dispatch('updated', event);
		} catch (err) {
			console.error('Failed to start event:', err);
		} finally {
			isStarting = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Are you sure you want to delete this event?')) return;

		try {
			await api.delete(`/events/${event.id}`);
			dispatch('deleted', event.id);
		} catch (err) {
			console.error('Failed to delete event:', err);
		}
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			weekday: 'long',
			month: 'long',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleTimeString('en-US', {
			hour: 'numeric',
			minute: '2-digit',
			hour12: true
		});
	}

	function isUpcoming(dateStr: string): boolean {
		return new Date(dateStr) > new Date();
	}

	function getInitials(username: string): string {
		return username
			.split(/[\s_-]/)
			.map(part => part[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function getAvatarColor(userId: string): string {
		const colors = ['#5865f2', '#57f287', '#fee75c', '#eb459e', '#ed4245', '#3ba55d'];
		let hash = 0;
		for (let i = 0; i < userId.length; i++) {
			hash = userId.charCodeAt(i) + ((hash << 5) - hash);
		}
		return colors[Math.abs(hash) % colors.length];
	}

	$: if (event) {
		loadEventUsers();
	}

	$: canStart = event.status === 1 && isUpcoming(event.scheduled_start);
</script>

<Modal open={true} title="" size="large" on:close={() => dispatch('close')}>
	<div class="event-detail">
		{#if event.image_url}
			<div class="event-banner">
				<img src={event.image_url} alt={event.name} />
				<div class="banner-overlay">
					<span class="entity-badge">
						{entityTypeIcons[event.entity_type]} {entityTypeLabels[event.entity_type]}
					</span>
				</div>
			</div>
		{/if}

		<div class="event-content">
			<div class="event-header-row">
				<div class="event-title-section">
					{#if !event.image_url}
						<span class="entity-badge no-banner">
							{entityTypeIcons[event.entity_type]} {entityTypeLabels[event.entity_type]}
						</span>
					{/if}
					<h2 class="event-title">{event.name}</h2>
					<span class="status-badge" style="color: {statusColors[event.status]}">
						{statusLabels[event.status]}
					</span>
				</div>
				{#if canManageEvents}
					<button class="delete-btn" on:click={handleDelete} type="button" title="Delete event">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
						</svg>
					</button>
				{/if}
			</div>

			<div class="event-meta-grid">
				<div class="meta-item">
					<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
						<path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"/>
						<path d="M12.5 7H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
					</svg>
					<div class="meta-text">
						<span class="meta-label">Starts</span>
						<span class="meta-value">{formatDate(event.scheduled_start)}</span>
						<span class="meta-subvalue">{formatTime(event.scheduled_start)}</span>
					</div>
				</div>

				{#if event.scheduled_end}
					<div class="meta-item">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"/>
							<path d="M12.5 7H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
						</svg>
						<div class="meta-text">
							<span class="meta-label">Ends</span>
							<span class="meta-value">{formatDate(event.scheduled_end)}</span>
							<span class="meta-subvalue">{formatTime(event.scheduled_end)}</span>
						</div>
					</div>
				{/if}

				{#if event.entity_type === 3 && event.location}
					<div class="meta-item">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>
						</svg>
						<div class="meta-text">
							<span class="meta-label">Location</span>
							<span class="meta-value">{event.location}</span>
						</div>
					</div>
				{/if}

				<div class="meta-item">
					<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
						<path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
					</svg>
					<div class="meta-text">
						<span class="meta-label">Attending</span>
						<span class="meta-value">{event.user_count} {event.user_count === 1 ? 'person' : 'people'}</span>
					</div>
				</div>
			</div>

			{#if event.description}
				<div class="event-description">
					<h3>About</h3>
					<p>{event.description}</p>
				</div>
			{/if}

			{#if event.status === 1}
				<div class="rsvp-section">
					<h3>Your RSVP</h3>
					<div class="rsvp-buttons">
						<button
							class="rsvp-btn"
							class:selected={currentRsvpStatus === 1}
							on:click={() => handleRsvp(1)}
							disabled={isRsvping}
							type="button"
						>
							<span class="rsvp-icon">✓</span>
							<span class="rsvp-label">Interested</span>
						</button>
						<button
							class="rsvp-btn"
							class:selected={currentRsvpStatus === 2}
							on:click={() => handleRsvp(2)}
							disabled={isRsvping}
							type="button"
						>
							<span class="rsvp-icon">🎉</span>
							<span class="rsvp-label">Going</span>
						</button>
						{#if currentRsvpStatus !== null}
							<button
								class="rsvp-remove"
								on:click={handleRemoveRsvp}
								disabled={isRsvping}
								type="button"
							>
								Remove RSVP
							</button>
						{/if}
					</div>
				</div>
			{/if}

			{#if canStart}
				<div class="start-section">
					<button
						class="start-btn"
						on:click={handleStartEvent}
						disabled={isStarting}
						type="button"
					>
						{isStarting ? 'Starting...' : 'Start Event Early'}
					</button>
				</div>
			{/if}

			{#if rsvpUsers.length > 0}
				<div class="attendees-section">
					<h3>People Interested ({rsvpUsers.length})</h3>
					<div class="attendees-grid">
						{#each rsvpUsers as rsvp (rsvp.user_id)}
							<div class="attendee-card">
								<div
									class="attendee-avatar"
									style="background: {getAvatarColor(rsvp.user_id)}"
								>
									{rsvp.user ? getInitials(rsvp.user.username) : '??'}
								</div>
								<div class="attendee-info">
									<span class="attendee-name">
										{rsvp.user?.username || 'Unknown User'}
									</span>
									<span class="rsvp-status">{rsvpLabels[rsvp.status]}</span>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	</div>
</Modal>

<style>
	.event-detail {
		display: flex;
		flex-direction: column;
	}

	.event-banner {
		position: relative;
		width: 100%;
		height: 160px;
		border-radius: 8px 8px 0 0;
		overflow: hidden;
		margin: -16px -16px 0 -16px;
	}

	.event-banner img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.banner-overlay {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		padding: 12px 16px;
		background: linear-gradient(transparent, rgba(0, 0, 0, 0.7));
	}

	.entity-badge {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		background: rgba(0, 0, 0, 0.6);
		color: white;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 500;
		backdrop-filter: blur(4px);
	}

	.entity-badge.no-banner {
		background: var(--bg-secondary, #2b2d31);
		margin-bottom: 8px;
	}

	.event-content {
		padding: 16px;
	}

	.event-header-row {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 16px;
	}

	.event-title-section {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.event-title {
		margin: 0;
		font-size: 22px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
	}

	.status-badge {
		font-size: 12px;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.delete-btn {
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		cursor: pointer;
		transition: background-color 0.15s, color 0.15s;
	}

	.delete-btn:hover {
		background: rgba(218, 55, 60, 0.1);
		color: var(--status-danger, #da373c);
	}

	.event-meta-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 16px;
		margin-bottom: 20px;
		padding: 16px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
	}

	.meta-item {
		display: flex;
		align-items: flex-start;
		gap: 10px;
	}

	.meta-item svg {
		flex-shrink: 0;
		color: var(--text-muted, #b5bac1);
		margin-top: 2px;
	}

	.meta-text {
		display: flex;
		flex-direction: column;
	}

	.meta-label {
		font-size: 11px;
		font-weight: 500;
		color: var(--text-faint, #6d6f78);
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.meta-value {
		font-size: 14px;
		font-weight: 500;
		color: var(--text-normal, #f2f3f5);
	}

	.meta-subvalue {
		font-size: 13px;
		color: var(--text-muted, #b5bac1);
	}

	.event-description {
		margin-bottom: 20px;
	}

	.event-description h3 {
		margin: 0 0 8px 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
	}

	.event-description p {
		margin: 0;
		font-size: 14px;
		color: var(--text-muted, #b5bac1);
		line-height: 1.6;
		white-space: pre-wrap;
	}

	.rsvp-section {
		margin-bottom: 20px;
	}

	.rsvp-section h3 {
		margin: 0 0 12px 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
	}

	.rsvp-buttons {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.rsvp-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 10px 16px;
		background: var(--bg-secondary, #2b2d31);
		border: 2px solid transparent;
		border-radius: 8px;
		color: var(--text-normal, #f2f3f5);
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: border-color 0.15s, background-color 0.15s;
	}

	.rsvp-btn:hover {
		background: var(--bg-modifier-hover, #35373c);
	}

	.rsvp-btn.selected {
		border-color: var(--brand-primary, #5865f2);
		background: rgba(88, 101, 242, 0.1);
	}

	.rsvp-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.rsvp-icon {
		font-size: 16px;
	}

	.rsvp-remove {
		padding: 10px 16px;
		background: transparent;
		border: none;
		color: var(--text-muted, #b5bac1);
		font-size: 13px;
		text-decoration: underline;
		cursor: pointer;
	}

	.rsvp-remove:hover {
		color: var(--status-danger, #da373c);
	}

	.rsvp-remove:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.start-section {
		margin-bottom: 20px;
	}

	.start-btn {
		width: 100%;
		padding: 12px;
		background: var(--status-success, #3ba55d);
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.start-btn:hover:not(:disabled) {
		background: #2d8a47;
	}

	.start-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.attendees-section {
		border-top: 1px solid var(--bg-secondary, #2b2d31);
		padding-top: 16px;
	}

	.attendees-section h3 {
		margin: 0 0 12px 0;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
	}

	.attendees-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 8px;
		max-height: 200px;
		overflow-y: auto;
	}

	.attendee-card {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 6px;
	}

	.attendee-avatar {
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 50%;
		color: white;
		font-size: 12px;
		font-weight: 600;
		flex-shrink: 0;
	}

	.attendee-info {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.attendee-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--text-normal, #f2f3f5);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.rsvp-status {
		font-size: 11px;
		color: var(--text-muted, #b5bac1);
	}
</style>
