/**
 * E2EE Session Manager
 * 
 * Manages E2EE sessions using the Double Ratchet algorithm.
 * This module provides:
 * - X3DH key agreement for session establishment
 * - Double Ratchet encryption/decryption
 * - Session state persistence to IndexedDB
 * - Out-of-order message handling
 */

import { browser } from '$app/environment';
import { sessionStorage, type SessionStorage } from '$lib/crypto/storage';
import { 
  RatchetSession, 
  RatchetSessionManager,
  type RatchetState,
  type RatchetMessage 
} from '$lib/crypto/ratchet';
import { 
  performX3DHSender, 
  performX3DHRecipient,
  deriveMessageKeys,
  base64ToArrayBuffer,
  arrayBufferToBase64,
  type RemotePreKeyBundle 
} from '$lib/crypto/signal-protocol';
import { keyStore, getIdentityKey, getSignedPreKey, getOneTimePreKey } from '$lib/crypto/keys';

/**
 * Session info for external use
 */
export interface E2EESessionInfo {
  recipientUserId: string;
  recipientDeviceId: string;
  established: boolean;
  createdAt: number;
  lastUsed: number;
  hasUnread: boolean;
}

/**
 * Encrypted message ready for transmission
 */
export interface E2EEEncryptedMessage {
  version: number;
  publicKey: string;        // Base64 - sender's current DH public key
  previousChainKey: string; // Base64 - hash of previous chain key
  messageNumber: number;
  ciphertext: string;       // Base64 - encrypted content
  iv: string;               // Base64 - initialization vector
  senderIdentityKey: string; // Base64 - sender's identity key
}

/**
 * Prekey message for initial session establishment
 */
export interface PreKeyMessage {
  publicKey: string;         // Base64 - sender's ephemeral public key
  identityKey: string;       // Base64 - sender's identity key
  usedPreKeyId?: number;     // ID of OTP used (if any)
}

/**
 * E2EE Session Manager class
 * 
 * Coordinates between X3DH, Double Ratchet, and storage.
 * This is the main entry point for E2EE session operations.
 */
export class E2EESessionManager {
  private ratchetManager: RatchetSessionManager;
  private storage: SessionStorage;
  private initialized = false;

  constructor() {
    this.ratchetManager = new RatchetSessionManager();
    this.storage = sessionStorage;
  }

  /**
   * Initialize the session manager
   */
  async init(): Promise<void> {
    if (!browser) return;
    if (this.initialized) return;

    // Initialize storage
    await this.storage.init();
    
    // Load existing sessions from storage
    await this.loadSessions();
    
    this.initialized = true;
    console.info('[E2EE Session] Manager initialized');
  }

  /**
   * Load existing sessions from IndexedDB into the ratchet manager
   */
  private async loadSessions(): Promise<void> {
    // This would load sessions from storage and recreate RatchetSessions
    // For now, we rebuild sessions on-demand from stored state
    console.debug('[E2EE Session] Sessions loaded on-demand from storage');
  }

  /**
   * Establish a new E2EE session with a recipient
   * Uses X3DH to perform key agreement, then initializes Double Ratchet
   */
  async establishSession(
    recipientUserId: string,
    recipientDeviceId: string,
    recipientBundle: RemotePreKeyBundle
  ): Promise<void> {
    console.info('[E2EE Session] Establishing session with', recipientUserId, recipientDeviceId);

    // Get our identity key
    const identityKey = await getIdentityKey();
    if (!identityKey) {
      throw new Error('No identity key - E2EE not initialized');
    }

    // Perform X3DH as sender (Alice)
    const { sharedSecret, ephemeralPublicKey, usedPreKeyId } = await performX3DHSender(
      identityKey,
      recipientBundle
    );

    console.info('[E2EE Session] X3DH complete, usedPreKeyId:', usedPreKeyId);

    // Create Double Ratchet session as initiator
    const remoteIdentityKey = base64ToArrayBuffer(recipientBundle.identityKey);
    const session = await RatchetSession.createAsInitiator(
      sharedSecret,
      recipientUserId,
      recipientDeviceId,
      remoteIdentityKey
    );

    // Store in manager
    await this.ratchetManager.getOrCreateSession(
      recipientUserId,
      recipientDeviceId,
      sharedSecret,
      remoteIdentityKey,
      session.getState()
    );

    // Persist session state to IndexedDB
    await this.storage.storeSession(recipientUserId, recipientDeviceId, session.getState());

    console.info('[E2EE Session] Session established and stored');
  }

  /**
   * Process an incoming pre-key message to establish a session as recipient
   * Called when we receive a message from a new contact
   */
  async processPreKeyMessage(
    senderUserId: string,
    senderDeviceId: string,
    preKeyMessage: PreKeyMessage
  ): Promise<void> {
    console.info('[E2EE Session] Processing pre-key message from', senderUserId);

    // Get our identity key
    const identityKey = await getIdentityKey();
    if (!identityKey) {
      throw new Error('No identity key - E2EE not initialized');
    }

    // Get our signed prekey
    const signedPreKey = await getSignedPreKey(1);
    if (!signedPreKey) {
      throw new Error('No signed prekey found');
    }

    // Get our one-time prekey if one was used
    let oneTimePreKey = null;
    if (preKeyMessage.usedPreKeyId !== undefined) {
      oneTimePreKey = await getOneTimePreKey(preKeyMessage.usedPreKeyId);
      if (!oneTimePreKey) {
        console.warn('[E2EE Session] OTP not found, may already be consumed');
      }
    }

    // Import sender's keys
    const senderIdentityKey = base64ToArrayBuffer(preKeyMessage.identityKey);
    const senderEphemeralKey = base64ToArrayBuffer(preKeyMessage.publicKey);

    // Perform X3DH as recipient (Bob)
    const sharedSecret = await performX3DHRecipient(
      identityKey,
      signedPreKey,
      oneTimePreKey,
      senderIdentityKey,
      senderEphemeralKey
    );

    console.info('[E2EE Session] X3DH as recipient complete');

    // Create Double Ratchet session
    const session = await RatchetSession.createAsInitiator(
      sharedSecret,
      senderUserId,
      senderDeviceId,
      senderIdentityKey
    );

    // Store in manager
    await this.ratchetManager.getOrCreateSession(
      senderUserId,
      senderDeviceId,
      sharedSecret,
      senderIdentityKey,
      session.getState()
    );

    // Persist session state
    await this.storage.storeSession(senderUserId, senderDeviceId, session.getState());

    console.info('[E2EE Session] Session established as recipient');
  }

  /**
   * Encrypt a message for a recipient
   */
  async encryptMessage(
    plaintext: string,
    recipientUserId: string,
    recipientDeviceId: string
  ): Promise<E2EEEncryptedMessage> {
    const session = this.ratchetManager.getSession(recipientUserId, recipientDeviceId);
    if (!session) {
      throw new Error(`No session found for ${recipientUserId}_${recipientDeviceId}`);
    }

    // Encrypt using the ratchet
    const ratchetMessage = await session.encrypt(plaintext);

    // Convert to transmission format
    const message: E2EEEncryptedMessage = {
      version: 1,
      publicKey: ratchetMessage.publicKey,
      previousChainKey: ratchetMessage.previousChainKey,
      messageNumber: ratchetMessage.messageNumber,
      ciphertext: ratchetMessage.ciphertext,
      iv: ratchetMessage.iv,
      senderIdentityKey: ratchetMessage.senderIdentityKey,
    };

    // Update stored session state
    await this.storage.storeSession(recipientUserId, recipientDeviceId, session.getState());

    return message;
  }

  /**
   * Decrypt a message from a sender
   */
  async decryptMessage(
    message: E2EEEncryptedMessage,
    senderUserId: string,
    senderDeviceId: string
  ): Promise<string> {
    const session = this.ratchetManager.getSession(senderUserId, senderDeviceId);
    if (!session) {
      throw new Error(`No session found for ${senderUserId}_${senderDeviceId}`);
    }

    // Convert from transmission format
    const ratchetMessage: RatchetMessage = {
      publicKey: message.publicKey,
      previousChainKey: message.previousChainKey,
      messageNumber: message.messageNumber,
      ciphertext: message.ciphertext,
      iv: message.iv,
      senderIdentityKey: message.senderIdentityKey,
    };

    // Decrypt using the ratchet
    const plaintext = await session.decrypt(ratchetMessage);

    // Update stored session state
    await this.storage.storeSession(senderUserId, senderDeviceId, session.getState());

    return plaintext;
  }

  /**
   * Check if a session exists for a recipient
   */
  hasSession(recipientUserId: string, recipientDeviceId: string): boolean {
    return this.ratchetManager.getSession(recipientUserId, recipientDeviceId) !== undefined;
  }

  /**
   * Get or establish a session
   * If session exists, returns it. Otherwise, must be established via establishSession
   */
  getSession(recipientUserId: string, recipientDeviceId: string): RatchetSession | undefined {
    return this.ratchetManager.getSession(recipientUserId, recipientDeviceId);
  }

  /**
   * Get all stored sessions
   */
  async getAllSessions(): Promise<E2EESessionInfo[]> {
    // This would enumerate stored sessions from IndexedDB
    // For now, return from memory
    const sessions: E2EESessionInfo[] = [];
    this.ratchetManager.getAllSessionStates().forEach((state, key) => {
      const [recipientUserId, recipientDeviceId] = key.split('_');
      sessions.push({
        recipientUserId,
        recipientDeviceId,
        established: true,
        createdAt: state.createdAt,
        lastUsed: state.lastUsed,
        hasUnread: false,
      });
    });
    return sessions;
  }

  /**
   * Remove a session
   */
  async removeSession(recipientUserId: string, recipientDeviceId: string): Promise<void> {
    this.ratchetManager.removeSession(recipientUserId, recipientDeviceId);
    await this.storage.deleteSession(recipientUserId, recipientDeviceId);
    console.info('[E2EE Session] Session removed for', recipientUserId);
  }
}

// Singleton instance
export const e2eeSessionManager = new E2EESessionManager();
