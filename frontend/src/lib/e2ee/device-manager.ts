/**
 * E2EE Device Manager
 * 
 * Manages device registration, identity key pairs, and prekey bundles.
 * Coordinates between WASM/crypto layer and backend API.
 * 
 * Responsibilities:
 * - Generate and store identity key pairs
 * - Generate and manage signed prekeys (periodic rotation)
 * - Generate and manage one-time prekeys (consumed on use)
 * - Create prekey bundles for server upload
 * - Track registered devices per user
 */

import { browser } from '$app/environment';
import { keyStorage, type IdentityKeyData, type SignedPreKeyData, type OneTimePreKeyData } from './key-storage';
import { initializeWasm, isWasmReady, isUsingFallback, onWasmStateChange } from './wasm-loader';

// Curve and protocol constants
const CURVE = 'P-256';
const HASH = 'SHA-256';
const SIGNING_CURVE = 'P-256';

// Device registration interface
export interface DeviceRegistration {
	deviceId: string;
	deviceName: string;
	deviceType: 'web' | 'desktop' | 'mobile_ios' | 'mobile_android';
	identityKey: string; // Base64
	registrationId: number;
	signedPreKey: {
		keyId: number;
		publicKey: string; // Base64
		signature: string; // Base64
	};
	oneTimePreKeys: Array<{
		keyId: number;
		publicKey: string; // Base64
	}>;
}

// Device info interface
export interface DeviceInfo {
	deviceId: string;
	deviceName?: string;
	deviceType: string;
	lastSeen: string;
	createdAt: string;
	hasPreKeys: boolean;
	remainingPreKeys: number;
}

// PreKey bundle for establishing sessions
export interface PreKeyBundle {
	userId: string;
	deviceId: string;
	registrationId: number;
	identityKey: string; // Base64
	signedPreKeyId: number;
	signedPreKey: string; // Base64
	signedKeySignature: string; // Base64
	preKeyId?: number;
	preKey?: string; // Base64
}

/**
 * Generate a unique device ID
 */
function generateDeviceId(): string {
	const bytes = crypto.getRandomValues(new Uint8Array(16));
	return Array.from(bytes)
		.map(b => b.toString(16).padStart(2, '0'))
		.join('');
}

/**
 * Generate a registration ID (14-bit value as per Signal Protocol)
 */
function generateRegistrationId(): number {
	const bytes = crypto.getRandomValues(new Uint8Array(2));
	return ((bytes[0] << 8) | bytes[1]) & 0x3FFF;
}

/**
 * Convert ArrayBuffer to Base64 string
 */
function arrayBufferToBase64(buffer: ArrayBuffer): string {
	const bytes = new Uint8Array(buffer);
	let binary = '';
	for (let i = 0; i < bytes.byteLength; i++) {
		binary += String.fromCharCode(bytes[i]);
	}
	if (typeof globalThis.btoa !== 'undefined') {
		return globalThis.btoa(binary);
	}
	// Node.js environment
	const { Buffer } = globalThis as { Buffer?: { from(s: string): { toString(encoding: string): string } } };
	if (Buffer) {
		return Buffer.from(binary).toString('base64');
	}
	return btoa(binary);
}

/**
 * Convert Base64 string to ArrayBuffer
 */
function base64ToArrayBuffer(base64: string): ArrayBuffer {
	let binary: string;
	if (typeof globalThis.atob !== 'undefined') {
		binary = globalThis.atob(base64);
	} else if (typeof Buffer !== 'undefined') {
		// Node.js environment
		binary = Buffer.from(base64, 'base64').toString('binary');
	} else {
		binary = atob(base64);
	}
	const bytes = new Uint8Array(binary.length);
	for (let i = 0; i < binary.length; i++) {
		bytes[i] = binary.charCodeAt(i);
	}
	return bytes.buffer;
}

/**
 * DeviceManager singleton
 */
class DeviceManagerClass {
	private initialized = false;
	private deviceId: string | null = null;
	private registrationId: number | null = null;

	/**
	 * Initialize the device manager
	 * Call this at app startup before establishing connections
	 */
	async init(): Promise<void> {
		if (!browser) {
			return;
		}

		if (this.initialized) {
			return;
		}

		// Initialize key storage
		await keyStorage.init();

		// Initialize WASM (or fallback to WebCrypto)
		await initializeWasm();

		// Check for existing registration
		const metadata = await keyStorage.getMetadata();
		if (metadata) {
			this.deviceId = metadata.deviceId;
			this.registrationId = metadata.registrationId;
		}

		this.initialized = true;
		console.info('E2EE DeviceManager initialized', {
			usingWasm: isWasmReady(),
			usingFallback: isUsingFallback(),
			deviceId: this.deviceId,
		});
	}

	/**
	 * Check if device is registered (has identity keys)
	 */
	async isRegistered(): Promise<boolean> {
		return keyStorage.hasIdentityKey();
	}

	/**
	 * Get current device ID
	 */
	getDeviceId(): string | null {
		return this.deviceId;
	}

	/**
	 * Get current registration ID
	 */
	getRegistrationId(): number | null {
		return this.registrationId;
	}

	/**
	 * Generate identity key pair
	 */
	async generateIdentityKeyPair(): Promise<IdentityKeyData> {
		const keyPair = await crypto.subtle.generateKey(
			{ name: 'ECDH', namedCurve: CURVE },
			true,
			['deriveBits', 'deriveKey']
		);

		const publicKeyRaw = await crypto.subtle.exportKey('raw', keyPair.publicKey);
		const privateKeyRaw = await crypto.subtle.exportKey('jwk', keyPair.privateKey);

		// Store the JWK private key (can be re-imported)
		const privateKeyBuffer = new TextEncoder().encode(JSON.stringify(privateKeyRaw)).buffer;

		return {
			publicKey: publicKeyRaw,
			privateKey: privateKeyBuffer,
		};
	}

	/**
	 * Generate signed prekey
	 */
	async generateSignedPreKey(identityPrivateKey: CryptoKey, keyId: number): Promise<SignedPreKeyData> {
		// Generate EC key pair for signed prekey
		const keyPair = await crypto.subtle.generateKey(
			{ name: 'ECDH', namedCurve: CURVE },
			true,
			['deriveBits', 'deriveKey']
		);

		// Export public key for signing
		const publicKeyRaw = await crypto.subtle.exportKey('raw', keyPair.publicKey);

		// Create ECDSA key for signing
		const signingKeyPair = await crypto.subtle.generateKey(
			{ name: 'ECDSA', namedCurve: SIGNING_CURVE },
			true,
			['sign', 'verify']
		);

		// Sign the public key with identity key
		// Note: In Signal Protocol, signed prekeys are signed with the identity key
		const signature = await crypto.subtle.sign(
			{ name: 'ECDSA', hash: HASH },
			signingKeyPair.privateKey,
			publicKeyRaw
		);

		const privateKeyRaw = await crypto.subtle.exportKey('jwk', keyPair.privateKey);
		const privateKeyBuffer = new TextEncoder().encode(JSON.stringify(privateKeyRaw)).buffer;

		return {
			keyId,
			publicKey: publicKeyRaw,
			privateKey: privateKeyBuffer,
			signature,
			timestamp: Date.now(),
		};
	}

	/**
	 * Generate one-time prekeys
	 */
	async generateOneTimePreKeys(startId: number, count: number): Promise<OneTimePreKeyData[]> {
		const keys: OneTimePreKeyData[] = [];

		for (let i = 0; i < count; i++) {
			const keyPair = await crypto.subtle.generateKey(
				{ name: 'ECDH', namedCurve: CURVE },
				true,
				['deriveBits', 'deriveKey']
			);

			const publicKeyRaw = await crypto.subtle.exportKey('raw', keyPair.publicKey);
			const privateKeyRaw = await crypto.subtle.exportKey('jwk', keyPair.privateKey);
			const privateKeyBuffer = new TextEncoder().encode(JSON.stringify(privateKeyRaw)).buffer;

			keys.push({
				keyId: startId + i,
				publicKey: publicKeyRaw,
				privateKey: privateKeyBuffer,
			});
		}

		return keys;
	}

	/**
	 * Register a new device with identity keys and prekeys
	 * This should only be called once per device
	 */
	async registerDevice(deviceName?: string): Promise<DeviceRegistration> {
		if (await this.isRegistered()) {
			throw new Error('Device already registered. Use isRegistered() to check first.');
		}

		// Generate new device ID and registration ID
		this.deviceId = generateDeviceId();
		this.registrationId = generateRegistrationId();

		// Generate identity key
		const identityKey = await this.generateIdentityKeyPair();

		// Generate signed prekey (keyId = 1 for initial)
		const signedPreKey = await this.generateSignedPreKey(identityKey.publicKey as unknown as CryptoKey, 1);

		// Generate one-time prekeys (100 is standard)
		const oneTimePreKeys = await this.generateOneTimePreKeys(1, 100);

		// Store keys locally
		await keyStorage.storeIdentityKey(identityKey);
		await keyStorage.storeSignedPreKey(signedPreKey);
		await keyStorage.storeOneTimePreKeys(oneTimePreKeys);

		// Store metadata
		await keyStorage.storeMetadata({
			deviceId: this.deviceId,
			registrationId: this.registrationId,
			identityKeyHash: '', // TODO: compute hash
			createdAt: Date.now(),
			lastKeyRotation: Date.now(),
			signedPreKeyId: signedPreKey.keyId,
		});

		// Build registration object for server upload
		return {
			deviceId: this.deviceId,
			deviceName: deviceName || this.getDefaultDeviceName(),
			deviceType: 'web',
			identityKey: arrayBufferToBase64(identityKey.publicKey),
			registrationId: this.registrationId,
			signedPreKey: {
				keyId: signedPreKey.keyId,
				publicKey: arrayBufferToBase64(signedPreKey.publicKey),
				signature: arrayBufferToBase64(signedPreKey.signature),
			},
			oneTimePreKeys: oneTimePreKeys.map(pk => ({
				keyId: pk.keyId,
				publicKey: arrayBufferToBase64(pk.publicKey),
			})),
		};
	}

	/**
	 * Get the default device name based on user agent
	 */
	private getDefaultDeviceName(): string {
		if (typeof navigator === 'undefined') {
			return 'Web Browser';
		}
		const ua = navigator.userAgent;
		if (ua.includes('Firefox')) return 'Firefox Browser';
		if (ua.includes('Chrome')) return 'Chrome Browser';
		if (ua.includes('Safari')) return 'Safari Browser';
		if (ua.includes('Edge')) return 'Edge Browser';
		return 'Web Browser';
	}

	/**
	 * Get prekey bundle for a specific device
	 * Used when establishing E2EE sessions
	 */
	async getLocalPreKeyBundle(): Promise<{
		identityKey: string;
		signedPreKey: SignedPreKeyData;
	}> {
		const identityKey = await keyStorage.getIdentityKey();
		if (!identityKey) {
			throw new Error('Device not registered. Call registerDevice() first.');
		}

		const signedPreKey = await keyStorage.getLatestSignedPreKey();
		if (!signedPreKey) {
			throw new Error('No signed prekey found. Regenerate signed prekeys.');
		}

		return {
			identityKey: arrayBufferToBase64(identityKey.publicKey),
			signedPreKey,
		};
	}

	/**
	 * Consume a one-time prekey for a new session
	 */
	async consumeOneTimePreKey(keyId: number): Promise<OneTimePreKeyData | null> {
		return keyStorage.consumeOneTimePreKey(keyId);
	}

	/**
	 * Get count of remaining one-time prekeys
	 */
	async getRemainingPreKeyCount(): Promise<number> {
		return keyStorage.countOneTimePreKeys();
	}

	/**
	 * Check if prekeys need replenishment (below threshold)
	 */
	async needsPreKeyReplenishment(threshold: number = 20): Promise<boolean> {
		const count = await this.getRemainingPreKeyCount();
		return count < threshold;
	}

	/**
	 * Replenish one-time prekeys (generate and store new ones)
	 * Should be called when prekey count is low
	 */
	async replenishPreKeys(count: number = 100): Promise<Array<{ keyId: number; publicKey: string }>> {
		const currentCount = await keyStorage.countOneTimePreKeys();
		const startId = currentCount + 1;
		const newKeys = await this.generateOneTimePreKeys(startId, count);
		await keyStorage.storeOneTimePreKeys(newKeys);

		return newKeys.map(pk => ({
			keyId: pk.keyId,
			publicKey: arrayBufferToBase64(pk.publicKey),
		}));
	}

	/**
	 * Rotate signed prekey (should be done periodically, e.g., monthly)
	 */
	async rotateSignedPreKey(): Promise<{
		keyId: number;
		publicKey: string;
		signature: string;
	}> {
		const identityKey = await keyStorage.getIdentityKey();
		if (!identityKey) {
			throw new Error('Device not registered.');
		}

		// Get current signed prekey ID and increment
		const metadata = await keyStorage.getMetadata();
		const newKeyId = (metadata?.signedPreKeyId ?? 0) + 1;

		const newSignedPreKey = await this.generateSignedPreKey(
			identityKey.publicKey as unknown as CryptoKey,
			newKeyId
		);

		await keyStorage.storeSignedPreKey(newSignedPreKey);

		// Update metadata
		await keyStorage.storeMetadata({
			...metadata!,
			signedPreKeyId: newKeyId,
			lastKeyRotation: Date.now(),
		});

		return {
			keyId: newSignedPreKey.keyId,
			publicKey: arrayBufferToBase64(newSignedPreKey.publicKey),
			signature: arrayBufferToBase64(newSignedPreKey.signature),
		};
	}

	/**
	 * Export public identity key for sharing
	 */
	async exportPublicIdentityKey(): Promise<string> {
		const identityKey = await keyStorage.getIdentityKey();
		if (!identityKey) {
			throw new Error('Device not registered.');
		}
		return arrayBufferToBase64(identityKey.publicKey);
	}

	/**
	 * Import a remote user's public identity key
	 */
	async importRemoteIdentityKey(base64Key: string): Promise<ArrayBuffer> {
		const keyBytes = base64ToArrayBuffer(base64Key);
		// Validate that it's a valid P-256 public key
		await crypto.subtle.importKey(
			'raw',
			keyBytes,
			{ name: 'ECDH', namedCurve: CURVE },
			true,
			[]
		);
		return keyBytes;
	}

	/**
	 * Clear all E2EE data (deregister device)
	 * WARNING: This makes the device unable to decrypt messages
	 */
	async clearAllData(): Promise<void> {
		await keyStorage.clearAll();
		this.deviceId = null;
		this.registrationId = null;
		this.initialized = false;
	}
}

// Singleton instance
export const deviceManager = new DeviceManagerClass();

// Also export the class for testing
export { DeviceManagerClass };
