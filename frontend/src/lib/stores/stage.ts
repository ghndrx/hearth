import { writable, derived } from 'svelte/store';
import { api, ApiError } from '$lib/api';

// Stage status constants
export const StageStatus = {
	SCHEDULED: 1,
	LIVE: 2,
	PAUSED: 3,
	ENDED: 4
} as const;

export type StageStatusType = typeof StageStatus[keyof typeof StageStatus];

// Stage role constants
export const StageRole = {
	AUDIENCE: 1,
	SPEAKER: 2,
	MODERATOR: 3,
	HOST: 4
} as const;

export type StageRoleType = typeof StageRole[keyof typeof StageRole];

// Stage interface
export interface Stage {
	id: string;
	channel_id: string;
	topic: string;
	description: string;
	status: StageStatusType;
	host_user_id: string;
	discovery_disabled: boolean;
	request_to_speak: boolean;
	moderator_only: boolean;
	max_speakers: number;
	speaker_count: number;
	audience_count: number;
	pending_request_count: number;
	created_at: string;
	started_at: string | null;
	ended_at: string | null;
}

// Participant interface
export interface StageParticipant {
	user_id: string;
	role: StageRoleType;
	joined_at: string;
	is_muted: boolean;
	is_deafened: boolean;
	has_pending_request: boolean;
	requested_at: string | null;
}

// Create stage request
export interface CreateStageRequest {
	topic: string;
	description?: string;
	discovery_disabled?: boolean;
	request_to_speak?: boolean;
	moderator_only?: boolean;
	max_speakers?: number;
}

// Update stage request
export interface UpdateStageRequest {
	topic?: string;
	description?: string;
}

// Stage configuration
export interface StageConfig {
	discovery_disabled?: boolean;
	request_to_speak?: boolean;
	moderator_only?: boolean;
	max_speakers?: number;
}

// Stage stores
export const currentStage = writable<Stage | null>(null);
export const stageParticipants = writable<StageParticipant[]>([]);
export const stageLoading = writable(false);
export const stageError = writable<string | null>(null);

// Derived stores
export const stageSpeakers = derived(stageParticipants, $participants => 
	$participants.filter(p => p.role === StageRole.SPEAKER || p.role === StageRole.MODERATOR || p.role === StageRole.HOST)
);

export const stageAudience = derived(stageParticipants, $participants => 
	$participants.filter(p => p.role === StageRole.AUDIENCE)
);

export const pendingSpeakers = derived(stageParticipants, $participants => 
	$participants.filter(p => p.has_pending_request)
);

// Helper function to check if a user is in the stage
export function isUserInStage(participants: StageParticipant[], userId: string): boolean {
	return participants.some(p => p.user_id === userId);
}

// API functions
export async function createStage(channelId: string, request: CreateStageRequest): Promise<Stage> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		const stage = await api.post<Stage>(`/channels/${channelId}/stage`, request);
		currentStage.set(stage);
		return stage;
	} catch (error) {
		console.error('Failed to create stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function getStage(channelId: string): Promise<Stage | null> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		const stage = await api.get<Stage>(`/channels/${channelId}/stage`);
		currentStage.set(stage);
		return stage;
	} catch (error) {
		if (error instanceof ApiError && error.status === 404) {
			currentStage.set(null);
			return null;
		}
		console.error('Failed to get stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function updateStage(stageId: string, request: UpdateStageRequest): Promise<Stage> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		const stage = await api.patch<Stage>(`/stages/${stageId}`, request);
		currentStage.set(stage);
		return stage;
	} catch (error) {
		console.error('Failed to update stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function updateStageConfig(stageId: string, config: StageConfig): Promise<Stage> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		const stage = await api.patch<Stage>(`/stages/${stageId}/config`, config);
		currentStage.set(stage);
		return stage;
	} catch (error) {
		console.error('Failed to update stage config:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function pauseStage(stageId: string): Promise<Stage> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		const stage = await api.post<Stage>(`/stages/${stageId}/pause`);
		currentStage.set(stage);
		return stage;
	} catch (error) {
		console.error('Failed to pause stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function resumeStage(stageId: string): Promise<Stage> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		const stage = await api.post<Stage>(`/stages/${stageId}/resume`);
		currentStage.set(stage);
		return stage;
	} catch (error) {
		console.error('Failed to resume stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function endStage(stageId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.delete(`/stages/${stageId}`);
		currentStage.set(null);
		stageParticipants.set([]);
	} catch (error) {
		console.error('Failed to end stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function joinStage(stageId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.post(`/stages/${stageId}/join`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to join stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function leaveStage(stageId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.post(`/stages/${stageId}/leave`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to leave stage:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function requestToSpeak(stageId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.post(`/stages/${stageId}/request-to-speak`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to request to speak:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function cancelRequestToSpeak(stageId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.delete(`/stages/${stageId}/request-to-speak`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to cancel request to speak:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function approveSpeaker(stageId: string, userId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.post(`/stages/${stageId}/approve/${userId}`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to approve speaker:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function denySpeaker(stageId: string, userId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.post(`/stages/${stageId}/deny/${userId}`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to deny speaker:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function promoteToSpeaker(stageId: string, userId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.post(`/stages/${stageId}/promote/${userId}`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to promote to speaker:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function demoteToAudience(stageId: string, userId: string): Promise<void> {
	stageLoading.set(true);
	stageError.set(null);
	
	try {
		await api.post(`/stages/${stageId}/demote/${userId}`);
		// Refresh participants
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to demote to audience:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	} finally {
		stageLoading.set(false);
	}
}

export async function loadParticipants(stageId: string): Promise<StageParticipant[]> {
	try {
		const participants = await api.get<StageParticipant[]>(`/stages/${stageId}/participants`);
		stageParticipants.set(participants);
		return participants;
	} catch (error) {
		console.error('Failed to load participants:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	}
}

export async function loadPendingRequests(stageId: string): Promise<StageParticipant[]> {
	try {
		const requests = await api.get<StageParticipant[]>(`/stages/${stageId}/requests`);
		return requests;
	} catch (error) {
		console.error('Failed to load pending requests:', error);
		if (error instanceof ApiError) {
			stageError.set(error.message);
		}
		throw error;
	}
}

export async function muteParticipant(stageId: string, userId: string): Promise<void> {
	try {
		await api.post(`/stages/${stageId}/mute/${userId}`);
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to mute participant:', error);
		throw error;
	}
}

export async function unmuteParticipant(stageId: string, userId: string): Promise<void> {
	try {
		await api.post(`/stages/${stageId}/unmute/${userId}`);
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to unmute participant:', error);
		throw error;
	}
}

export async function addModerator(stageId: string, userId: string): Promise<void> {
	try {
		await api.post(`/stages/${stageId}/moderators/${userId}`);
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to add moderator:', error);
		throw error;
	}
}

export async function removeModerator(stageId: string, userId: string): Promise<void> {
	try {
		await api.delete(`/stages/${stageId}/moderators/${userId}`);
		await loadParticipants(stageId);
	} catch (error) {
		console.error('Failed to remove moderator:', error);
		throw error;
	}
}

// Helper to check if user is host or moderator
export function isHostOrModerator(participant: StageParticipant | undefined): boolean {
	if (!participant) return false;
	return participant.role === StageRole.HOST || participant.role === StageRole.MODERATOR;
}

// Helper to check if user is host
export function isHost(participant: StageParticipant | undefined): boolean {
	if (!participant) return false;
	return participant.role === StageRole.HOST;
}

// Helper to get status label
export function getStatusLabel(status: StageStatusType): string {
	switch (status) {
		case StageStatus.SCHEDULED:
			return 'Scheduled';
		case StageStatus.LIVE:
			return 'Live';
		case StageStatus.PAUSED:
			return 'Paused';
		case StageStatus.ENDED:
			return 'Ended';
		default:
			return 'Unknown';
	}
}

// Helper to get role label
export function getRoleLabel(role: StageRoleType): string {
	switch (role) {
		case StageRole.AUDIENCE:
			return 'Audience';
		case StageRole.SPEAKER:
			return 'Speaker';
		case StageRole.MODERATOR:
			return 'Moderator';
		case StageRole.HOST:
			return 'Host';
		default:
			return 'Unknown';
	}
}
