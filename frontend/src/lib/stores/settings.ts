import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { theme as themeStore, type ThemeChoice } from './theme';
export type { ResolvedTheme } from './theme';
import { api } from '$lib/api';

export type Theme = ThemeChoice;
export type MessageDisplay = 'cozy' | 'compact';
export type NotificationLevel = 'all' | 'mentions' | 'none';

export type ThreadNotificationLevel = 'all' | 'mentions' | 'none';

export interface NotificationSettings {
	desktopEnabled: boolean;
	soundsEnabled: boolean;
	soundVolume: number;
	mentionSound: boolean;
	messageSound: boolean;
	flashTaskbar: boolean;
	showPreviews: boolean;
	muteDMs: boolean;
	muteGroupDMs: boolean;
	mentionEveryone: boolean;
	mentionRoles: boolean;
	mentionHighlight: boolean;
	suppressDND: boolean;
	// FEAT-001: Thread notification preferences
	threadNotifications: ThreadNotificationLevel;
	threadAutoFollow: boolean;
	threadFollowOnReply: boolean;
}

export interface AppSettings {
	theme: ThemeChoice;
	messageDisplay: MessageDisplay;
	compactMode: boolean;
	showSendButton: boolean;
	enableAnimations: boolean;
	enableSounds: boolean;
	notificationsEnabled: boolean;
	fontSize: number;
	developerMode: boolean;
	notifications: NotificationSettings;
}

export interface SettingsState {
	isOpen: boolean;
	isServerSettingsOpen: boolean;
	activeSection: string;
	app: AppSettings;
	loadedFromBackend: boolean;
}

// Backend API response shape
interface BackendUserSettings {
	user_id: string;
	theme: string;
	message_display: string;
	compact_mode: boolean;
	developer_mode: boolean;
	inline_embeds: boolean;
	inline_attachments: boolean;
	render_reactions: boolean;
	animate_emoji: boolean;
	enable_tts: boolean;
	custom_css?: string;
	notifications_enabled: boolean;
	notifications_sound: boolean;
	notifications_desktop: boolean;
	notifications_mentions_only: boolean;
	notifications_dm: boolean;
	notifications_server_defaults: boolean;
	privacy_dm_from_servers: boolean;
	privacy_dm_from_friends_only: boolean;
	privacy_show_activity: boolean;
	privacy_friend_requests_all: boolean;
	privacy_read_receipts: boolean;
	locale: string;
	thread_auto_follow: boolean;
	thread_follow_on_reply: boolean;
	thread_default_notification_level: string;
	updated_at: string;
}

const defaultNotificationSettings: NotificationSettings = {
	desktopEnabled: true,
	soundsEnabled: true,
	soundVolume: 80,
	mentionSound: true,
	messageSound: true,
	flashTaskbar: true,
	showPreviews: true,
	muteDMs: false,
	muteGroupDMs: false,
	mentionEveryone: true,
	mentionRoles: true,
	mentionHighlight: true,
	suppressDND: false,
	// FEAT-001: Thread notification defaults
	threadNotifications: 'all',
	threadAutoFollow: true,
	threadFollowOnReply: true
};

const defaultSettings: AppSettings = {
	theme: 'system',
	messageDisplay: 'cozy',
	compactMode: false,
	showSendButton: false,
	enableAnimations: true,
	enableSounds: true,
	notificationsEnabled: true,
	fontSize: 16,
	developerMode: false,
	notifications: defaultNotificationSettings
};

function isValidTheme(value: unknown): value is ThemeChoice {
	return typeof value === 'string' && ['system', 'dark', 'light', 'midnight'].includes(value);
}

function isValidMessageDisplay(value: unknown): value is MessageDisplay {
	return typeof value === 'string' && ['cozy', 'compact'].includes(value);
}

function isValidFontSize(value: unknown): value is number {
	return typeof value === 'number' && value >= 12 && value <= 24;
}

function isValidVolume(value: unknown): value is number {
	return typeof value === 'number' && value >= 0 && value <= 100;
}

function isValidThreadNotificationLevel(value: unknown): value is ThreadNotificationLevel {
	return typeof value === 'string' && ['all', 'mentions', 'none'].includes(value);
}

function loadNotificationSettings(parsed: Record<string, unknown>): NotificationSettings {
	const n = typeof parsed.notifications === 'object' && parsed.notifications !== null 
		? parsed.notifications as Record<string, unknown>
		: {};
	
	return {
		desktopEnabled: typeof n.desktopEnabled === 'boolean' ? n.desktopEnabled : defaultNotificationSettings.desktopEnabled,
		soundsEnabled: typeof n.soundsEnabled === 'boolean' ? n.soundsEnabled : defaultNotificationSettings.soundsEnabled,
		soundVolume: isValidVolume(n.soundVolume) ? n.soundVolume : defaultNotificationSettings.soundVolume,
		mentionSound: typeof n.mentionSound === 'boolean' ? n.mentionSound : defaultNotificationSettings.mentionSound,
		messageSound: typeof n.messageSound === 'boolean' ? n.messageSound : defaultNotificationSettings.messageSound,
		flashTaskbar: typeof n.flashTaskbar === 'boolean' ? n.flashTaskbar : defaultNotificationSettings.flashTaskbar,
		showPreviews: typeof n.showPreviews === 'boolean' ? n.showPreviews : defaultNotificationSettings.showPreviews,
		muteDMs: typeof n.muteDMs === 'boolean' ? n.muteDMs : defaultNotificationSettings.muteDMs,
		muteGroupDMs: typeof n.muteGroupDMs === 'boolean' ? n.muteGroupDMs : defaultNotificationSettings.muteGroupDMs,
		mentionEveryone: typeof n.mentionEveryone === 'boolean' ? n.mentionEveryone : defaultNotificationSettings.mentionEveryone,
		mentionRoles: typeof n.mentionRoles === 'boolean' ? n.mentionRoles : defaultNotificationSettings.mentionRoles,
		mentionHighlight: typeof n.mentionHighlight === 'boolean' ? n.mentionHighlight : defaultNotificationSettings.mentionHighlight,
		suppressDND: typeof n.suppressDND === 'boolean' ? n.suppressDND : defaultNotificationSettings.suppressDND,
		// FEAT-001: Thread notification settings
		threadNotifications: isValidThreadNotificationLevel(n.threadNotifications) ? n.threadNotifications : defaultNotificationSettings.threadNotifications,
		threadAutoFollow: typeof n.threadAutoFollow === 'boolean' ? n.threadAutoFollow : defaultNotificationSettings.threadAutoFollow,
		threadFollowOnReply: typeof n.threadFollowOnReply === 'boolean' ? n.threadFollowOnReply : defaultNotificationSettings.threadFollowOnReply
	};
}

function loadSettings(): AppSettings {
	if (!browser) return defaultSettings;
	
	try {
		const stored = localStorage.getItem('hearth_settings');
		if (stored) {
			const parsed = JSON.parse(stored);
			
			// Validate parsed data is an object
			if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
				console.warn('Invalid settings format, using defaults');
				return defaultSettings;
			}
			
			// Merge with defaults, validating each property
			return {
				theme: isValidTheme(parsed.theme) ? parsed.theme : defaultSettings.theme,
				messageDisplay: isValidMessageDisplay(parsed.messageDisplay) ? parsed.messageDisplay : defaultSettings.messageDisplay,
				compactMode: typeof parsed.compactMode === 'boolean' ? parsed.compactMode : defaultSettings.compactMode,
				showSendButton: typeof parsed.showSendButton === 'boolean' ? parsed.showSendButton : defaultSettings.showSendButton,
				enableAnimations: typeof parsed.enableAnimations === 'boolean' ? parsed.enableAnimations : defaultSettings.enableAnimations,
				enableSounds: typeof parsed.enableSounds === 'boolean' ? parsed.enableSounds : defaultSettings.enableSounds,
				notificationsEnabled: typeof parsed.notificationsEnabled === 'boolean' ? parsed.notificationsEnabled : defaultSettings.notificationsEnabled,
				fontSize: isValidFontSize(parsed.fontSize) ? parsed.fontSize : defaultSettings.fontSize,
				developerMode: typeof parsed.developerMode === 'boolean' ? parsed.developerMode : defaultSettings.developerMode,
				notifications: loadNotificationSettings(parsed)
			};
		}
	} catch (error) {
		console.error('Failed to load settings:', error);
	}
	return defaultSettings;
}

function saveSettings(settings: AppSettings) {
	if (!browser) return;
	
	try {
		localStorage.setItem('hearth_settings', JSON.stringify(settings));
	} catch (error) {
		if (error instanceof Error && error.name === 'QuotaExceededError') {
			console.error('Failed to save settings: localStorage quota exceeded');
		} else {
			console.error('Failed to save settings:', error);
		}
	}
}

// Map backend response to frontend AppSettings
function mapBackendToFrontend(backend: BackendUserSettings): Partial<AppSettings> {
	return {
		theme: isValidTheme(backend.theme) ? backend.theme : defaultSettings.theme,
		messageDisplay: isValidMessageDisplay(backend.message_display) ? backend.message_display as MessageDisplay : defaultSettings.messageDisplay,
		compactMode: typeof backend.compact_mode === 'boolean' ? backend.compact_mode : defaultSettings.compactMode,
		developerMode: typeof backend.developer_mode === 'boolean' ? backend.developer_mode : defaultSettings.developerMode,
		notificationsEnabled: typeof backend.notifications_enabled === 'boolean' ? backend.notifications_enabled : defaultSettings.notificationsEnabled,
		notifications: {
			desktopEnabled: typeof backend.notifications_desktop === 'boolean' ? backend.notifications_desktop : defaultNotificationSettings.desktopEnabled,
			soundsEnabled: typeof backend.notifications_sound === 'boolean' ? backend.notifications_sound : defaultNotificationSettings.soundsEnabled,
			soundVolume: defaultNotificationSettings.soundVolume,
			mentionSound: defaultNotificationSettings.mentionSound,
			messageSound: defaultNotificationSettings.messageSound,
			flashTaskbar: defaultNotificationSettings.flashTaskbar,
			showPreviews: defaultNotificationSettings.showPreviews,
			muteDMs: typeof backend.notifications_dm === 'boolean' ? !backend.notifications_dm : defaultNotificationSettings.muteDMs,
			muteGroupDMs: defaultNotificationSettings.muteGroupDMs,
			mentionEveryone: typeof backend.notifications_mentions_only === 'boolean' ? backend.notifications_mentions_only : defaultNotificationSettings.mentionEveryone,
			mentionRoles: defaultNotificationSettings.mentionRoles,
			mentionHighlight: defaultNotificationSettings.mentionHighlight,
			suppressDND: defaultNotificationSettings.suppressDND,
			threadNotifications: isValidThreadNotificationLevel(backend.thread_default_notification_level) 
				? backend.thread_default_notification_level as ThreadNotificationLevel 
				: defaultNotificationSettings.threadNotifications,
			threadAutoFollow: typeof backend.thread_auto_follow === 'boolean' ? backend.thread_auto_follow : defaultNotificationSettings.threadAutoFollow,
			threadFollowOnReply: typeof backend.thread_follow_on_reply === 'boolean' ? backend.thread_follow_on_reply : defaultNotificationSettings.threadFollowOnReply
		}
	};
}

// Map frontend partial settings to backend update request
function mapFrontendToBackend(updates: Partial<AppSettings>): Record<string, unknown> {
	const backend: Record<string, unknown> = {};
	
	if (updates.theme !== undefined) backend['theme'] = updates.theme;
	if (updates.messageDisplay !== undefined) backend['message_display'] = updates.messageDisplay;
	if (updates.compactMode !== undefined) backend['compact_mode'] = updates.compactMode;
	if (updates.developerMode !== undefined) backend['developer_mode'] = updates.developerMode;
	if (updates.notificationsEnabled !== undefined) backend['notifications_enabled'] = updates.notificationsEnabled;
	
	if (updates.notifications) {
		const n = updates.notifications;
		if (n.desktopEnabled !== undefined) backend['notifications_desktop'] = n.desktopEnabled;
		if (n.soundsEnabled !== undefined) backend['notifications_sound'] = n.soundsEnabled;
		if (n.muteDMs !== undefined) backend['notifications_dm'] = !n.muteDMs;
		if (n.mentionEveryone !== undefined) backend['notifications_mentions_only'] = n.mentionEveryone;
		// FEAT-001: Thread settings
		if (n.threadAutoFollow !== undefined) backend['thread_auto_follow'] = n.threadAutoFollow;
		if (n.threadFollowOnReply !== undefined) backend['thread_follow_on_reply'] = n.threadFollowOnReply;
		if (n.threadNotifications !== undefined) backend['thread_default_notification_level'] = n.threadNotifications;
	}
	
	return backend;
}

// Fetch user settings from backend API
export async function fetchUserSettings(settingsStore: ReturnType<typeof createSettingsStore>): Promise<void> {
	if (!browser) return;
	
	try {
		const backendSettings = await api.get<BackendUserSettings | undefined>('/users/@me/settings');
		if (!backendSettings) {
			// No backend settings (user not authenticated or API error)
			settingsStore.update(s => ({ ...s, loadedFromBackend: true }));
			return;
		}
		const mapped = mapBackendToFrontend(backendSettings);
		
		// Merge with localStorage settings (localStorage takes precedence for fields not on backend)
		const local = loadSettings();
		const merged: AppSettings = {
			...local,
			...mapped,
			notifications: {
				...defaultNotificationSettings,
				...local.notifications,
				...mapped.notifications
			}
		};
		
		settingsStore.update(s => {
			const newApp = merged;
			saveSettings(newApp);
			themeStore.set(newApp.theme);
			if (browser) {
				document.documentElement.style.setProperty('--message-font-size', `${newApp.fontSize}px`);
			}
			return { ...s, app: newApp, loadedFromBackend: true };
		});
	} catch (error) {
		console.warn('Failed to fetch user settings from backend, using local defaults:', error);
		// Still mark as loaded so we don't keep retrying
		settingsStore.update(s => ({ ...s, loadedFromBackend: true }));
	}
}

// Sync settings to backend API (fire-and-forget for thread-specific settings only)
// Non-thread notification settings (desktopEnabled, soundsEnabled, etc.) remain local-only
async function syncSettingsToBackend(updates: Partial<AppSettings>): Promise<void> {
	if (!browser) return;
	
	try {
		// Only sync thread-specific settings to backend (FEAT-001)
		// Only include fields that were explicitly changed in updates
		const backendUpdates: Record<string, unknown> = {};
		
		if (updates.notifications) {
			const n = updates.notifications;
			// Only sync thread settings if explicitly changed
			if (Object.prototype.hasOwnProperty.call(n, 'threadAutoFollow')) backendUpdates['thread_auto_follow'] = n.threadAutoFollow;
			if (Object.prototype.hasOwnProperty.call(n, 'threadFollowOnReply')) backendUpdates['thread_follow_on_reply'] = n.threadFollowOnReply;
			if (Object.prototype.hasOwnProperty.call(n, 'threadNotifications')) backendUpdates['thread_default_notification_level'] = n.threadNotifications;
		}
		
		// Also sync top-level app settings that map to backend
		if (updates.theme !== undefined) backendUpdates['theme'] = updates.theme;
		if (updates.messageDisplay !== undefined) backendUpdates['message_display'] = updates.messageDisplay;
		if (updates.compactMode !== undefined) backendUpdates['compact_mode'] = updates.compactMode;
		if (updates.developerMode !== undefined) backendUpdates['developer_mode'] = updates.developerMode;
		if (updates.notificationsEnabled !== undefined) backendUpdates['notifications_enabled'] = updates.notificationsEnabled;
		
		if (Object.keys(backendUpdates).length > 0) {
			await api.patch('/users/@me/settings', backendUpdates);
		}
	} catch (error) {
		// Non-critical: log but don't throw
		console.warn('Failed to sync settings to backend:', error);
	}
}

const initialState: SettingsState = {
	isOpen: false,
	isServerSettingsOpen: false,
	activeSection: 'account',
	app: loadSettings(),
	loadedFromBackend: false
};

// Sync initial theme from settings to theme store
if (browser) {
	themeStore.set(initialState.app.theme);
	// Apply initial font size
	document.documentElement.style.setProperty('--message-font-size', `${initialState.app.fontSize}px`);
}

function createSettingsStore() {
	const { subscribe, update } = writable<SettingsState>(initialState);
	
	return {
		subscribe,
		update,
		
		open(section = 'account') {
			update(s => ({ ...s, isOpen: true, activeSection: section }));
		},
		
		close() {
			update(s => ({ ...s, isOpen: false }));
		},
		
		openServerSettings() {
			update(s => ({ ...s, isServerSettingsOpen: true }));
		},
		
		closeServerSettings() {
			update(s => ({ ...s, isServerSettingsOpen: false }));
		},
		
		setSection(section: string) {
			update(s => ({ ...s, activeSection: section }));
		},
		
		updateApp(updates: Partial<AppSettings>) {
			update(s => {
				const newApp = { ...s.app, ...updates };
				saveSettings(newApp);
				
				// Delegate theme to the theme store
				if (updates.theme !== undefined) {
					themeStore.set(updates.theme);
				}
				
				// Apply font size
				if (updates.fontSize !== undefined && browser) {
					document.documentElement.style.setProperty('--message-font-size', `${updates.fontSize}px`);
				}
				
				// Sync non-blocking to backend
				syncSettingsToBackend(updates);
				
				return { ...s, app: newApp };
			});
		},
		
		updateNotifications(updates: Partial<NotificationSettings>) {
			update(s => {
				const newNotifications = { ...s.app.notifications, ...updates };
				const newApp = { ...s.app, notifications: newNotifications };
				saveSettings(newApp);
				
				// Only sync thread-specific settings to backend (FEAT-001)
				// If only non-thread notification fields are changed, skip backend sync
				const hasThreadUpdate = 
					Object.prototype.hasOwnProperty.call(updates, 'threadAutoFollow') ||
					Object.prototype.hasOwnProperty.call(updates, 'threadFollowOnReply') ||
					Object.prototype.hasOwnProperty.call(updates, 'threadNotifications');
				
				if (hasThreadUpdate) {
					syncSettingsToBackend({ notifications: updates as NotificationSettings });
				}
				
				return { ...s, app: newApp };
			});
		},
		
		reset() {
			update(s => {
				saveSettings(defaultSettings);
				themeStore.set(defaultSettings.theme);
				if (browser) {
					document.documentElement.style.setProperty('--message-font-size', `${defaultSettings.fontSize}px`);
				}
				// TODO: Also reset backend settings
				return { ...s, app: defaultSettings };
			});
		}
	};
}

export const settings = createSettingsStore();

// Attempt to load settings from backend (non-blocking)
// This must be called after 'settings' is created
if (browser) {
	fetchUserSettings(settings);
}

// Convenience derived stores
export const isSettingsOpen = derived(settings, $s => $s.isOpen);
export const isServerSettingsOpen = derived(settings, $s => $s.isServerSettingsOpen);
export const activeSection = derived(settings, $s => $s.activeSection);
export const appSettings = derived(settings, $s => $s.app);
export const currentTheme = derived(settings, $s => $s.app.theme);
export const notificationSettings = derived(settings, $s => $s.app.notifications);
