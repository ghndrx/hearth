import { writable, derived } from 'svelte/store';
import { api } from '$lib/api';
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

export interface User {
	id: string;
	username: string;
	display_name: string | null;
	avatar: string | null;
}

export const channels = writable<Channel[]>([]);
export const currentChannel = writable<Channel | null>(null);

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

export async function loadServerChannels(serverId: string) {
	try {
		const data = await api.get<Channel[]>(`/servers/${serverId}/channels`);
		channels.update(c => {
			// Remove old channels for this server, add new ones
			const other = c.filter(ch => ch.server_id !== serverId);
			return [...other, ...data];
		});
	} catch (error) {
		console.error('Failed to load channels:', error);
	}
}

export async function loadDMChannels() {
	try {
		const data = await api.get<Channel[]>('/users/@me/channels');
		channels.update(c => {
			// Remove old DM channels, add new ones
			const serverChs = c.filter(ch => ch.server_id !== null);
			return [...serverChs, ...data];
		});
	} catch (error) {
		console.error('Failed to load DM channels:', error);
	}
}

export async function createChannel(serverId: string, name: string, type: number = 0) {
	try {
		const channel = await api.post<Channel>(`/servers/${serverId}/channels`, { name, type });
		channels.update(c => [...c, channel]);
		return channel;
	} catch (error) {
		console.error('Failed to create channel:', error);
		throw error;
	}
}

export async function updateChannel(id: string, updates: Partial<Channel>) {
	try {
		const channel = await api.patch<Channel>(`/channels/${id}`, updates);
		channels.update(c => c.map(ch => ch.id === id ? channel : ch));
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
		const channel = await api.post<Channel>('/users/@me/channels', { recipient_id: userId });
		channels.update(c => {
			if (c.find(ch => ch.id === channel.id)) return c;
			return [...c, channel];
		});
		return channel;
	} catch (error) {
		console.error('Failed to create DM:', error);
		throw error;
	}
}
