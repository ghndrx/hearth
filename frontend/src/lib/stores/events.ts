import { writable } from 'svelte/store';
import { api, ApiError } from '$lib/api';

export type EventType = 1 | 2 | 3;
export type EventStatus = 1 | 2 | 3 | 4;

export interface Event {
	id: string;
	server_id: string;
	channel_id: string | null;
	creator_id: string;
	name: string;
	description: string;
	image_url: string | null;
	scheduled_start: string;
	scheduled_end: string | null;
	entity_type: EventType;
	location: string;
	status: EventStatus;
	user_count: number;
	recurrence_rule?: unknown;
	created_at: string;
}

export interface EventRSVP {
	event_id: string;
	user_id: string;
	status: number;
	created_at: string;
	user?: PublicUser;
}

export interface PublicUser {
	id: string;
	username: string;
	avatar: string | null;
	discriminator: string;
}

export interface CreateEventRequest {
	name: string;
	description?: string;
	image_url?: string | null;
	scheduled_start: string;
	scheduled_end?: string | null;
	entity_type: EventType;
	channel_id?: string | null;
	location?: string;
	recurrence_rule?: unknown;
}

export interface UpdateEventRequest {
	name?: string;
	description?: string;
	image_url?: string | null;
	scheduled_start?: string;
	scheduled_end?: string | null;
	entity_type?: EventType;
	channel_id?: string | null;
	location?: string;
	status?: EventStatus;
	recurrence_rule?: unknown;
}

// Event type labels
export const EVENT_TYPE_LABELS: Record<EventType, string> = {
	1: 'Stage',
	2: 'Voice',
	3: 'External',
};

// Event status labels
export const EVENT_STATUS_LABELS: Record<EventStatus, string> = {
	1: 'Scheduled',
	2: 'Active',
	3: 'Completed',
	4: 'Cancelled',
};

export const events = writable<Event[]>([]);
export const currentEvent = writable<Event | null>(null);
export const eventsLoading = writable(false);
export const eventsError = writable<string | null>(null);

export async function loadServerEvents(serverId: string, status?: number): Promise<Event[]> {
	eventsLoading.set(true);
	eventsError.set(null);

	try {
		const params = status ? `?status=${status}` : '';
		const data = await api.get<Event[]>(`/servers/${serverId}/events${params}`);
		events.set(data);
		return data;
	} catch (error) {
		console.error('Failed to load server events:', error);
		if (error instanceof ApiError) {
			eventsError.set(error.message);
		}
		throw error;
	} finally {
		eventsLoading.set(false);
	}
}

export async function getEvent(eventId: string): Promise<Event | null> {
	try {
		const data = await api.get<Event>(`/events/${eventId}`);
		currentEvent.set(data);
		return data;
	} catch (error) {
		if (error instanceof ApiError && error.status === 404) {
			return null;
		}
		console.error('Failed to get event:', error);
		throw error;
	}
}

export async function createEvent(serverId: string, request: CreateEventRequest): Promise<Event> {
	try {
		const data = await api.post<Event>(`/servers/${serverId}/events`, request);
		events.update(e => [...e, data]);
		return data;
	} catch (error) {
		console.error('Failed to create event:', error);
		throw error;
	}
}

export async function updateEvent(eventId: string, request: UpdateEventRequest): Promise<Event> {
	try {
		const data = await api.patch<Event>(`/events/${eventId}`, request);
		events.update(e => e.map(ev => ev.id === eventId ? data : ev));
		currentEvent.update(ev => ev?.id === eventId ? data : ev);
		return data;
	} catch (error) {
		console.error('Failed to update event:', error);
		throw error;
	}
}

export async function deleteEvent(eventId: string): Promise<void> {
	try {
		await api.delete(`/events/${eventId}`);
		events.update(e => e.filter(ev => ev.id !== eventId));
		currentEvent.update(ev => ev?.id === eventId ? null : ev);
	} catch (error) {
		console.error('Failed to delete event:', error);
		throw error;
	}
}

export async function rsvpToEvent(eventId: string, status: number = 1): Promise<void> {
	try {
		await api.post(`/events/${eventId}/rsvp`, { status });
	} catch (error) {
		console.error('Failed to RSVP to event:', error);
		throw error;
	}
}

export async function removeRsvp(eventId: string): Promise<void> {
	try {
		await api.delete(`/events/${eventId}/rsvp`);
	} catch (error) {
		console.error('Failed to remove RSVP:', error);
		throw error;
	}
}

export async function getEventUsers(eventId: string): Promise<EventRSVP[]> {
	try {
		return await api.get<EventRSVP[]>(`/events/${eventId}/users`);
	} catch (error) {
		console.error('Failed to get event users:', error);
		throw error;
	}
}

export async function startEvent(eventId: string): Promise<void> {
	try {
		await api.post(`/events/${eventId}/start`);
	} catch (error) {
		console.error('Failed to start event:', error);
		throw error;
	}
}

export function formatEventTime(dateStr: string): string {
	const date = new Date(dateStr);
	const now = new Date();
	const diff = date.getTime() - now.getTime();
	const days = Math.floor(diff / (1000 * 60 * 60 * 24));

	if (days === 0) {
		return `Today at ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
	} else if (days === 1) {
		return `Tomorrow at ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
	} else if (days === -1) {
		return `Yesterday at ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
	} else if (days > 0 && days <= 7) {
		return `${date.toLocaleDateString([], { weekday: 'long' })} at ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
	} else {
		return date.toLocaleDateString([], { month: 'short', day: 'numeric' }) + 
			` at ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
	}
}
