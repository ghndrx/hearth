import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { api } from '$lib/api';

export type ChannelNotificationLevel = 'all_messages' | 'mentions_only' | 'nothing';

export interface ChannelNotificationOverride {
	id: string;
	user_id: string;
	channel_id: string;
	notification_level: ChannelNotificationLevel;
	created_at: string;
	updated_at: string;
}

interface ChannelOverridesState {
	overrides: Map<string, ChannelNotificationLevel>;
	loading: boolean;
	error: string | null;
}

const initialState: ChannelOverridesState = {
	overrides: new Map(),
	loading: false,
	error: null
};

function createChannelOverridesStore() {
	const { subscribe, set, update } = writable<ChannelOverridesState>(initialState);

	async function loadOverrides() {
		if (!browser) return;

		update(s => ({ ...s, loading: true, error: null }));

		try {
			const response = await api.get<{ overrides: ChannelNotificationOverride[] }>('/users/@me/notification-overrides');
			const overridesMap = new Map<string, ChannelNotificationLevel>();
			
			for (const override of response.overrides) {
				overridesMap.set(override.channel_id, override.notification_level);
			}

			update(s => ({
				...s,
				overrides: overridesMap,
				loading: false
			}));
		} catch (err) {
			update(s => ({
				...s,
				loading: false,
				error: err instanceof Error ? err.message : 'Failed to load overrides'
			}));
		}
	}

	async function setOverride(channelId: string, level: ChannelNotificationLevel) {
		if (!browser) return;

		update(s => ({ ...s, loading: true, error: null }));

		try {
			await api.put(`/users/@me/notification-overrides/${channelId}`, {
				notification_level: level
			});

			update(s => {
				const newOverrides = new Map(s.overrides);
				newOverrides.set(channelId, level);
				return {
					...s,
					overrides: newOverrides,
					loading: false
				};
			});
		} catch (err) {
			update(s => ({
				...s,
				loading: false,
				error: err instanceof Error ? err.message : 'Failed to set override'
			}));
		}
	}

	async function clearOverride(channelId: string) {
		if (!browser) return;

		update(s => ({ ...s, loading: true, error: null }));

		try {
			await api.delete(`/users/@me/notification-overrides/${channelId}`);

			update(s => {
				const newOverrides = new Map(s.overrides);
				newOverrides.delete(channelId);
				return {
					...s,
					overrides: newOverrides,
					loading: false
				};
			});
		} catch (err) {
			update(s => ({
				...s,
				loading: false,
				error: err instanceof Error ? err.message : 'Failed to clear override'
			}));
		}
	}

	function getOverride(channelId: string): ChannelNotificationLevel {
		return get({ subscribe }).overrides.get(channelId) ?? 'all_messages';
	}

	function hasOverride(channelId: string): boolean {
		return get({ subscribe }).overrides.has(channelId);
	}

	function reset() {
		set(initialState);
	}

	return {
		subscribe,
		loadOverrides,
		setOverride,
		clearOverride,
		getOverride,
		hasOverride,
		reset
	};
}

export const channelNotificationOverrides = createChannelOverridesStore();

// Derived store for checking if a channel has any override
export function createChannelOverrideDerived(channelId: string) {
	return derived(channelNotificationOverrides, $overrides => ({
		level: $overrides.overrides.get(channelId) ?? 'all_messages',
		hasOverride: $overrides.overrides.has(channelId),
		isMuted: $overrides.overrides.get(channelId) === 'nothing',
		isMentionsOnly: $overrides.overrides.get(channelId) === 'mentions_only'
	}));
}

// Helper function to get effective notification level for a channel
export function getEffectiveNotificationLevel(channelId: string): ChannelNotificationLevel {
	return channelNotificationOverrides.getOverride(channelId);
}

// Helper to check if notifications should be shown for a channel
export function shouldShowNotification(channelId: string, isMention: boolean): boolean {
	const level = getEffectiveNotificationLevel(channelId);
	
	switch (level) {
		case 'nothing':
			return false;
		case 'mentions_only':
			return isMention;
		case 'all_messages':
		default:
			return true;
	}
}
