import { writable, derived, type Readable } from 'svelte/store';
import { api } from '$lib/api';
import { servers } from './servers';

export interface UnreadChannel {
	channel_id: string;
	has_unread: boolean;
	unread_count: number;
	mention_count: number;
}

export interface UnreadState {
	total_unread: number;
	total_mentions: number;
	channels: UnreadChannel[];
}

// Server-level unread info (from /servers/:id/unread endpoint)
export interface ServerUnreadInfo {
	server_id: string;
	has_unread: boolean;
	unread_count: number;
	mention_count: number;
}

// Map of channel_id -> UnreadChannel
const unreadMap = writable<Map<string, UnreadChannel>>(new Map());

// Raw unread state store
export const unreadStore = writable<UnreadState>({
	total_unread: 0,
	total_mentions: 0,
	channels: []
});

// Derived store: Set of channel IDs with unread messages
export const unreadChannels: Readable<Set<string>> = derived(
	unreadMap,
	($map) => {
		const unread = new Set<string>();
		for (const [channelId, state] of $map) {
			if (state.has_unread) {
				unread.add(channelId);
			}
		}
		return unread;
	}
);

// Derived store: Map of channel_id -> boolean for quick lookup
export const hasUnreadForChannel: Readable<Map<string, boolean>> = derived(
	unreadMap,
	($map) => {
		const result = new Map<string, boolean>();
		for (const [channelId, state] of $map) {
			result.set(channelId, state.has_unread);
		}
		return result;
	}
);

// Server-level unread state
const serverUnreadMap = writable<Map<string, ServerUnreadInfo>>(new Map());

// Exported server-level unread store
export const serverUnreadStore = derived(
	serverUnreadMap,
	($map) => $map
);

// Derived store: Set of server IDs with unread messages
export const unreadServerIds: Readable<Set<string>> = derived(
	serverUnreadMap,
	($map) => {
		const unread = new Set<string>();
		for (const [serverId, info] of $map) {
			if (info.has_unread) {
				unread.add(serverId);
			}
		}
		return unread;
	}
);

// Derived store: Map of server_id -> has_unread for quick lookup
export const hasUnreadForServer: Readable<Map<string, boolean>> = derived(
	serverUnreadMap,
	($map) => {
		const result = new Map<string, boolean>();
		for (const [serverId, info] of $map) {
			result.set(serverId, info.has_unread);
		}
		return result;
	}
);

// Derived store: Map of server_id -> mention_count for badges
export const mentionCountForServer: Readable<Map<string, number>> = derived(
	serverUnreadMap,
	($map) => {
		const result = new Map<string, number>();
		for (const [serverId, info] of $map) {
			result.set(serverId, info.mention_count);
		}
		return result;
	}
);

/**
 * Fetch unread state from the API (channel-level)
 */
export async function fetchUnreadState(): Promise<void> {
	try {
		const state = await api.get<UnreadState>('/users/@me/unread');
		
		unreadStore.set(state);
		
		// Update the map
		unreadMap.update(() => {
			const map = new Map<string, UnreadChannel>();
			for (const channel of state.channels) {
				map.set(channel.channel_id, channel);
			}
			return map;
		});
	} catch (error) {
		console.error('Failed to fetch unread state:', error);
	}
}

/**
 * Fetch server-level unread state for all servers the user is in
 * Calls /servers/:id/unread for each server and aggregates the results
 */
export async function fetchServerUnreadState(): Promise<void> {
	try {
		const serverList = await api.get<{ id: string }[]>('/users/@me/servers');
		
		// Fetch unread state for each server in parallel
		const results = await Promise.allSettled(
			serverList.map(async (server: { id: string }) => {
				const unread = await api.get<{ 
					total_unread: number; 
					total_mentions: number;
					channels: Array<{ channel_id: string; unread_count: number; mention_count: number }>;
				}>(`/servers/${server.id}/unread`);
				return {
					serverId: server.id,
					info: {
						server_id: server.id,
						has_unread: unread.total_unread > 0,
						unread_count: unread.total_unread,
						mention_count: unread.total_mentions
					}
				};
			})
		);

		// Update the server unread map
		serverUnreadMap.update((map) => {
			const newMap = new Map<string, ServerUnreadInfo>();
			for (const result of results) {
				if (result.status === 'fulfilled') {
					newMap.set(result.value.serverId, result.value.info);
				}
			}
			return newMap;
		});
	} catch (error) {
		console.error('Failed to fetch server unread state:', error);
	}
}

/**
 * Mark a channel as read (acknowledge all messages)
 */
export async function markChannelRead(channelId: string): Promise<void> {
	try {
		await api.post(`/channels/${channelId}/ack`);
		
		// Update local state
		unreadMap.update((map) => {
			const channel = map.get(channelId);
			if (channel) {
				map.set(channelId, {
					...channel,
					has_unread: false,
					unread_count: 0
				});
			}
			return new Map(map);
		});
		
		// Also update the full store
		unreadStore.update((state) => {
			const channelIndex = state.channels.findIndex((c) => c.channel_id === channelId);
			if (channelIndex >= 0) {
				const updatedChannels = [...state.channels];
				updatedChannels[channelIndex] = {
					...updatedChannels[channelIndex],
					has_unread: false,
					unread_count: 0
				};
				return {
					...state,
					channels: updatedChannels
				};
			}
			return state;
		});
	} catch (error) {
		console.error('Failed to mark channel as read:', error);
		throw error;
	}
}

/**
 * Update unread state for a specific channel (e.g., when a new message arrives)
 */
export function updateChannelUnread(channelId: string, hasUnread: boolean, unreadCount: number, mentionCount: number): void {
	unreadMap.update((map) => {
		map.set(channelId, {
			channel_id: channelId,
			has_unread: hasUnread,
			unread_count: unreadCount,
			mention_count: mentionCount
		});
		return new Map(map);
	});
}

/**
 * Update server-level unread state (e.g., when a new message arrives in any channel)
 */
export function updateServerUnread(serverId: string, hasUnread: boolean, unreadCount: number, mentionCount: number): void {
	serverUnreadMap.update((map) => {
		map.set(serverId, {
			server_id: serverId,
			has_unread: hasUnread,
			unread_count: unreadCount,
			mention_count: mentionCount
		});
		return new Map(map);
	});
}

/**
 * Get unread count for a specific channel
 */
export function getUnreadCount(channelId: string): number {
	let count = 0;
	unreadMap.subscribe((map) => {
		const channel = map.get(channelId);
		if (channel) {
			count = channel.unread_count;
		}
	})();
	return count;
}

/**
 * Get unread status for a specific channel (for ChannelList)
 */
export function getChannelUnread(channelId: string): boolean {
	let hasUnread = false;
	unreadMap.subscribe((map) => {
		const channel = map.get(channelId);
		hasUnread = channel ? channel.unread_count > 0 : false;
	})();
	return hasUnread;
}

/**
 * Get mention count for a specific server
 */
export function getServerMentionCount(serverId: string): number {
	let count = 0;
	serverUnreadMap.subscribe((map) => {
		const info = map.get(serverId);
		if (info) {
			count = info.mention_count;
		}
	})();
	return count;
}
