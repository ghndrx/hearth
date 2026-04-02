import { writable, derived } from 'svelte/store';
import { api, ApiError } from '$lib/api';
import { currentServer } from './servers';

export interface Channel {
	id: string;
	server_id: string | null;
	name: string;
	topic: string | null;
	type: number; // 0=text, 1=dm, 2=voice, 3=group_dm, 4=category
	position: number;
	parent_id: string | null;
	slowmode: number;
	nsfw: boolean;
	e2ee_enabled: boolean;
	recipients?: User[];
	last_message_id: string | null;
	last_message_at: string | null;
}

// Backend channel format (for normalization)
interface BackendChannel {
	id: string;
	server_id?: string | null;
	guild_id?: string | null;
	name: string;
	topic?: string | null;
	type: number | string; // Backend sends string, frontend expects number
	position?: number;
	parent_id?: string | null;
	slowmode?: number;
	rate_limit_per_user?: number;
	nsfw?: boolean;
	e2ee_enabled?: boolean;
	recipients?: User[];
	last_message_id?: string | null;
	last_message_at?: string | null;
}

export interface User {
	id: string;
	username: string;
	display_name: string | null;
	avatar: string | null;
}

export const channels = writable<Channel[]>([]);
export const currentChannel = writable<Channel | null>(null);
export const channelsLoading = writable(false);
export const channelsError = writable<string | null>(null);

// Channel type mapping from backend strings to frontend numbers
const CHANNEL_TYPE_MAP: Record<string, number> = {
	'text': 0,
	'dm': 1,
	'voice': 2,
	'group_dm': 3,
	'category': 4,
	'announcement': 5,
	'forum': 6,
	'stage': 7
};

// Normalize backend channel to frontend format
function normalizeChannel(ch: BackendChannel): Channel {
	// Convert string type to numeric
	let channelType: number;
	if (typeof ch.type === 'string') {
		channelType = CHANNEL_TYPE_MAP[ch.type] ?? 0;
	} else {
		channelType = ch.type;
	}

	return {
		id: ch.id,
		server_id: ch.server_id || ch.guild_id || null,
		name: ch.name,
		topic: ch.topic ?? null,
		type: channelType,
		position: ch.position ?? 0,
		parent_id: ch.parent_id ?? null,
		slowmode: ch.slowmode ?? ch.rate_limit_per_user ?? 0,
		nsfw: ch.nsfw ?? false,
		e2ee_enabled: ch.e2ee_enabled ?? false,
		recipients: ch.recipients,
		last_message_id: ch.last_message_id ?? null,
		last_message_at: ch.last_message_at ?? null,
	};
}

// Derived store for current server's channels
export const serverChannels = derived(
	[channels, currentServer],
	([$channels, $currentServer]) => {
		if (!$currentServer) return [];
		return $channels.filter(c => c.server_id === $currentServer.id);
	}
);

// Derived store for DM channels
export const dmChannels = derived(channels, $channels => 
	$channels.filter(c => c.type === 1 || c.type === 3)
);

export async function loadServerChannels(serverId: string): Promise<Channel[]> {
	channelsLoading.set(true);
	channelsError.set(null);
	
	try {
		const data = await api.get<BackendChannel[]>(`/servers/${serverId}/channels`);
		const normalized = data.map(normalizeChannel);
		
		channels.update(c => {
			// Remove old channels for this server, add new ones
			const other = c.filter(ch => ch.server_id !== serverId);
			return [...other, ...normalized];
		});
		
		return normalized;
	} catch (error) {
		console.error('Failed to load channels for server:', serverId, error);
		if (error instanceof ApiError) {
			channelsError.set(error.message);
		}
		throw error;
	} finally {
		channelsLoading.set(false);
	}
}

export async function loadDMChannels(): Promise<Channel[]> {
	channelsLoading.set(true);
	channelsError.set(null);
	
	try {
		const data = await api.get<BackendChannel[]>('/users/@me/channels');
		const normalized = data.map(normalizeChannel);
		
		channels.update(c => {
			// Remove old DM channels, add new ones
			const serverChs = c.filter(ch => ch.server_id !== null);
			return [...serverChs, ...normalized];
		});
		
		return normalized;
	} catch (error) {
		console.error('Failed to load DM channels:', error);
		if (error instanceof ApiError) {
			channelsError.set(error.message);
		}
		throw error;
	} finally {
		channelsLoading.set(false);
	}
}

// Derived store for channels grouped by category
export const categorizedChannels = derived(
	[channels, currentServer],
	([$channels, $currentServer]) => {
		if (!$currentServer) return { categories: [], uncategorized: [] };
		const serverChs = $channels
			.filter(c => c.server_id === $currentServer.id)
			.sort((a, b) => a.position - b.position);

		const categories = serverChs.filter(c => c.type === 4);
		const uncategorized = serverChs.filter(c => c.type !== 4 && !c.parent_id);

		return {
			categories: categories.map(cat => ({
				...cat,
				channels: serverChs
					.filter(c => c.parent_id === cat.id && c.type !== 4)
					.sort((a, b) => a.position - b.position)
			})),
			uncategorized
		};
	}
);

export interface ReorderEntry {
	id: string;
	category_id: string | null;
	position: number;
}

export async function reorderChannels(entries: ReorderEntry[]) {
	try {
		await api.patch('/channels/reorder', { channels: entries });

		// Update local state optimistically
		channels.update(c => {
			const updated = [...c];
			for (const entry of entries) {
				const idx = updated.findIndex(ch => ch.id === entry.id);
				if (idx !== -1) {
					updated[idx] = {
						...updated[idx],
						position: entry.position,
						parent_id: entry.category_id
					};
				}
			}
			return updated;
		});
	} catch (error) {
		console.error('Failed to reorder channels:', error);
		throw error;
	}
}

export type ChannelTypeString = 'text' | 'voice' | 'announcement' | 'forum' | 'category';

export interface CreateChannelOptions {
	name: string;
	type?: ChannelTypeString;
	topic?: string;
	parentId?: string;
	nsfw?: boolean;
}

export async function createChannel(serverId: string, options: CreateChannelOptions) {
	try {
		const payload: {
			name: string;
			type: string;
			topic?: string;
			parent_id?: string;
			nsfw?: boolean;
		} = {
			name: options.name,
			type: options.type || 'text'
		};
		
		if (options.topic) {
			payload.topic = options.topic;
		}
		if (options.parentId) {
			payload.parent_id = options.parentId;
		}
		if (options.nsfw !== undefined) {
			payload.nsfw = options.nsfw;
		}
		
		const response = await api.post<BackendChannel>(`/servers/${serverId}/channels`, payload);
		const channel = normalizeChannel(response);
		
		channels.update(c => [...c, channel]);
		return channel;
	} catch (error) {
		console.error('Failed to create channel:', error);
		throw error;
	}
}

export async function updateChannel(id: string, updates: Partial<Pick<Channel, 'name' | 'topic' | 'position' | 'parent_id' | 'nsfw' | 'slowmode'>>) {
	try {
		// Map frontend field names to backend if needed
		const payload: Record<string, unknown> = { ...updates };
		if ('slowmode' in updates) {
			payload.rate_limit_per_user = updates.slowmode;
			delete payload.slowmode;
		}
		
		const response = await api.patch<BackendChannel>(`/channels/${id}`, payload);
		const channel = normalizeChannel(response);
		
		channels.update(c => c.map(ch => ch.id === id ? channel : ch));
		currentChannel.update(ch => ch?.id === id ? channel : ch);
		return channel;
	} catch (error) {
		console.error('Failed to update channel:', error);
		throw error;
	}
}

export async function deleteChannel(id: string) {
	try {
		await api.delete(`/channels/${id}`);
		channels.update(c => c.filter(ch => ch.id !== id));
		currentChannel.update(ch => ch?.id === id ? null : ch);
	} catch (error) {
		console.error('Failed to delete channel:', error);
		throw error;
	}
}

export async function createDM(userId: string) {
	try {
		// POST to /users/@me/channels creates/gets a DM channel
		const response = await api.post<BackendChannel>('/users/@me/channels', { recipient_id: userId });
		const channel = normalizeChannel(response);
		
		channels.update(c => {
			// Avoid duplicates
			if (c.find(ch => ch.id === channel.id)) return c;
			return [...c, channel];
		});
		return channel;
	} catch (error) {
		console.error('Failed to create DM:', error);
		throw error;
	}
}

export async function getChannel(id: string): Promise<Channel | null> {
	try {
		const response = await api.get<BackendChannel>(`/channels/${id}`);
		return normalizeChannel(response);
	} catch (error) {
		if (error instanceof ApiError && error.status === 404) {
			return null;
		}
		throw error;
	}
}

export async function createGroupDM(recipientIds: string[], name?: string) {
	try {
		const payload: { recipient_ids: string[]; name?: string } = { recipient_ids: recipientIds };
		if (name) payload.name = name;
		const response = await api.post<BackendChannel>('/dms/group', payload);
		const channel = normalizeChannel(response);

		channels.update(c => {
			if (c.find(ch => ch.id === channel.id)) return c;
			return [...c, channel];
		});
		return channel;
	} catch (error) {
		console.error('Failed to create group DM:', error);
		throw error;
	}
}

export async function addGroupDMParticipant(channelId: string, userId: string) {
	try {
		const response = await api.put<BackendChannel>(`/dms/${channelId}/participants`, { user_id: userId });
		const channel = normalizeChannel(response);
		channels.update(c => c.map(ch => ch.id === channelId ? channel : ch));
		return channel;
	} catch (error) {
		console.error('Failed to add participant to group DM:', error);
		throw error;
	}
}

export async function removeGroupDMParticipant(channelId: string, userId: string) {
	try {
		await api.delete(`/dms/${channelId}/participants?user_id=${userId}`);
		// Reload DM channels to get updated recipients
		await loadDMChannels();
	} catch (error) {
		console.error('Failed to remove participant from group DM:', error);
		throw error;
	}
}

export async function leaveDM(channelId: string) {
	try {
		await api.delete(`/dms/${channelId}/leave`);
		channels.update(c => c.filter(ch => ch.id !== channelId));
		currentChannel.update(ch => ch?.id === channelId ? null : ch);
	} catch (error) {
		console.error('Failed to leave DM:', error);
		throw error;
	}
}

export async function transferGroupDMOwnership(channelId: string, newOwnerId: string) {
	try {
		const response = await api.patch<BackendChannel>(`/dms/${channelId}/owner`, { user_id: newOwnerId });
		const channel = normalizeChannel(response);
		channels.update(c => c.map(ch => ch.id === channelId ? channel : ch));
		return channel;
	} catch (error) {
		console.error('Failed to transfer group DM ownership:', error);
		throw error;
	}
}

// Announcement channel following
export interface ChannelFollower {
	id: string;
	name: string;
	channel_id: string;
	guild_id: string;
	type: number;
	source_channel_id?: string;
	source_guild_id?: string;
}

export interface FollowChannelRequest {
	webhook_channel_id: string;
}

export async function getChannelFollowers(channelId: string): Promise<ChannelFollower[]> {
	try {
		return await api.get<ChannelFollower[]>(`/channels/${channelId}/followers`);
	} catch (error) {
		console.error('Failed to get channel followers:', error);
		throw error;
	}
}

export async function followChannel(channelId: string, targetChannelId: string): Promise<ChannelFollower> {
	try {
		return await api.post<ChannelFollower, FollowChannelRequest>(`/channels/${channelId}/followers`, {
			webhook_channel_id: targetChannelId
		});
	} catch (error) {
		console.error('Failed to follow channel:', error);
		throw error;
	}
}

export async function unfollowChannel(channelId: string, webhookId: string): Promise<void> {
	try {
		await api.delete(`/channels/${channelId}/followers/${webhookId}`);
	} catch (error) {
		console.error('Failed to unfollow channel:', error);
		throw error;
	}
}

export async function crosspostMessage(channelId: string, messageId: string) {
	try {
		return await api.post<unknown>(`/channels/${channelId}/messages/${messageId}/crosspost`);
	} catch (error) {
		console.error('Failed to crosspost message:', error);
		throw error;
	}
}
