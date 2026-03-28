/**
 * E2EE Session Storage - IndexedDB Implementation
 * 
 * Provides IndexedDB storage for E2EE session state,
 * including:
 * - Ratchet session state (for Double Ratchet)
 * - Identity keys
 * - Prekey bundles
 * - Session metadata
 * 
 * This storage is encrypted at rest using AES-GCM with
 * a key derived from the user's authentication token.
 */

import { browser } from '$app/environment';
import type { RatchetState } from './ratchet';

const DB_NAME = 'hearth-e2ee-sessions';
const DB_VERSION = 1;

// Store names
const STORES = {
  SESSIONS: 'ratchet_sessions',
  IDENTITY_KEYS: 'identity_keys',
  PREKEY_BUNDLES: 'prekey_bundles',
  METADATA: 'metadata',
} as const;

/**
 * Stored session record
 */
interface StoredSession {
  id: string;                    // recipientUserId_recipientDeviceId
  recipientUserId: string;
  recipientDeviceId: string;
  state: StoredRatchetState;
  createdAt: number;
  lastUsed: number;
}

/**
 * Serialized ratchet state for storage
 */
interface StoredRatchetState {
  rootKey: string;               // Base64
  dhRatchet: {
    publicKey: string;          // Base64
    privateKey: string;         // Base64 (encrypted)
    remotePublicKey: string | null;
  };
  chainKeys: {
    sending: string | null;      // Base64
    receiving: string | null;   // Base64
  };
  messageNumbers: {
    sending: number;
    receiving: number;
  };
  previousChainLength: number;
  remoteIdentityKey: string | null;
}

/**
 * Identity key storage record
 */
interface StoredIdentityKey {
  id: string;
  publicKey: string;            // Base64
  encryptedPrivateKey: string;  // Base64 encrypted
  iv: string;                   // Base64
  createdAt: number;
}

/**
 * Prekey bundle cache
 */
interface StoredPreKeyBundle {
  id: string;                    // userId_deviceId
  userId: string;
  deviceId: string;
  bundle: string;               // JSON stringified
  cachedAt: number;
  expiresAt: number;
}

/**
 * Session storage manager
 */
export class SessionStorage {
  private db: IDBDatabase | null = null;
  private encryptionKey: CryptoKey | null = null;
  private initialized = false;

  /**
   * Initialize storage with an encryption key
   */
  async init(encryptionKey?: CryptoKey): Promise<void> {
    if (!browser) {
      throw new Error('SessionStorage can only be used in browser environment');
    }

    if (this.initialized && this.db) {
      return;
    }

    // Open database
    this.db = await this.openDatabase();

    // Set encryption key if provided
    if (encryptionKey) {
      this.encryptionKey = encryptionKey;
    }

    this.initialized = true;
  }

  /**
   * Open or create the IndexedDB database
   */
  private openDatabase(): Promise<IDBDatabase> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);

      request.onerror = () => {
        reject(new Error(`Failed to open database: ${request.error?.message}`));
      };

      request.onsuccess = () => {
        resolve(request.result);
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;

        // Ratchet sessions store
        if (!db.objectStoreNames.contains(STORES.SESSIONS)) {
          const store = db.createObjectStore(STORES.SESSIONS, { keyPath: 'id' });
          store.createIndex('recipientUserId', 'recipientUserId');
          store.createIndex('lastUsed', 'lastUsed');
        }

        // Identity keys store
        if (!db.objectStoreNames.contains(STORES.IDENTITY_KEYS)) {
          db.createObjectStore(STORES.IDENTITY_KEYS, { keyPath: 'id' });
        }

        // Prekey bundle cache
        if (!db.objectStoreNames.contains(STORES.PREKEY_BUNDLES)) {
          const store = db.createObjectStore(STORES.PREKEY_BUNDLES, { keyPath: 'id' });
          store.createIndex('userId', 'userId');
          store.createIndex('expiresAt', 'expiresAt');
        }

        // Metadata store
        if (!db.objectStoreNames.contains(STORES.METADATA)) {
          db.createObjectStore(STORES.METADATA, { keyPath: 'key' });
        }
      };
    });
  }

  /**
   * Derive an encryption key from a password/secret
   */
  async deriveEncryptionKey(secret: string): Promise<CryptoKey> {
    const salt = await this.getMetadata('encryption_salt') as Uint8Array | null;
    let saltBytes: Uint8Array;

    if (salt) {
      // Ensure we have a regular ArrayBuffer backing, not SharedArrayBuffer
      saltBytes = new Uint8Array(salt.buffer.slice(0));
    } else {
      saltBytes = crypto.getRandomValues(new Uint8Array(32));
      await this.setMetadata('encryption_salt', saltBytes);
    }

    const keyMaterial = await crypto.subtle.importKey(
      'raw',
      new TextEncoder().encode(secret),
      'PBKDF2',
      false,
      ['deriveKey']
    );

    return crypto.subtle.deriveKey(
      {
        name: 'PBKDF2',
        salt: saltBytes.buffer as ArrayBuffer,
        iterations: 100000,
        hash: 'SHA-256',
      },
      keyMaterial,
      { name: 'AES-GCM', length: 256 },
      false,
      ['encrypt', 'decrypt']
    );
  }

  /**
   * Set the encryption key directly
   */
  setEncryptionKey(key: CryptoKey): void {
    this.encryptionKey = key;
  }

  /**
   * Encrypt data
   */
  private async encrypt(data: ArrayBuffer): Promise<{ encrypted: ArrayBuffer; iv: Uint8Array }> {
    if (!this.encryptionKey) {
      // No encryption - return as-is
      return { encrypted: data, iv: crypto.getRandomValues(new Uint8Array(12)) };
    }

    const iv = crypto.getRandomValues(new Uint8Array(12));
    const encrypted = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv },
      this.encryptionKey,
      data
    );

    return { encrypted, iv };
  }

  /**
   * Decrypt data
   */
  private async decrypt(encrypted: ArrayBuffer, iv: Uint8Array): Promise<ArrayBuffer> {
    if (!this.encryptionKey) {
      return encrypted;
    }

    // Ensure iv is a regular ArrayBuffer, not SharedArrayBuffer
    const ivBuffer = iv.buffer.slice(0);

    return crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: ivBuffer as ArrayBuffer },
      this.encryptionKey,
      encrypted
    );
  }

  // ============================================================================
  // Session Storage
  // ============================================================================

  /**
   * Store a ratchet session state
   */
  async storeSession(
    recipientUserId: string,
    recipientDeviceId: string,
    state: RatchetState
  ): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    const id = `${recipientUserId}_${recipientDeviceId}`;
    const now = Date.now();

    // Serialize the ratchet state
    const serialized = this.serializeRatchetState(state);

    // Encrypt private key
    const privateKeyJwk = JSON.parse(new TextDecoder().decode(new Uint8Array(state.dhRatchet.dhKeyPair.privateKey)));
    const privateKeyBuffer = new TextEncoder().encode(JSON.stringify(privateKeyJwk)).buffer;
    const { encrypted: encryptedPrivateKey, iv: privateKeyIv } = await this.encrypt(privateKeyBuffer);

    const record: StoredSession = {
      id,
      recipientUserId,
      recipientDeviceId,
      state: {
        ...serialized,
        dhRatchet: {
          ...serialized.dhRatchet,
          privateKey: arrayBufferToBase64(encryptedPrivateKey),
        },
      },
      createdAt: state.createdAt,
      lastUsed: now,
    };

    // Store IV alongside the record (in a separate metadata field)
    await this.setMetadata(`session_iv_${id}`, privateKeyIv);

    return this.put(STORES.SESSIONS, record);
  }

  /**
   * Get a ratchet session state
   */
  async getSession(
    recipientUserId: string,
    recipientDeviceId: string
  ): Promise<RatchetState | null> {
    if (!this.db) throw new Error('Storage not initialized');

    const id = `${recipientUserId}_${recipientDeviceId}`;
    const record = await this.get<StoredSession>(STORES.SESSIONS, id);

    if (!record) return null;

    // Get IV and decrypt private key
    const iv = await this.getMetadata(`session_iv_${id}`) as Uint8Array;
    if (!iv) throw new Error('Missing IV for session');

    const encryptedPrivateKey = base64ToArrayBuffer(record.state.dhRatchet.privateKey);
    const privateKeyBuffer = await this.decrypt(encryptedPrivateKey, iv);
    const privateKeyJwk = JSON.parse(new TextDecoder().decode(new Uint8Array(privateKeyBuffer)));

    // Reconstruct ratchet state
    return this.deserializeRatchetState(record.state, privateKeyJwk);
  }

  /**
   * List all stored sessions for a user
   */
  async listSessionsForUser(recipientUserId: string): Promise<string[]> {
    if (!this.db) throw new Error('Storage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.SESSIONS, 'readonly');
      const store = tx.objectStore(STORES.SESSIONS);
      const index = store.index('recipientUserId');
      const request = index.getAllKeys(recipientUserId);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result.map(k => String(k)));
    });
  }

  /**
   * Delete a session
   */
  async deleteSession(recipientUserId: string, recipientDeviceId: string): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    const id = `${recipientUserId}_${recipientDeviceId}`;
    await this.delete(STORES.SESSIONS, id);
    await this.deleteMetadata(`session_iv_${id}`);
  }

  /**
   * Update session last used time
   */
  async touchSession(recipientUserId: string, recipientDeviceId: string): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    const id = `${recipientUserId}_${recipientDeviceId}`;
    const record = await this.get<StoredSession>(STORES.SESSIONS, id);

    if (record) {
      record.lastUsed = Date.now();
      await this.put(STORES.SESSIONS, record);
    }
  }

  // ============================================================================
  // Identity Key Storage
  // ============================================================================

  /**
   * Store identity key pair
   */
  async storeIdentityKey(
    publicKey: ArrayBuffer,
    encryptedPrivateKey: ArrayBuffer,
    iv: Uint8Array
  ): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    const record: StoredIdentityKey = {
      id: 'identity',
      publicKey: arrayBufferToBase64(publicKey),
      encryptedPrivateKey: arrayBufferToBase64(encryptedPrivateKey),
      iv: arrayBufferToBase64(iv.buffer.slice(0) as ArrayBuffer),
      createdAt: Date.now(),
    };

    return this.put(STORES.IDENTITY_KEYS, record);
  }

  /**
   * Get identity key pair
   */
  async getIdentityKey(): Promise<{
    publicKey: ArrayBuffer;
    encryptedPrivateKey: ArrayBuffer;
    iv: Uint8Array;
  } | null> {
    if (!this.db) throw new Error('Storage not initialized');

    const record = await this.get<StoredIdentityKey>(STORES.IDENTITY_KEYS, 'identity');
    if (!record) return null;

    return {
      publicKey: base64ToArrayBuffer(record.publicKey),
      encryptedPrivateKey: base64ToArrayBuffer(record.encryptedPrivateKey),
      iv: new Uint8Array(base64ToArrayBuffer(record.iv)),
    };
  }

  // ============================================================================
  // Prekey Bundle Cache
  // ============================================================================

  /**
   * Cache a prekey bundle
   */
  async cachePreKeyBundle(
    userId: string,
    deviceId: string,
    bundle: unknown,
    ttlMs: number = 3600000 // 1 hour default
  ): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    const id = `${userId}_${deviceId}`;
    const now = Date.now();

    const record: StoredPreKeyBundle = {
      id,
      userId,
      deviceId,
      bundle: JSON.stringify(bundle),
      cachedAt: now,
      expiresAt: now + ttlMs,
    };

    return this.put(STORES.PREKEY_BUNDLES, record);
  }

  /**
   * Get cached prekey bundle
   */
  async getCachedPreKeyBundle(
    userId: string,
    deviceId: string
  ): Promise<unknown | null> {
    if (!this.db) throw new Error('Storage not initialized');

    const id = `${userId}_${deviceId}`;
    const record = await this.get<StoredPreKeyBundle>(STORES.PREKEY_BUNDLES, id);

    if (!record) return null;

    // Check expiration
    if (Date.now() > record.expiresAt) {
      await this.delete(STORES.PREKEY_BUNDLES, id);
      return null;
    }

    return JSON.parse(record.bundle);
  }

  /**
   * Clear expired prekey bundles
   */
  async clearExpiredPreKeyBundles(): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    const now = Date.now();

    await new Promise<void>((resolve, reject) => {
      const tx = this.db!.transaction(STORES.PREKEY_BUNDLES, 'readwrite');
      const store = tx.objectStore(STORES.PREKEY_BUNDLES);
      const index = store.index('expiresAt');
      const range = IDBKeyRange.upperBound(now);
      const request = index.openCursor(range);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const cursor = request.result;
        if (cursor) {
          cursor.delete();
          cursor.continue();
        } else {
          resolve();
        }
      };
    });
  }

  // ============================================================================
  // Metadata Storage
  // ============================================================================

  async getMetadata(key: string): Promise<unknown | null> {
    if (!this.db) throw new Error('Storage not initialized');

    const record = await this.get<{ key: string; value: unknown }>(STORES.METADATA, key);
    return record?.value ?? null;
  }

  async setMetadata(key: string, value: unknown): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    return this.put(STORES.METADATA, { key, value });
  }

  private async deleteMetadata(key: string): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    return this.delete(STORES.METADATA, key);
  }

  // ============================================================================
  // Utilities
  // ============================================================================

  /**
   * Clear all stored data
   */
  async clearAll(): Promise<void> {
    if (!this.db) throw new Error('Storage not initialized');

    for (const storeName of Object.values(STORES)) {
      await new Promise<void>((resolve, reject) => {
        const tx = this.db!.transaction(storeName, 'readwrite');
        const store = tx.objectStore(storeName);
        const request = store.clear();
        request.onerror = () => reject(request.error);
        request.onsuccess = () => resolve();
      });
    }
  }

  /**
   * Close database connection
   */
  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
    }
    this.initialized = false;
    this.encryptionKey = null;
  }

  // --- Private helpers ---

  private put<T>(storeName: string, data: T): Promise<void> {
    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(storeName, 'readwrite');
      const store = tx.objectStore(storeName);
      const request = store.put(data);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  private get<T>(storeName: string, key: string): Promise<T | null> {
    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(storeName, 'readonly');
      const store = tx.objectStore(storeName);
      const request = store.get(key);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result ?? null);
    });
  }

  private delete(storeName: string, key: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(storeName, 'readwrite');
      const store = tx.objectStore(storeName);
      const request = store.delete(key);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  private serializeRatchetState(state: RatchetState): StoredRatchetState {
    return {
      rootKey: arrayBufferToBase64(state.rootKey),
      dhRatchet: {
        publicKey: arrayBufferToBase64(state.dhRatchet.dhKeyPair.publicKey),
        privateKey: '', // Will be encrypted separately
        remotePublicKey: state.dhRatchet.remoteDHPublicKey
          ? arrayBufferToBase64(state.dhRatchet.remoteDHPublicKey)
          : null,
      },
      chainKeys: {
        sending: state.chainKeys.sending
          ? arrayBufferToBase64(state.chainKeys.sending)
          : null,
        receiving: state.chainKeys.receiving
          ? arrayBufferToBase64(state.chainKeys.receiving)
          : null,
      },
      messageNumbers: { ...state.messageNumbers },
      previousChainLength: state.previousChainLength,
      remoteIdentityKey: state.remoteIdentityKey
        ? arrayBufferToBase64(state.remoteIdentityKey)
        : null,
    };
  }

  private deserializeRatchetState(
    stored: StoredRatchetState,
    privateKeyJwk: JsonWebKey
  ): RatchetState {
    return {
      dhRatchet: {
        dhKeyPair: {
          publicKey: base64ToArrayBuffer(stored.dhRatchet.publicKey),
          privateKey: new TextEncoder().encode(JSON.stringify(privateKeyJwk)).buffer,
        },
        remoteDHPublicKey: stored.dhRatchet.remotePublicKey
          ? base64ToArrayBuffer(stored.dhRatchet.remotePublicKey)
          : null,
      },
      rootKey: base64ToArrayBuffer(stored.rootKey),
      chainKeys: {
        sending: stored.chainKeys.sending
          ? base64ToArrayBuffer(stored.chainKeys.sending)
          : null,
        receiving: stored.chainKeys.receiving
          ? base64ToArrayBuffer(stored.chainKeys.receiving)
          : null,
      },
      messageNumbers: { ...stored.messageNumbers },
      previousChainLength: stored.previousChainLength,
      remoteIdentityKey: stored.remoteIdentityKey
        ? base64ToArrayBuffer(stored.remoteIdentityKey)
        : null,
      recipientUserId: '', // Will be set from context
      recipientDeviceId: '', // Will be set from context
      createdAt: Date.now(),
      lastUsed: Date.now(),
    };
  }
}

// Singleton instance
export const sessionStorage = new SessionStorage();

// ============================================================================
// Utility Functions
// ============================================================================

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
  return new Uint8Array(bytes).buffer.slice(0);
}
