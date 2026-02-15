import { writable, derived } from 'svelte/store';
import { api } from '$lib/api';
import { currentServer } from './servers';
import { gateway } from './gateway';

export interface Member {
	id: string;
	user_id: string;
	server_id: string;
	nickname: string | null;
	roles: string[];
	joined_at: string;
	user: {
		id: string;
		username: string;
		display_name: string | null;
		avatar: string | null;
		bot: boolean;
	};
}

export interface Role {
	id: string;
	server_id: string;
	name: string;
	color: string;
	position: number;
	permissions: string;
	hoist: boolean;
	mentionable: boolean;
}

// Map of server_id -> members
const membersMap = writable<Record<string, Member[]>>({});

// Map of server_id -> roles
const rolesMap = writable<Record<string, Role[]>>({});

// Loading state
export const loadingMembers = writable<Record<string, boolean>>({});

// Derived store for current server's members
export const members = derived(
	[membersMap, currentServer],
	([$membersMap, $currentServer]) => {
		if (!$currentServer) return [];
		return $membersMap[$currentServer.id] || [];
	}
);

// Derived store for current server's roles
export const roles = derived(
	[rolesMap, currentServer],
	([$rolesMap, $currentServer]) => {
		if (!$currentServer) return [];
		return $rolesMap[$currentServer.id] || [];
	}
);

export async function loadServerMembers(serverId: string) {
	loadingMembers.update(l => ({ ...l, [serverId]: true }));
	
	try {
		const data = await api.get<Member[]>(`/servers/${serverId}/members`);
		membersMap.update(m => ({
			...m,
			[serverId]: data
		}));
	} catch (error) {
		console.error('Failed to load members:', error);
	} finally {
		loadingMembers.update(l => ({ ...l, [serverId]: false }));
	}
}

export async function loadServerRoles(serverId: string) {
	try {
		const data = await api.get<Role[]>(`/servers/${serverId}/roles`);
		rolesMap.update(r => ({
			...r,
			[serverId]: data.sort((a, b) => b.position - a.position)
		}));
	} catch (error) {
		console.error('Failed to load roles:', error);
	}
}

export async function updateMember(serverId: string, userId: string, updates: { nickname?: string; roles?: string[] }) {
	try {
		const member = await api.patch<Member>(`/servers/${serverId}/members/${userId}`, updates);
		membersMap.update(m => {
			const serverMembers = m[serverId] || [];
			return {
				...m,
				[serverId]: serverMembers.map(mem => 
					mem.user_id === userId ? member : mem
				)
			};
		});
		return member;
	} catch (error) {
		console.error('Failed to update member:', error);
		throw error;
	}
}

export async function kickMember(serverId: string, userId: string) {
	try {
		await api.delete(`/servers/${serverId}/members/${userId}`);
		membersMap.update(m => {
			const serverMembers = m[serverId] || [];
			return {
				...m,
				[serverId]: serverMembers.filter(mem => mem.user_id !== userId)
			};
		});
	} catch (error) {
		console.error('Failed to kick member:', error);
		throw error;
	}
}

export async function banMember(serverId: string, userId: string, reason?: string, deleteDays?: number) {
	try {
		await api.put(`/servers/${serverId}/bans/${userId}`, { reason, delete_message_days: deleteDays });
		membersMap.update(m => {
			const serverMembers = m[serverId] || [];
			return {
				...m,
				[serverId]: serverMembers.filter(mem => mem.user_id !== userId)
			};
		});
	} catch (error) {
		console.error('Failed to ban member:', error);
		throw error;
	}
}

// Handle gateway events for member updates
gateway.on('GUILD_MEMBER_ADD', (data) => {
	const member = data as Member;
	membersMap.update(m => {
		const serverMembers = m[member.server_id] || [];
		if (serverMembers.find(mem => mem.user_id === member.user_id)) {
			return m;
		}
		return {
			...m,
			[member.server_id]: [...serverMembers, member]
		};
	});
});

gateway.on('GUILD_MEMBER_REMOVE', (data) => {
	const { server_id, user } = data as { server_id: string; user: { id: string } };
	membersMap.update(m => {
		const serverMembers = m[server_id] || [];
		return {
			...m,
			[server_id]: serverMembers.filter(mem => mem.user_id !== user.id)
		};
	});
});

gateway.on('GUILD_MEMBER_UPDATE', (data) => {
	const member = data as Member;
	membersMap.update(m => {
		const serverMembers = m[member.server_id] || [];
		return {
			...m,
			[member.server_id]: serverMembers.map(mem =>
				mem.user_id === member.user_id ? member : mem
			)
		};
	});
});
