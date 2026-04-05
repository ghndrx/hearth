import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { fetchApi } from '$lib/api';
import { onGatewayEvent, gateway, Op } from '$lib/stores/gateway';

// Voice Activity Types
export type VoiceActivityType = 'poker' | 'chess' | 'watch_together';
export type VoiceActivityStatus = 'active' | 'finished' | 'cancelled';

export interface VoiceActivityParticipant {
	user_id: string;
	username: string;
	display_name?: string;
	avatar?: string;
	joined_at: string;
}

export interface VoiceActivity {
	id: string;
	channel_id: string;
	server_id: string;
	creator_id: string;
	activity_type: VoiceActivityType;
	status: VoiceActivityStatus;
	max_participants: number;
	metadata: Record<string, unknown>;
	participants: VoiceActivityParticipant[];
	created_at: string;
	updated_at: string;
	ended_at?: string;
}

export interface GameState {
	activity_id: string;
	state: unknown;
	version: number;
	updated_at?: string;
}

// Poker specific types
export interface PokerState {
	phase: string;
	pot: number;
	community_cards: string[];
	current_turn?: string;
	dealer_index: number;
	small_blind: number;
	big_blind: number;
	players: PokerPlayer[];
	round: number;
}

export interface PokerPlayer {
	user_id: string;
	hand: string[];
	chips: number;
	bet: number;
	folded: boolean;
	all_in: boolean;
	is_dealer: boolean;
}

// Chess specific types
export interface ChessState {
	board: string; // FEN
	white_player?: string;
	black_player?: string;
	current_turn: 'white' | 'black';
	move_history: string[];
	status: string;
	winner?: string;
}

// Watch Together specific types
export interface WatchTogetherState {
	video_url: string;
	video_title?: string;
	is_playing: boolean;
	current_time: number;
	playback_rate: number;
	updated_by?: string;
	queue: WatchTogetherQueueItem[];
}

export interface WatchTogetherQueueItem {
	url: string;
	title?: string;
	added_by: string;
	added_at: string;
}

export interface VoiceActivityStore {
	currentActivity: VoiceActivity | null;
	gameState: unknown | null;
	gameStateVersion: number;
	loading: boolean;
	error: string | null;
}

const initialState: VoiceActivityStore = {
	currentActivity: null,
	gameState: null,
	gameStateVersion: 0,
	loading: false,
	error: null
};

function createVoiceActivityStore() {
	const { subscribe, set, update } = writable<VoiceActivityStore>(initialState);

	return {
		subscribe,
		set,
		update,

		reset() {
			set(initialState);
		},

		setActivity(activity: VoiceActivity | null) {
			update(state => ({
				...state,
				currentActivity: activity,
				error: null
			}));
		},

		setGameState(gameState: unknown, version: number) {
			update(state => ({
				...state,
				gameState,
				gameStateVersion: version
			}));
		},

		setLoading(loading: boolean) {
			update(state => ({ ...state, loading }));
		},

		setError(error: string | null) {
			update(state => ({ ...state, error, loading: false }));
		}
	};
}

export const voiceActivityState = createVoiceActivityStore();

// Derived stores
export const isInActivity = derived(voiceActivityState, $state => $state.currentActivity !== null);
export const currentActivityType = derived(voiceActivityState, $state => $state.currentActivity?.activity_type ?? null);
export const activityParticipants = derived(voiceActivityState, $state => $state.currentActivity?.participants ?? []);

// API functions
export async function startActivity(channelId: string, activityType: VoiceActivityType, metadata?: Record<string, unknown>): Promise<VoiceActivity | null> {
	voiceActivityState.setLoading(true);
	try {
		const res = await fetchApi(`/channels/${channelId}/activities`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				activity_type: activityType,
				metadata: metadata ?? {}
			})
		});
		if (!res.ok) {
			const err = await res.json();
			voiceActivityState.setError(err.error || 'Failed to start activity');
			return null;
		}
		const activity: VoiceActivity = await res.json();
		voiceActivityState.setActivity(activity);
		return activity;
	} catch (e) {
		voiceActivityState.setError('Failed to start activity');
		return null;
	}
}

export async function joinActivity(activityId: string): Promise<VoiceActivity | null> {
	voiceActivityState.setLoading(true);
	try {
		const res = await fetchApi(`/activities/${activityId}/join`, { method: 'POST' });
		if (!res.ok) {
			const err = await res.json();
			voiceActivityState.setError(err.error || 'Failed to join activity');
			return null;
		}
		const activity: VoiceActivity = await res.json();
		voiceActivityState.setActivity(activity);
		return activity;
	} catch (e) {
		voiceActivityState.setError('Failed to join activity');
		return null;
	}
}

export async function leaveActivity(activityId: string): Promise<void> {
	try {
		await fetchApi(`/activities/${activityId}/participants/@me`, { method: 'DELETE' });
		voiceActivityState.reset();
	} catch (e) {
		voiceActivityState.setError('Failed to leave activity');
	}
}

export async function endActivity(activityId: string): Promise<void> {
	try {
		await fetchApi(`/activities/${activityId}`, { method: 'DELETE' });
		voiceActivityState.reset();
	} catch (e) {
		voiceActivityState.setError('Failed to end activity');
	}
}

export async function getChannelActivity(channelId: string): Promise<VoiceActivity | null> {
	try {
		const res = await fetchApi(`/channels/${channelId}/activities`);
		if (res.status === 404) return null;
		if (!res.ok) return null;
		return await res.json();
	} catch {
		return null;
	}
}

export async function getGameState(activityId: string): Promise<GameState | null> {
	try {
		const res = await fetchApi(`/activities/${activityId}/state`);
		if (!res.ok) return null;
		return await res.json();
	} catch {
		return null;
	}
}

export async function sendGameMove(activityId: string, action: string, data?: unknown): Promise<GameState | null> {
	try {
		const res = await fetchApi(`/activities/${activityId}/moves`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ action, data: data ?? {} })
		});
		if (!res.ok) {
			const err = await res.json();
			voiceActivityState.setError(err.error || 'Move failed');
			return null;
		}
		const state: GameState = await res.json();
		voiceActivityState.setGameState(state.state, state.version);
		return state;
	} catch (e) {
		voiceActivityState.setError('Failed to send move');
		return null;
	}
}

// WebSocket-based real-time functions
export function sendActivityViaWS(type: string, data: Record<string, unknown>): void {
	const gw = get(gateway);
	if (!gw.connected) return;

	gateway.send({
		op: Op.DISPATCH,
		d: { t: type, d: data }
	});
}

export function sendGameMoveViaWS(activityId: string, action: string, data?: unknown): void {
	sendActivityViaWS('ACTIVITY_GAME_MOVE', {
		activity_id: activityId,
		action,
		data: data ?? {}
	});
}

export function requestStateSync(activityId: string): void {
	sendActivityViaWS('ACTIVITY_SYNC', { activity_id: activityId });
}

// Gateway event listeners
if (browser) {
	onGatewayEvent('ACTIVITY_START', (data: unknown) => {
		const d = data as VoiceActivity;
		voiceActivityState.setActivity(d);
	});

	onGatewayEvent('ACTIVITY_PARTICIPANT_JOIN', (data: unknown) => {
		const d = data as { activity_id: string; user_id: string };
		voiceActivityState.update(state => {
			if (state.currentActivity && state.currentActivity.id === d.activity_id) {
				return { ...state };
			}
			return state;
		});
	});

	onGatewayEvent('ACTIVITY_PARTICIPANT_LEAVE', (data: unknown) => {
		const d = data as { activity_id: string; user_id: string };
		voiceActivityState.update(state => {
			if (state.currentActivity && state.currentActivity.id === d.activity_id) {
				return {
					...state,
					currentActivity: {
						...state.currentActivity,
						participants: state.currentActivity.participants.filter(p => p.user_id !== d.user_id)
					}
				};
			}
			return state;
		});
	});

	onGatewayEvent('ACTIVITY_END', (data: unknown) => {
		const d = data as { activity_id: string };
		voiceActivityState.update(state => {
			if (state.currentActivity && state.currentActivity.id === d.activity_id) {
				return initialState;
			}
			return state;
		});
	});

	onGatewayEvent('ACTIVITY_GAME_MOVE', (data: unknown) => {
		const d = data as { activity_id: string; state: GameState };
		voiceActivityState.update(state => {
			if (state.currentActivity && state.currentActivity.id === d.activity_id && d.state) {
				return {
					...state,
					gameState: d.state.state,
					gameStateVersion: d.state.version
				};
			}
			return state;
		});
	});

	onGatewayEvent('ACTIVITY_STATE_SYNC', (data: unknown) => {
		const d = data as { activity_id: string; state: GameState };
		if (d.state) {
			voiceActivityState.setGameState(d.state.state, d.state.version);
		}
	});
}
