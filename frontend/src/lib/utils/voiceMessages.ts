import type { Message } from '$lib/stores/messages';

export interface VoiceMessageData {
	voice_message: true;
	duration_ms: number;
	waveform_data: number[];
}

/**
 * Check if a message is a voice message by examining its content
 */
export function isVoiceMessage(message: Message): boolean {
	if (!message.content) return false;
	try {
		const data = JSON.parse(message.content);
		return data.voice_message === true;
	} catch {
		return false;
	}
}

/**
 * Parse voice message data from a message's content
 */
export function parseVoiceMessageData(message: Message): VoiceMessageData | null {
	if (!message.content) return null;
	try {
		const data = JSON.parse(message.content);
		if (data.voice_message === true) {
			return {
				voice_message: true,
				duration_ms: data.duration_ms || 0,
				waveform_data: data.waveform_data || []
			};
		}
		return null;
	} catch {
		return null;
	}
}
