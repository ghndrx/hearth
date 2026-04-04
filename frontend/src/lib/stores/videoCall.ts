import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { gateway, Op, onGatewayEvent } from './gateway';
import { user as authUser } from '$lib/stores/auth';

// Video call types
export type VideoCallType = 'direct' | 'group' | 'channel';

// Video call states
export type VideoCallState = 
	| 'idle'
	| 'ringing_out'
	| 'ringing_in'
	| 'connecting'
	| 'connected'
	| 'reconnecting'
	| 'ended';

export interface VideoParticipant {
	id: string;
	username: string;
	display_name: string | null;
	avatar: string | null;
	isCameraOn: boolean;
	isScreenShare: boolean;
	isMuted: boolean;
	isSpeaking: boolean;
	joinedAt: Date;
}

export interface VideoCallData {
	call_id: string;
	channel_id: string;
	server_id?: string;
	call_type: VideoCallType;
	state: VideoCallState;
	initiator_id?: string;
	participants?: VideoParticipant[];
	started_at?: string;
	ended_at?: string;
}

export interface VideoRingData {
	call_id: string;
	channel_id: string;
	server_id?: string;
	call_type: VideoCallType;
	from_user_id: string;
	from_username: string;
}

export interface VideoState {
	isActive: boolean;
	callId: string | null;
	channelId: string | null;
	serverId: string | null;
	callType: VideoCallType;
	state: VideoCallState;
	initiatorId: string | null;
	participants: VideoParticipant[];
	startedAt: Date | null;
	
	// Local state
	localCameraOn: boolean;
	localMuted: boolean;
	localScreenShare: boolean;
	connectionState: 'disconnected' | 'connecting' | 'connected' | 'reconnecting';
	
	// Incoming ring
	incomingRing: VideoRingData | null;
	
	error: string | null;
}

const initialState: VideoState = {
	isActive: false,
	callId: null,
	channelId: null,
	serverId: null,
	callType: 'direct',
	state: 'idle',
	initiatorId: null,
	participants: [],
	startedAt: null,
	localCameraOn: false,
	localMuted: true,
	localScreenShare: false,
	connectionState: 'disconnected',
	incomingRing: null,
	error: null
};

function createVideoCallStore() {
	const { subscribe, set, update } = writable<VideoState>(initialState);

	return {
		subscribe,

		// Reset store to initial state
		reset() {
			set(initialState);
		},

		// Start a video call
		startCall(callData: {
			channelId: string;
			serverId?: string;
			callType: VideoCallType;
			toUserId?: string;
		}) {
			update(state => ({
				...state,
				isActive: true,
				channelId: callData.channelId,
				serverId: callData.serverId ?? null,
				callType: callData.callType,
				state: 'ringing_out',
				connectionState: 'connecting',
				error: null
			}));

			// Send ring message
			gateway.send({
				op: Op.DISPATCH,
				d: {
					t: 'VIDEO_RING',
					d: {
						channel_id: callData.channelId,
						server_id: callData.serverId,
						call_type: callData.callType,
						to_user_id: callData.toUserId,
						is_group: callData.callType === 'group'
					}
				}
			});
		},

		// Accept incoming call
		acceptCall() {
			const state = get({ subscribe });
			if (!state.incomingRing) return;

			update(s => ({
				...s,
				isActive: true,
				callId: state.incomingRing!.call_id,
				channelId: state.incomingRing!.channel_id,
				serverId: state.incomingRing!.server_id ?? null,
				callType: state.incomingRing!.call_type,
				state: 'connecting',
				initiatorId: state.incomingRing!.from_user_id,
				incomingRing: null,
				connectionState: 'connecting'
			}));

			// Send accept
			gateway.send({
				op: Op.DISPATCH,
				d: {
					t: 'VIDEO_RING_RESPONSE',
					d: {
						call_id: state.incomingRing!.call_id,
						accept: true,
						to_user_id: state.incomingRing!.from_user_id
					}
				}
			});
		},

		// Decline incoming call
		declineCall() {
			const state = get({ subscribe });
			if (!state.incomingRing) return;

			// Send decline
			gateway.send({
				op: Op.DISPATCH,
				d: {
					t: 'VIDEO_RING_RESPONSE',
					d: {
						call_id: state.incomingRing!.call_id,
						accept: false,
						to_user_id: state.incomingRing!.from_user_id
					}
				}
			});

			update(s => ({
				...s,
				incomingRing: null
			}));
		},

		// End current call
		endCall() {
			const state = get({ subscribe });
			if (!state.callId) return;

			// Send leave
			gateway.send({
				op: Op.DISPATCH,
				d: {
					t: 'VIDEO_LEAVE',
					d: {
						call_id: state.callId,
						channel_id: state.channelId,
						server_id: state.serverId
					}
				}
			});

			set(initialState);
		},

		// Toggle camera
		toggleCamera() {
			update(state => {
				const newCameraOn = !state.localCameraOn;
				
				// Send state update if in a call
				if (state.callId) {
					gateway.send({
						op: Op.DISPATCH,
						d: {
							t: 'VIDEO_STATE_UPDATE',
							d: {
								call_id: state.callId,
								is_camera_on: newCameraOn,
								is_muted: state.localMuted,
								is_screen_share: state.localScreenShare
							}
						}
					});
				}

				return { ...state, localCameraOn: newCameraOn };
			});
		},

		// Toggle mute
		toggleMute() {
			update(state => {
				const newMuted = !state.localMuted;
				
				// Send state update if in a call
				if (state.callId) {
					gateway.send({
						op: Op.DISPATCH,
						d: {
							t: 'VIDEO_STATE_UPDATE',
							d: {
								call_id: state.callId,
								is_camera_on: state.localCameraOn,
								is_muted: newMuted,
								is_screen_share: state.localScreenShare
							}
						}
					});
				}

				return { ...state, localMuted: newMuted };
			});
		},

		// Toggle screen share
		toggleScreenShare() {
			update(state => {
				const newScreenShare = !state.localScreenShare;
				
				// Send screen share start/stop
				if (state.callId) {
					gateway.send({
						op: Op.DISPATCH,
						d: {
							t: newScreenShare ? 'VIDEO_SCREEN_START' : 'VIDEO_SCREEN_STOP',
							d: {
								call_id: state.callId,
								is_screen_share: newScreenShare
							}
						}
					});
				}

				return { ...state, localScreenShare: newScreenShare };
			});
		},

		// Set connection state
		setConnectionState(connectionState: VideoState['connectionState']) {
			update(state => ({ ...state, connectionState }));
		},

		// Set call state
		setCallState(state: VideoCallState) {
			update(s => ({ ...s, state }));
		},

		// Add participant
		addParticipant(participant: VideoParticipant) {
			update(state => {
				if (state.participants.some(p => p.id === participant.id)) {
					return state;
				}
				return {
					...state,
					participants: [...state.participants, participant]
				};
			});
		},

		// Remove participant
		removeParticipant(userId: string) {
			update(state => ({
				...state,
				participants: state.participants.filter(p => p.id !== userId)
			}));
		},

		// Update participant
		updateParticipant(userId: string, updates: Partial<VideoParticipant>) {
			update(state => ({
				...state,
				participants: state.participants.map(p =>
					p.id === userId ? { ...p, ...updates } : p
				)
			}));
		},

		// Set incoming ring
		setIncomingRing(ring: VideoRingData | null) {
			update(state => ({ ...state, incomingRing: ring }));
		},

		// Set error
		setError(error: string | null) {
			update(state => ({ ...state, error }));
		},

		// Set participants from server update
		setParticipants(participants: VideoParticipant[]) {
			update(state => ({ ...state, participants }));
		},

		// Set call info
		setCallInfo(info: {
			callId: string;
			channelId: string;
			serverId?: string;
			callType: VideoCallType;
			state: VideoCallState;
			initiatorId?: string;
			startedAt?: string;
		}) {
			update(state => ({
				...state,
				callId: info.callId,
				channelId: info.channelId,
				serverId: info.serverId ?? null,
				callType: info.callType,
				state: info.state,
				initiatorId: info.initiatorId ?? null,
				startedAt: info.startedAt ? new Date(info.startedAt) : null,
				isActive: info.state !== 'idle' && info.state !== 'ended'
			}));
		}
	};
}

export const videoCallStore = createVideoCallStore();

// Derived stores
export const isInVideoCall = derived(videoCallStore, $state => $state.isActive);
export const videoCallParticipants = derived(videoCallStore, $state => $state.participants);
export const videoCallState = derived(videoCallStore, $state => $state.state);
export const incomingVideoRing = derived(videoCallStore, $state => $state.incomingRing);
export const isScreenSharing = derived(videoCallStore, $state => $state.localScreenShare);
export const isCameraOn = derived(videoCallStore, $state => $state.localCameraOn);
export const isMuted = derived(videoCallStore, $state => $state.localMuted);

// Setup gateway event listeners
if (browser) {
	// Incoming ring
	onGatewayEvent('VIDEO_RING_START', (data: unknown) => {
		const ring = data as VideoRingData;
		videoCallStore.setIncomingRing(ring);
	});

	// Call accepted by target
	onGatewayEvent('VIDEO_RING_ACCEPT', (data: unknown) => {
		const accept = data as { call_id: string; from_user_id: string; from_username: string };
		videoCallStore.setCallState('connecting');
	});

	// Call declined by target
	onGatewayEvent('VIDEO_RING_DECLINE', (data: unknown) => {
		const decline = data as { call_id: string; from_user_id: string; from_username: string };
		videoCallStore.setError('Call declined');
		videoCallStore.setCallState('ended');
		// Reset after a short delay
		setTimeout(() => videoCallStore.reset(), 2000);
	});

	// Call ended
	onGatewayEvent('VIDEO_RING_END', (data: unknown) => {
		const end = data as { call_id: string; reason?: string };
		videoCallStore.setCallState('ended');
		setTimeout(() => videoCallStore.reset(), 2000);
	});

	// Full call state update
	onGatewayEvent('VIDEO_SERVER_UPDATE', (data: unknown) => {
		const update = data as VideoCallData;
		
		if (update.call_id) {
			videoCallStore.setCallInfo({
				callId: update.call_id,
				channelId: update.channel_id,
				serverId: update.server_id,
				callType: update.call_type,
				state: update.state,
				initiatorId: update.initiator_id,
				startedAt: update.started_at
			});
		}

		if (update.participants) {
			videoCallStore.setParticipants(
				update.participants.map(p => ({
					id: p.id,
					username: p.username,
					display_name: p.display_name ?? null,
					avatar: p.avatar ?? null,
					isCameraOn: p.isCameraOn,
					isScreenShare: p.isScreenShare,
					isMuted: p.isMuted,
					isSpeaking: false,
					joinedAt: new Date()
				}))
			);
		}
	});

	// Partial state update (participant joined/left, state changed)
	onGatewayEvent('VIDEO_STATE_UPDATE', (data: unknown) => {
		const update = data as {
			call_id: string;
			user_id?: string;
			is_joined?: boolean;
			is_left?: boolean;
			is_camera_on?: boolean;
			is_muted?: boolean;
			is_screen_share?: boolean;
		};

		const currentUser = get(authUser);

		if (update.is_left && update.user_id) {
			videoCallStore.removeParticipant(update.user_id);
		} else if (update.is_joined && update.user_id) {
			videoCallStore.addParticipant({
				id: update.user_id,
				username: 'Unknown',
				display_name: null,
				avatar: null,
				isCameraOn: update.is_camera_on ?? false,
				isScreenShare: update.is_screen_share ?? false,
				isMuted: update.is_muted ?? true,
				isSpeaking: false,
				joinedAt: new Date()
			});
		} else if (update.user_id && update.user_id !== currentUser?.id) {
			// Update another participant's state
			videoCallStore.updateParticipant(update.user_id, {
				isCameraOn: update.is_camera_on,
				isScreenShare: update.is_screen_share,
				isMuted: update.is_muted
			});
		}
	});
}

// Video call actions
export const videoCallActions = {
	// Start a video call
	startCall(channelId: string, toUserId?: string, serverId?: string) {
		videoCallStore.startCall({
			channelId,
			serverId,
			callType: toUserId ? 'direct' : 'channel',
			toUserId
		});
	},

	// Accept incoming call
	acceptCall() {
		videoCallStore.acceptCall();
	},

	// Decline incoming call
	declineCall() {
		videoCallStore.declineCall();
	},

	// End current call
	endCall() {
		videoCallStore.endCall();
	},

	// Toggle camera
	toggleCamera() {
		videoCallStore.toggleCamera();
	},

	// Toggle mute
	toggleMute() {
		videoCallStore.toggleMute();
	},

	// Toggle screen share
	toggleScreenShare() {
		videoCallStore.toggleScreenShare();
	}
};
