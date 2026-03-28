/**
 * X3DH Key Exchange Service
 * 
 * Phase 1 uses the WebCrypto-based X3DH implementation from crypto/signal-protocol.ts.
 * The libsignal-client WASM implementation is deferred to a future enhancement.
 * 
 * This module provides the complete X3DH key agreement for establishing E2EE sessions.
 */

// Re-export X3DH functions from the WebCrypto implementation
export {
  performX3DHSender,
  performX3DHRecipient,
  type RemotePreKeyBundle,
} from '$lib/crypto/signal-protocol';

// Import the key storage functions from crypto/keys
import { keyStore } from '$lib/crypto/keys';

export { keyStore };

// Re-export types for compatibility
export type { RemotePreKeyBundle as PreKeyBundle } from '$lib/crypto/signal-protocol';

/**
 * Initiate X3DH key agreement with a remote user (sender side)
 * 
 * Called when we want to send an encrypted message to a recipient.
 * Performs X3DH as Alice (sender) to establish a shared secret.
 * 
 * @param recipientUserId - The recipient's user ID
 * @param recipientDeviceId - The recipient's device ID
 * @returns The X3DH result including shared secret and ephemeral public key
 */
export async function initiateX3DH(
  recipientUserId: string,
  recipientDeviceId: string
): Promise<{ 
  sharedSecret: ArrayBuffer; 
  ephemeralPublicKey: ArrayBuffer;
  usedPreKeyId?: number;
}> {
  // Import the WebCrypto X3DH implementation
  const { performX3DHSender } = await import('$lib/crypto/signal-protocol');
  const { getIdentityKey } = await import('$lib/crypto/keys');
  const { e2eeApi } = await import('$lib/crypto/e2ee-api');
  
  // Get our identity key
  const identityKey = await getIdentityKey();
  if (!identityKey) {
    throw new Error('No identity key found - E2EE not initialized');
  }
  
  // Get recipient's prekey bundle
  const bundle = await e2eeApi.getPreKeyBundle(recipientUserId, recipientDeviceId);
  
  // Perform X3DH as sender
  const result = await performX3DHSender(identityKey, bundle);
  
  return {
    sharedSecret: result.sharedSecret,
    ephemeralPublicKey: result.ephemeralPublicKey,
    usedPreKeyId: result.usedPreKeyId,
  };
}

/**
 * Process an incoming X3DH initial message (recipient side)
 * 
 * Called when we receive a pre-key message from a new contact.
 * Performs X3DH as Bob (recipient) to establish a shared secret.
 * 
 * @param senderIdentityKey - The sender's identity key (raw ArrayBuffer)
 * @param ephemeralPublicKey - The ephemeral public key from the sender (ArrayBuffer)
 * @param usedPreKeyId - The prekey ID that was used (if any)
 * @returns The derived shared secret
 */
export async function processPreKeyMessage(
  senderIdentityKey: ArrayBuffer,
  ephemeralPublicKey: ArrayBuffer,
  usedPreKeyId?: number
): Promise<ArrayBuffer> {
  const { performX3DHRecipient } = await import('$lib/crypto/signal-protocol');
  const { getIdentityKey, getSignedPreKey, getOneTimePreKey } = await import('$lib/crypto/keys');
  
  // Get our identity key
  const identityKey = await getIdentityKey();
  if (!identityKey) {
    throw new Error('No identity key found - E2EE not initialized');
  }
  
  // Get our signed prekey (keyId = 1 for initial signed prekey)
  const signedPreKey = await getSignedPreKey(1);
  if (!signedPreKey) {
    throw new Error('No signed prekey found - please regenerate signed prekeys');
  }
  
  // Get our one-time prekey if one was used
  let oneTimePreKey = null;
  if (usedPreKeyId !== undefined) {
    oneTimePreKey = await getOneTimePreKey(usedPreKeyId);
    if (!oneTimePreKey) {
      throw new Error(`One-time prekey ${usedPreKeyId} not found`);
    }
  }
  
  // Perform X3DH as recipient
  const sharedSecret = await performX3DHRecipient(
    identityKey,
    signedPreKey,
    oneTimePreKey,
    senderIdentityKey,
    ephemeralPublicKey
  );
  
  return sharedSecret;
}

/**
 * Complete X3DH as recipient when we have all the necessary data
 * 
 * This is the lower-level version that takes all parameters directly.
 * Use this when you already have the bundle and key data.
 * 
 * @param localIdentityKey - Our identity key pair
 * @param localSignedPreKey - Our signed prekey
 * @param localOneTimePreKey - Our one-time prekey (optional)
 * @param remoteIdentityKey - Sender's identity key (ArrayBuffer)
 * @param ephemeralPublicKey - Sender's ephemeral public key (ArrayBuffer)
 * @returns The derived shared secret
 */
export async function completeX3DHAsRecipient(
  localIdentityKey: CryptoKey,
  localSignedPreKey: { keyId: number; publicKey: CryptoKey; privateKey: CryptoKey },
  localOneTimePreKey: { keyId: number; publicKey: CryptoKey; privateKey: CryptoKey } | null,
  remoteIdentityKey: ArrayBuffer,
  ephemeralPublicKey: ArrayBuffer
): Promise<ArrayBuffer> {
  const { performX3DHRecipient } = await import('$lib/crypto/signal-protocol');
  
  // Convert CryptoKey-based keys to the format expected by signal-protocol
  const identityKeyPair = {
    publicKey: localIdentityKey,
    privateKey: localIdentityKey, // For ECDH, we use the same key for both
  };
  
  const signedPreKey = {
    keyId: localSignedPreKey.keyId,
    publicKey: localSignedPreKey.publicKey,
    privateKey: localSignedPreKey.privateKey,
    signature: new ArrayBuffer(0), // Not needed for recipient side
    timestamp: Date.now(),
  };
  
  const oneTimePreKey = localOneTimePreKey ? {
    keyId: localOneTimePreKey.keyId,
    publicKey: localOneTimePreKey.publicKey,
    privateKey: localOneTimePreKey.privateKey,
  } : null;
  
  return performX3DHRecipient(
    identityKeyPair,
    signedPreKey,
    oneTimePreKey,
    remoteIdentityKey,
    ephemeralPublicKey
  );
}

/**
 * Derive session keys from X3DH shared secret
 * 
 * Takes the X3DH shared secret and derives the initial chain key and message key.
 * 
 * @param sharedSecret - The X3DH shared secret
 * @returns The derived encryption and MAC keys
 */
export async function deriveSessionKeys(
  sharedSecret: ArrayBuffer
): Promise<{
  encryptionKey: CryptoKey;
  macKey: CryptoKey;
}> {
  const { deriveMessageKeys } = await import('$lib/crypto/signal-protocol');
  return deriveMessageKeys(sharedSecret);
}
