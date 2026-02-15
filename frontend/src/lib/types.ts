// User types
export interface User {
	id: string;
	username: string;
	email?: string;
	display_name: string | null;
	avatar: string | null;
	avatarUrl?: string;
	role_color?: string;
	status?: 'online' | 'idle' | 'dnd' | 'offline';
	createdAt?: string;
	created_at?: string;
}

// Server (Guild) types
export interface Server {
	id: string;
	name: string;
	iconUrl?: string;
	ownerId: string;
	createdAt: string;
}

// Channel types
export type ChannelType = 'text' | 'voice' | 'announcement';

export interface Channel {
	id: string;
	serverId: string;
	name: string;
	type: ChannelType;
	topic?: string;
	position: number;
	createdAt: string;
}

// Message types
export type MessageType = 'default' | 'reply' | 'recipient_add' | 'recipient_remove' | 'call' | 'channel_name_change' | 'channel_icon_change' | 'pinned' | 'member_join' | 'thread_created';

export interface Message {
	id: string;
	channel_id: string;
	author_id: string;
	server_id?: string;
	content: string;
	encrypted_content?: string;
	encrypted: boolean;
	type: MessageType;
	reply_to?: string | null;
	reply_to_id?: string;
	thread_id?: string;
	pinned: boolean;
	tts: boolean;
	mention_everyone: boolean;
	flags: number;
	created_at: string;
	edited_at?: string;
	author?: User;
	attachments?: Attachment[];
	embeds?: Embed[];
	reactions?: Reaction[];
	mentions?: string[];
	mention_roles?: string[];
	referenced_message?: Message;
}

export interface Embed {
	type?: string;
	title?: string;
	description?: string;
	url?: string;
	timestamp?: string;
	color?: number;
	footer?: { text: string; icon_url?: string };
	image?: { url: string; proxy_url?: string; width?: number; height?: number };
	thumbnail?: { url: string; proxy_url?: string; width?: number; height?: number };
	video?: { url: string; proxy_url?: string; width?: number; height?: number };
	provider?: { name?: string; url?: string };
	author?: { name: string; url?: string; icon_url?: string };
	fields?: { name: string; value: string; inline?: boolean }[];
}

export interface Attachment {
	id: string;
	message_id: string;
	filename: string;
	url: string;
	proxy_url?: string;
	size: number;
	content_type?: string;
	width?: number;
	height?: number;
	ephemeral: boolean;
	encrypted: boolean;
	encrypted_key?: string;
	iv?: string;
	created_at: string;
}

export interface Reaction {
	message_id: string;
	emoji: string;
	count: number;
	me: boolean;
	user_ids?: string[];
}

// Member types
export interface Member {
	userId: string;
	serverId: string;
	user?: User;
	nickname?: string;
	joinedAt: string;
	roles: string[];
}

// Role types
export interface Role {
	id: string;
	serverId: string;
	name: string;
	color: string;
	position: number;
	permissions: string[];
}

// WebSocket event types
export type WSEventType =
	| 'MESSAGE_CREATE'
	| 'MESSAGE_UPDATE'
	| 'MESSAGE_DELETE'
	| 'CHANNEL_CREATE'
	| 'CHANNEL_UPDATE'
	| 'CHANNEL_DELETE'
	| 'MEMBER_JOIN'
	| 'MEMBER_LEAVE'
	| 'MEMBER_UPDATE'
	| 'PRESENCE_UPDATE'
	| 'TYPING_START'
	| 'READY';

export interface WSEvent<T = unknown> {
	type: WSEventType;
	data: T;
	timestamp: string;
}

// Auth types
export interface LoginRequest {
	email: string;
	password: string;
}

export interface RegisterRequest {
	username: string;
	email: string;
	password: string;
}

export interface AuthResponse {
	user: User;
	token: string;
}
