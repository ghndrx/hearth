/**
 * Double Ratchet Protocol Implementation
 * 
 * Implements the Double Ratchet algorithm for message encryption,
 * providing forward secrecy and break-in recovery.
 * 
 * The Double Ratchet combines:
 * - Diffie-Hellman Ratchet: Provides break-in recovery via asymmetric ratcheting
 * - Symmetric Key Ratchet: Provides forward secrecy via symmetric key updates
 * 
 * Each message uses a unique message key derived from the chain key,
 * which is advanced after each message.
 * 
 * This implementation follows the Signal Protocol specification
 * with HKDF-based key derivation.
 */

const HASH = 'SHA-256';
const MESSAGE_KEY_LENGTH = 32;
const CHAIN_KEY_LENGTH = 32;

/**
 * Ratchet state for a session
 */
export interface RatchetState {
  // DH ratchet state
  dhRatchet: {
    dhKeyPair: {
      publicKey: ArrayBuffer;
      privateKey: ArrayBuffer;
    };
    remoteDHPublicKey: ArrayBuffer | null;
  };
  
  // Root key (used to derive chain keys)
  rootKey: ArrayBuffer;
  
  // Chain keys (one per sending/receiving chain)
  chainKeys: {
    sending: ArrayBuffer | null;
    receiving: ArrayBuffer | null;
  };
  
  // Message numbers
  messageNumbers: {
    sending: number;
    receiving: number;
  };
  
  // Previous chain length (for DH ratchet)
  previousChainLength: number;
  
  // Remote identity key (for verification)
  remoteIdentityKey: ArrayBuffer | null;
  
  // Session metadata
  recipientUserId: string;
  recipientDeviceId: string;
  createdAt: number;
  lastUsed: number;
}

/**
 * Encrypted message with metadata
 */
export interface RatchetMessage {
  // Public fields (needed for ratchet processing)
  publicKey: string;        // Sender's current DH public key (Base64)
  previousChainKey: string; // Previous chain's last message key hash (Base64)
  messageNumber: number;
  
  // Encrypted payload
  ciphertext: string;       // Base64 encrypted message
  iv: string;              // Base64 IV
  
  // Metadata
  senderIdentityKey: string; // Sender's identity key (Base64)
}

/**
 * Ratchet session for encrypting/decrypting messages
 */
export class RatchetSession {
  private state: RatchetState;
  private initialized = false;

  /**
   * Create a new ratchet session (initiator side)
   */
  static async createAsInitiator(
    sharedSecret: ArrayBuffer,
    recipientUserId: string,
    recipientDeviceId: string,
    remoteIdentityKey: ArrayBuffer
  ): Promise<RatchetSession> {
    // Generate our DH key pair for the initial ratchet step
    const dhKeyPair = await crypto.subtle.generateKey(
      { name: 'ECDH', namedCurve: 'P-256' },
      true,
      ['deriveBits', 'deriveKey']
    );

    const publicKeyRaw = await crypto.subtle.exportKey('raw', dhKeyPair.publicKey);
    const privateKeyJwk = await crypto.subtle.exportKey('jwk', dhKeyPair.privateKey);

    // Derive initial root key and chain keys from shared secret
    const { rootKey, chainKey } = await deriveRootAndChainKeys(sharedSecret, publicKeyRaw);

    const state: RatchetState = {
      dhRatchet: {
        dhKeyPair: {
          publicKey: publicKeyRaw,
          privateKey: new TextEncoder().encode(JSON.stringify(privateKeyJwk)).buffer,
        },
        remoteDHPublicKey: null,
      },
      rootKey,
      chainKeys: {
        sending: chainKey,
        receiving: null,
      },
      messageNumbers: {
        sending: 0,
        receiving: 0,
      },
      previousChainLength: 0,
      remoteIdentityKey,
      recipientUserId,
      recipientDeviceId,
      createdAt: Date.now(),
      lastUsed: Date.now(),
    };

    const session = new RatchetSession(state);
    return session;
  }

  /**
   * Create a session from stored state (recipient side)
   */
  static async createFromState(state: RatchetState): Promise<RatchetSession> {
    const session = new RatchetSession(state);
    return session;
  }

  private constructor(state: RatchetState) {
    this.state = state;
    this.initialized = true;
  }

  /**
   * Get current session state for storage
   */
  getState(): RatchetState {
    return { ...this.state };
  }

  /**
   * Get the current message number for sending
   */
  getMessageNumber(): number {
    return this.state.messageNumbers.sending;
  }

  /**
   * Get the remote party's current public key
   */
  getRemotePublicKey(): ArrayBuffer | null {
    return this.state.dhRatchet.remoteDHPublicKey;
  }

  /**
   * Encrypt a message
   */
  async encrypt(plaintext: string): Promise<RatchetMessage> {
    if (!this.initialized) {
      throw new Error('Ratchet session not initialized');
    }

    // Derive message key from sending chain
    const { messageKey, chainKey } = await deriveMessageKey(this.state.chainKeys.sending!);
    this.state.chainKeys.sending = chainKey;

    // Export our current DH public key
    const dhKeyPair = this.state.dhRatchet.dhKeyPair;
    const publicKeyBase64 = arrayBufferToBase64(dhKeyPair.publicKey);

    // The identity key is the same as the DH key in this simplified implementation
    const identityKeyBase64 = publicKeyBase64;

    // Generate IV and encrypt
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const plaintextBytes = new TextEncoder().encode(plaintext);

    const ciphertext = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv },
      messageKey,
      plaintextBytes
    );

    // Update message number
    const messageNumber = this.state.messageNumbers.sending;
    this.state.messageNumbers.sending++;
    this.state.lastUsed = Date.now();

    // Calculate hash of previous chain key for continuity
    const previousChainKeyHash = await hashArrayBuffer(this.state.chainKeys.sending!);

    return {
      publicKey: publicKeyBase64,
      previousChainKey: arrayBufferToBase64(previousChainKeyHash),
      messageNumber,
      ciphertext: arrayBufferToBase64(ciphertext),
      iv: arrayBufferToBase64(iv.buffer),
      senderIdentityKey: identityKeyBase64,
    };
  }

  /**
   * Decrypt a message
   */
  async decrypt(message: RatchetMessage): Promise<string> {
    if (!this.initialized) {
      throw new Error('Ratchet session not initialized');
    }

    const remotePublicKey = base64ToArrayBuffer(message.publicKey);
    const senderIdentityKey = base64ToArrayBuffer(message.senderIdentityKey);
    const ciphertext = base64ToArrayBuffer(message.ciphertext);
    const iv = base64ToArrayBuffer(message.iv);

    // Check if this is a new DH ratchet step
    if (this.state.dhRatchet.remoteDHPublicKey === null ||
        arrayBufferEqual(remotePublicKey, this.state.dhRatchet.remoteDHPublicKey)) {
      // Same chain - symmetric key ratchet
      await this.symmetricRatchet(remotePublicKey, senderIdentityKey);
    } else {
      // New chain - DH ratchet
      await this.dhRatchet(remotePublicKey, senderIdentityKey);
    }

    // Verify message number
    if (message.messageNumber < this.state.messageNumbers.receiving) {
      throw new Error('Message number too low - possible replay attack');
    }

    // Skip ahead if needed (message number gap)
    while (this.state.messageNumbers.receiving < message.messageNumber) {
      const { chainKey } = await deriveMessageKey(this.state.chainKeys.receiving!);
      this.state.chainKeys.receiving = chainKey;
      this.state.messageNumbers.receiving++;
    }

    // Derive message key and advance chain
    const { messageKey, chainKey } = await deriveMessageKey(this.state.chainKeys.receiving!);
    this.state.chainKeys.receiving = chainKey;
    this.state.messageNumbers.receiving++;
    this.state.lastUsed = Date.now();

    // Decrypt
    const plaintextBytes = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv },
      messageKey,
      ciphertext
    );

    return new TextDecoder().decode(plaintextBytes);
  }

  /**
   * Perform symmetric key ratchet step
   * Called when receiving a message with the same DH public key
   */
  private async symmetricRatchet(
    remotePublicKey: ArrayBuffer,
    remoteIdentityKey: ArrayBuffer
  ): Promise<void> {
    // Update remote identity if provided
    if (remoteIdentityKey) {
      this.state.remoteIdentityKey = remoteIdentityKey;
    }

    // Skip receiving chain ahead to the message number
    while (this.state.messageNumbers.receiving < this.state.messageNumbers.sending) {
      const { chainKey } = await deriveMessageKey(this.state.chainKeys.receiving!);
      this.state.chainKeys.receiving = chainKey;
      this.state.messageNumbers.receiving++;
    }
  }

  /**
   * Perform DH ratchet step
   * Called when receiving a message with a new DH public key
   */
  private async dhRatchet(
    remotePublicKey: ArrayBuffer,
    remoteIdentityKey: ArrayBuffer
  ): Promise<void> {
    // Store previous chain length
    this.state.previousChainLength = this.state.messageNumbers.sending;

    // Update remote public key
    this.state.dhRatchet.remoteDHPublicKey = remotePublicKey;
    this.state.remoteIdentityKey = remoteIdentityKey;

    // Reset receiving chain
    this.state.messageNumbers.receiving = 0;

    // Perform DH with remote key
    const dhKeyPair = this.state.dhRatchet.dhKeyPair;
    const privateKeyJwk = JSON.parse(new TextDecoder().decode(new Uint8Array(dhKeyPair.privateKey)));
    const privateKey = await crypto.subtle.importKey(
      'jwk',
      privateKeyJwk,
      { name: 'ECDH', namedCurve: 'P-256' },
      true,
      ['deriveBits', 'deriveKey']
    );

    const sharedSecret = await crypto.subtle.deriveBits(
      { name: 'ECDH', public: await crypto.subtle.importKey(
        'raw',
        remotePublicKey,
        { name: 'ECDH', namedCurve: 'P-256' },
        false,
        []
      )},
      privateKey,
      256
    );

    // Derive new root key and receiving chain key
    const { rootKey, chainKey } = await dhRatchetRootKey(this.state.rootKey, sharedSecret);
    this.state.rootKey = rootKey;
    this.state.chainKeys.receiving = chainKey;

    // Generate new DH key pair and derive sending chain key
    const newDhKeyPair = await crypto.subtle.generateKey(
      { name: 'ECDH', namedCurve: 'P-256' },
      true,
      ['deriveBits', 'deriveKey']
    );

    const newPublicKeyRaw = await crypto.subtle.exportKey('raw', newDhKeyPair.publicKey);
    const newPrivateKeyJwk = await crypto.subtle.exportKey('jwk', newDhKeyPair.privateKey);

    // Derive sending chain key
    const sendingSharedSecret = await crypto.subtle.deriveBits(
      { name: 'ECDH', public: await crypto.subtle.importKey(
        'raw',
        remotePublicKey,
        { name: 'ECDH', namedCurve: 'P-256' },
        false,
        []
      )},
      newDhKeyPair.privateKey,
      256
    );

    const { rootKey: newRootKey, chainKey: sendingChainKey } = await dhRatchetRootKey(
      this.state.rootKey,
      sendingSharedSecret
    );

    this.state.rootKey = newRootKey;
    this.state.chainKeys.sending = sendingChainKey;
    this.state.dhRatchet.dhKeyPair = {
      publicKey: newPublicKeyRaw,
      privateKey: new TextEncoder().encode(JSON.stringify(newPrivateKeyJwk)).buffer,
    };
    this.state.messageNumbers.sending = 0;
  }
}

/**
 * Derive root key and chain key from shared secret
 */
async function deriveRootAndChainKeys(
  sharedSecret: ArrayBuffer,
  publicKey: ArrayBuffer
): Promise<{ rootKey: ArrayBuffer; chainKey: ArrayBuffer }> {
  const inputKey = concatArrayBuffers([sharedSecret, publicKey]);
  
  const hkdfKey = await crypto.subtle.importKey(
    'raw',
    inputKey,
    'HKDF',
    false,
    ['deriveKey']
  );

  const derived = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new Uint8Array(32),
      info: new TextEncoder().encode('HearthRatchet'),
    },
    hkdfKey,
    64 * 8 // 64 bytes = 32 for root key + 32 for chain key
  );

  return {
    rootKey: derived.slice(0, 32),
    chainKey: derived.slice(32, 64),
  };
}

/**
 * Derive message key from chain key
 */
async function deriveMessageKey(chainKey: ArrayBuffer): Promise<{
  messageKey: CryptoKey;
  chainKey: ArrayBuffer;
}> {
  const hkdfKey = await crypto.subtle.importKey(
    'raw',
    chainKey,
    'HKDF',
    false,
    ['deriveKey']
  );

  // Derive next chain key
  const nextChainKey = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new Uint8Array(32),
      info: new TextEncoder().encode('HearthChainKey'),
    },
    hkdfKey,
    CHAIN_KEY_LENGTH * 8
  );

  // Derive message key
  const messageKeyRaw = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new Uint8Array(32),
      info: new TextEncoder().encode('HearthMessageKey'),
    },
    hkdfKey,
    MESSAGE_KEY_LENGTH * 8
  );

  const messageKey = await crypto.subtle.importKey(
    'raw',
    messageKeyRaw,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );

  return { messageKey, chainKey: nextChainKey };
}

/**
 * DH ratchet root key derivation
 */
async function dhRatchetRootKey(
  rootKey: ArrayBuffer,
  sharedSecret: ArrayBuffer
): Promise<{ rootKey: ArrayBuffer; chainKey: ArrayBuffer }> {
  const inputKey = concatArrayBuffers([rootKey, sharedSecret]);
  
  const hkdfKey = await crypto.subtle.importKey(
    'raw',
    inputKey,
    'HKDF',
    false,
    ['deriveKey']
  );

  const derived = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: HASH,
      salt: new Uint8Array(32),
      info: new TextEncoder().encode('HearthDHRatchet'),
    },
    hkdfKey,
    64 * 8
  );

  return {
    rootKey: derived.slice(0, 32),
    chainKey: derived.slice(32, 64),
  };
}

/**
 * Compare two ArrayBuffers for equality
 */
function arrayBufferEqual(a: ArrayBuffer, b: ArrayBuffer): boolean {
  if (a.byteLength !== b.byteLength) return false;
  const aBytes = new Uint8Array(a);
  const bBytes = new Uint8Array(b);
  for (let i = 0; i < aBytes.length; i++) {
    if (aBytes[i] !== bBytes[i]) return false;
  }
  return true;
}

/**
 * Hash an ArrayBuffer with SHA-256
 */
async function hashArrayBuffer(buffer: ArrayBuffer): Promise<ArrayBuffer> {
  return crypto.subtle.digest('SHA-256', buffer);
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
  return new Uint8Array(bytes).buffer.slice(0);
}

/**
 * Concatenate ArrayBuffers
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
 * Session manager for handling multiple ratchet sessions
 */
export class RatchetSessionManager {
  private sessions: Map<string, RatchetSession> = new Map();

  /**
   * Get or create a session for a recipient
   */
  async getOrCreateSession(
    recipientUserId: string,
    recipientDeviceId: string,
    sharedSecret: ArrayBuffer,
    remoteIdentityKey: ArrayBuffer,
    existingState?: RatchetState
  ): Promise<RatchetSession> {
    const key = `${recipientUserId}_${recipientDeviceId}`;
    
    if (existingState) {
      const session = await RatchetSession.createFromState(existingState);
      this.sessions.set(key, session);
      return session;
    }

    if (this.sessions.has(key)) {
      return this.sessions.get(key)!;
    }

    const session = await RatchetSession.createAsInitiator(
      sharedSecret,
      recipientUserId,
      recipientDeviceId,
      remoteIdentityKey
    );
    this.sessions.set(key, session);
    return session;
  }

  /**
   * Get an existing session
   */
  getSession(recipientUserId: string, recipientDeviceId: string): RatchetSession | undefined {
    return this.sessions.get(`${recipientUserId}_${recipientDeviceId}`);
  }

  /**
   * Remove a session
   */
  removeSession(recipientUserId: string, recipientDeviceId: string): void {
    this.sessions.delete(`${recipientUserId}_${recipientDeviceId}`);
  }

  /**
   * Get all session states for persistence
   */
  getAllSessionStates(): Map<string, RatchetState> {
    const states = new Map<string, RatchetState>();
    this.sessions.forEach((session, key) => {
      states.set(key, session.getState());
    });
    return states;
  }
}

// Singleton session manager
export const sessionManager = new RatchetSessionManager();
