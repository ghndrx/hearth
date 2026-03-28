/**
 * E2EE Key Storage
 * 
 * IndexedDB-based storage for E2EE keys and session state.
 * This module provides the local storage layer for the device manager.
 * 
 * Storage Schema:
 * - identityKeys: Device identity key pairs (long-term)
 * - signedPreKeys: Signed prekey pairs (rotated periodically)
 * - oneTimePreKeys: One-time prekey pairs (consumed on use)
 * - sessions: Established E2EE sessions (per recipient/device)
 * - metadata: Device registration info and protocol state
 */

import { browser } from '$app/environment';

// Re-export types from crypto modules for compatibility
export type { IdentityKeyPair, SignedPreKey, OneTimePreKey } from '$lib/crypto/keys';
export type { SessionState } from '$lib/crypto/signal-protocol';

// Import the WebCrypto-based KeyStore from crypto/keys
import { KeyStore, keyStore as cryptoKeyStore } from '$lib/crypto/keys';
import type { IdentityKeyPair, SignedPreKey, OneTimePreKey } from '$lib/crypto/keys';

// Legacy types for backward compatibility
export interface IdentityKeyData {
  publicKey: ArrayBuffer;
  privateKey: ArrayBuffer;
}

export interface SignedPreKeyData {
  keyId: number;
  publicKey: ArrayBuffer;
  privateKey: ArrayBuffer;
  signature: ArrayBuffer;
  timestamp: number;
}

export interface OneTimePreKeyData {
  keyId: number;
  publicKey: ArrayBuffer;
  privateKey: ArrayBuffer;
}

export interface SessionData {
  recipientUserId: string;
  recipientDeviceId: string;
  rootKey: ArrayBuffer;
  chainKey: ArrayBuffer;
  messageNumber: number;
  previousChainLength: number;
  remoteIdentityKey: ArrayBuffer;
  established: boolean;
  createdAt: number;
  lastUsed: number;
}

export interface DeviceMetadata {
  deviceId: string;
  registrationId: number;
  identityKeyHash: string;
  createdAt: number;
  lastKeyRotation: number;
  signedPreKeyId: number;
}

// Storage constants
const DB_NAME = 'hearth-e2ee-keys';
const DB_VERSION = 1;

// Store names
const STORES = {
  IDENTITY: 'identity_keys',
  SIGNED_PREKEYS: 'signed_prekeys',
  ONE_TIME_PREKEYS: 'one_time_prekeys',
  SESSIONS: 'sessions',
  METADATA: 'metadata',
} as const;

/**
 * KeyStorage class - IndexedDB storage for E2EE keys
 * 
 * This class wraps IndexedDB operations for storing:
 * - Identity key pairs
 * - Signed prekeys  
 * - One-time prekeys
 * - Session state
 * - Device metadata
 */
export class KeyStorage {
  private dbName = DB_NAME;
  private dbVersion = DB_VERSION;
  private db: IDBDatabase | null = null;
  private initialized = false;

  /**
   * Initialize the key storage
   */
  async init(): Promise<void> {
    if (!browser) return;
    if (this.initialized) return;

    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.dbName, this.dbVersion);

      request.onerror = () => {
        reject(new Error(`Failed to open key storage: ${request.error?.message}`));
      };

      request.onsuccess = () => {
        this.db = request.result;
        this.initialized = true;
        resolve();
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;

        // Identity keys store
        if (!db.objectStoreNames.contains(STORES.IDENTITY)) {
          db.createObjectStore(STORES.IDENTITY, { keyPath: 'id' });
        }

        // Signed prekeys store
        if (!db.objectStoreNames.contains(STORES.SIGNED_PREKEYS)) {
          const store = db.createObjectStore(STORES.SIGNED_PREKEYS, { keyPath: 'keyId' });
          store.createIndex('createdAt', 'createdAt');
        }

        // One-time prekeys store
        if (!db.objectStoreNames.contains(STORES.ONE_TIME_PREKEYS)) {
          const store = db.createObjectStore(STORES.ONE_TIME_PREKEYS, { keyPath: 'keyId' });
          store.createIndex('keyId', 'keyId');
        }

        // Sessions store
        if (!db.objectStoreNames.contains(STORES.SESSIONS)) {
          const store = db.createObjectStore(STORES.SESSIONS, { keyPath: 'id' });
          store.createIndex('recipientUserId', 'recipientUserId');
          store.createIndex('lastUsed', 'lastUsed');
        }

        // Metadata store
        if (!db.objectStoreNames.contains(STORES.METADATA)) {
          db.createObjectStore(STORES.METADATA, { keyPath: 'key' });
        }
      };
    });
  }

  /**
   * Check if the store has been initialized
   */
  isInitialized(): boolean {
    return this.initialized;
  }

  /**
   * Store identity key pair
   */
  async storeIdentityKey(keyData: IdentityKeyData): Promise<void> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.IDENTITY, 'readwrite');
      const store = tx.objectStore(STORES.IDENTITY);

      const record = {
        id: 'identity',
        publicKey: keyData.publicKey,
        privateKey: keyData.privateKey,
        createdAt: Date.now(),
      };

      const request = store.put(record);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  /**
   * Get identity key pair
   */
  async getIdentityKey(): Promise<IdentityKeyData | null> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.IDENTITY, 'readonly');
      const store = tx.objectStore(STORES.IDENTITY);
      const request = store.get('identity');

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const result = request.result;
        if (!result) {
          resolve(null);
          return;
        }
        resolve({
          publicKey: result.publicKey,
          privateKey: result.privateKey,
        });
      };
    });
  }

  /**
   * Check if identity key exists
   */
  async hasIdentityKey(): Promise<boolean> {
    const key = await this.getIdentityKey();
    return key !== null;
  }

  /**
   * Store signed prekey
   */
  async storeSignedPreKey(preKey: SignedPreKeyData): Promise<void> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.SIGNED_PREKEYS, 'readwrite');
      const store = tx.objectStore(STORES.SIGNED_PREKEYS);

      const record = {
        keyId: preKey.keyId,
        publicKey: preKey.publicKey,
        privateKey: preKey.privateKey,
        signature: preKey.signature,
        createdAt: preKey.timestamp || Date.now(),
      };

      const request = store.put(record);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  /**
   * Get signed prekey by ID
   */
  async getSignedPreKey(keyId: number): Promise<SignedPreKeyData | null> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.SIGNED_PREKEYS, 'readonly');
      const store = tx.objectStore(STORES.SIGNED_PREKEYS);
      const request = store.get(keyId);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const result = request.result;
        if (!result) {
          resolve(null);
          return;
        }
        resolve({
          keyId: result.keyId,
          publicKey: result.publicKey,
          privateKey: result.privateKey,
          signature: result.signature,
          timestamp: result.createdAt,
        });
      };
    });
  }

  /**
   * Get latest signed prekey (by creation time)
   */
  async getLatestSignedPreKey(): Promise<SignedPreKeyData | null> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.SIGNED_PREKEYS, 'readonly');
      const store = tx.objectStore(STORES.SIGNED_PREKEYS);
      const index = store.index('createdAt');
      const request = index.openCursor(null, 'prev');

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const cursor = request.result;
        if (!cursor) {
          resolve(null);
          return;
        }
        const result = cursor.value;
        resolve({
          keyId: result.keyId,
          publicKey: result.publicKey,
          privateKey: result.privateKey,
          signature: result.signature,
          timestamp: result.createdAt,
        });
      };
    });
  }

  /**
   * Store one-time prekeys
   */
  async storeOneTimePreKeys(preKeys: OneTimePreKeyData[]): Promise<void> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.ONE_TIME_PREKEYS, 'readwrite');
      const store = tx.objectStore(STORES.ONE_TIME_PREKEYS);

      for (const preKey of preKeys) {
        store.put({
          keyId: preKey.keyId,
          publicKey: preKey.publicKey,
          privateKey: preKey.privateKey,
        });
      }

      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
  }

  /**
   * Store a single one-time prekey
   */
  async storeOneTimePreKey(preKey: OneTimePreKeyData): Promise<void> {
    return this.storeOneTimePreKeys([preKey]);
  }

  /**
   * Get and consume (delete) a one-time prekey
   */
  async consumeOneTimePreKey(keyId: number): Promise<OneTimePreKeyData | null> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    // First get the key
    const preKey = await this.getOneTimePreKey(keyId);
    if (!preKey) return null;

    // Then delete it (consume)
    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.ONE_TIME_PREKEYS, 'readwrite');
      const store = tx.objectStore(STORES.ONE_TIME_PREKEYS);
      const request = store.delete(keyId);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(preKey);
    });
  }

  /**
   * Get a one-time prekey by ID
   */
  async getOneTimePreKey(keyId: number): Promise<OneTimePreKeyData | null> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.ONE_TIME_PREKEYS, 'readonly');
      const store = tx.objectStore(STORES.ONE_TIME_PREKEYS);
      const request = store.get(keyId);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const result = request.result;
        if (!result) {
          resolve(null);
          return;
        }
        resolve({
          keyId: result.keyId,
          publicKey: result.publicKey,
          privateKey: result.privateKey,
        });
      };
    });
  }

  /**
   * Count remaining one-time prekeys
   */
  async countOneTimePreKeys(): Promise<number> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.ONE_TIME_PREKEYS, 'readonly');
      const store = tx.objectStore(STORES.ONE_TIME_PREKEYS);
      const request = store.count();

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve(request.result);
    });
  }

  /**
   * Store session state
   */
  async storeSession(session: SessionData): Promise<void> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    const id = `${session.recipientUserId}_${session.recipientDeviceId}`;

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.SESSIONS, 'readwrite');
      const store = tx.objectStore(STORES.SESSIONS);

      const record = {
        id,
        recipientUserId: session.recipientUserId,
        recipientDeviceId: session.recipientDeviceId,
        rootKey: session.rootKey,
        chainKey: session.chainKey,
        messageNumber: session.messageNumber,
        previousChainLength: session.previousChainLength,
        remoteIdentityKey: session.remoteIdentityKey,
        established: session.established,
        createdAt: session.createdAt || Date.now(),
        lastUsed: session.lastUsed || Date.now(),
      };

      const request = store.put(record);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  /**
   * Get session state
   */
  async getSession(recipientUserId: string, recipientDeviceId: string): Promise<SessionData | null> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    const id = `${recipientUserId}_${recipientDeviceId}`;

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.SESSIONS, 'readonly');
      const store = tx.objectStore(STORES.SESSIONS);
      const request = store.get(id);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const result = request.result;
        if (!result) {
          resolve(null);
          return;
        }
        resolve({
          recipientUserId: result.recipientUserId,
          recipientDeviceId: result.recipientDeviceId,
          rootKey: result.rootKey,
          chainKey: result.chainKey,
          messageNumber: result.messageNumber,
          previousChainLength: result.previousChainLength,
          remoteIdentityKey: result.remoteIdentityKey,
          established: result.established,
          createdAt: result.createdAt,
          lastUsed: result.lastUsed,
        });
      };
    });
  }

  /**
   * Delete session
   */
  async deleteSession(recipientUserId: string, recipientDeviceId: string): Promise<void> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    const id = `${recipientUserId}_${recipientDeviceId}`;

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.SESSIONS, 'readwrite');
      const store = tx.objectStore(STORES.SESSIONS);
      const request = store.delete(id);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  /**
   * Store metadata
   */
  async storeMetadata(metadata: DeviceMetadata): Promise<void> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.METADATA, 'readwrite');
      const store = tx.objectStore(STORES.METADATA);

      const record = {
        key: 'device',
        ...metadata,
      };

      const request = store.put(record);
      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  /**
   * Get metadata
   */
  async getMetadata(): Promise<DeviceMetadata | null> {
    if (!this.db) throw new Error('KeyStorage not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction(STORES.METADATA, 'readonly');
      const store = tx.objectStore(STORES.METADATA);
      const request = store.get('device');

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const result = request.result;
        if (!result) {
          resolve(null);
          return;
        }
        resolve({
          deviceId: result.deviceId,
          registrationId: result.registrationId,
          identityKeyHash: result.identityKeyHash,
          createdAt: result.createdAt,
          lastKeyRotation: result.lastKeyRotation,
          signedPreKeyId: result.signedPreKeyId,
        });
      };
    });
  }

  /**
   * Clear all stored data
   */
  async clearAll(): Promise<void> {
    if (!this.db) return;

    const stores = Object.values(STORES);

    for (const storeName of stores) {
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
      this.initialized = false;
    }
  }
}

// Singleton instance
export const keyStorage = new KeyStorage();
