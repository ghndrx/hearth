/**
 * Safety Number Generation & Verification
 * 
 * Safety numbers provide a cryptographic shortcut for verifying that a pairwise
 * E2EE channel is secure and that neither party's identity key has been tampered
 * with or replaced.
 * 
 * The safety number is a 60-digit number (displayed as 12 groups of 5 digits)
 * derived from a SHA-256 hash of both users' identity public keys, sorted
 * lexicographically by user ID.
 * 
 * Reference: Signal Protocol safety numbers specification
 */

import { exportPublicKey } from './keys';

/**
 * Compute SHA-256 hash
 */
async function sha256(data: BufferSource): Promise<ArrayBuffer> {
  return crypto.subtle.digest('SHA-256', data);
}

/**
 * Convert a string to UTF-8 bytes
 */
function stringToBytes(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

/**
 * Compare two Uint8Arrays lexicographically
 */
function compareBytes(a: Uint8Array, b: Uint8Array): number {
  const len = Math.min(a.length, b.length);
  for (let i = 0; i < len; i++) {
    if (a[i] < b[i]) return -1;
    if (a[i] > b[i]) return 1;
  }
  return a.length - b.length;
}

/**
 * Generate a safety number for a pairwise E2EE channel.
 * 
 * The safety number is computed as:
 *   SHA256(sorted_user_id_1 + sorted_user_id_2 + identity_key_1 + identity_key_2)
 * 
 * Where sorted_user_ids are sorted lexicographically by user ID string.
 * This ensures both parties compute the same safety number.
 * 
 * @param localIdentityKey - Local user's P-256 identity public key
 * @param remoteIdentityKey - Remote user's P-256 identity public key
 * @param localUserId - Local user's UUID string
 * @param remoteUserId - Remote user's UUID string
 * @returns 60-digit safety number formatted as "XXXXX XXXXX XXXXX XXXXX XXXXX XXXXX"
 */
export async function generateSafetyNumber(
  localIdentityKey: CryptoKey,
  remoteIdentityKey: CryptoKey,
  localUserId: string,
  remoteUserId: string
): Promise<string> {
  // Sort user IDs lexicographically to ensure both parties compute the same hash
  const [userId1, userId2] = localUserId < remoteUserId
    ? [localUserId, remoteUserId]
    : [remoteUserId, localUserId];

  // Export identity public keys as raw bytes
  const localKeyBytes = await exportPublicKey(localIdentityKey);
  const remoteKeyBytes = await exportPublicKey(remoteIdentityKey);

  // Concatenate key material in sorted order
  const combined = new Uint8Array(
    stringToBytes(userId1).length +
    stringToBytes(userId2).length +
    localKeyBytes.byteLength +
    remoteKeyBytes.byteLength
  );

  let offset = 0;
  
  // Add sorted user IDs
  const userId1Bytes = stringToBytes(userId1);
  combined.set(userId1Bytes, offset);
  offset += userId1Bytes.length;

  const userId2Bytes = stringToBytes(userId2);
  combined.set(userId2Bytes, offset);
  offset += userId2Bytes.length;

  // Add identity keys in sorted order (same order as user IDs)
  if (localUserId < remoteUserId) {
    combined.set(new Uint8Array(localKeyBytes), offset);
    offset += localKeyBytes.byteLength;
    combined.set(new Uint8Array(remoteKeyBytes), offset);
  } else {
    combined.set(new Uint8Array(remoteKeyBytes), offset);
    offset += remoteKeyBytes.byteLength;
    combined.set(new Uint8Array(localKeyBytes), offset);
  }

  // Compute SHA-256 hash
  const hash = await sha256(combined);
  const hashBytes = new Uint8Array(hash);

  // Extract 60 decimal digits from 30 bytes (15 x 16-bit values)
  // Each 2 bytes gives us ~5 digits (0-65535 range)
  const digits: string[] = [];
  for (let i = 0; i < 15; i++) {
    const byte1 = hashBytes[i * 2];
    const byte2 = hashBytes[i * 2 + 1];
    // Combine two bytes into a 16-bit unsigned integer
    const value = (byte1 << 8) | byte2;
    // Take modulo 100000 to get exactly 5 digits
    const fiveDigits = String(value % 100000).padStart(5, '0');
    digits.push(fiveDigits);
  }

  // Format as 6 groups of 5 digits: "12345 67890 12345 67890 12345 67890"
  return digits.join(' ');
}

/**
 * Verify a safety number by comparing two versions.
 * 
 * @param localSafetyNumber - Safety number computed locally
 * @param remoteSafetyNumber - Safety number from remote party (entered or scanned)
 * @returns true if the safety numbers match exactly
 */
export function verifySafetyNumber(
  localSafetyNumber: string,
  remoteSafetyNumber: string
): boolean {
  // Remove all whitespace and compare
  const normalize = (s: string) => s.replace(/\s+/g, '');
  return normalize(localSafetyNumber) === normalize(remoteSafetyNumber);
}

/**
 * Generate a shorter display-friendly version of safety number.
 * Uses first 30 digits only, formatted as QR-friendly blocks.
 */
export function getShortSafetyNumber(safetyNumber: string): string {
  const normalized = safetyNumber.replace(/\s+/g, '');
  // Return first 30 digits as 6 groups of 5
  const short = normalized.slice(0, 30);
  return short.match(/.{1,5}/g)?.join(' ') ?? short;
}

/**
 * Generate bytes suitable for QR code encoding.
 * The QR code encodes the raw hash bytes (no formatting).
 */
export async function safetyNumberToQRBytes(
  localIdentityKey: CryptoKey,
  remoteIdentityKey: CryptoKey,
  localUserId: string,
  remoteUserId: string
): Promise<Uint8Array> {
  const [userId1, userId2] = localUserId < remoteUserId
    ? [localUserId, remoteUserId]
    : [remoteUserId, localUserId];

  const localKeyBytes = await exportPublicKey(localIdentityKey);
  const remoteKeyBytes = await exportPublicKey(remoteIdentityKey);

  const combined = new Uint8Array(
    stringToBytes(userId1).length +
    stringToBytes(userId2).length +
    localKeyBytes.byteLength +
    remoteKeyBytes.byteLength
  );

  let offset = 0;
  const userId1Bytes = stringToBytes(userId1);
  combined.set(userId1Bytes, offset);
  offset += userId1Bytes.length;

  const userId2Bytes = stringToBytes(userId2);
  combined.set(userId2Bytes, offset);
  offset += userId2Bytes.length;

  if (localUserId < remoteUserId) {
    combined.set(new Uint8Array(localKeyBytes), offset);
    offset += localKeyBytes.byteLength;
    combined.set(new Uint8Array(remoteKeyBytes), offset);
  } else {
    combined.set(new Uint8Array(remoteKeyBytes), offset);
    offset += remoteKeyBytes.byteLength;
    combined.set(new Uint8Array(localKeyBytes), offset);
  }

  return new Uint8Array(await sha256(combined));
}

/**
 * Safety number matcher - checks if any of a user's devices
 * have matching safety numbers.
 */
export interface SafetyNumberMatch {
  deviceId: string;
  safetyNumber: string;
}

/**
 * Generate safety numbers for all device combinations between two users.
 */
export async function generateMultiDeviceSafetyNumbers(
  localIdentityKey: CryptoKey,
  remoteDevices: Array<{ deviceId: string; identityKey: CryptoKey }>,
  localUserId: string,
  remoteUserId: string
): Promise<SafetyNumberMatch[]> {
  const matches: SafetyNumberMatch[] = [];

  for (const remote of remoteDevices) {
    const sn = await generateSafetyNumber(
      localIdentityKey,
      remote.identityKey,
      localUserId,
      remoteUserId
    );
    matches.push({
      deviceId: remote.deviceId,
      safetyNumber: sn
    });
  }

  return matches;
}
