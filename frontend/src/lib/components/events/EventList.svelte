<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { 
		events, 
		loadServerEvents, 
		formatEventTime, 
		EVENT_TYPE_LABELS,
		EVENT_STATUS_LABELS,
		type Event 
	} from '$lib/stores/events';

	export let serverId: string;
	export let showCreateButton = true;

	const dispatch = createEventDispatcher<{
		viewEvent: { event: Event };
		createEvent: void;
	}>();

	let loading = false;
	let error: string | null = null;
	let filterStatus: number | null = null;

	$: filteredEvents = filterStatus !== null
		? $events.filter(e => e.status === filterStatus)
		: $events;

	$: upcomingEvents = filteredEvents.filter(e => e.status === 1);
	$: pastEvents = filteredEvents.filter(e => e.status !== 1);

	async function loadEvents() {
		loading = true;
		error = null;
		try {
			await loadServerEvents(serverId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load events';
		} finally {
			loading = false;
		}
	}

	function handleViewEvent(event: Event) {
		dispatch('viewEvent', { event });
	}

	function handleCreateEvent() {
		dispatch('createEvent');
	}

	function formatEventDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function formatEventTimeOfDay(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
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

	$: if (serverId) {
		loadEvents();
	}
</script>

<div class="event-list">
	{#if error}
		<div class="error-banner">
			<span>{error}</span>
			<button on:click={loadEvents}>Retry</button>
		</div>
	{/if}

	<div class="list-header">
		<h2>Events</h2>
		{#if showCreateButton}
			<button class="create-btn" on:click={handleCreateEvent}>
				<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
					<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
				</svg>
				New Event
			</button>
		{/if}
	</div>

	<div class="filter-bar">
		<button
			class="filter-btn"
			class:active={filterStatus === null}
			on:click={() => filterStatus = null}
		>
			All
		</button>
		<button
			class="filter-btn"
			class:active={filterStatus === 1}
			on:click={() => filterStatus = 1}
		>
			Scheduled
		</button>
		<button
			class="filter-btn"
			class:active={filterStatus === 2}
			on:click={() => filterStatus = 2}
		>
			Active
		</button>
		<button
			class="filter-btn"
			class:active={filterStatus === 3}
			on:click={() => filterStatus = 3}
		>
			Completed
		</button>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<span>Loading events...</span>
		</div>
	{:else if filteredEvents.length === 0}
		<div class="empty-state">
			<div class="empty-icon">📅</div>
			<h3>No events scheduled</h3>
			<p>Create an event to schedule activities for your community.</p>
			{#if showCreateButton}
				<button class="btn primary" on:click={handleCreateEvent}>Create Event</button>
			{/if}
		</div>
	{:else}
		<div class="events-grid">
			{#each upcomingEvents as event (event.id)}
				<button class="event-card upcoming" on:click={() => handleViewEvent(event)}>
					{#if event.image_url}
						<div class="card-cover" style="background-image: url({event.image_url})"></div>
					{/if}
					<div class="card-content">
						<div class="card-header">
							<span class="event-type-icon">{entityTypeIcons[event.entity_type]}</span>
							<span class="upcoming-tag">{getRelativeTime(event.scheduled_start)}</span>
						</div>
						<h3 class="event-name">{event.name}</h3>
						<div class="event-time">
							<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
								<path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"/>
								<path d="M12.5 7H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
							</svg>
							<span>{formatEventDate(event.scheduled_start)} at {formatEventTimeOfDay(event.scheduled_start)}</span>
						</div>
						<div class="event-meta">
							<span class="rsvp-count">
								<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
									<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
								</svg>
								{event.user_count} interested
							</span>
						</div>
					</div>
				</button>
			{/each}

			{#each pastEvents as event (event.id)}
				<button class="event-card past" on:click={() => handleViewEvent(event)}>
					{#if event.image_url}
						<div class="card-cover" style="background-image: url({event.image_url})"></div>
					{/if}
					<div class="card-content">
						<div class="card-header">
							<span class="event-type-icon">{entityTypeIcons[event.entity_type]}</span>
							<span class="status-badge" style="color: {statusColors[event.status]}">
								{EVENT_STATUS_LABELS[event.status]}
							</span>
						</div>
						<h3 class="event-name">{event.name}</h3>
						<div class="event-time">
							<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
								<path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"/>
								<path d="M12.5 7H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
							</svg>
							<span>{formatEventDate(event.scheduled_start)} at {formatEventTimeOfDay(event.scheduled_start)}</span>
						</div>
						<div class="event-meta">
							<span class="rsvp-count">
								<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
									<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
								</svg>
								{event.user_count} interested
							</span>
						</div>
					</div>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.event-list {
		display: flex;
		flex-direction: column;
		gap: 16px;
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
	}

	.error-banner button {
		background: none;
		border: none;
		color: var(--status-danger, #da373c);
		text-decoration: underline;
		cursor: pointer;
	}

	.list-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.list-header h2 {
		margin: 0;
		font-size: 18px;
		font-weight: 600;
		color: var(--text-primary);
	}

	.create-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: 4px;
		color: white;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s;
	}

	.create-btn:hover {
		background: var(--brand-hover, #4752c4);
	}

	.filter-bar {
		display: flex;
		gap: 4px;
		padding: 4px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 6px;
		width: fit-content;
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

	.loading-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
		text-align: center;
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--bg-tertiary, #1e1f22);
		border-top-color: var(--brand-primary, #5865f2);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.loading-state span {
		margin-top: 12px;
		color: var(--text-muted, #b5bac1);
		font-size: 14px;
	}

	.empty-icon {
		font-size: 48px;
		margin-bottom: 16px;
	}

	.empty-state h3 {
		margin: 0 0 8px 0;
		font-size: 18px;
		font-weight: 600;
	}

	.empty-state p {
		margin: 0 0 16px 0;
		color: var(--text-muted, #b5bac1);
		font-size: 14px;
		max-width: 300px;
	}

	.btn.primary {
		background: var(--brand-primary, #5865f2);
		border: none;
		border-radius: 4px;
		color: white;
		padding: 10px 20px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
	}

	.btn.primary:hover {
		background: var(--brand-hover, #4752c4);
	}

	.events-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 16px;
	}

	.event-card {
		display: flex;
		flex-direction: column;
		background: var(--bg-secondary, #2b2d31);
		border: none;
		border-radius: 8px;
		overflow: hidden;
		cursor: pointer;
		transition: background-color 0.15s, transform 0.15s;
		text-align: left;
	}

	.event-card:hover {
		background: var(--bg-modifier-hover, #35373c);
		transform: translateY(-2px);
	}

	.event-card.past {
		opacity: 0.7;
	}

	.event-card.past:hover {
		opacity: 1;
	}

	.card-cover {
		width: 100%;
		height: 100px;
		background-size: cover;
		background-position: center;
	}

	.card-content {
		padding: 12px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.event-type-icon {
		font-size: 18px;
	}

	.upcoming-tag {
		padding: 2px 8px;
		background: rgba(88, 101, 242, 0.2);
		color: var(--brand-primary, #5865f2);
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
	}

	.status-badge {
		font-size: 11px;
		font-weight: 500;
		text-transform: uppercase;
	}

	.event-name {
		margin: 0;
		font-size: 15px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.event-time {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		color: var(--text-muted, #b5bac1);
	}

	.event-time svg {
		flex-shrink: 0;
	}

	.event-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-top: 4px;
	}

	.rsvp-count {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--text-faint, #6d6f78);
	}
</style>
