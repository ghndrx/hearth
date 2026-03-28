/**
 * E2EE Key Storage - IndexedDB Layer
 * 
 * This module provides IndexedDB storage for E2EE keys.
 * 
 * For Phase 1 (DM Foundation), we use the WebCrypto-based implementation
 * in crypto/keys.ts. The libsignal-client WASM integration is a future
 * enhancement for when we need the full Signal Protocol implementation.
 * 
 * Storage Schema:
 * - identityKeys: Device identity key pairs
 * - signedPreKeys: Signed prekey pairs (rotated periodically)
 * - oneTimePreKeys: One-time prekey pairs (consumed on use)
 * - sessions: Established E2EE sessions (per recipient/device)
 * - metadata: Device registration info and protocol state
 */

// Re-export WebCrypto-based types for compatibility
export type { IdentityKeyPair, SignedPreKey, OneTimePreKey } from '$lib/crypto/keys';

// Import the WebCrypto-based KeyStore from crypto/keys
import { KeyStore, keyStore, type IdentityKeyPair, type SignedPreKey, type OneTimePreKey } from '$lib/crypto/keys';

export { keyStore };

// Re-export key generation functions from crypto/keys
export {
  generateIdentityKeyPair,
  generateSignedPreKey,
  generateOneTimePreKeys,
  exportPublicKey,
  importPublicKey,
  deriveSharedSecret,
  deriveMessageKey,
} from '$lib/crypto/keys';

// ============================================================================
// IndexedDB Storage - Delegated to crypto/keys.ts
// ============================================================================

// The KeyStore class from crypto/keys.ts provides IndexedDB storage
// using the raw IndexedDB API (no external dependencies)

export { KeyStore };

// ============================================================================
// libsignal-client WASM Integration (Future Enhancement)
// ============================================================================
// The following functions are stubs for libsignal-client WASM integration.
// They will be implemented when we upgrade from WebCrypto to libsignal-client
// for full Signal Protocol compliance.
//
// For now, we use the WebCrypto-based X3DH implementation in crypto/signal-protocol.ts

/**
 * Stub: Initialize libsignal-client WASM module
 * 
 * This will be implemented when we integrate libsignal-client WASM.
 * The WASM module provides:
 * - Curve25519 key operations (instead of P-256 in WebCrypto)
 * - Double Ratchet algorithm implementation
 * - Proper Signal Protocol session management
 */
export async function initializeWasm(): Promise<void> {
  console.debug('libsignal-client WASM initialization deferred to future enhancement');
  // TODO: Load and initialize libsignal-client WASM
  // import init from 'libsignal-client';
  // await init();
}

/**
 * Stub: Generate identity key using libsignal-client
 */
export async function generateIdentityKey(): Promise<IdentityKeyPair> {
  // Fall back to WebCrypto implementation
  const keyPair = await crypto.subtle.generateKey(
    { name: 'ECDH', namedCurve: 'P-256' },
    true,
    ['deriveBits', 'deriveKey']
  );
  return {
    publicKey: keyPair.publicKey,
    privateKey: keyPair.privateKey,
  } as IdentityKeyPair;
}

/**
 * Stub: Store session record (libsignal-client format)
 * 
 * libsignal-client stores sessions as serialized SessionRecord objects.
 * For now, we store session state in our own format in crypto/secure-storage.ts
 */
export async function storeSessionRecord(): Promise<void> {
  throw new Error('libsignal-client WASM not yet integrated - use WebCrypto session management');
}

/**
 * Stub: Get session record (libsignal-client format)
 */
export async function getSessionRecord(): Promise<unknown> {
  throw new Error('libsignal-client WASM not yet integrated - use WebCrypto session management');
}

/**
 * Stub: Process prekey bundle and create session
 */
export async function processPreKeyBundle(): Promise<void> {
  throw new Error('libsignal-client WASM not yet integrated - use WebCrypto X3DH');
}
