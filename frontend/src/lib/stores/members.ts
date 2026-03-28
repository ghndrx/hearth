/**
 * Server Members Store
 * 
 * Manages server member data including roles and membership loading.
 */

import { writable } from 'svelte/store';
import { api, ApiError } from '$lib/api';

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
		display_name: string;
		avatar: string | null;
		banner?: string | null;
		bio?: string | null;
		pronouns?: string | null;
		bot?: boolean;
		created_at?: string;
	};
}

export interface Role {
	id: string;
	name: string;
	color: string;
	position: number;
	hoist: boolean;
	permissions?: number;
}

// Backend raw response types (Color is int on backend, not hex string)
interface BackendMember {
	id?: string; // not present on backend — we use user_id
	user_id: string;
	server_id: string;
	nickname: string | null;
	roles: string[]; // UUIDs serialized as strings
	joined_at: string;
	user: {
		id: string;
		username: string;
		display_name: string | null;
		avatar: string | null;
		banner?: string | null;
		bio?: string | null;
		pronouns?: string | null;
		bot?: boolean;
		created_at?: string;
	};
}

interface BackendRole {
	id: string;
	name: string;
	color: number; // RGB integer from backend
	position: number;
	hoist: boolean;
	permissions?: number;
}

export const members = writable<Member[]>([]);
export const roles = writable<Role[]>([]);

function intToHexColor(color: number): string {
	return '#' + color.toString(16).padStart(6, '0');
}

export async function loadServerMembers(serverId: string): Promise<void> {
	try {
		const data = await api.get<BackendMember[]>(`/servers/${serverId}/members`);
		// Normalize: backend uses user_id at top level; frontend uses id as user_id alias
		const normalized: Member[] = (data || []).map(m => ({
			id: m.user_id,
			user_id: m.user_id,
			server_id: m.server_id,
			nickname: m.nickname,
			roles: m.roles || [],
			joined_at: m.joined_at,
			user: {
				id: m.user?.id || m.user_id,
				username: m.user?.username || '',
				display_name: m.user?.display_name || m.user?.username || '',
				avatar: m.user?.avatar || null,
				banner: m.user?.banner ?? null,
				bio: m.user?.bio ?? null,
				pronouns: m.user?.pronouns ?? null,
				bot: m.user?.bot ?? false,
				created_at: m.user?.created_at,
			},
		}));
		members.set(normalized);
	} catch (err) {
		console.error('[members] loadServerMembers failed:', err);
		if (err instanceof ApiError) {
			throw err;
		}
	}
}

export async function loadServerRoles(serverId: string): Promise<void> {
	try {
		const data = await api.get<BackendRole[]>(`/servers/${serverId}/roles`);
		// Normalize: backend returns color as int; frontend expects hex string
		const normalized: Role[] = (data || []).map(r => ({
			id: r.id,
			name: r.name,
			color: r.color ? intToHexColor(r.color) : '#99aab5',
			position: r.position,
			hoist: r.hoist,
			permissions: r.permissions,
		}));
		roles.set(normalized);
	} catch (err) {
		console.error('[members] loadServerRoles failed:', err);
		if (err instanceof ApiError) {
			throw err;
		}
	}
}
