<script lang="ts">
	import { onMount, createEventDispatcher } from 'svelte';
	import { 
		events, 
		loadServerEvents, 
		formatEventTime, 
		EVENT_TYPE_LABELS,
		type Event 
	} from '$lib/stores/events';
	import { currentServer } from '$lib/stores/servers';
	import EventCreateModal from './EventCreateModal.svelte';
	import EventDetail from './EventDetail.svelte';

	const dispatch = createEventDispatcher<{
		createEvent: void;
		viewEvent: { event: Event };
	}>();

	let upcomingEvents: Event[] = [];
	let showCreateModal = false;
	let showEventDetail = false;
	let selectedEvent: Event | null = null;
	let loading = false;
	let error: string | null = null;

	$: if ($currentServer) {
		loadUpcomingEvents($currentServer.id);
	}

	// Subscribe to events store
	$: upcomingEvents = $events.filter(e => e.status === 1).slice(0, 3);

	async function loadUpcomingEvents(serverId: string) {
		loading = true;
		error = null;
		try {
			// Only load scheduled events
			await loadServerEvents(serverId, 1);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load events';
		} finally {
			loading = false;
		}
	}

	function handleViewEvent(event: Event) {
		selectedEvent = event;
		showEventDetail = true;
		dispatch('viewEvent', { event });
	}

	function handleCloseDetail() {
		showEventDetail = false;
		selectedEvent = null;
	}

	function handleEventCreated() {
		showCreateModal = false;
		if ($currentServer) {
			loadUpcomingEvents($currentServer.id);
		}
	}
</script>

{#if upcomingEvents.length > 0}
	<div class="event-banner">
		<div class="event-banner-header">
			<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
				<path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2zm-8 4H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"/>
			</svg>
			<span class="event-banner-title">Upcoming Events</span>
			<button class="create-event-btn" on:click={() => showCreateModal = true}>
				<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
					<path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
				</svg>
				New
			</button>
		</div>

		{#each upcomingEvents as event (event.id)}
			<button class="event-item" on:click={() => handleViewEvent(event)}>
				<div class="event-info">
					<span class="event-name">{event.name}</span>
					<span class="event-time">{formatEventTime(event.scheduled_start)}</span>
				</div>
				<span class="event-type-badge type-{event.entity_type}">
					{EVENT_TYPE_LABELS[event.entity_type]}
				</span>
			</button>
		{/each}
	</div>
{/if}

{#if showCreateModal && $currentServer}
	<EventCreateModal 
		serverId={$currentServer.id} 
		on:close={() => showCreateModal = false}
		on:created={handleEventCreated}
	/>
{/if}

{#if showEventDetail && selectedEvent}
	<EventDetail 
		event={selectedEvent} 
		on:close={handleCloseDetail}
		on:updated={() => $currentServer && loadUpcomingEvents($currentServer.id)}
		on:deleted={() => { handleCloseDetail(); if ($currentServer) loadUpcomingEvents($currentServer.id); }}
	/>
{/if}

<style>
	.event-banner {
		background: var(--bg-secondary);
		border-bottom: 1px solid rgba(0, 0, 0, 0.24);
		padding: 8px 8px;
	}

	.event-banner-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 8px;
		color: var(--text-primary);
		font-size: 13px;
		font-weight: 600;
	}

	.event-banner-header svg {
		color: var(--text-muted);
	}

	.event-banner-title {
		flex: 1;
	}

	.create-event-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 4px 8px;
		background: var(--brand-primary);
		border: none;
		border-radius: 3px;
		color: white;
		font-size: 12px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.create-event-btn:hover {
		background: var(--brand-hover);
	}

	.event-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 8px 10px;
		margin-top: 2px;
		background: rgba(79, 84, 92, 0.16);
		border: none;
		border-radius: 4px;
		cursor: pointer;
		transition: background 0.15s ease;
		text-align: left;
	}

	.event-item:hover {
		background: rgba(79, 84, 92, 0.32);
	}

	.event-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.event-name {
		color: var(--text-primary);
		font-size: 13px;
		font-weight: 500;
	}

	.event-time {
		color: var(--text-muted);
		font-size: 11px;
	}

	.event-type-badge {
		padding: 2px 6px;
		border-radius: 3px;
		font-size: 10px;
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
</style>
