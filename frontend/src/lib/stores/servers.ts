import { writable, derived } from 'svelte/store';
import { api } from '$lib/api';

export interface Server {
	id: string;
	name: string;
	icon: string | null;
	banner: string | null;
	description: string | null;
	owner_id: string;
	created_at: string;
}

export const servers = writable<Server[]>([]);
export const currentServer = writable<Server | null>(null);

export async function loadServers() {
	try {
		const data = await api.get('/users/@me/servers');
		servers.set(data);
	} catch (error) {
		console.error('Failed to load servers:', error);
	}
}

export async function createServer(name: string, icon?: string) {
	try {
		const server = await api.post('/servers', { name, icon });
		servers.update(s => [...s, server]);
		return server;
	} catch (error) {
		console.error('Failed to create server:', error);
		throw error;
	}
}

export async function updateServer(id: string, updates: Partial<Server>) {
	try {
		const server = await api.patch(`/servers/${id}`, updates);
		servers.update(s => s.map(srv => srv.id === id ? server : srv));
		return server;
	} catch (error) {
		console.error('Failed to update server:', error);
		throw error;
	}
}

export async function deleteServer(id: string) {
	try {
		await api.delete(`/servers/${id}`);
		servers.update(s => s.filter(srv => srv.id !== id));
		currentServer.update(s => s?.id === id ? null : s);
	} catch (error) {
		console.error('Failed to delete server:', error);
		throw error;
	}
}

export async function leaveServer(id: string) {
	try {
		await api.delete(`/servers/${id}/members/@me`);
		servers.update(s => s.filter(srv => srv.id !== id));
		currentServer.update(s => s?.id === id ? null : s);
	} catch (error) {
		console.error('Failed to leave server:', error);
		throw error;
	}
}

export async function joinServer(inviteCode: string) {
	try {
		const server = await api.post(`/invites/${inviteCode}`);
		servers.update(s => [...s, server]);
		return server;
	} catch (error) {
		console.error('Failed to join server:', error);
		throw error;
	}
}
