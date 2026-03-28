/**
 * E2EE Service - Prekey Bundle Fetching and Key Management
 * 
 * This service provides high-level E2EE operations:
 * - Fetching prekey bundles for establishing sessions
 * - Uploading device keys to the server
 * - Managing device registration
 * 
 * Backend routes:
 * - POST /api/v1/keys/upload - upload device keys
 * - GET /api/v1/keys/{userId}/devices - list user devices
 * - GET /api/v1/keys/{userId}/devices/{id}/bundle - get prekey bundle
 * - GET /api/v1/keys/{userId}/capabilities - check E2EE support
 */

import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { e2eeApi, type DeviceInfo, type E2EECapabilities, E2EEApiError } from '$lib/crypto/e2ee-api';
import { auth } from '$lib/stores/auth';
import { keyStorage } from '$lib/e2ee/key-storage';
import { deviceManager, type DeviceRegistration } from '$lib/e2ee/device-manager';
import type { RemotePreKeyBundle } from '$lib/crypto/signal-protocol';

// ============================================================================
// Types
// ============================================================================

/**
 * Result of fetching a prekey bundle
 */
export interface PreKeyBundleResult {
  success: boolean;
  bundle?: RemotePreKeyBundle;
  error?: string;
  isNewDevice?: boolean;
}

/**
 * Result of registering device keys
 */
export interface KeyUploadResult {
  success: boolean;
  deviceId?: string;
  error?: string;
}

/**
 * Session establishment result
 */
export interface SessionEstablishmentResult {
  success: boolean;
  sharedSecret?: ArrayBuffer;
  error?: string;
}

// ============================================================================
// Prekey Bundle Fetching
// ============================================================================

/**
 * Fetch a prekey bundle for a specific user's device
 * 
 * This is called when establishing an E2EE session with a recipient.
 * The bundle contains the recipient's identity key and prekeys needed
 * for X3DH key agreement.
 * 
 * @param userId - The recipient user's ID
 * @param deviceId - The specific device ID to fetch the bundle for
 * @returns The prekey bundle result
 */
export async function fetchPrekeyBundle(
  userId: string,
  deviceId: string
): Promise<PreKeyBundleResult> {
  if (!browser) {
    return { success: false, error: 'Cannot fetch prekey bundle in SSR' };
  }

  try {
    const bundle = await e2eeApi.getPreKeyBundle(userId, deviceId);
    return {
      success: true,
      bundle,
      isNewDevice: false,
    };
  } catch (error) {
    if (error instanceof E2EEApiError) {
      if (error.statusCode === 404) {
        // Device not found - this is a new device scenario
        return {
          success: false,
          error: 'Device not found - may be a new device',
          isNewDevice: true,
        };
      }
      return {
        success: false,
        error: error.message,
      };
    }
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

/**
 * Fetch prekey bundles for all of a user's devices
 * 
 * @param userId - The recipient user's ID
 * @returns Array of prekey bundles for all devices
 */
export async function fetchAllPrekeyBundles(
  userId: string
): Promise<{
  success: boolean;
  bundles?: RemotePreKeyBundle[];
  error?: string;
}> {
  if (!browser) {
    return { success: false, error: 'Cannot fetch prekey bundles in SSR' };
  }

  try {
    const bundles = await e2eeApi.getAllPreKeyBundles(userId);
    return {
      success: true,
      bundles,
    };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

/**
 * Get a user's registered devices
 * 
 * @param userId - The user ID to get devices for
 * @returns Array of device info
 */
export async function getUserDevices(
  userId: string
): Promise<{
  success: boolean;
  devices?: DeviceInfo[];
  error?: string;
}> {
  if (!browser) {
    return { success: false, error: 'Cannot get user devices in SSR' };
  }

  try {
    const devices = await e2eeApi.getUserDevices(userId);
    return {
      success: true,
      devices,
    };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

// ============================================================================
// Device Key Management
// ============================================================================

/**
 * Upload device keys to the server
 * 
 * This is called during device registration to upload the initial
 * identity key and prekeys to the server.
 * 
 * @param registration - The device registration data to upload
 * @returns The upload result
 */
export async function uploadDeviceKeys(
  registration: DeviceRegistration
): Promise<KeyUploadResult> {
  if (!browser) {
    return { success: false, error: 'Cannot upload keys in SSR' };
  }

  try {
    const result = await e2eeApi.uploadKeys(registration);
    return {
      success: true,
      deviceId: result.device_id,
    };
  } catch (error) {
    if (error instanceof E2EEApiError) {
      return {
        success: false,
        error: error.message,
      };
    }
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

/**
 * Register the current device and upload keys
 * 
 * This generates new identity keys and prekeys, stores them locally,
 * and uploads the public keys to the server.
 * 
 * @param deviceName - Optional custom device name
 * @returns The registration result
 */
export async function registerDevice(
  deviceName?: string
): Promise<{
  success: boolean;
  deviceId?: string;
  error?: string;
}> {
  if (!browser) {
    return { success: false, error: 'Cannot register device in SSR' };
  }

  try {
    // Check if already registered
    const isRegistered = await deviceManager.isRegistered();
    if (isRegistered) {
      const existingDeviceId = deviceManager.getDeviceId();
      return {
        success: true,
        deviceId: existingDeviceId || undefined,
      };
    }

    // Register new device
    const registration = await deviceManager.registerDevice(deviceName);

    // Upload keys to server
    await uploadDeviceKeys(registration);

    return {
      success: true,
      deviceId: registration.deviceId,
    };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

/**
 * Check if E2EE is supported for a user
 * 
 * @param userId - The user ID to check
 * @returns Whether the user supports E2EE
 */
export async function checkUserE2EESupport(
  userId: string
): Promise<boolean> {
  if (!browser) return false;

  try {
    return await e2eeApi.supportsE2EE(userId);
  } catch {
    return false;
  }
}

/**
 * Get E2EE capabilities for a user
 * 
 * @param userId - The user ID to check
 * @returns The E2EE capabilities or null if unavailable
 */
export async function getUserE2EECapabilities(
  userId: string
): Promise<E2EECapabilities | null> {
  if (!browser) return null;

  try {
    return await e2eeApi.getCapabilities(userId);
  } catch {
    return null;
  }
}

/**
 * Delete a registered device
 * 
 * @param deviceId - The device ID to delete
 */
export async function deleteDevice(
  deviceId: string
): Promise<{
  success: boolean;
  error?: string;
}> {
  if (!browser) {
    return { success: false, error: 'Cannot delete device in SSR' };
  }

  try {
    await e2eeApi.deleteDevice(deviceId);
    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

/**
 * Get the current device's remaining prekey count
 * 
 * @returns The number of remaining one-time prekeys
 */
export async function getRemainingPreKeyCount(): Promise<number> {
  if (!browser) return 0;
  return deviceManager.getRemainingPreKeyCount();
}

/**
 * Check if prekeys need replenishment
 * 
 * @param threshold - The threshold below which replenishment is needed
 * @returns Whether prekeys need replenishment
 */
export async function needsPreKeyReplenishment(
  threshold: number = 20
): Promise<boolean> {
  if (!browser) return false;
  return deviceManager.needsPreKeyReplenishment(threshold);
}

/**
 * Replenish one-time prekeys
 * 
 * Generates new prekeys and uploads them to the server.
 * Should be called when prekey count is low.
 * 
 * @param count - Number of prekeys to generate
 * @returns The new prekey public parts
 */
export async function replenishPreKeys(
  count: number = 100
): Promise<{
  success: boolean;
  preKeys?: Array<{ keyId: number; publicKey: string }>;
  error?: string;
}> {
  if (!browser) {
    return { success: false, error: 'Cannot replenish prekeys in SSR' };
  }

  try {
    const newKeys = await deviceManager.replenishPreKeys(count);
    return {
      success: true,
      preKeys: newKeys,
    };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

// ============================================================================
// Session Establishment Support
// ============================================================================

/**
 * Prepare for session establishment with a user
 * 
 * This function checks if we can establish a session with the given user
 * by verifying they support E2EE and have available prekeys.
 * 
 * @param userId - The user to establish a session with
 * @returns Whether session establishment is possible
 */
export async function canEstablishSession(
  userId: string
): Promise<{
  possible: boolean;
  reason?: string;
  devices?: DeviceInfo[];
}> {
  // Check if user supports E2EE
  const supportsE2EE = await checkUserE2EESupport(userId);
  if (!supportsE2EE) {
    return {
      possible: false,
      reason: 'User does not support E2EE',
    };
  }

  // Get user's devices
  const result = await getUserDevices(userId);
  if (!result.success || !result.devices) {
    return {
      possible: false,
      reason: result.error || 'Failed to get user devices',
    };
  }

  // Check if any device has prekeys
  const devicesWithPrekeys = result.devices.filter(d => d.has_pre_keys);
  if (devicesWithPrekeys.length === 0) {
    return {
      possible: false,
      reason: 'User has no devices with prekeys available',
      devices: result.devices,
    };
  }

  return {
    possible: true,
    devices: result.devices,
  };
}

// ============================================================================
// Service Initialization
// ============================================================================

/**
 * Initialize the E2EE service
 * 
 * This sets up the auth token for API requests and initializes
 * the key storage. Should be called at app startup.
 */
export async function initializeE2EEService(): Promise<void> {
  if (!browser) return;

  // Set auth token from auth store
  const $auth = get(auth);
  if ($auth.token) {
    e2eeApi.setAuthToken($auth.token);
  }

  // Subscribe to auth changes to keep token updated
  auth.subscribe(($auth) => {
    if ($auth.token) {
      e2eeApi.setAuthToken($auth.token);
    }
  });
}

// Export the API client for direct use if needed
export { e2eeApi };
