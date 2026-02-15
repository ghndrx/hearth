import { writable, derived, get } from 'svelte/store';
import { user } from './auth';

export interface VoiceUser {
	id: string;
	username: string;
	display_name: string | null;
	avatar: string | null;
	muted: boolean;
	deafened: boolean;
	speaking: boolean;
	video: boolean;
	streaming: boolean;
}

export interface VoiceState {
	channelId: string | null;
	serverId: string | null;
	muted: boolean;
	deafened: boolean;
	video: boolean;
	streaming: boolean;
}

export interface ChannelVoiceState {
	channelId: string;
	users: VoiceUser[];
}

// Current user's voice state
const initialVoiceState: VoiceState = {
	channelId: null,
	serverId: null,
	muted: false,
	deafened: false,
	video: false,
	streaming: false
};

export const voiceState = writable<VoiceState>(initialVoiceState);

// All voice states per channel
export const channelVoiceStates = writable<Record<string, VoiceUser[]>>({});

// Derived store: users in current voice channel
export const currentVoiceUsers = derived(
	[voiceState, channelVoiceStates],
	([$voiceState, $channelVoiceStates]) => {
		if (!$voiceState.channelId) return [];
		return $channelVoiceStates[$voiceState.channelId] || [];
	}
);

// Derived store: is user connected to voice
export const isVoiceConnected = derived(
	voiceState,
	$voiceState => $voiceState.channelId !== null
);

// Get users in a specific channel
export function getChannelUsers(channelId: string): VoiceUser[] {
	const states = get(channelVoiceStates);
	return states[channelId] || [];
}

// Voice actions
export function joinVoiceChannel(channelId: string, serverId: string | null = null) {
	const currentUser = get(user);
	if (!currentUser) return;

	// Leave current channel if connected
	const currentState = get(voiceState);
	if (currentState.channelId) {
		leaveVoiceChannel();
	}

	// Update voice state
	voiceState.update(state => ({
		...state,
		channelId,
		serverId
	}));

	// Add current user to channel's voice state
	channelVoiceStates.update(states => {
		const currentUsers = states[channelId] || [];
		const userVoice: VoiceUser = {
			id: currentUser.id,
			username: currentUser.username,
			display_name: currentUser.display_name,
			avatar: currentUser.avatar,
			muted: get(voiceState).muted,
			deafened: get(voiceState).deafened,
			speaking: false,
			video: false,
			streaming: false
		};

		return {
			...states,
			[channelId]: [...currentUsers.filter(u => u.id !== currentUser.id), userVoice]
		};
	});

	// TODO: Actually connect to voice via WebRTC
	console.log(`Joined voice channel: ${channelId}`);
}

export function leaveVoiceChannel() {
	const currentUser = get(user);
	const currentState = get(voiceState);
	
	if (!currentUser || !currentState.channelId) return;

	const channelId = currentState.channelId;

	// Remove user from channel's voice state
	channelVoiceStates.update(states => {
		const currentUsers = states[channelId] || [];
		return {
			...states,
			[channelId]: currentUsers.filter(u => u.id !== currentUser.id)
		};
	});

	// Reset voice state
	voiceState.set(initialVoiceState);

	// TODO: Actually disconnect from voice via WebRTC
	console.log(`Left voice channel: ${channelId}`);
}

export function toggleMute() {
	const currentUser = get(user);
	const currentState = get(voiceState);
	
	voiceState.update(state => ({ ...state, muted: !state.muted }));

	// Update user in channel state
	if (currentUser && currentState.channelId) {
		channelVoiceStates.update(states => {
			const users = states[currentState.channelId!] || [];
			return {
				...states,
				[currentState.channelId!]: users.map(u => 
					u.id === currentUser.id ? { ...u, muted: !currentState.muted } : u
				)
			};
		});
	}
}

export function toggleDeafen() {
	const currentUser = get(user);
	const currentState = get(voiceState);
	const willDeafen = !currentState.deafened;
	
	voiceState.update(state => ({
		...state,
		deafened: willDeafen,
		// When deafening, also mute
		muted: willDeafen ? true : state.muted
	}));

	// Update user in channel state
	if (currentUser && currentState.channelId) {
		channelVoiceStates.update(states => {
			const users = states[currentState.channelId!] || [];
			return {
				...states,
				[currentState.channelId!]: users.map(u => 
					u.id === currentUser.id 
						? { ...u, deafened: willDeafen, muted: willDeafen ? true : u.muted } 
						: u
				)
			};
		});
	}
}

export function toggleVideo() {
	voiceState.update(state => ({ ...state, video: !state.video }));
}

export function toggleStreaming() {
	voiceState.update(state => ({ ...state, streaming: !state.streaming }));
}

export function setSpeaking(speaking: boolean) {
	const currentUser = get(user);
	const currentState = get(voiceState);

	if (!currentUser || !currentState.channelId) return;

	channelVoiceStates.update(states => {
		const users = states[currentState.channelId!] || [];
		return {
			...states,
			[currentState.channelId!]: users.map(u => 
				u.id === currentUser.id ? { ...u, speaking } : u
			)
		};
	});
}

// Update voice states from gateway events
export function updateVoiceState(channelId: string, voiceUser: VoiceUser) {
	channelVoiceStates.update(states => {
		const users = states[channelId] || [];
		const existingIndex = users.findIndex(u => u.id === voiceUser.id);
		
		if (existingIndex >= 0) {
			users[existingIndex] = voiceUser;
			return { ...states, [channelId]: [...users] };
		} else {
			return { ...states, [channelId]: [...users, voiceUser] };
		}
	});
}

export function removeVoiceUser(channelId: string, userId: string) {
	channelVoiceStates.update(states => {
		const users = states[channelId] || [];
		return {
			...states,
			[channelId]: users.filter(u => u.id !== userId)
		};
	});
}

// Clear all voice states (on disconnect)
export function clearVoiceStates() {
	voiceState.set(initialVoiceState);
	channelVoiceStates.set({});
}
