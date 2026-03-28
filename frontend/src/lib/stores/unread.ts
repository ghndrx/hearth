import { writable, derived, type Readable } from 'svelte/store';
import { api } from '$lib/api';

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

/**
 * Fetch unread state from the API
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
