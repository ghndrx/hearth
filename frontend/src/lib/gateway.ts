import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { handleMessageCreate, handleMessageUpdate, handleMessageDelete } from './stores/messages';

export type GatewayState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

export const gatewayState = writable<GatewayState>('disconnected');

// Event emitter for components to subscribe to raw gateway events
type GatewayEventHandler = (data: unknown) => void;
const eventHandlers = new Map<string, Set<GatewayEventHandler>>();

export function onGatewayEvent(eventType: string, handler: GatewayEventHandler): () => void {
	if (!eventHandlers.has(eventType)) {
		eventHandlers.set(eventType, new Set());
	}
	eventHandlers.get(eventType)!.add(handler);
	return () => {
		eventHandlers.get(eventType)?.delete(handler);
	};
}

function emitGatewayEvent(eventType: string, data: unknown) {
	const handlers = eventHandlers.get(eventType);
	if (handlers) {
		handlers.forEach(handler => handler(data));
	}
	// Also emit to wildcard listeners
	const wildcardHandlers = eventHandlers.get('*');
	if (wildcardHandlers) {
		wildcardHandlers.forEach(handler => handler({ type: eventType, data }));
	}
}

// Gateway opcodes (match backend)
const Op = {
	DISPATCH: 0,
	HEARTBEAT: 1,
	IDENTIFY: 2,
	PRESENCE_UPDATE: 3,
	VOICE_STATE_UPDATE: 4,
	RESUME: 6,
	RECONNECT: 7,
	REQUEST_GUILD_MEMBERS: 8,
	INVALID_SESSION: 9,
	HELLO: 10,
	HEARTBEAT_ACK: 11,
} as const;

interface GatewayMessage {
	op: number;      // Opcode
	d?: unknown;     // Data
	s?: number;      // Sequence (for dispatch events)
	t?: string;      // Event type (for dispatch events)
}

interface HelloPayload {
	heartbeat_interval: number;
}

class Gateway {
	private ws: WebSocket | null = null;
	private reconnectAttempts = 0;
	private maxReconnectAttempts = 5;
	private reconnectDelay = 1000;
	private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
	private heartbeatIntervalMs = 30000;
	private sequence: number | null = null;
	private sessionId: string | null = null;
	private token: string | null = null;
	private heartbeatAcked = true;
	
	connect(token: string) {
		if (!browser) return;
		
		this.token = token;
		gatewayState.set('connecting');
		
		const wsUrl = this.getWebSocketUrl();
		this.ws = new WebSocket(`${wsUrl}?token=${token}`);
		
		this.ws.onopen = () => {
			// Wait for HELLO before starting heartbeat
			this.reconnectAttempts = 0;
		};
		
		this.ws.onmessage = (event) => {
			try {
				const msg: GatewayMessage = JSON.parse(event.data);
				this.handleMessage(msg);
			} catch (error) {
				console.error('Failed to parse gateway message:', error);
			}
		};
		
		this.ws.onclose = (event) => {
			this.stopHeartbeat();
			
			if (event.code !== 1000 && this.token) {
				this.attemptReconnect();
			} else {
				gatewayState.set('disconnected');
			}
		};
		
		this.ws.onerror = (error) => {
			console.error('Gateway error:', error);
		};
	}
	
	disconnect() {
		this.token = null;
		if (this.ws) {
			this.ws.close(1000);
			this.ws = null;
		}
		this.stopHeartbeat();
		this.sessionId = null;
		this.sequence = null;
		gatewayState.set('disconnected');
	}
	
	send(op: number, data?: unknown) {
		if (this.ws?.readyState === WebSocket.OPEN) {
			const msg: GatewayMessage = { op, d: data };
			this.ws.send(JSON.stringify(msg));
		}
	}
	
	private getWebSocketUrl(): string {
		const apiUrl = import.meta.env.VITE_API_URL || '';
		if (apiUrl.startsWith('http')) {
			return apiUrl.replace(/^http/, 'ws').replace('/api/v1', '/gateway');
		}
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		return `${protocol}//${window.location.host}/gateway`;
	}
	
	private handleMessage(msg: GatewayMessage) {
		switch (msg.op) {
			case Op.HELLO:
				this.handleHello(msg.d as HelloPayload);
				break;
				
			case Op.HEARTBEAT_ACK:
				this.heartbeatAcked = true;
				break;
				
			case Op.HEARTBEAT:
				// Server requests heartbeat
				this.sendHeartbeat();
				break;
				
			case Op.RECONNECT:
				// Server wants us to reconnect
				this.ws?.close();
				this.attemptReconnect();
				break;
				
			case Op.INVALID_SESSION:
				// Session invalid, need to re-identify
				this.sessionId = null;
				this.sequence = null;
				if (this.token) {
					setTimeout(() => this.sendIdentify(), 1000 + Math.random() * 4000);
				}
				break;
				
			case Op.DISPATCH:
				if (msg.s !== undefined) {
					this.sequence = msg.s;
				}
				if (msg.t) {
					this.handleDispatch(msg.t, msg.d);
				}
				break;
		}
	}
	
	private handleHello(data: HelloPayload) {
		this.heartbeatIntervalMs = data.heartbeat_interval;
		this.startHeartbeat();
		
		// If we have a session, try to resume
		if (this.sessionId && this.sequence !== null) {
			this.sendResume();
		} else {
			this.sendIdentify();
		}
	}
	
	private sendIdentify() {
		this.send(Op.IDENTIFY, {
			token: this.token,
			properties: {
				$os: 'browser',
				$browser: 'hearth-web',
				$device: 'hearth-web'
			}
		});
	}
	
	private sendResume() {
		this.send(Op.RESUME, {
			token: this.token,
			session_id: this.sessionId,
			seq: this.sequence
		});
	}
	
	private handleDispatch(eventType: string, data: unknown) {
		console.log('[Gateway] Dispatch event:', eventType, data);
		
		// Emit to event listeners
		emitGatewayEvent(eventType, data);
		
		switch (eventType) {
			case 'READY': {
				const readyData = data as { session_id: string };
				this.sessionId = readyData.session_id;
				gatewayState.set('connected');
				console.log('[Gateway] Ready with session:', this.sessionId);
				break;
			}
			
			case 'RESUMED':
				gatewayState.set('connected');
				console.log('[Gateway] Resumed');
				break;
				
			case 'MESSAGE_CREATE':
				console.log('[Gateway] MESSAGE_CREATE received:', data);
				handleMessageCreate(this.normalizeMessage(data));
				break;
				
			case 'MESSAGE_UPDATE':
				console.log('[Gateway] MESSAGE_UPDATE received:', data);
				handleMessageUpdate(this.normalizeMessage(data));
				break;
				
			case 'MESSAGE_DELETE':
				console.log('[Gateway] MESSAGE_DELETE received:', data);
				handleMessageDelete(data as { id: string; channel_id: string });
				break;
				
			case 'TYPING_START':
				this.handleTyping(data as { channel_id: string; user_id: string });
				break;
				
			case 'PRESENCE_UPDATE':
				this.handlePresence(data as { user: { id: string }; status: string });
				break;
				
			// Guild events (backend uses GUILD_ prefix)
			case 'GUILD_CREATE':
			case 'GUILD_UPDATE':
			case 'GUILD_DELETE':
			case 'CHANNEL_CREATE':
			case 'CHANNEL_UPDATE':
			case 'CHANNEL_DELETE':
			case 'GUILD_MEMBER_ADD':
			case 'GUILD_MEMBER_REMOVE':
			case 'GUILD_MEMBER_UPDATE':
				console.log('[Gateway] Guild event:', eventType, data);
				break;
				
			default:
				console.log('[Gateway] Unknown event:', eventType, data);
		}
	}
	
	// Normalize backend message format to frontend format
	private normalizeMessage(data: unknown): Record<string, unknown> {
		const msg = data as Record<string, unknown>;
		return {
			...msg,
			// Map backend field names to frontend expectations
			created_at: msg.timestamp || msg.created_at,
			edited_at: msg.edited_timestamp || msg.edited_at,
			server_id: msg.guild_id || msg.server_id,
			author_id: (msg.author as Record<string, unknown>)?.id || msg.author_id,
			author: msg.author,
			reply_to: msg.referenced_message_id || msg.reply_to,
		};
	}
	
	private handleTyping(data: { channel_id: string; user_id: string }) {
		// TODO: Show typing indicator
		console.log(`User ${data.user_id} is typing in ${data.channel_id}`);
	}
	
	private handlePresence(data: { user: { id: string }; status: string }) {
		// TODO: Update presence store
		console.log(`User ${data.user.id} is now ${data.status}`);
	}
	
	private sendHeartbeat() {
		this.send(Op.HEARTBEAT, this.sequence);
	}
	
	private startHeartbeat() {
		this.heartbeatAcked = true;
		// Send first heartbeat after jitter
		const jitter = Math.random() * this.heartbeatIntervalMs;
		setTimeout(() => {
			this.sendHeartbeat();
			
			this.heartbeatInterval = setInterval(() => {
				if (!this.heartbeatAcked) {
					// No ack received, zombie connection
					console.warn('Heartbeat not acknowledged, reconnecting...');
					this.ws?.close();
					return;
				}
				this.heartbeatAcked = false;
				this.sendHeartbeat();
			}, this.heartbeatIntervalMs);
		}, jitter);
	}
	
	private stopHeartbeat() {
		if (this.heartbeatInterval) {
			clearInterval(this.heartbeatInterval);
			this.heartbeatInterval = null;
		}
	}
	
	private attemptReconnect() {
		if (this.reconnectAttempts >= this.maxReconnectAttempts) {
			gatewayState.set('disconnected');
			console.error('Max reconnection attempts reached');
			return;
		}
		
		gatewayState.set('reconnecting');
		this.reconnectAttempts++;
		
		const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
		console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
		
		setTimeout(() => {
			if (this.token) {
				this.connect(this.token);
			}
		}, delay);
	}
	
	// Update presence status
	updatePresence(status: string, activities: unknown[] = []) {
		this.send(Op.PRESENCE_UPDATE, {
			status,
			activities,
			since: status === 'idle' ? Date.now() : null,
			afk: status === 'idle'
		});
	}
	
	// Subscribe to a channel for real-time events
	subscribeChannel(channelId: string) {
		console.log('[Gateway] Subscribing to channel:', channelId);
		// Send as dispatch event type SUBSCRIBE
		this.send(Op.DISPATCH, {
			t: 'SUBSCRIBE',
			d: { channel_id: channelId }
		});
	}
	
	// Unsubscribe from a channel
	unsubscribeChannel(channelId: string) {
		console.log('[Gateway] Unsubscribing from channel:', channelId);
		this.send(Op.DISPATCH, {
			t: 'UNSUBSCRIBE', 
			d: { channel_id: channelId }
		});
	}
	
	// Subscribe to a server for real-time events
	subscribeServer(serverId: string) {
		console.log('[Gateway] Subscribing to server:', serverId);
		this.send(Op.DISPATCH, {
			t: 'SUBSCRIBE',
			d: { server_id: serverId }
		});
	}
}

export const gateway = new Gateway();
