import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { api } from '$lib/api';
import { currentServer } from './servers';
import { voiceState } from './voice';

export interface ServerAudioSettings {
	user_id: string;
	server_id: string;
	input_device_id: string;
	output_device_id: string;
	input_volume: number;
	output_volume: number;
	push_to_talk_enabled: boolean;
	push_to_talk_key: string;
	updated_at: string;
}

export interface AudioSettingsState {
	// Map of server_id -> settings
	serverSettings: Record<string, ServerAudioSettings>;
	// Currently active audio devices (applied from current server's settings)
	activeInputDeviceId: string;
	activeOutputDeviceId: string;
	activeInputVolume: number;
	activeOutputVolume: number;
	// Push-to-talk state
	pushToTalkEnabled: boolean;
	pushToTalkKey: string;
	pushToTalkActive: boolean;
	// Loading state
	loading: boolean;
}

const initialState: AudioSettingsState = {
	serverSettings: {},
	activeInputDeviceId: '',
	activeOutputDeviceId: '',
	activeInputVolume: 100,
	activeOutputVolume: 100,
	pushToTalkEnabled: false,
	pushToTalkKey: '',
	pushToTalkActive: false,
	loading: false,
};

function createAudioSettingsStore() {
	const { subscribe, set, update } = writable<AudioSettingsState>(initialState);

	let pttKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
	let pttKeyupHandler: ((e: KeyboardEvent) => void) | null = null;

	function setupPushToTalk(key: string, enabled: boolean) {
		// Clean up old listeners
		if (pttKeydownHandler) {
			window.removeEventListener('keydown', pttKeydownHandler);
			pttKeydownHandler = null;
		}
		if (pttKeyupHandler) {
			window.removeEventListener('keyup', pttKeyupHandler);
			pttKeyupHandler = null;
		}

		if (!enabled || !key) return;

		pttKeydownHandler = (e: KeyboardEvent) => {
			if (e.code === key && !e.repeat) {
				update(s => ({ ...s, pushToTalkActive: true }));
			}
		};
		pttKeyupHandler = (e: KeyboardEvent) => {
			if (e.code === key) {
				update(s => ({ ...s, pushToTalkActive: false }));
			}
		};

		window.addEventListener('keydown', pttKeydownHandler);
		window.addEventListener('keyup', pttKeyupHandler);
	}

	function applyServerSettings(settings: ServerAudioSettings | null) {
		update(s => {
			const inputDeviceId = settings?.input_device_id ?? '';
			const outputDeviceId = settings?.output_device_id ?? '';
			const inputVolume = settings?.input_volume ?? 100;
			const outputVolume = settings?.output_volume ?? 100;
			const pttEnabled = settings?.push_to_talk_enabled ?? false;
			const pttKey = settings?.push_to_talk_key ?? '';

			if (browser) {
				setupPushToTalk(pttKey, pttEnabled);
			}

			return {
				...s,
				activeInputDeviceId: inputDeviceId,
				activeOutputDeviceId: outputDeviceId,
				activeInputVolume: inputVolume,
				activeOutputVolume: outputVolume,
				pushToTalkEnabled: pttEnabled,
				pushToTalkKey: pttKey,
				pushToTalkActive: false,
			};
		});
	}

	return {
		subscribe,

		// Load all server audio settings for the current user
		async loadAll() {
			update(s => ({ ...s, loading: true }));
			try {
				const data = await api.get<ServerAudioSettings[]>('/users/@me/audio-settings');
				const map: Record<string, ServerAudioSettings> = {};
				for (const s of data) {
					map[s.server_id] = s;
				}
				update(s => ({ ...s, serverSettings: map, loading: false }));
			} catch {
				update(s => ({ ...s, loading: false }));
			}
		},

		// Load settings for a specific server
		async loadForServer(serverId: string) {
			try {
				const data = await api.get<ServerAudioSettings>(`/servers/${serverId}/audio-settings`);
				update(s => ({
					...s,
					serverSettings: { ...s.serverSettings, [serverId]: data },
				}));
				return data;
			} catch {
				return null;
			}
		},

		// Update settings for a specific server
		async updateForServer(serverId: string, updates: Partial<Omit<ServerAudioSettings, 'user_id' | 'server_id' | 'updated_at'>>) {
			try {
				const data = await api.patch<ServerAudioSettings>(`/servers/${serverId}/audio-settings`, updates);
				update(s => ({
					...s,
					serverSettings: { ...s.serverSettings, [serverId]: data },
				}));

				// If this is the currently active server, apply settings
				const voice = get(voiceState);
				if (voice.serverId === serverId) {
					applyServerSettings(data);
				}

				return data;
			} catch {
				return null;
			}
		},

		// Apply settings for a given server (called on server switch)
		switchToServer(serverId: string) {
			const state = get({ subscribe });
			const settings = state.serverSettings[serverId] || null;
			applyServerSettings(settings);
		},

		// Reset settings for a server
		async resetForServer(serverId: string) {
			try {
				await api.delete(`/servers/${serverId}/audio-settings`);
				update(s => {
					const { [serverId]: _, ...rest } = s.serverSettings;
					return { ...s, serverSettings: rest };
				});
			} catch {
				// ignore
			}
		},

		// Set push-to-talk active state (for manual control)
		setPushToTalkActive(active: boolean) {
			update(s => ({ ...s, pushToTalkActive: active }));
		},

		reset() {
			if (pttKeydownHandler) {
				window.removeEventListener('keydown', pttKeydownHandler);
				pttKeydownHandler = null;
			}
			if (pttKeyupHandler) {
				window.removeEventListener('keyup', pttKeyupHandler);
				pttKeyupHandler = null;
			}
			set(initialState);
		},
	};
}

export const audioSettings = createAudioSettingsStore();

// Derived store: current server's audio settings
export const currentServerAudioSettings = derived(
	[audioSettings, currentServer],
	([$audioSettings, $currentServer]) => {
		if (!$currentServer) return null;
		return $audioSettings.serverSettings[$currentServer.id] || null;
	}
);

// Derived store: is push-to-talk currently active (key held)
export const isPushToTalkActive = derived(
	audioSettings,
	$s => $s.pushToTalkEnabled && $s.pushToTalkActive
);

// Derived store: is push-to-talk enabled for current context
export const isPushToTalkEnabled = derived(
	audioSettings,
	$s => $s.pushToTalkEnabled
);

// Auto-switch audio devices when the voice channel's server changes
if (browser) {
	let lastVoiceServerId: string | null = null;

	voiceState.subscribe(($voice) => {
		if ($voice.serverId && $voice.serverId !== lastVoiceServerId && $voice.isConnected) {
			lastVoiceServerId = $voice.serverId;
			audioSettings.switchToServer($voice.serverId);
		}
		if (!$voice.isConnected) {
			lastVoiceServerId = null;
		}
	});
}
