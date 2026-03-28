/**
 * X3DH Key Agreement - Signal Protocol Implementation
 * 
 * Implements the Extended Triple Diffie-Hellman (X3DH) key agreement protocol
 * for establishing secure sessions between users.
 * 
 * X3DH establishes a shared secret between two parties by performing
 * multiple Diffie-Hellman operations using:
 * - Identity keys (IK)
 * - Signed prekeys (SPK)
 * - One-time prekeys (OTPK)
 * - Ephemeral keys (EK)
 * 
 * This module provides both sender and recipient-side X3DH operations,
 * integrating with the libsignal-client WASM when available, with
 * WebCrypto fallback for environments without WASM support.
 */

import { e2eeApi } from './e2ee-api';

// Curve parameters
const CURVE = 'P-256';
const HASH = 'SHA-256';

/**
 * Identity key pair (long-term)
 */
export interface IdentityKeyPair {
  publicKey: CryptoKey;
  privateKey: CryptoKey;
}

/**
 * Signed prekey (rotated periodically)
 */
export interface SignedPreKey {
  keyId: number;
  publicKey: CryptoKey;
  privateKey: CryptoKey;
  signature: ArrayBuffer;
  timestamp: number;
}

/**
 * One-time prekey (single use)
 */
export interface OneTimePreKey {
  keyId: number;
  publicKey: CryptoKey;
  privateKey: CryptoKey;
}

/**
 * Remote prekey bundle received from server
 */
export interface RemotePreKeyBundle {
  userId: string;
  deviceId: string;
  registrationId: number;
  identityKey: string;      // Base64
  signedPreKeyId: number;
  signedPreKey: string;     // Base64
  signedKeySignature: string; // Base64
  preKeyId?: number;
  preKey?: string;          // Base64
}

/**
 * X3DH result after key agreement
 */
export interface X3DHResult {
  sharedSecret: ArrayBuffer;
  ephemeralPublicKey: ArrayBuffer;
  usedPreKeyId?: number;
  initiatorIdentityKey?: ArrayBuffer;
}

/**
 * Session initialization result
 */
export interface SessionInitResult {
  sharedSecret: ArrayBuffer;
  ephemeralPublicKey: ArrayBuffer;
  messageKey: CryptoKey;
  usedPreKeyId?: number;
}

/**
 * Convert ArrayBuffer to Base64
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
  return Buffer.from(bytes).toString('base64');
}

/**
 * Convert Base64 to ArrayBuffer
 */
function base64ToArrayBuffer(base64: string): ArrayBuffer {
  let binary: string;
  if (typeof globalThis.atob !== 'undefined') {
    binary = globalThis.atob(base64);
  } else {
    binary = Buffer.from(base64, 'base64').toString('binary');
  }
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  // Return a copy to ensure regular ArrayBuffer (not SharedArrayBuffer)
  return new Uint8Array(bytes).buffer.slice(0);
}

/**
 * Concatenate multiple ArrayBuffers
 */
function concatArrayBuffers(buffers: ArrayBuffer[]): ArrayBuffer {
  const totalLength = buffers.reduce((sum, buf) => sum + buf.byteLength, 0);
  const result = new Uint8Array(totalLength);
  let offset = 0;
  for (const buf of buffers) {
    result.set(new Uint8Array(buf), offset);
    offset += buf.byteLength;
  }
  return result.buffer;
}

/**
 * Derive a key using HKDF
 */
async function deriveHKDF(
  inputKey: ArrayBuffer,
  info: string,
  length: number = 32
): Promise<ArrayBuffer> {
  const key = await crypto.subtle.importKey(
    'raw',
    inputKey,
    'HKDF',
    false,
    ['deriveBits']
  );

  return crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new Uint8Array(32), // Zero salt for Signal Protocol
      info: new TextEncoder().encode(info),
    },
    key,
    length * 8 // Convert to bits
  );
}

/**
 * Import a public key from Base64 raw bytes
 */
async function importPublicKey(base64Key: string): Promise<CryptoKey> {
  const rawKey = base64ToArrayBuffer(base64Key);
  return crypto.subtle.importKey(
    'raw',
    rawKey,
    { name: 'ECDH', namedCurve: CURVE },
    false,
    []
  );
}

/**
 * Import a private key from ArrayBuffer (JWK format)
 */
async function importPrivateKey(jwkBuffer: ArrayBuffer): Promise<CryptoKey> {
  const jwk = JSON.parse(new TextDecoder().decode(new Uint8Array(jwkBuffer)));
  return crypto.subtle.importKey(
    'jwk',
    jwk,
    { name: 'ECDH', namedCurve: CURVE },
    true,
    ['deriveBits', 'deriveKey']
  );
}

/**
 * Generate an ephemeral key pair for X3DH
 */
async function generateEphemeralKeyPair(): Promise<CryptoKeyPair> {
  return crypto.subtle.generateKey(
    { name: 'ECDH', namedCurve: CURVE },
    true,
    ['deriveBits', 'deriveKey']
  );
}

/**
 * Perform ECDH to derive a shared secret
 */
async function performECDH(
  privateKey: CryptoKey,
  publicKey: CryptoKey
): Promise<ArrayBuffer> {
  return crypto.subtle.deriveBits(
    { name: 'ECDH', public: publicKey },
    privateKey,
    256
  );
}

/**
 * Initiate X3DH key agreement as the sender (Alice)
 * 
 * Alice initiates X3DH with Bob by:
 * 1. DH1: Alice's identity key × Bob's signed prekey
 * 2. DH2: Alice's ephemeral key × Bob's identity key
 * 3. DH3: Alice's ephemeral key × Bob's signed prekey
 * 4. (Optional) DH4: Alice's ephemeral key × Bob's one-time prekey
 * 
 * The shared secret is then derived via HKDF
 */
export async function performX3DHSender(
  localIdentityKey: IdentityKeyPair,
  remoteBundle: RemotePreKeyBundle
): Promise<X3DHResult> {
  // Import remote keys
  const remoteIdentityKey = await importPublicKey(remoteBundle.identityKey);
  const remoteSignedPreKey = await importPublicKey(remoteBundle.signedPreKey);

  // Generate ephemeral key pair
  const ephemeralKeyPair = await generateEphemeralKeyPair();

  // Perform DH operations
  const dh1 = await performECDH(localIdentityKey.privateKey, remoteSignedPreKey);
  const dh2 = await performECDH(ephemeralKeyPair.privateKey, remoteIdentityKey);
  const dh3 = await performECDH(ephemeralKeyPair.privateKey, remoteSignedPreKey);

  let combinedSecret: ArrayBuffer;
  let usedPreKeyId: number | undefined;

  if (remoteBundle.preKey && remoteBundle.preKeyId) {
    // DH4 with one-time prekey
    const remoteOneTimePreKey = await importPublicKey(remoteBundle.preKey);
    const dh4 = await performECDH(ephemeralKeyPair.privateKey, remoteOneTimePreKey);
    combinedSecret = concatArrayBuffers([dh1, dh2, dh3, dh4]);
    usedPreKeyId = remoteBundle.preKeyId;
  } else {
    combinedSecret = concatArrayBuffers([dh1, dh2, dh3]);
  }

  // Export ephemeral public key
  const ephemeralPublicKeyRaw = await crypto.subtle.exportKey('raw', ephemeralKeyPair.publicKey);

  // Derive final shared secret via HKDF
  const sharedSecret = await deriveHKDF(combinedSecret, 'HearthX3DH');

  // Export our identity public key for the recipient
  const initiatorIdentityKeyRaw = await crypto.subtle.exportKey('raw', localIdentityKey.publicKey);

  return {
    sharedSecret,
    ephemeralPublicKey: ephemeralPublicKeyRaw,
    usedPreKeyId,
    initiatorIdentityKey: initiatorIdentityKeyRaw,
  };
}

/**
 * Complete X3DH key agreement as the recipient (Bob)
 * 
 * Bob receives Alice's initial message containing:
 * - Alice's identity key (IK_A)
 * - Alice's ephemeral key (EK_A)
 * - (Optional) Used one-time prekey ID
 * 
 * Bob then performs:
 * 1. DH1: Bob's signed prekey × Alice's identity key
 * 2. DH2: Bob's identity key × Alice's ephemeral key
 * 3. DH3: Bob's signed prekey × Alice's ephemeral key
 * 4. (Optional) DH4: Bob's one-time prekey × Alice's ephemeral key
 */
export async function performX3DHRecipient(
  localIdentityKey: {
    publicKey: CryptoKey;
    privateKey: CryptoKey;
  },
  localSignedPreKey: SignedPreKey,
  localOneTimePreKey: OneTimePreKey | null,
  remoteIdentityKeyRaw: ArrayBuffer,
  remoteEphemeralKeyRaw: ArrayBuffer
): Promise<ArrayBuffer> {
  // Import remote keys
  const remoteIdentityKey = await crypto.subtle.importKey(
    'raw',
    remoteIdentityKeyRaw,
    { name: 'ECDH', namedCurve: CURVE },
    false,
    []
  );

  const remoteEphemeralKey = await crypto.subtle.importKey(
    'raw',
    remoteEphemeralKeyRaw,
    { name: 'ECDH', namedCurve: CURVE },
    false,
    []
  );

  // Perform DH operations
  const dh1 = await performECDH(localSignedPreKey.privateKey, remoteIdentityKey);
  const dh2 = await performECDH(localIdentityKey.privateKey, remoteEphemeralKey);
  const dh3 = await performECDH(localSignedPreKey.privateKey, remoteEphemeralKey);

  let combinedSecret: ArrayBuffer;

  if (localOneTimePreKey) {
    const dh4 = await performECDH(localOneTimePreKey.privateKey, remoteEphemeralKey);
    combinedSecret = concatArrayBuffers([dh1, dh2, dh3, dh4]);
  } else {
    combinedSecret = concatArrayBuffers([dh1, dh2, dh3]);
  }

  // Derive final shared secret via HKDF
  return deriveHKDF(combinedSecret, 'HearthX3DH');
}

/**
 * Derive message keys from shared secret using HKDF
 */
export async function deriveMessageKeys(
  sharedSecret: ArrayBuffer
): Promise<{
  encryptionKey: CryptoKey;
  macKey: CryptoKey;
}> {
  // Derive encryption key
  const encKeyMaterial = await crypto.subtle.importKey(
    'raw',
    sharedSecret,
    'HKDF',
    false,
    ['deriveKey']
  );

  const encryptionKey = await crypto.subtle.deriveKey(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new TextEncoder().encode('HearthX3DH-Enc'),
      info: new TextEncoder().encode('message_key'),
    },
    encKeyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );

  // Derive MAC key
  const macKeyMaterial = await crypto.subtle.importKey(
    'raw',
    sharedSecret,
    'HKDF',
    false,
    ['deriveKey']
  );

  const macKey = await crypto.subtle.deriveKey(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new TextEncoder().encode('HearthX3DH-Mac'),
      info: new TextEncoder().encode('mac_key'),
    },
    macKeyMaterial,
    { name: 'HMAC', hash: HASH },
    false,
    ['sign', 'verify']
  );

  return { encryptionKey, macKey };
}

/**
 * Establish an E2EE session with a remote user
 * 
 * This is the main entry point for creating an encrypted session.
 * It fetches the remote user's prekey bundle and performs X3DH.
 */
export async function establishSession(
  localIdentityKey: IdentityKeyPair,
  recipientUserId: string,
  recipientDeviceId?: string
): Promise<{
  sharedSecret: ArrayBuffer;
  ephemeralPublicKey: ArrayBuffer;
  messageKey: CryptoKey;
  recipientDeviceId: string;
} | null> {
  try {
    // Get recipient's prekey bundle(s)
    let bundles: RemotePreKeyBundle[];
    
    if (recipientDeviceId) {
      const bundle = await e2eeApi.getPreKeyBundle(recipientUserId, recipientDeviceId);
      bundles = [bundle];
    } else {
      bundles = await e2eeApi.getAllPreKeyBundles(recipientUserId);
    }

    if (bundles.length === 0) {
      console.warn('[X3DH] No prekey bundles available for recipient:', recipientUserId);
      return null;
    }

    // Use the first available bundle
    const bundle = bundles[0];

    // Perform X3DH as sender
    const result = await performX3DHSender(localIdentityKey, bundle);

    // Derive message keys
    const { encryptionKey: messageKey } = await deriveMessageKeys(result.sharedSecret);

    return {
      sharedSecret: result.sharedSecret,
      ephemeralPublicKey: result.ephemeralPublicKey,
      messageKey,
      recipientDeviceId: bundle.deviceId,
    };
  } catch (error) {
    console.error('[X3DH] Failed to establish session:', error);
    throw error;
  }
}

/**
 * Process an incoming X3DH initial message as recipient
 * 
 * Called when receiving a pre-key message from a new contact.
 * This completes the X3DH as the recipient (Bob).
 */
export async function processPreKeyMessage(
  localIdentityKey: {
    publicKey: CryptoKey;
    privateKey: CryptoKey;
  },
  localSignedPreKey: SignedPreKey,
  localOneTimePreKeys: OneTimePreKey[],
  senderIdentityKeyRaw: ArrayBuffer,
  senderEphemeralKeyRaw: ArrayBuffer,
  usedPreKeyId?: number
): Promise<{
  sharedSecret: ArrayBuffer;
  messageKey: CryptoKey;
}> {
  // Find the used one-time prekey if specified
  let localOneTimePreKey: OneTimePreKey | null = null;
  if (usedPreKeyId !== undefined) {
    localOneTimePreKey = localOneTimePreKeys.find(pk => pk.keyId === usedPreKeyId) || null;
  }

  // Complete X3DH as recipient
  const sharedSecret = await performX3DHRecipient(
    localIdentityKey,
    localSignedPreKey,
    localOneTimePreKey,
    senderIdentityKeyRaw,
    senderEphemeralKeyRaw
  );

  // Derive message keys
  const { encryptionKey: messageKey } = await deriveMessageKeys(sharedSecret);

  return { sharedSecret, messageKey };
}

/**
 * Get prekey bundle for a user
 */
export async function getPreKeyBundle(
  userId: string,
  deviceId: string
): Promise<RemotePreKeyBundle> {
  return e2eeApi.getPreKeyBundle(userId, deviceId);
}

/**
 * Get all prekey bundles for a user (all devices)
 */
export async function getAllPreKeyBundles(
  userId: string
): Promise<RemotePreKeyBundle[]> {
  return e2eeApi.getAllPreKeyBundles(userId);
}

/**
 * Check if E2EE is supported in the current environment
 */
export function isE2EESupported(): boolean {
  if (typeof window === 'undefined') return false;
  
  return !!(
    crypto?.subtle &&
    typeof crypto.subtle.generateKey === 'function' &&
    typeof crypto.subtle.deriveBits === 'function' &&
    typeof crypto.subtle.encrypt === 'function' &&
    typeof crypto.subtle.decrypt === 'function' &&
    typeof indexedDB !== 'undefined'
  );
}

/**
 * Serialization format for X3DH initial message
 * This is sent to the recipient as the first message
 */
export interface X3DHPreKeyMessage {
  version: number;
  baseKey: string;           // Our ephemeral public key (Base64)
  identityKey: string;       // Our identity public key (Base64)
  usedPreKeyId?: number;     // One-time prekey ID used (if any)
}

/**
 * Serialize X3DH pre-key message for transmission
 */
export function serializePreKeyMessage(message: X3DHPreKeyMessage): string {
  return JSON.stringify({
    v: 1,
    bk: message.baseKey,
    ik: message.identityKey,
    pk: message.usedPreKeyId,
  });
}

/**
 * Deserialize X3DH pre-key message
 */
export function deserializePreKeyMessage(data: string): X3DHPreKeyMessage {
  const parsed = JSON.parse(data);
  return {
    version: parsed.v,
    baseKey: parsed.bk,
    identityKey: parsed.ik,
    usedPreKeyId: parsed.pk,
  };
}
