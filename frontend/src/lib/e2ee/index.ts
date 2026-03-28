/**
 * E2EE Module - End-to-End Encryption Foundation
 * 
 * This module provides the foundational E2EE infrastructure for Hearth.
 * It supports both libsignal-client WASM and WebCrypto fallback.
 * 
 * @module e2ee
 */

import { browser } from '$app/environment';

// ============================================================================
// WASM Loader & State Management
// ============================================================================

export {
	initializeWasm,
	getWasmState,
	isWasmReady,
	isUsingFallback,
	isWasmAvailable,
	destroyWasm,
	getInitError,
	onWasmStateChange,
	getWasmModule,
} from './wasm-loader';

export type { 
	WasmState,
	X3DHResult, 
	SessionHandle,
} from './wasm-loader';

// ============================================================================
// Key Storage
// ============================================================================

export { keyStorage } from './key-storage';

export type {
	IdentityKeyData,
	SignedPreKeyData,
	OneTimePreKeyData,
	SessionData,
	DeviceMetadata,
} from './key-storage';

// Re-export KeyStore from crypto/keys for compatibility
export { KeyStore, keyStore } from '$lib/crypto/keys';

// ============================================================================
// Device Manager
// ============================================================================

export { deviceManager, DeviceManagerClass } from './device-manager';

export type {
	DeviceRegistration,
	DeviceInfo,
	PreKeyBundle,
} from './device-manager';

// ============================================================================
// Initialization
// ============================================================================

export { initE2EE, getE2EEStatus, isE2EEReady } from './init';

export type { E2EEInitResult } from './init';

// ============================================================================
// Signal Protocol Utilities (for message encryption/decryption)
// ============================================================================

export {
	signalEncrypt,
	signalDecrypt,
	serializeEncryptedPayload,
	deserializeEncryptedPayload,
	isSignalE2EESupported,
	isInitialized,
	decryptMessage,
} from './signal';

export type { EncryptedPayload } from './signal';

// ============================================================================
// X3DH Key Exchange (from crypto/signal-protocol)
// ============================================================================

export {
	performX3DHSender,
	performX3DHRecipient,
	generateDeviceId,
	generateRegistrationId,
	generateDeviceKeys,
	exportDeviceKeysForUpload,
	deriveMessageKeys,
	isE2EESupported,
} from '$lib/crypto/signal-protocol';

export type {
	RemotePreKeyBundle,
	DeviceRegistration as SignalDeviceRegistration,
	SessionState,
} from '$lib/crypto/signal-protocol';

// ============================================================================
// Session Manager (Double Ratchet)
// ============================================================================

export { 
	e2eeSessionManager, 
	E2EESessionManager,
	type E2EESessionInfo,
	type E2EEEncryptedMessage,
	type PreKeyMessage,
} from './session-manager';

// Re-export Double Ratchet types for convenience
export type { 
	RatchetState,
	RatchetMessage,
} from '$lib/crypto/ratchet';

export { RatchetSession, RatchetSessionManager } from '$lib/crypto/ratchet';

// ============================================================================
// E2EE Store (lazy loaded to avoid circular dependencies)
// ============================================================================

// Lazy load to avoid circular dependencies
let e2eeStore: typeof import('$lib/stores/e2ee').e2ee | null = null;
let e2eeReadyStore: typeof import('$lib/stores/e2ee').e2eeReady | null = null;

/**
 * Get the E2EE store
 */
export async function getE2EEStore() {
	if (!e2eeStore) {
		const module = await import('$lib/stores/e2ee');
		e2eeStore = module.e2ee;
		e2eeReadyStore = module.e2eeReady;
	}
	return { e2ee: e2eeStore, e2eeReady: e2eeReadyStore };
}

/**
 * Check if E2EE store is loaded
 */
export function isE2EEInitialized(): boolean {
	if (!browser) return false;
	return e2eeStore !== null;
}
