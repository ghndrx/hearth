/**
 * libsignal-client WASM Loader
 * 
 * Initializes and manages the libsignal-client WASM module.
 * Provides graceful fallback to WebCrypto if WASM is unavailable.
 * 
 * The WASM module provides:
 * - Curve25519 key operations (vs P-256 in WebCrypto)
 * - Proper X3DH implementation
 * - Double Ratchet session management
 * - Session record serialization
 * 
 * Note: Full WASM integration is a future enhancement. Phase 1 uses WebCrypto.
 */

import { browser } from '$app/environment';

// WASM state enum
export enum WasmState {
	UNINITIALIZED = 'uninitialized',
	INITIALIZING = 'initializing',
	READY = 'ready',
	FALLBACK = 'fallback',
	ERROR = 'error',
}

// Key types for E2EE
export interface IdentityKeyPair {
	publicKey: ArrayBuffer;
	privateKey: ArrayBuffer;
}

export interface SignedPreKey {
	keyId: number;
	publicKey: ArrayBuffer;
	privateKey: ArrayBuffer;
	signature: ArrayBuffer;
	timestamp: number;
}

export interface OneTimePreKey {
	keyId: number;
	publicKey: ArrayBuffer;
	privateKey: ArrayBuffer;
}

export interface X3DHResult {
	sharedSecret: ArrayBuffer;
	ephemeralPublicKey: ArrayBuffer;
	usedPreKeyId?: number;
}

export interface SessionHandle {
	// Session state that can be serialized
	serialize(): ArrayBuffer;
}

// WasmModule type - represents the loaded WASM module
export interface WasmModule {
	// Placeholder for when we integrate libsignal-client fully
	readonly isLoaded: boolean;
}

// Internal state
let wasmState: WasmState = WasmState.UNINITIALIZED;
let wasmModule: WasmModule | null = null;
let initError: Error | null = null;
let wasmStateChangeHandlers: Array<(state: WasmState) => void> = [];

/**
 * Set WASM state and notify listeners
 */
function setWasmState(newState: WasmState): void {
	if (wasmState !== newState) {
		wasmState = newState;
		console.debug('[WASM] State changed to:', newState);
		wasmStateChangeHandlers.forEach((handler) => {
			try {
				handler(newState);
			} catch (err) {
				console.error('[WASM] State change handler error:', err);
			}
		});
	}
}

/**
 * Initialize the libsignal-client WASM module
 * 
 * This is called automatically by initE2EE(). Manual calling is usually not needed.
 * 
 * Phase 1: This always falls back to WebCrypto since full WASM integration
 * is deferred to a future enhancement.
 * 
 * @returns Promise that resolves when WASM is ready (or fallback is active)
 */
export async function initializeWasm(): Promise<void> {
	if (!browser) {
		console.debug('[WASM] Skipping initialization (not in browser)');
		setWasmState(WasmState.FALLBACK);
		return;
	}

	if (wasmState === WasmState.READY) {
		console.debug('[WASM] Already initialized');
		return;
	}

	if (wasmState === WasmState.INITIALIZING) {
		// Wait for existing initialization to complete
		await waitForWasmReady();
		return;
	}

	setWasmState(WasmState.INITIALIZING);
	initError = null;

	try {
		console.info('[WASM] Attempting to load libsignal-client...');

		// Phase 1: We don't actually load WASM yet - use WebCrypto fallback
		// Full WASM integration is deferred until needed for advanced features
		
		// In the future, this would look like:
		// const libsignal = await import('@signalapp/libsignal-client');
		// await libsignal.initializeAll();
		// wasmModule = { isLoaded: true, ...libsignal };
		
		// For now, mark as fallback
		wasmModule = { isLoaded: false };
		
		console.info('[WASM] Using WebCrypto fallback (libsignal-client WASM deferred)');
		setWasmState(WasmState.FALLBACK);
	} catch (error) {
		const err = error instanceof Error ? error : new Error(String(error));
		console.warn('[WASM] Failed to initialize:', err.message);
		console.info('[WASM] Falling back to WebCrypto implementation');

		initError = err;
		wasmModule = null;
		setWasmState(WasmState.FALLBACK);
	}
}

/**
 * Wait for WASM to be ready (or fallback to be active)
 */
function waitForWasmReady(): Promise<void> {
	return new Promise((resolve) => {
		if (wasmState === WasmState.READY || wasmState === WasmState.FALLBACK || wasmState === WasmState.ERROR) {
			resolve();
			return;
		}

		// Poll until ready or fallback
		const checkInterval = setInterval(() => {
			if (wasmState === WasmState.READY || wasmState === WasmState.FALLBACK || wasmState === WasmState.ERROR) {
				clearInterval(checkInterval);
				resolve();
			}
		}, 50);
	});
}

/**
 * Check if WASM module is ready (fully loaded and initialized)
 */
export function isWasmReady(): boolean {
	return wasmState === WasmState.READY;
}

/**
 * Check if we're using WebCrypto fallback instead of WASM
 */
export function isUsingFallback(): boolean {
	return wasmState === WasmState.FALLBACK || wasmState === WasmState.ERROR || wasmState === WasmState.UNINITIALIZED;
}

/**
 * Check if WASM is available (but not necessarily initialized)
 * Returns true if WebAssembly is supported in the environment
 */
export function isWasmAvailable(): boolean {
	if (!browser) return false;

	// Check if WebAssembly is supported
	try {
		if (typeof WebAssembly !== 'object') {
			return false;
		}
		// Check for WASM support in the environment
		// Node.js and modern browsers support it
		return true;
	} catch {
		return false;
	}
}

/**
 * Get the WASM module if loaded
 */
export function getWasmModule(): WasmModule | null {
	return wasmModule;
}

/**
 * Get the current WASM state
 */
export function getWasmState(): WasmState {
	return wasmState;
}

/**
 * Get the initialization error if any
 */
export function getInitError(): Error | null {
	return initError;
}

/**
 * Subscribe to WASM state changes
 * 
 * @param handler - Callback function called when state changes
 * @returns Unsubscribe function
 */
export function onWasmStateChange(handler: (state: WasmState) => void): () => void {
	wasmStateChangeHandlers.push(handler);

	// Immediately call with current state
	handler(wasmState);

	// Return unsubscribe function
	return () => {
		const index = wasmStateChangeHandlers.indexOf(handler);
		if (index > -1) {
			wasmStateChangeHandlers.splice(index, 1);
		}
	};
}

/**
 * Destroy the WASM module and free resources
 * 
 * Call this when E2EE is no longer needed to prevent memory leaks.
 */
export async function destroyWasm(): Promise<void> {
	// In Phase 1, there's nothing to destroy since we use WebCrypto
	// In the future, this would clean up WASM resources
	
	wasmModule = null;
	setWasmState(WasmState.UNINITIALIZED);
}
