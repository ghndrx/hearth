import { writable, derived } from 'svelte/store';
import { api } from '$lib/api';
import { onGatewayEvent } from './gateway';
import { browser } from '$app/environment';

// Types matching backend models
export type CallType = 'direct' | 'group' | 'channel';
export type CallStatus = 'ringing' | 'active' | 'ended';

export interface CallParticipant {
	id: string;
	call_id: string;
	user_id: string;
	joined_at: string;
	left_at: string | null;
	is_muted: boolean;
	is_video_on: boolean;
	username: string;
	display_name: string | null;
	avatar: string | null;
}

export interface Call {
	id: string;
	channel_id: string;
	server_id: string | null;
	initiator_id: string;
	type: CallType;
	status: CallStatus;
	started_at: string;
	ended_at: string | null;
	end_reason: string;
	created_at: string;
	participants: CallParticipant[];
}

export interface ICEServer {
	urls: string[];
	username?: string;
	credential?: string;
}

export interface JoinCallResponse {
	call_id: string;
	user_id: string;
	joined_at: string;
	ice_servers: ICEServer[];
}

// Store for active calls indexed by channel ID
export const activeCalls = writable<Map<string, Call>>(new Map());
export const callsLoading = writable(false);
export const callsError = writable<string | null>(null);

// Derived: check if a specific channel has an active call
export const activeCallForChannel = derived(activeCalls, ($activeCalls) => {
	return (channelId: string): Call | undefined => $activeCalls.get(channelId);
});

// Create a new call via REST API
export async function createCall(
	channelId: string,
	type: CallType = 'direct',
	targetUserId?: string,
	serverId?: string
): Promise<Call> {
	callsLoading.set(true);
	callsError.set(null);

	try {
		const call = await api.post<Call>('/calls', {
			channel_id: channelId,
			server_id: serverId,
			type,
			target_user_id: targetUserId
		});

		activeCalls.update((map) => {
			map.set(call.channel_id, call);
			return new Map(map);
		});

		return call;
	} catch (err) {
		const message = err instanceof Error ? err.message : 'Failed to create call';
		callsError.set(message);
		throw err;
	} finally {
		callsLoading.set(false);
	}
}

// Get call details
export async function getCall(callId: string): Promise<Call> {
	const call = await api.get<Call>(`/calls/${callId}`);
	if (call.status !== 'ended') {
		activeCalls.update((map) => {
			map.set(call.channel_id, call);
			return new Map(map);
		});
	}
	return call;
}

// Join an existing call
export async function joinCall(callId: string): Promise<JoinCallResponse> {
	callsLoading.set(true);
	callsError.set(null);

	try {
		const resp = await api.post<JoinCallResponse>(`/calls/${callId}/join`);
		// Refresh call state after joining
		await getCall(callId);
		return resp;
	} catch (err) {
		const message = err instanceof Error ? err.message : 'Failed to join call';
		callsError.set(message);
		throw err;
	} finally {
		callsLoading.set(false);
	}
}

// Leave a call
export async function leaveCall(callId: string): Promise<void> {
	try {
		await api.post(`/calls/${callId}/leave`);
		// Remove from active calls
		activeCalls.update((map) => {
			for (const [channelId, call] of map) {
				if (call.id === callId) {
					map.delete(channelId);
					break;
				}
			}
			return new Map(map);
		});
	} catch (err) {
		const message = err instanceof Error ? err.message : 'Failed to leave call';
		callsError.set(message);
		throw err;
	}
}

// Listen for gateway events to keep active calls in sync
if (browser) {
	// TODO: Wire up CALL_CREATED and CALL_ENDED events once the backend
	// dispatches them via the gateway bridge. Currently, call state is
	// managed through the videoCall.ts store's WebSocket events.
	onGatewayEvent('CALL_CREATED', (data: unknown) => {
		const call = data as Call;
		activeCalls.update((map) => {
			map.set(call.channel_id, call);
			return new Map(map);
		});
	});

	onGatewayEvent('CALL_ENDED', (data: unknown) => {
		const ended = data as { call_id: string; channel_id: string };
		activeCalls.update((map) => {
			map.delete(ended.channel_id);
			return new Map(map);
		});
	});
}
