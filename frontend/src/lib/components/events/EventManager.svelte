<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { api } from '$lib/api';
	import EventCreationModal from './EventCreationModal.svelte';
	import EventDetailModal from './EventDetailModal.svelte';

	export let serverId: string;
	export let canManageEvents = false;

	const dispatch = createEventDispatcher();

	interface Event {
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
	}

	// Entity types: 1=stage, 2=voice, 3=external
	const entityTypeLabels: Record<number, string> = {
		1: 'Stage',
		2: 'Voice',
		3: 'External'
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

	let events: Event[] = [];
	let loading = true;
	let error = '';
	let showCreateModal = false;
	let showDetailModal = false;
	let selectedEvent: Event | null = null;
	let filterStatus: number | null = null;
	let deleting: string | null = null;

	async function loadEvents() {
		loading = true;
		error = '';
		try {
			const params = new URLSearchParams();
			if (filterStatus !== null) {
				params.set('status', String(filterStatus));
			}
			const query = params.toString() ? `?${params.toString()}` : '';
			events = await api.get<Event[]>(`/servers/${serverId}/events${query}`);
		} catch (err) {
			error = 'Failed to load events';
			console.error('Failed to load events:', err);
		} finally {
			loading = false;
		}
	}

	function openEventDetail(event: Event) {
		selectedEvent = event;
		showDetailModal = true;
	}

	async function deleteEvent(eventId: string) {
		if (deleting) return;
		deleting = eventId;
		try {
			await api.delete(`/events/${eventId}`);
			events = events.filter(e => e.id !== eventId);
		} catch (err) {
			console.error('Failed to delete event:', err);
		} finally {
			deleting = null;
		}
	}

	function handleEventCreated(event: Event) {
		events = [event, ...events];
		showCreateModal = false;
	}

	function handleEventUpdated(event: Event) {
		events = events.map(e => e.id === event.id ? event : e);
		showDetailModal = false;
	}

	function handleEventDeleted(eventId: string) {
		events = events.filter(e => e.id !== eventId);
		showDetailModal = false;
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			month: 'short',
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

	function formatDateTime(dateStr: string): string {
		return `${formatDate(dateStr)} at ${formatTime(dateStr)}`;
	}

	function isUpcoming(dateStr: string): boolean {
		return new Date(dateStr) > new Date();
	}

	function getRelativeTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = date.getTime() - now.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

		if (diffDays < 0) return 'Past';
		if (diffDays === 0) return 'Today';
		if (diffDays === 1) return 'Tomorrow';
		if (diffDays < 7) return `In ${diffDays} days`;
		if (diffDays < 30) return `In ${Math.floor(diffDays / 7)} weeks`;
		return `In ${Math.floor(diffDays / 30)} months`;
	}

	$: if (serverId) {
		loadEvents();
	}

	$: filteredEvents = filterStatus !== null
		? events.filter(e => e.status === filterStatus)
		: events;

	$: upcomingEvents = events.filter(e => e.status === 1 && isUpcoming(e.scheduled_start));
	$: pastEvents = events.filter(e => e.status !== 1 || !isUpcoming(e.scheduled_start));
</script>

<div class="event-manager">
	<div class="event-header">
		<div class="header-top">
			<h1>Scheduled Events</h1>
			{#if canManageEvents}
				<button class="btn primary" on:click={() => (showCreateModal = true)} type="button">
					<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
						<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
					</svg>
					Create Event
				</button>
			{/if}
		</div>
		<p class="header-desc">
			Plan and schedule events for your community. Members can RSVP to show interest.
		</p>

		{#if upcomingEvents.length > 0}
			<div class="upcoming-badge">
				{upcomingEvents.length} upcoming
			</div>
		{/if}
	</div>

	{#if error}
		<div class="error-banner">
			<span>{error}</span>
			<button on:click={loadEvents}>Retry</button>
		</div>
	{/if}

	<div class="filter-bar">
		<button
			class="filter-btn"
			class:active={filterStatus === null}
			on:click={() => (filterStatus = null)}
			type="button"
		>
			All
		</button>
		<button
			class="filter-btn"
			class:active={filterStatus === 1}
			on:click={() => (filterStatus = 1)}
			type="button"
		>
			Scheduled
		</button>
		<button
			class="filter-btn"
			class:active={filterStatus === 2}
			on:click={() => (filterStatus = 2)}
			type="button"
		>
			Active
		</button>
		<button
			class="filter-btn"
			class:active={filterStatus === 3}
			on:click={() => (filterStatus = 3)}
			type="button"
		>
			Completed
		</button>
	</div>

	{#if loading}
		<div class="loading">
			<div class="spinner"></div>
			<span>Loading events...</span>
		</div>
	{:else if filteredEvents.length === 0}
		<div class="empty">
			<div class="empty-icon">📅</div>
			<h3>No events scheduled</h3>
			<p>
				{#if filterStatus !== null}
					No events with status "{statusLabels[filterStatus]}".
				{:else}
					Create an event to schedule activities for your community.
				{/if}
			</p>
			{#if canManageEvents && filterStatus === null}
				<button class="btn primary" on:click={() => (showCreateModal = true)} type="button">
					Create Event
				</button>
			{/if}
		</div>
	{:else}
		<div class="events-list">
			{#each filteredEvents as event (event.id)}
				<div class="event-card" on:click={() => openEventDetail(event)} on:keypress role="button" tabindex="0">
					<div class="event-icon" style="background: {statusColors[event.status]}20; color: {statusColors[event.status]}">
						{entityTypeIcons[event.entity_type] || '📅'}
					</div>
					<div class="event-info">
						<div class="event-name-row">
							<span class="event-name">{event.name}</span>
							<span class="event-status" style="color: {statusColors[event.status]}">
								{statusLabels[event.status]}
							</span>
						</div>
						{#if event.description}
							<span class="event-desc">{event.description}</span>
						{/if}
						<div class="event-meta">
							<span class="meta-item">
								<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
									<path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"/>
									<path d="M12.5 7H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
								</svg>
								{formatDateTime(event.scheduled_start)}
							</span>
							<span class="meta-item">
								<span class="event-type-badge">{entityTypeLabels[event.entity_type]}</span>
							</span>
							{#if event.entity_type === 3 && event.location}
								<span class="meta-item">
									<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
										<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>
									</svg>
									{event.location}
								</span>
							{/if}
						</div>
						{#if event.status === 1 && isUpcoming(event.scheduled_start)}
							<span class="upcoming-tag">{getRelativeTime(event.scheduled_start)}</span>
						{/if}
					</div>
					<div class="event-actions">
						<span class="rsvp-count" title="{event.user_count} interested">
							<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
								<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
							</svg>
							{event.user_count}
						</span>
						{#if canManageEvents}
							<button
								class="action-btn danger"
								on:click|stopPropagation={() => deleteEvent(event.id)}
								disabled={deleting === event.id}
								title="Delete event"
								type="button"
							>
								{#if deleting === event.id}
									<span class="spinner-sm"></span>
								{:else}
									<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
										<path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
									</svg>
								{/if}
							</button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showCreateModal}
	<EventCreationModal
		{serverId}
		on:close={() => (showCreateModal = false)}
		on:created={(e) => handleEventCreated(e.detail)}
	/>
{/if}

{#if showDetailModal && selectedEvent}
	<EventDetailModal
		event={selectedEvent}
		{canManageEvents}
		on:close={() => { showDetailModal = false; selectedEvent = null; }}
		on:updated={(e) => handleEventUpdated(e.detail)}
		on:deleted={(e) => handleEventDeleted(e.detail)}
		on:rsvpUpdated={loadEvents}
	/>
{/if}

<style>
	.event-manager {
		padding: 16px;
	}

	.event-header {
		margin-bottom: 16px;
	}

	.header-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 8px;
	}

	.event-header h1 {
		margin: 0;
		font-size: 20px;
		font-weight: 600;
	}

	.header-desc {
		margin: 0 0 12px 0;
		font-size: 14px;
		color: var(--text-muted, #b5bac1);
	}

	.upcoming-badge {
		display: inline-block;
		padding: 4px 10px;
		background: var(--brand-primary, #5865f2);
		color: white;
		border-radius: 12px;
		font-size: 12px;
		font-weight: 500;
	}

	.error-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 12px;
		background: rgba(218, 55, 60, 0.1);
		border: 1px solid rgba(218, 55, 60, 0.3);
		border-radius: 4px;
		color: var(--status-danger, #da373c);
		font-size: 13px;
		margin-bottom: 16px;
	}

	.error-banner button {
		background: none;
		border: none;
		color: var(--status-danger, #da373c);
		text-decoration: underline;
		cursor: pointer;
	}

	.filter-bar {
		display: flex;
		gap: 4px;
		margin-bottom: 16px;
		padding: 4px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 6px;
	}

	.filter-btn {
		padding: 6px 12px;
		background: transparent;
		border: none;
		border-radius: 4px;
		color: var(--text-muted, #b5bac1);
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background-color 0.15s, color 0.15s;
	}

	.filter-btn:hover {
		background: var(--bg-modifier-hover, #35373c);
		color: var(--text-normal, #f2f3f5);
	}

	.filter-btn.active {
		background: var(--bg-modifier-selected, #404249);
		color: var(--text-normal, #f2f3f5);
	}

	.loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		color: var(--text-muted, #b5bac1);
		gap: 12px;
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--bg-tertiary, #1e1f22);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.spinner-sm {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		text-align: center;
	}

	.empty-icon {
		font-size: 48px;
		margin-bottom: 16px;
	}

	.empty h3 {
		margin: 0 0 8px 0;
		font-size: 18px;
		font-weight: 600;
	}

	.empty p {
		margin: 0 0 16px 0;
		color: var(--text-muted, #b5bac1);
		font-size: 14px;
		max-width: 300px;
	}

	.events-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.event-card {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 12px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		cursor: pointer;
		transition: background-color 0.15s;
	}

	.event-card:hover {
		background: var(--bg-modifier-hover, #35373c);
	}

	.event-icon {
		flex-shrink: 0;
		width: 48px;
		height: 48px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 8px;
		font-size: 20px;
	}

	.event-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.event-name-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.event-name {
		font-size: 14px;
		font-weight: 600;
		color: var(--text-normal, #f2f3f5);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.event-status {
		font-size: 11px;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.02em;
	}

	.event-desc {
		font-size: 13px;
		color: var(--text-muted, #b5bac1);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.event-meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px;
		margin-top: 4px;
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--text-faint, #6d6f78);
	}

	.meta-item svg {
		flex-shrink: 0;
	}

	.event-type-badge {
		padding: 2px 6px;
		background: var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
	}

	.upcoming-tag {
		display: inline-block;
		margin-top: 4px;
		padding: 2px 8px;
		background: rgba(88, 101, 242, 0.2);
		color: var(--brand-primary, #5865f2);
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
		width: fit-content;
	}

	.event-actions {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.rsvp-count {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 13px;
		color: var(--text-muted, #b5bac1);
	}

	.action-btn {
		width: 32px;
		height: 32px;
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

	.action-btn:hover {
		background: var(--bg-modifier-hover, #35373c);
		color: var(--text-normal, #f2f3f5);
	}

	.action-btn.danger:hover {
		background: rgba(218, 55, 60, 0.1);
		color: var(--status-danger, #da373c);
	}

	.action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-width: 96px;
		min-height: 38px;
		padding: 8px 16px;
		border-radius: 3px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		border: none;
		transition: background-color 0.1s ease;
	}

	.btn.primary {
		background: var(--blurple, #5865f2);
		color: white;
	}

	.btn.primary:hover {
		background: var(--blurple-hover, #4752c4);
	}
</style>
