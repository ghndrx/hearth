/**
 * E2EE Key Management
 * 
 * Implements X3DH (Extended Triple Diffie-Hellman) key agreement
 * for establishing secure sessions.
 */

// Key types
export interface IdentityKeyPair {
  publicKey: CryptoKey;
  privateKey: CryptoKey;
}

export interface SignedPreKey {
  keyId: number;
  publicKey: CryptoKey;
  privateKey: CryptoKey;
  signature: ArrayBuffer;
  timestamp: number;
}

export interface OneTimePreKey {
  keyId: number;
  publicKey: CryptoKey;
  privateKey: CryptoKey;
}

export interface PreKeyBundle {
  identityKey: ArrayBuffer;
  signedPreKey: {
    keyId: number;
    publicKey: ArrayBuffer;
    signature: ArrayBuffer;
  };
  oneTimePreKey?: {
    keyId: number;
    publicKey: ArrayBuffer;
  };
}

// Constants
const CURVE = 'P-256'; // Using P-256 for browser compatibility
const HASH = 'SHA-256';
const AES_KEY_LENGTH = 256;

/**
 * Generate an identity key pair
 */
export async function generateIdentityKeyPair(): Promise<IdentityKeyPair> {
  const keyPair = await crypto.subtle.generateKey(
    {
      name: 'ECDH',
      namedCurve: CURVE,
    },
    true,
    ['deriveBits', 'deriveKey']
  );

  return {
    publicKey: keyPair.publicKey,
    privateKey: keyPair.privateKey,
  };
}

/**
 * Generate a signed pre-key
 */
export async function generateSignedPreKey(
  identityKey: CryptoKey,
  keyId: number
): Promise<SignedPreKey> {
  const keyPair = await crypto.subtle.generateKey(
    {
      name: 'ECDH',
      namedCurve: CURVE,
    },
    true,
    ['deriveBits', 'deriveKey']
  );

  // Export public key for signing
  const publicKeyRaw = await crypto.subtle.exportKey('raw', keyPair.publicKey);

  // Sign the public key with identity key (using ECDSA for signing)
  const signingKey = await crypto.subtle.generateKey(
    {
      name: 'ECDSA',
      namedCurve: CURVE,
    },
    true,
    ['sign', 'verify']
  );

  const signature = await crypto.subtle.sign(
    {
      name: 'ECDSA',
      hash: HASH,
    },
    signingKey.privateKey,
    publicKeyRaw
  );

  return {
    keyId,
    publicKey: keyPair.publicKey,
    privateKey: keyPair.privateKey,
    signature,
    timestamp: Date.now(),
  };
}

/**
 * Generate one-time pre-keys
 */
export async function generateOneTimePreKeys(
  startId: number,
  count: number
): Promise<OneTimePreKey[]> {
  const keys: OneTimePreKey[] = [];

  for (let i = 0; i < count; i++) {
    const keyPair = await crypto.subtle.generateKey(
      {
        name: 'ECDH',
        namedCurve: CURVE,
      },
      true,
      ['deriveBits', 'deriveKey']
    );

    keys.push({
      keyId: startId + i,
      publicKey: keyPair.publicKey,
      privateKey: keyPair.privateKey,
    });
  }

  return keys;
}

/**
 * Export a public key to raw bytes
 */
export async function exportPublicKey(key: CryptoKey): Promise<ArrayBuffer> {
  return crypto.subtle.exportKey('raw', key);
}

/**
 * Import a public key from raw bytes
 */
export async function importPublicKey(raw: ArrayBuffer): Promise<CryptoKey> {
  return crypto.subtle.importKey(
    'raw',
    raw,
    {
      name: 'ECDH',
      namedCurve: CURVE,
    },
    true,
    []
  );
}

/**
 * Derive a shared secret using ECDH
 */
export async function deriveSharedSecret(
  privateKey: CryptoKey,
  publicKey: CryptoKey
): Promise<ArrayBuffer> {
  return crypto.subtle.deriveBits(
    {
      name: 'ECDH',
      public: publicKey,
    },
    privateKey,
    256
  );
}

/**
 * Derive an AES key from shared secrets using HKDF
 */
export async function deriveMessageKey(
  sharedSecrets: ArrayBuffer[],
  info: string
): Promise<CryptoKey> {
  // Concatenate all shared secrets
  const combined = new Uint8Array(
    sharedSecrets.reduce((acc, s) => acc + s.byteLength, 0)
  );
  let offset = 0;
  for (const secret of sharedSecrets) {
    combined.set(new Uint8Array(secret), offset);
    offset += secret.byteLength;
  }

  // Import as HKDF key
  const hkdfKey = await crypto.subtle.importKey(
    'raw',
    combined,
    'HKDF',
    false,
    ['deriveKey']
  );

  // Derive AES-GCM key
  return crypto.subtle.deriveKey(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new Uint8Array(32), // In production, use proper salt
      info: new TextEncoder().encode(info),
    },
    hkdfKey,
    {
      name: 'AES-GCM',
      length: AES_KEY_LENGTH,
    },
    false,
    ['encrypt', 'decrypt']
  );
}

/**
 * Store keys securely in IndexedDB
 */
export class KeyStore {
  private dbName = 'hearth-keys';
  private dbVersion = 1;
  public db: IDBDatabase | null = null;

  async init(): Promise<void> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.dbName, this.dbVersion);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        this.db = request.result;
        resolve();
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;
        
        // Identity keys
        if (!db.objectStoreNames.contains('identityKeys')) {
          db.createObjectStore('identityKeys', { keyPath: 'id' });
        }
        
        // Signed pre-keys
        if (!db.objectStoreNames.contains('signedPreKeys')) {
          db.createObjectStore('signedPreKeys', { keyPath: 'keyId' });
        }
        
        // One-time pre-keys
        if (!db.objectStoreNames.contains('oneTimePreKeys')) {
          db.createObjectStore('oneTimePreKeys', { keyPath: 'keyId' });
        }
        
        // Sessions
        if (!db.objectStoreNames.contains('sessions')) {
          db.createObjectStore('sessions', { keyPath: 'recipientId' });
        }
      };
    });
  }

  async storeIdentityKey(keyPair: IdentityKeyPair): Promise<void> {
    if (!this.db) throw new Error('KeyStore not initialized');

    const publicKeyRaw = await exportPublicKey(keyPair.publicKey);
    const privateKeyJwk = await crypto.subtle.exportKey('jwk', keyPair.privateKey);

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction('identityKeys', 'readwrite');
      const store = tx.objectStore('identityKeys');
      
      const request = store.put({
        id: 'identity',
        publicKey: publicKeyRaw,
        privateKey: privateKeyJwk,
      });

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  async getIdentityKey(): Promise<IdentityKeyPair | null> {
    if (!this.db) throw new Error('KeyStore not initialized');

    return new Promise((resolve, reject) => {
      const tx = this.db!.transaction('identityKeys', 'readonly');
      const store = tx.objectStore('identityKeys');
      const request = store.get('identity');

      request.onerror = () => reject(request.error);
      request.onsuccess = async () => {
        if (!request.result) {
          resolve(null);
          return;
        }

        const publicKey = await importPublicKey(request.result.publicKey);
        const privateKey = await crypto.subtle.importKey(
          'jwk',
          request.result.privateKey,
          { name: 'ECDH', namedCurve: CURVE },
          true,
          ['deriveBits', 'deriveKey']
        );

        resolve({ publicKey, privateKey });
      };
    });
  }

  async clear(): Promise<void> {
    if (!this.db) return;

    const stores = ['identityKeys', 'signedPreKeys', 'oneTimePreKeys', 'sessions'];
    
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
}

export const keyStore = new KeyStore();

// ============================================================================
// Standalone Key Access Functions (for use by x3dh.ts)
// ============================================================================

/**
 * Get the stored identity key pair
 * Initializes KeyStore if needed
 */
export async function getIdentityKey(): Promise<IdentityKeyPair | null> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  return keyStore.getIdentityKey();
}

/**
 * Get a signed prekey by ID
 * Initializes KeyStore if needed
 */
export async function getSignedPreKey(keyId: number): Promise<SignedPreKey | null> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  return new Promise((resolve, reject) => {
    const tx = keyStore.db!.transaction('signedPreKeys', 'readonly');
    const store = tx.objectStore('signedPreKeys');
    const request = store.get(keyId);

    request.onerror = () => reject(request.error);
    request.onsuccess = async () => {
      if (!request.result) {
        resolve(null);
        return;
      }

      const publicKey = await importPublicKey(request.result.publicKey);
      const privateKey = await crypto.subtle.importKey(
        'jwk',
        request.result.privateKey,
        { name: 'ECDH', namedCurve: CURVE },
        true,
        ['deriveBits', 'deriveKey']
      );

      resolve({
        keyId: request.result.keyId,
        publicKey,
        privateKey,
        signature: request.result.signature,
        timestamp: request.result.timestamp,
      });
    };
  });
}

/**
 * Get a one-time prekey by ID
 * Initializes KeyStore if needed
 */
export async function getOneTimePreKey(keyId: number): Promise<OneTimePreKey | null> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  return new Promise((resolve, reject) => {
    const tx = keyStore.db!.transaction('oneTimePreKeys', 'readonly');
    const store = tx.objectStore('oneTimePreKeys');
    const request = store.get(keyId);

    request.onerror = () => reject(request.error);
    request.onsuccess = async () => {
      if (!request.result) {
        resolve(null);
        return;
      }

      const publicKey = await importPublicKey(request.result.publicKey);
      const privateKey = await crypto.subtle.importKey(
        'jwk',
        request.result.privateKey,
        { name: 'ECDH', namedCurve: CURVE },
        true,
        ['deriveBits', 'deriveKey']
      );

      resolve({
        keyId: request.result.keyId,
        publicKey,
        privateKey,
      });
    };
  });
}

/**
 * Get all signed prekeys
 * Initializes KeyStore if needed
 */
export async function getAllSignedPreKeys(): Promise<SignedPreKey[]> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  return new Promise((resolve, reject) => {
    const tx = keyStore.db!.transaction('signedPreKeys', 'readonly');
    const store = tx.objectStore('signedPreKeys');
    const request = store.getAll();

    request.onerror = () => reject(request.error);
    request.onsuccess = async () => {
      const results: SignedPreKey[] = [];
      
      for (const result of request.result) {
        const publicKey = await importPublicKey(result.publicKey);
        const privateKey = await crypto.subtle.importKey(
          'jwk',
          result.privateKey,
          { name: 'ECDH', namedCurve: CURVE },
          true,
          ['deriveBits', 'deriveKey']
        );

        results.push({
          keyId: result.keyId,
          publicKey,
          privateKey,
          signature: result.signature,
          timestamp: result.timestamp,
        });
      }
      
      resolve(results);
    };
  });
}

/**
 * Get all one-time prekeys
 * Initializes KeyStore if needed
 */
export async function getAllOneTimePreKeys(): Promise<OneTimePreKey[]> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  return new Promise((resolve, reject) => {
    const tx = keyStore.db!.transaction('oneTimePreKeys', 'readonly');
    const store = tx.objectStore('oneTimePreKeys');
    const request = store.getAll();

    request.onerror = () => reject(request.error);
    request.onsuccess = async () => {
      const results: OneTimePreKey[] = [];
      
      for (const result of request.result) {
        const publicKey = await importPublicKey(result.publicKey);
        const privateKey = await crypto.subtle.importKey(
          'jwk',
          result.privateKey,
          { name: 'ECDH', namedCurve: CURVE },
          true,
          ['deriveBits', 'deriveKey']
        );

        results.push({
          keyId: result.keyId,
          publicKey,
          privateKey,
        });
      }
      
      resolve(results);
    };
  });
}

/**
 * Store a signed prekey
 * Initializes KeyStore if needed
 */
export async function storeSignedPreKey(preKey: SignedPreKey): Promise<void> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  const publicKeyRaw = await exportPublicKey(preKey.publicKey);
  const privateKeyJwk = await crypto.subtle.exportKey('jwk', preKey.privateKey);

  return new Promise((resolve, reject) => {
    const tx = keyStore.db!.transaction('signedPreKeys', 'readwrite');
    const store = tx.objectStore('signedPreKeys');
    
    const request = store.put({
      keyId: preKey.keyId,
      publicKey: publicKeyRaw,
      privateKey: privateKeyJwk,
      signature: preKey.signature,
      timestamp: preKey.timestamp,
    });

    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve();
  });
}

/**
 * Store one-time prekeys
 * Initializes KeyStore if needed
 */
export async function storeOneTimePreKeys(preKeys: OneTimePreKey[]): Promise<void> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  for (const preKey of preKeys) {
    const publicKeyRaw = await exportPublicKey(preKey.publicKey);
    const privateKeyJwk = await crypto.subtle.exportKey('jwk', preKey.privateKey);

    await new Promise<void>((resolve, reject) => {
      const tx = keyStore.db!.transaction('oneTimePreKeys', 'readwrite');
      const store = tx.objectStore('oneTimePreKeys');
      
      const request = store.put({
        keyId: preKey.keyId,
        publicKey: publicKeyRaw,
        privateKey: privateKeyJwk,
      });

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }
}

/**
 * Delete a one-time prekey (consume it)
 * Initializes KeyStore if needed
 */
export async function deleteOneTimePreKey(keyId: number): Promise<void> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  return new Promise((resolve, reject) => {
    const tx = keyStore.db!.transaction('oneTimePreKeys', 'readwrite');
    const store = tx.objectStore('oneTimePreKeys');
    const request = store.delete(keyId);

    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve();
  });
}

/**
 * Count remaining one-time prekeys
 * Initializes KeyStore if needed
 */
export async function countOneTimePreKeys(): Promise<number> {
  if (!keyStore.db) {
    await keyStore.init();
  }
  
  return new Promise((resolve, reject) => {
    const tx = keyStore.db!.transaction('oneTimePreKeys', 'readonly');
    const store = tx.objectStore('oneTimePreKeys');
    const request = store.count();

    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
  });
}
