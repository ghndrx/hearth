import { writable } from 'svelte/store';
import { api, ApiError } from '$lib/api';

// Types
export interface Rule {
	id: string;
	order: number;
	title: string;
	description: string;
}

export interface ScreeningQuestion {
	id: string;
	order: number;
	question: string;
	required: boolean;
	type: 'text' | 'select' | 'agree';
	options?: string[];
}

export interface ScreeningAnswer {
	question_id: string;
	answer: string;
}

export interface WelcomeScreen {
	id: string;
	server_id: string;
	enabled: boolean;
	title: string;
	description: string;
	welcome_channels: string[];
	updated_at: string;
	created_at: string;
}

export interface WelcomeScreenConfig extends WelcomeScreen {
	rules: Rule[];
	questions: ScreeningQuestion[];
}

export interface MemberScreening {
	id: string;
	user_id: string;
	server_id: string;
	answers: ScreeningAnswer[];
	rules_read: boolean;
	status: 'pending' | 'approved' | 'rejected';
	created_at: string;
	updated_at: string;
}

// Stores
export const welcomeScreenConfig = writable<WelcomeScreenConfig | null>(null);
export const myScreening = writable<MemberScreening | null>(null);
export const pendingScreenings = writable<MemberScreening[]>([]);
export const welcomeLoading = writable(false);
export const welcomeError = writable<string | null>(null);

// API functions
export async function loadWelcomeScreen(serverId: string): Promise<WelcomeScreenConfig | null> {
	welcomeLoading.set(true);
	welcomeError.set(null);

	try {
		const config = await api.get<WelcomeScreenConfig>(`/servers/${serverId}/welcome`);
		welcomeScreenConfig.set(config);
		return config;
	} catch (error) {
		console.error('Failed to load welcome screen:', error);
		if (error instanceof ApiError && error.status !== 404) {
			welcomeError.set(error.message);
		}
		return null;
	} finally {
		welcomeLoading.set(false);
	}
}

export async function updateWelcomeScreen(
	serverId: string,
	updates: Partial<WelcomeScreenConfig>
): Promise<WelcomeScreenConfig> {
	welcomeLoading.set(true);
	welcomeError.set(null);

	try {
		const config = await api.put<WelcomeScreenConfig>(`/servers/${serverId}/welcome`, updates);
		welcomeScreenConfig.set(config);
		return config;
	} catch (error) {
		console.error('Failed to update welcome screen:', error);
		if (error instanceof ApiError) {
			welcomeError.set(error.message);
		}
		throw error;
	} finally {
		welcomeLoading.set(false);
	}
}

export async function submitScreening(
	serverId: string,
	answers: ScreeningAnswer[],
	rulesRead: boolean
): Promise<MemberScreening> {
	const screening = await api.post<MemberScreening>(`/servers/${serverId}/screening`, {
		answers,
		rules_read: rulesRead,
	});
	myScreening.set(screening);
	return screening;
}

export async function getMyScreening(serverId: string): Promise<MemberScreening | null> {
	try {
		const screening = await api.get<MemberScreening>(`/servers/${serverId}/screening/me`);
		myScreening.set(screening);
		return screening;
	} catch (error) {
		if (error instanceof ApiError && error.status === 404) {
			myScreening.set(null);
			return null;
		}
		throw error;
	}
}

export async function loadPendingScreenings(
	serverId: string,
	limit = 50,
	offset = 0
): Promise<MemberScreening[]> {
	welcomeLoading.set(true);
	welcomeError.set(null);

	try {
		const screenings = await api.get<MemberScreening[]>(
			`/servers/${serverId}/screening/pending?limit=${limit}&offset=${offset}`
		);
		pendingScreenings.set(screenings);
		return screenings;
	} catch (error) {
		console.error('Failed to load pending screenings:', error);
		if (error instanceof ApiError) {
			welcomeError.set(error.message);
		}
		throw error;
	} finally {
		welcomeLoading.set(false);
	}
}

export async function approveScreening(serverId: string, userId: string): Promise<void> {
	await api.post(`/servers/${serverId}/screening/${userId}/approve`);
	pendingScreenings.update((s) => s.filter((screen) => screen.user_id !== userId));
}

export async function rejectScreening(serverId: string, userId: string, reason?: string): Promise<void> {
	await api.post(`/servers/${serverId}/screening/${userId}/reject`, { reason });
	pendingScreenings.update((s) => s.filter((screen) => screen.user_id !== userId));
}
