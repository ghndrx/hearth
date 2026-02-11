import { writable, derived, get } from 'svelte/store';
import type { Server, Channel, Member } from '$lib/types';
import { auth } from './auth';

function createServersStore() {
	const { subscribe, set, update } = writable<{
		servers: Server[];
		channels: Map<string, Channel[]>;
		members: Map<string, Member[]>;
		activeServerId: string | null;
		activeChannelId: string | null;
		loading: boolean;
	}>({
		servers: [],
		channels: new Map(),
		members: new Map(),
		activeServerId: null,
		activeChannelId: null,
		loading: false
	});

	async function fetchWithAuth(url: string, options: RequestInit = {}) {
		const token = get(auth).token;
		return fetch(url, {
			...options,
			headers: {
				...options.headers,
				Authorization: `Bearer ${token}`,
				'Content-Type': 'application/json'
			}
		});
	}

	return {
		subscribe,

		loadServers: async () => {
			update((s) => ({ ...s, loading: true }));
			try {
				const res = await fetchWithAuth('/api/servers');
				if (res.ok) {
					const servers = await res.json();
					update((s) => ({ ...s, servers, loading: false }));
				}
			} catch (e) {
				console.error('Failed to load servers:', e);
				update((s) => ({ ...s, loading: false }));
			}
		},

		loadChannels: async (serverId: string) => {
			try {
				const res = await fetchWithAuth(`/api/servers/${serverId}/channels`);
				if (res.ok) {
					const channels = await res.json();
					update((s) => {
						const newChannels = new Map(s.channels);
						newChannels.set(serverId, channels);
						return { ...s, channels: newChannels };
					});
				}
			} catch (e) {
				console.error('Failed to load channels:', e);
			}
		},

		loadMembers: async (serverId: string) => {
			try {
				const res = await fetchWithAuth(`/api/servers/${serverId}/members`);
				if (res.ok) {
					const members = await res.json();
					update((s) => {
						const newMembers = new Map(s.members);
						newMembers.set(serverId, members);
						return { ...s, members: newMembers };
					});
				}
			} catch (e) {
				console.error('Failed to load members:', e);
			}
		},

		setActiveServer: (serverId: string | null) => {
			update((s) => ({ ...s, activeServerId: serverId, activeChannelId: null }));
		},

		setActiveChannel: (channelId: string | null) => {
			update((s) => ({ ...s, activeChannelId: channelId }));
		},

		addServer: (server: Server) => {
			update((s) => ({ ...s, servers: [...s.servers, server] }));
		},

		updateServer: (server: Server) => {
			update((s) => ({
				...s,
				servers: s.servers.map((srv) => (srv.id === server.id ? server : srv))
			}));
		},

		removeServer: (serverId: string) => {
			update((s) => ({
				...s,
				servers: s.servers.filter((srv) => srv.id !== serverId),
				activeServerId: s.activeServerId === serverId ? null : s.activeServerId
			}));
		},

		addChannel: (channel: Channel) => {
			update((s) => {
				const newChannels = new Map(s.channels);
				const serverChannels = newChannels.get(channel.serverId) || [];
				newChannels.set(channel.serverId, [...serverChannels, channel]);
				return { ...s, channels: newChannels };
			});
		},

		updateChannel: (channel: Channel) => {
			update((s) => {
				const newChannels = new Map(s.channels);
				const serverChannels = newChannels.get(channel.serverId) || [];
				newChannels.set(
					channel.serverId,
					serverChannels.map((ch) => (ch.id === channel.id ? channel : ch))
				);
				return { ...s, channels: newChannels };
			});
		},

		removeChannel: (channelId: string, serverId: string) => {
			update((s) => {
				const newChannels = new Map(s.channels);
				const serverChannels = newChannels.get(serverId) || [];
				newChannels.set(
					serverId,
					serverChannels.filter((ch) => ch.id !== channelId)
				);
				return {
					...s,
					channels: newChannels,
					activeChannelId: s.activeChannelId === channelId ? null : s.activeChannelId
				};
			});
		},

		reset: () => {
			set({
				servers: [],
				channels: new Map(),
				members: new Map(),
				activeServerId: null,
				activeChannelId: null,
				loading: false
			});
		}
	};
}

export const servers = createServersStore();

export const activeServer = derived(servers, ($servers) =>
	$servers.servers.find((s) => s.id === $servers.activeServerId) || null
);

export const activeChannels = derived(servers, ($servers) =>
	$servers.activeServerId ? $servers.channels.get($servers.activeServerId) || [] : []
);

export const activeChannel = derived(servers, ($servers) => {
	if (!$servers.activeServerId || !$servers.activeChannelId) return null;
	const channels = $servers.channels.get($servers.activeServerId) || [];
	return channels.find((c) => c.id === $servers.activeChannelId) || null;
});

export const activeMembers = derived(servers, ($servers) =>
	$servers.activeServerId ? $servers.members.get($servers.activeServerId) || [] : []
);
