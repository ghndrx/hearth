/**
 * E2EE Service
 * 
 * High-level service for E2EE operations that integrates with the
 * message sending/receiving flow.
 * 
 * Responsibilities:
 * - Initialize E2EE on app startup
 * - Encrypt messages before sending
 * - Decrypt messages when received
 * - Manage session state per conversation
 */

import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { e2ee, e2eeReady, type E2EESession } from '$lib/stores/e2ee';
import { channels, type Channel } from '$lib/stores/channels';
import { encryptMessage, decryptMessage, type EncryptedMessage } from '$lib/crypto/encryption';
import { isE2EESupported } from '$lib/crypto/signal-protocol';
import { secureKeyStore } from '$lib/crypto/secure-storage';
import { initializeWasm, isWasmReady, isUsingFallback, onWasmStateChange, type WasmState } from './wasm-loader';

// Backend message format for encrypted content
export interface BackendEncryptedContent {
	version: number;
	ciphertext: string;  // Base64
	iv: string;          // Base64
	sender_key_id: number;
	recipient_key_id: number;
}

let initialized = false;
let initPromise: Promise<boolean> | null = null;

/**
 * Initialize E2EE for the current user
 * Should be called once on app startup after authentication
 */
export async function initializeE2EE(): Promise<boolean> {
	if (!browser) return false;
	
	if (initialized) {
		return e2eeReady ? get(e2eeReady) : false;
	}
	
	if (initPromise) {
		return initPromise;
	}
	
	initPromise = doInitialize();
	return initPromise;
}

async function doInitialize(): Promise<boolean> {
	console.info('[E2EE Service] Initializing...');
	
	// Check browser support
	if (!isE2EESupported()) {
		console.warn('[E2EE Service] E2EE not supported in this browser');
		return false;
	}
	
	try {
		// Initialize secure storage
		await secureKeyStore.init();
		
		// Initialize WASM (optional - will fallback to WebCrypto if unavailable)
		await initializeWasm();
		
		// Initialize the E2EE store (loads/generates keys)
		const success = await e2ee.initialize();
		
		if (success) {
			initialized = true;
			console.info('[E2EE Service] Initialized successfully');
			console.info('[E2EE Service] Using WASM:', isWasmReady(), '| Fallback:', isUsingFallback());
		} else {
			console.warn('[E2EE Service] Initialization returned false');
		}
		
		return success;
	} catch (error) {
		console.error('[E2EE Service] Initialization failed:', error);
		return false;
	}
}

/**
 * Subscribe to E2EE state changes
 */
export function onE2EEStateChange(handler: (state: WasmState) => void): () => void {
	return onWasmStateChange((state) => handler(state));
}

/**
 * Check if a channel has E2EE enabled
 */
export function isChannelE2EEEnabled(channelId: string): boolean {
	const $channels = get(channels);
	const channel = $channels.find(c => c.id === channelId);
	return channel?.e2ee_enabled ?? false;
}

/**
 * Encrypt a message for a specific channel
 * 
 * @param plaintext - The message text to encrypt
 * @param channelId - The channel ID to encrypt for
 * @returns The encrypted message data to send to backend, or null if encryption fails
 */
export async function encryptMessageForChannel(
	plaintext: string,
	channelId: string,
	recipientUserId: string
): Promise<{
	encrypted_content: string;
	encrypted: boolean;
} | null> {
	const $e2ee = get(e2ee);
	
	if (!$e2ee.initialized) {
		console.warn('[E2EE Service] E2EE not initialized, cannot encrypt');
		return null;
	}
	
	try {
		// Get or establish session with the recipient
		const session = await e2ee.getOrEstablishSession(recipientUserId);
		
		if (!session || !session.encryptionKey) {
			console.error('[E2EE Service] No session available for encryption');
			return null;
		}
		
		// Encrypt using the session's encryption key
		const encrypted = await encryptMessage(
			plaintext,
			session.encryptionKey,
			$e2ee.registrationId || 0,
			session.messageNumber
		);
		
		// Format for backend - the backend expects encrypted_content as a JSON string
		const backendFormat: BackendEncryptedContent = {
			version: encrypted.version,
			ciphertext: encrypted.ciphertext,
			iv: encrypted.iv,
			sender_key_id: encrypted.senderKeyId,
			recipient_key_id: encrypted.recipientKeyId,
		};
		
		return {
			encrypted_content: JSON.stringify(backendFormat),
			encrypted: true,
		};
	} catch (error) {
		console.error('[E2EE Service] Encryption failed:', error);
		return null;
	}
}

/**
 * Decrypt a message from a specific sender
 * 
 * @param encryptedContent - The encrypted content string from backend
 * @param senderUserId - The user ID of the message sender
 * @param senderDeviceId - The device ID of the sender
 * @returns The decrypted plaintext, or null if decryption fails
 */
export async function decryptMessageFromSender(
	encryptedContent: string,
	senderUserId: string,
	senderDeviceId: string
): Promise<string | null> {
	const $e2ee = get(e2ee);
	
	if (!$e2ee.initialized) {
		console.warn('[E2EE Service] E2EE not initialized, cannot decrypt');
		return null;
	}
	
	try {
		// Parse the encrypted content
		let parsed: BackendEncryptedContent;
		try {
			parsed = JSON.parse(encryptedContent);
		} catch {
			console.error('[E2EE Service] Failed to parse encrypted content');
			return null;
		}
		
		// Convert to frontend format
		const encrypted: EncryptedMessage = {
			version: parsed.version,
			ciphertext: parsed.ciphertext,
			iv: parsed.iv,
			tag: '',
			senderKeyId: parsed.sender_key_id,
			recipientKeyId: parsed.recipient_key_id,
		};
		
		// Decrypt using the session
		const plaintext = await e2ee.decryptMessageFromUser(
			encrypted,
			senderUserId,
			senderDeviceId
		);
		
		return plaintext;
	} catch (error) {
		console.error('[E2EE Service] Decryption failed:', error);
		return null;
	}
}

/**
 * Handle an incoming message that may be encrypted
 * Returns the decrypted content if encrypted, or the original content if not
 */
export async function handleIncomingMessage(
	message: {
		content: string;
		encrypted: boolean;
		encrypted_content?: string;
		author_id?: string;
		author?: { id: string };
	},
	senderDeviceId?: string
): Promise<string> {
	// If not encrypted, return original content
	if (!message.encrypted || !message.encrypted_content) {
		return message.content;
	}
	
	// Get sender user ID
	const senderUserId = message.author_id || message.author?.id;
	if (!senderUserId) {
		console.warn('[E2EE Service] Cannot decrypt: no sender user ID');
		return message.content;
	}
	
	// Try to decrypt
	const decrypted = await decryptMessageFromSender(
		message.encrypted_content,
		senderUserId,
		senderDeviceId || 'default'
	);
	
	return decrypted ?? message.content;
}

/**
 * Prepare a message for sending - encrypt if E2EE is enabled for the channel
 */
export async function prepareMessageForSending(
	plaintext: string,
	channelId: string,
	recipientUserId: string
): Promise<{
	content: string;
	encrypted: boolean;
	encrypted_content?: string;
}> {
	// Check if channel has E2EE enabled
	const $channels = get(channels);
	const channel = $channels.find(c => c.id === channelId);
	
	// If not an E2EE channel, send as plaintext
	if (!channel?.e2ee_enabled) {
		return {
			content: plaintext,
			encrypted: false,
		};
	}
	
	// Try to encrypt
	const encrypted = await encryptMessageForChannel(plaintext, channelId, recipientUserId);
	
	if (encrypted) {
		// Return encrypted format
		return {
			content: '', // Backend expects empty content for encrypted messages
			encrypted: true,
			encrypted_content: encrypted.encrypted_content,
		};
	}
	
	// Encryption failed but channel requires E2EE
	// This should not happen in normal flow
	console.error('[E2EE Service] Failed to encrypt for E2EE channel, falling back');
	return {
		content: plaintext,
		encrypted: false,
	};
}

/**
 * Get session info for debugging
 */
export function getSessionInfo(): {
	initialized: boolean;
	ready: boolean;
	wasmState: string;
	sessions: number;
} {
	const $e2ee = get(e2ee);
	return {
		initialized: $e2ee.initialized,
		ready: $e2ee.supported,
		wasmState: isUsingFallback() ? 'webcrypto' : (isWasmReady() ? 'wasm' : 'uninitialized'),
		sessions: $e2ee.sessions.size,
	};
}

/**
 * Reset E2EE state (for logout)
 */
export async function resetE2EE(): Promise<void> {
	initialized = false;
	initPromise = null;
	await e2ee.clear();
}
