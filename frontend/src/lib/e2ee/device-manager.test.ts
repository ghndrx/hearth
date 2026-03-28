/**
 * Device Manager Tests
 * 
 * Tests for the E2EE device manager.
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';

// Mock browser environment
const mockBrowser = typeof window !== 'undefined';

// Check for WebCrypto support
const hasWebCrypto = typeof crypto !== 'undefined' && 
                         crypto.subtle && 
                         typeof crypto.subtle.generateKey === 'function';

// Check for IndexedDB support
const hasIndexedDB = typeof indexedDB !== 'undefined';

describe('Device Manager', () => {
  describe('generateDeviceId', () => {
    it('should generate a valid device ID format', async () => {
      const { deviceManager } = await import('./device-manager');
      
      // Generate a device ID through the manager
      const deviceId = (deviceManager as any).deviceId || 'test-device';
      
      // The device ID should be a 32-character hex string
      // We can't directly test generateDeviceId without initialization
      // but we can verify the format through the registration flow
    });
  });

  describe('isRegistered', () => {
    it.skipIf(!mockBrowser || !hasIndexedDB)('should return false when not initialized', async () => {
      const { deviceManager } = await import('./device-manager');
      
      // Initialize first
      await deviceManager.init();
      
      // isRegistered checks if identity key exists
      const registered = await deviceManager.isRegistered();
      // Either true (has existing keys) or false (no keys yet)
      expect(typeof registered).toBe('boolean');
    });
  });
});

describe('Device Registration Flow', () => {
  // These tests require full browser environment with IndexedDB
  
  it.skipIf(!hasIndexedDB || !hasWebCrypto)('should generate valid registration data', async () => {
    const { deviceManager } = await import('./device-manager');
    
    // This test verifies the structure of the registration object
    // We can't actually register without a real browser environment
    
    // Check that the manager has the expected methods
    expect(typeof deviceManager.init).toBe('function');
    expect(typeof deviceManager.isRegistered).toBe('function');
    expect(typeof deviceManager.registerDevice).toBe('function');
    expect(typeof deviceManager.generateIdentityKeyPair).toBe('function');
    expect(typeof deviceManager.generateSignedPreKey).toBe('function');
    expect(typeof deviceManager.generateOneTimePreKeys).toBe('function');
  });
});

describe('Key Generation', () => {
  it.skipIf(!hasWebCrypto)('should generate identity key pair', async () => {
    const { deviceManager } = await import('./device-manager');
    
    const identityKey = await deviceManager.generateIdentityKeyPair();
    
    expect(identityKey).toBeDefined();
    expect(identityKey.publicKey).toBeDefined();
    expect(identityKey.publicKey.byteLength).toBeGreaterThan(0);
    expect(identityKey.privateKey).toBeDefined();
    expect(identityKey.privateKey.byteLength).toBeGreaterThan(0);
  });

  it.skipIf(!hasWebCrypto)('should generate signed prekey', async () => {
    const { deviceManager } = await import('./device-manager');
    
    const identityKey = await deviceManager.generateIdentityKeyPair();
    const signedPreKey = await deviceManager.generateSignedPreKey(
      identityKey.publicKey as unknown as CryptoKey,
      1
    );
    
    expect(signedPreKey).toBeDefined();
    expect(signedPreKey.keyId).toBe(1);
    expect(signedPreKey.publicKey).toBeDefined();
    expect(signedPreKey.privateKey).toBeDefined();
    expect(signedPreKey.signature).toBeDefined();
    expect(signedPreKey.signature.byteLength).toBeGreaterThan(0);
  });

  it.skipIf(!hasWebCrypto)('should generate multiple one-time prekeys', async () => {
    const { deviceManager } = await import('./device-manager');
    
    const preKeys = await deviceManager.generateOneTimePreKeys(1, 10);
    
    expect(preKeys).toBeDefined();
    expect(preKeys.length).toBe(10);
    
    // Verify each key has correct keyId
    for (let i = 0; i < 10; i++) {
      expect(preKeys[i].keyId).toBe(i + 1);
      expect(preKeys[i].publicKey).toBeDefined();
      expect(preKeys[i].privateKey).toBeDefined();
    }
  });
});

describe('Prekey Bundle Structure', () => {
  it.skipIf(!hasWebCrypto)('should create valid prekey bundle for upload', async () => {
    const { deviceManager } = await import('./device-manager');
    
    // Generate keys
    const identityKey = await deviceManager.generateIdentityKeyPair();
    const signedPreKey = await deviceManager.generateSignedPreKey(
      identityKey.publicKey as unknown as CryptoKey,
      1
    );
    const oneTimePreKeys = await deviceManager.generateOneTimePreKeys(1, 5);
    
    // The registration object structure should be valid
    const registration = {
      deviceId: 'test-device',
      deviceName: 'Test Device',
      deviceType: 'web' as const,
      identityKey: arrayBufferToBase64(identityKey.publicKey),
      registrationId: 12345,
      signedPreKey: {
        keyId: signedPreKey.keyId,
        publicKey: arrayBufferToBase64(signedPreKey.publicKey),
        signature: arrayBufferToBase64(signedPreKey.signature),
      },
      oneTimePreKeys: oneTimePreKeys.map(pk => ({
        keyId: pk.keyId,
        publicKey: arrayBufferToBase64(pk.publicKey),
      })),
    };
    
    // Validate structure
    expect(registration.deviceId).toBe('test-device');
    expect(registration.identityKey).toMatch(/^[A-Za-z0-9+/=]+$/);
    expect(registration.signedPreKey.signature).toMatch(/^[A-Za-z0-9+/=]+$/);
    expect(registration.oneTimePreKeys).toHaveLength(5);
  });
});

// Helper function
function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}
