/**
 * E2EE Initialization Module
 * 
 * Coordinates E2EE initialization with app startup.
 * This should be imported early in the app lifecycle to ensure
 * E2EE is ready before establishing connections.
 * 
 * Usage:
 *   import { initE2EE } from '$lib/e2ee';
 *   await initE2EE();
 */

import { browser } from '$app/environment';
import { deviceManager } from './device-manager';
import { initializeWasm, isWasmReady, isUsingFallback, getInitError } from './wasm-loader';
import { keyStorage } from './key-storage';

export interface E2EEInitResult {
	success: boolean;
	usingWasm: boolean;
	usingFallback: boolean;
	deviceRegistered: boolean;
	deviceId: string | null;
	error?: Error;
}

/**
 * Initialize E2EE subsystem
 * 
 * This should be called once at app startup, ideally before
 * establishing the WebSocket connection to the gateway.
 * 
 * Initialization steps:
 * 1. Initialize key storage (IndexedDB)
 * 2. Initialize WASM module (or fallback to WebCrypto)
 * 3. Initialize device manager
 * 
 * @param options Options for initialization
 * @param options.autoRegister Whether to automatically register device if not registered
 * @param options.deviceName Custom device name
 */
export async function initE2EE(options: {
	autoRegister?: boolean;
	deviceName?: string;
} = {}): Promise<E2EEInitResult> {
	if (!browser) {
		return {
			success: false,
			usingWasm: false,
			usingFallback: false,
			deviceRegistered: false,
			deviceId: null,
			error: new Error('E2EE initialization skipped: not in browser'),
		};
	}

	try {
		console.info('[E2EE] Starting initialization...');

		// Step 1: Initialize key storage
		console.info('[E2EE] Initializing key storage...');
		await keyStorage.init();
		console.info('[E2EE] Key storage ready');

		// Step 2: Initialize WASM (or fallback)
		console.info('[E2EE] Initializing WASM module...');
		await initializeWasm();
		console.info('[E2EE] WASM state:', {
			usingWasm: isWasmReady(),
			usingFallback: isUsingFallback(),
		});

		// Step 3: Initialize device manager
		console.info('[E2EE] Initializing device manager...');
		await deviceManager.init();
		console.info('[E2EE] Device manager ready');

		// Check if device is registered
		const isRegistered = await deviceManager.isRegistered();
		const deviceId = deviceManager.getDeviceId();

		if (!isRegistered && options.autoRegister) {
			console.info('[E2EE] Device not registered, registering...');
			const registration = await deviceManager.registerDevice(options.deviceName);
			console.info('[E2EE] Device registered:', {
				deviceId: registration.deviceId,
				preKeyCount: registration.oneTimePreKeys.length,
			});
			return {
				success: true,
				usingWasm: isWasmReady(),
				usingFallback: isUsingFallback(),
				deviceRegistered: true,
				deviceId: registration.deviceId,
			};
		}

		return {
			success: true,
			usingWasm: isWasmReady(),
			usingFallback: isUsingFallback(),
			deviceRegistered: isRegistered,
			deviceId,
		};
	} catch (error) {
		const err = error instanceof Error ? error : new Error(String(error));
		console.error('[E2EE] Initialization failed:', err);

		return {
			success: false,
			usingWasm: isWasmReady(),
			usingFallback: isUsingFallback(),
			deviceRegistered: false,
			deviceId: null,
			error: err,
		};
	}
}

/**
 * Get E2EE status without initializing
 */
export function getE2EEStatus(): {
	initialized: boolean;
	usingWasm: boolean;
	usingFallback: boolean;
	error: Error | null;
} {
	return {
		initialized: deviceManager.getDeviceId() !== null || false,
		usingWasm: isWasmReady(),
		usingFallback: isUsingFallback(),
		error: getInitError(),
	};
}

/**
 * Check if E2EE is ready to use
 */
export function isE2EEReady(): boolean {
	return deviceManager.getDeviceId() !== null;
}
