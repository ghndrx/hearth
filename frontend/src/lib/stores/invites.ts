import { writable } from 'svelte/store';
import { api } from '$lib/api';

export interface Invite {
	code: string;
	server_id: string;
	channel_id: string;
	inviter_id: string;
	max_uses: number;
	uses: number;
	max_age: number;
	temporary: boolean;
	created_at: string;
	expires_at: string | null;
	server?: {
		id: string;
		name: string;
		icon: string | null;
	};
}

export const invites = writable<Invite[]>([]);

export async function loadServerInvites(serverId: string) {
	try {
		const data = await api.get<Invite[]>(`/servers/${serverId}/invites`);
		invites.set(data);
		return data;
	} catch (error) {
		console.error('Failed to load invites:', error);
		throw error;
	}
}

export async function createInvite(channelId: string, options: {
	max_age?: number;
	max_uses?: number;
	temporary?: boolean;
} = {}): Promise<Invite> {
	try {
		const invite = await api.post<Invite>(`/channels/${channelId}/invites`, {
			max_age: options.max_age ?? 86400, // 24 hours default
			max_uses: options.max_uses ?? 0,   // unlimited default
			temporary: options.temporary ?? false
		});
		invites.update(i => [...i, invite]);
		return invite;
	} catch (error) {
		console.error('Failed to create invite:', error);
		throw error;
	}
}

export async function deleteInvite(code: string) {
	try {
		await api.delete(`/invites/${code}`);
		invites.update(i => i.filter(inv => inv.code !== code));
	} catch (error) {
		console.error('Failed to delete invite:', error);
		throw error;
	}
}

export async function getInvite(code: string): Promise<Invite> {
	try {
		return await api.get<Invite>(`/invites/${code}`);
	} catch (error) {
		console.error('Failed to get invite:', error);
		throw error;
	}
}

export async function acceptInvite(code: string) {
	try {
		return await api.post<{ id: string; name: string }>(`/invites/${code}`);
	} catch (error) {
		console.error('Failed to accept invite:', error);
		throw error;
	}
}
