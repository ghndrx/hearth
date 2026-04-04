import { writable, get } from 'svelte/store';
import { api } from '$lib/api';
import { gateway, onGatewayEvent } from '$lib/stores/gateway';

export interface SoundboardSound {
	id: string;
	name: string;
	emoji_name?: string;
	volume: number;
	audio_url: string;
	duration_ms: number;
	available: boolean;
	creator_id: string;
	created_at: string;
	guild_id?: string;
}

export interface SoundboardSoundPack {
	id: string;
	guild_id?: string;
	name: string;
	emoji_name?: string;
	sounds: SoundboardSound[];
	is_default: boolean;
	position: number;
	created_at: string;
	updated_at: string;
}

export interface SoundboardPlayEvent {
	sound_id: string;
	sound_name: string;
	emoji_name?: string;
	audio_url: string;
	volume: number;
	duration_ms: number;
	user_id: string;
	channel_id: string;
	server_id: string;
}

// Soundboard API functions
export async function fetchServerSounds(serverId: string): Promise<SoundboardSound[]> {
	const response = await api.get<SoundboardSound[]>(`/servers/${serverId}/soundboard`);
	return response || [];
}

export async function fetchDefaultSounds(): Promise<SoundboardSound[]> {
	const response = await api.get<SoundboardSound[]>('/soundboard/defaults');
	return response || [];
}

export async function fetchServerPacks(serverId: string): Promise<SoundboardSoundPack[]> {
	const response = await api.get<SoundboardSoundPack[]>(`/servers/${serverId}/soundboard/packs`);
	return response || [];
}

export async function createSound(serverId: string, data: {
	name: string;
	emoji_name?: string;
	volume?: number;
	audio_url?: string;
}): Promise<SoundboardSound> {
	return api.post<SoundboardSound>(`/servers/${serverId}/soundboard`, data);
}

export async function createPack(serverId: string, data: {
	name: string;
	emoji_name?: string;
	is_default?: boolean;
}): Promise<SoundboardSoundPack> {
	return api.post<SoundboardSoundPack>(`/servers/${serverId}/soundboard/packs`, data);
}

// Soundboard WebSocket signaling
export function sendSoundboardPlay(channelId: string, serverId: string, soundId: string, volume?: number): void {
	gateway.send({
		op: 0, // Dispatch
		d: {
			t: 'SOUNDBOARD_PLAY',
			d: {
				sound_id: soundId,
				channel_id: channelId,
				server_id: serverId,
				volume: volume ?? 1.0,
			},
		},
	});
}

export function sendSoundboardStop(channelId: string, serverId: string, soundId: string): void {
	gateway.send({
		op: 0, // Dispatch
		d: {
			t: 'SOUNDBOARD_STOP',
			d: {
				sound_id: soundId,
				channel_id: channelId,
				server_id: serverId,
			},
		},
	});
}

// Soundboard hotkey store
export interface HotkeyMapping {
	soundId: string;
	soundName: string;
	emojiName?: string;
	audioUrl: string;
	volume: number;
	durationMs: number;
}

export const soundboardHotkeys = writable<Map<number, HotkeyMapping>>(new Map());

// F1-F9 key codes
const HOTKEY_KEYS = [
	'F1', 'F2', 'F3', 'F4', 'F5', 'F6', 'F7', 'F8', 'F9'
];

let currentHotkeyCleanup: (() => void) | null = null;

export function setupSoundboardHotkeys(
	sounds: SoundboardSound[],
	channelId: string,
	serverId: string,
	onPlay?: (sound: SoundboardSound) => void
): () => void {
	// Clear previous hotkeys
	if (currentHotkeyCleanup) {
		currentHotkeyCleanup();
		currentHotkeyCleanup = null;
	}

	// Map first 9 sounds to F1-F9
	const mappings = new Map<number, HotkeyMapping>();
	sounds.slice(0, 9).forEach((sound, index) => {
		mappings.set(index + 1, {
			soundId: sound.id,
			soundName: sound.name,
			emojiName: sound.emoji_name,
			audioUrl: sound.audio_url,
			volume: sound.volume,
			durationMs: sound.duration_ms,
		});
	});
	soundboardHotkeys.set(mappings);

	// Audio element for playing sounds
	let currentAudio: HTMLAudioElement | null = null;

	function playSound(mapping: HotkeyMapping) {
		// Stop any current sound
		if (currentAudio) {
			currentAudio.pause();
			currentAudio = null;
		}

		// Play the sound
		currentAudio = new Audio(mapping.audioUrl);
		currentAudio.volume = mapping.volume;
		currentAudio.play().catch(err => {
			console.error('[Soundboard] Failed to play sound:', err);
		});

		// Send WebSocket event
		sendSoundboardPlay(channelId, serverId, mapping.soundId, mapping.volume);

		if (onPlay) {
			onPlay({
				id: mapping.soundId,
				name: mapping.soundName,
				emoji_name: mapping.emojiName,
				audio_url: mapping.audioUrl,
				volume: mapping.volume,
				duration_ms: mapping.durationMs,
				available: true,
				creator_id: '',
				created_at: '',
			});
		}

		// Auto-stop after duration
		setTimeout(() => {
			if (currentAudio) {
				currentAudio.pause();
				currentAudio = null;
			}
		}, mapping.durationMs);
	}

	function handleKeydown(event: KeyboardEvent) {
		// Check if we're in an input field
		const target = event.target as HTMLElement;
		if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
			return;
		}

		const keyIndex = HOTKEY_KEYS.indexOf(event.key);
		if (keyIndex !== -1) {
			event.preventDefault();
			const mapping = mappings.get(keyIndex + 1);
			if (mapping) {
				playSound(mapping);
			}
		}
	}

	document.addEventListener('keydown', handleKeydown);

	currentHotkeyCleanup = () => {
		document.removeEventListener('keydown', handleKeydown);
		if (currentAudio) {
			currentAudio.pause();
			currentAudio = null;
		}
	};

	return currentHotkeyCleanup;
}

export function cleanupSoundboardHotkeys(): void {
	if (currentHotkeyCleanup) {
		currentHotkeyCleanup();
		currentHotkeyCleanup = null;
	}
	soundboardHotkeys.set(new Map());
}
