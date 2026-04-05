/**
 * E2EE Key Storage - IndexedDB Layer
 * 
 * This module provides IndexedDB storage for E2EE keys.
 * Uses the WebCrypto-based implementation in crypto/keys.ts.
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


