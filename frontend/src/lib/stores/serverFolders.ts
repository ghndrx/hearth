import { writable, derived, get } from 'svelte/store';
import { api, ApiError } from '$lib/api';
import { servers } from './servers';
import type { Server } from './servers';

export interface ServerFolder {
	id: string;
	user_id: string;
	parent_id: string | null;
	name: string;
	position: number;
	is_collapsed: boolean;
	depth: number;
	children?: ServerFolder[];
	servers?: ServerInFolder[];
	created_at: string;
	updated_at: string;
}

export interface ServerInFolder {
	server_id: string;
	folder_id: string | null;
	position: number;
	assigned_at: string;
	server?: Server;
}

export interface ServerFolderTree {
	folders: ServerFolder[];
	servers: ServerInFolder[]; // Servers not in any folder
}

export interface CreateFolderRequest {
	name: string;
	parent_id?: string;
	position?: number;
}

export interface UpdateFolderRequest {
	name?: string;
	position?: number;
	is_collapsed?: boolean;
	parent_id?: string | null;
}

export interface MoveServersRequest {
	server_ids: string[];
	folder_id?: string | null;
}

export interface ServerPosition {
	server_id: string;
	position: number;
}

export interface ReorderServersRequest {
	folder_id?: string | null;
	server_positions: ServerPosition[];
}

// Store for the folder tree
export const serverFolderTree = writable<ServerFolderTree | null>(null);
export const serverFoldersLoading = writable(false);
export const serverFoldersError = writable<string | null>(null);

// Derived store for folders with their servers (flat list for rendering)
export const serverFolders = derived(serverFolderTree, ($tree) => $tree?.folders ?? []);

// Derived store for unassigned servers (not in any folder)
export const unassignedServers = derived(
	[serverFolderTree, servers],
	([$tree, $servers]) => {
		if (!$tree) return $servers;
		
		// Get all server IDs that are in folders
		const folderServerIds = new Set<string>();
		const collectServerIds = (folders: ServerFolder[]) => {
			for (const folder of folders) {
				if (folder.servers) {
					for (const sif of folder.servers) {
						folderServerIds.add(sif.server_id);
					}
				}
				if (folder.children) {
					collectServerIds(folder.children);
				}
			}
		};
		collectServerIds($tree.folders);
		
		// Return servers not in any folder
		return $servers.filter(s => !folderServerIds.has(s.id));
	}
);

function normalizeServerInFolder(sif: any): ServerInFolder {
	return {
		server_id: sif.server_id,
		folder_id: sif.folder_id ?? null,
		position: sif.position,
		assigned_at: sif.assigned_at,
		server: sif.server ? {
			id: sif.server.id,
			name: sif.server.name,
			icon: sif.server.icon ?? sif.server.icon_url ?? null,
			banner: sif.server.banner ?? sif.server.banner_url ?? null,
			description: sif.server.description,
			owner_id: sif.server.owner_id,
			created_at: sif.server.created_at
		} : undefined
	};
}

function normalizeFolder(folder: any): ServerFolder {
	return {
		id: folder.id,
		user_id: folder.user_id,
		parent_id: folder.parent_id ?? null,
		name: folder.name,
		position: folder.position,
		is_collapsed: folder.is_collapsed ?? false,
		depth: folder.depth ?? 0,
		children: folder.children?.map(normalizeFolder),
		servers: folder.servers?.map(normalizeServerInFolder),
		created_at: folder.created_at,
		updated_at: folder.updated_at
	};
}

// Load all folders and unassigned servers for the current user
export async function loadServerFolders() {
	serverFoldersLoading.set(true);
	serverFoldersError.set(null);
	
	try {
		const data = await api.get<ServerFolderTree>('/users/me/server-folders');
		
		// Normalize the folder tree
		const normalized: ServerFolderTree = {
			folders: data.folders.map(normalizeFolder),
			servers: data.servers.map(normalizeServerInFolder)
		};
		
		serverFolderTree.set(normalized);
	} catch (error) {
		console.error('Failed to load server folders:', error);
		if (error instanceof ApiError) {
			serverFoldersError.set(error.message);
		}
		throw error;
	} finally {
		serverFoldersLoading.set(false);
	}
}

// Create a new folder
export async function createServerFolder(request: CreateFolderRequest): Promise<ServerFolder> {
	try {
		const folder = await api.post<ServerFolder>('/users/me/server-folders', request);
		// Reload the tree to get updated data
		await loadServerFolders();
		return normalizeFolder(folder);
	} catch (error) {
		console.error('Failed to create server folder:', error);
		throw error;
	}
}

// Update a folder
export async function updateServerFolder(id: string, request: UpdateFolderRequest): Promise<ServerFolder> {
	try {
		const folder = await api.patch<ServerFolder>(`/users/me/server-folders/${id}`, request);
		// Reload the tree to get updated data
		await loadServerFolders();
		return normalizeFolder(folder);
	} catch (error) {
		console.error('Failed to update server folder:', error);
		throw error;
	}
}

// Delete a folder
export async function deleteServerFolder(id: string): Promise<void> {
	try {
		await api.delete(`/users/me/server-folders/${id}`);
		// Reload the tree to get updated data
		await loadServerFolders();
	} catch (error) {
		console.error('Failed to delete server folder:', error);
		throw error;
	}
}

// Move a single server to a folder
export async function moveServerToFolder(serverId: string, folderId: string | null): Promise<void> {
	try {
		await api.post('/users/me/server-folders/move', {
			server_id: serverId,
			folder_id: folderId
		});
		// Reload the tree to get updated data
		await loadServerFolders();
	} catch (error) {
		console.error('Failed to move server to folder:', error);
		throw error;
	}
}

// Move multiple servers to a folder
export async function moveServersToFolder(request: MoveServersRequest): Promise<void> {
	try {
		await api.post('/users/me/server-folders/move-batch', request);
		// Reload the tree to get updated data
		await loadServerFolders();
	} catch (error) {
		console.error('Failed to move servers to folder:', error);
		throw error;
	}
}

// Reorder servers within a folder
export async function reorderServersInFolder(request: ReorderServersRequest): Promise<void> {
	try {
		await api.post('/users/me/server-folders/reorder', request);
		// Reload the tree to get updated data
		await loadServerFolders();
	} catch (error) {
		console.error('Failed to reorder servers:', error);
		throw error;
	}
}

// Toggle folder collapsed state
export async function toggleFolderCollapsed(id: string, isCollapsed: boolean): Promise<void> {
	await updateServerFolder(id, { is_collapsed: isCollapsed });
}

// Get a folder by ID from the current tree
export function getFolderById(id: string): ServerFolder | null {
	const tree = get(serverFolderTree);
	if (!tree) return null;
	
	const findFolder = (folders: ServerFolder[]): ServerFolder | null => {
		for (const folder of folders) {
			if (folder.id === id) return folder;
			if (folder.children) {
				const found = findFolder(folder.children);
				if (found) return found;
			}
		}
		return null;
	};
	
	return findFolder(tree.folders);
}
