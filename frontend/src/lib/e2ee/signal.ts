/**
 * E2EE Signal Protocol Service
 * 
 * Phase 1 uses the WebCrypto-based Signal Protocol implementation from crypto/.
 * The libsignal-client WASM implementation is deferred to a future enhancement.
 * 
 * This module provides the high-level E2EE API by delegating to:
 * - crypto/signal-protocol.ts: X3DH key agreement
 * - crypto/encryption.ts: AES-GCM message encryption
 * - crypto/secure-storage.ts: Encrypted key storage
 * - stores/e2ee.ts: E2EE state management
 */

import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { e2ee, type E2EESession } from '$lib/stores/e2ee';
import { 
  encryptMessage as webcryptoEncrypt, 
  decryptMessage as webcryptoDecrypt,
  type EncryptedMessage 
} from '$lib/crypto/encryption';
import { isE2EESupported } from '$lib/crypto/signal-protocol';

// Re-export types for compatibility
export type { EncryptedMessage };

export interface EncryptedPayload {
  ciphertext: string;
  iv: string;
  type: number;
}

// Serialization format for encrypted payloads
const PAYLOAD_VERSION = 1;

/**
 * Serialize an encrypted payload to a string for transmission
 */
export function serializeEncryptedPayload(
  ciphertext: ArrayBuffer,
  iv: ArrayBuffer,
  type: number
): string {
  const data = {
    v: PAYLOAD_VERSION,
    ct: arrayBufferToBase64(ciphertext),
    iv: arrayBufferToBase64(iv),
    t: type,
  };
  return JSON.stringify(data);
}

/**
 * Deserialize an encrypted payload from a string
 */
export function deserializeEncryptedPayload(serialized: string): EncryptedPayload {
  const data = JSON.parse(serialized);
  if (data.v !== PAYLOAD_VERSION) {
    throw new Error(`Unsupported payload version: ${data.v}`);
  }
  return {
    ciphertext: data.ct,
    iv: data.iv,
    type: data.t,
  };
}

/**
 * Encrypt a message for a recipient
 * 
 * Uses the E2EE session established via X3DH.
 */
export async function signalEncrypt(
  plaintext: string,
  recipientUserId: string,
  recipientDeviceId: number
): Promise<ArrayBuffer> {
  const state = get(e2ee);
  
  // Get or establish session
  const session = await e2ee.getOrEstablishSession(recipientUserId, String(recipientDeviceId));
  if (!session || !session.encryptionKey) {
    throw new Error('No E2EE session available');
  }
  
  // Encrypt using AES-GCM
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const plaintextBytes = new TextEncoder().encode(plaintext);
  
  const ciphertext = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    session.encryptionKey,
    plaintextBytes
  );
  
  // Combine IV and ciphertext
  const combined = new Uint8Array(iv.length + ciphertext.byteLength);
  combined.set(iv);
  combined.set(new Uint8Array(ciphertext), iv.length);
  
  return combined.buffer;
}

/**
 * Decrypt a message from a sender
 */
export async function signalDecrypt(
  ciphertext: ArrayBuffer,
  senderUserId: string,
  senderDeviceId: number,
  messageType: number
): Promise<string> {
  const state = get(e2ee);
  
  // Get or establish session
  const session = await e2ee.getOrEstablishSession(senderUserId, String(senderDeviceId));
  if (!session || !session.encryptionKey) {
    throw new Error('No E2EE session available');
  }
  
  // Extract IV and ciphertext
  const data = new Uint8Array(ciphertext);
  const iv = data.slice(0, 12);
  const ct = data.slice(12);
  
  // Decrypt using AES-GCM
  const plaintext = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv },
    session.encryptionKey,
    ct
  );
  
  return new TextDecoder().decode(plaintext);
}

/**
 * Check if E2EE is supported and initialized
 */
export function isSignalE2EESupported(): boolean {
  if (!browser) return false;
  return isE2EESupported();
}

// ============================================================================
// Utility Functions
// ============================================================================

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer.slice(0);
}

// ============================================================================
// Public API Functions (for messages.ts compatibility)
// ============================================================================

/**
 * Check if E2EE is initialized and ready
 */
export function isInitialized(): boolean {
  if (!browser) return false;
  try {
    const $e2ee = get(e2ee);
    return $e2ee.initialized && $e2ee.supported;
  } catch {
    return false;
  }
}

/**
 * Decrypt a message payload
 * 
 * This is a convenience function that wraps signalDecrypt for use
 * in the message handling flow.
 * 
 * @param encryptedContent - The encrypted content string (Base64 ciphertext)
 * @param senderUserId - The sender's user ID
 * @param senderDeviceId - The sender's device ID
 * @param messageType - The message type (for protocol handling)
 */
export async function decryptMessage(
  encryptedContent: string,
  senderUserId: string,
  senderDeviceId: number,
  messageType: number
): Promise<string | null> {
  try {
    // Get ciphertext as ArrayBuffer
    const ciphertext = base64ToArrayBuffer(encryptedContent);
    
    // Decrypt using signal protocol
    const plaintext = await signalDecrypt(
      ciphertext,
      senderUserId,
      senderDeviceId,
      messageType
    );
    
    return plaintext;
  } catch (error) {
    console.error('[E2EE] decryptMessage failed:', error);
    return null;
  }
}
