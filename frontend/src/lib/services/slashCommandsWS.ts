import { browser } from '$app/environment';
import { api } from '$lib/api';
import { slashCommandUI } from '$lib/stores/slashCommandsUI';
import { slashCommandAutocomplete } from '$lib/services/slashCommands';
import type { SlashCommand, AutocompleteResult, CommandChoice } from '$lib/services/slashCommands';

// Event types for slash commands over WebSocket
export const WS_EVENT_COMMAND_EXECUTE = 'COMMAND_EXECUTE';
export const WS_EVENT_COMMAND_RESPONSE = 'COMMAND_RESPONSE';
export const WS_EVENT_AUTOCOMPLETE = 'AUTOCOMPLETE';
export const WS_EVENT_INTERACTION_CREATE = 'INTERACTION_CREATE';
export const WS_EVENT_INTERACTION_UPDATE = 'INTERACTION_UPDATE';

// Interface for command response events
export interface CommandResponseEvent {
	command_id: string;
	command_name: string;
	channel_id: string;
	guild_id?: string;
	user_id: string;
	status: 'success' | 'error' | 'denied';
	response?: {
		type: number;
		content?: string;
		embeds?: unknown[];
		components?: unknown[];
		flags?: number;
	};
	error_message?: string;
	timestamp: string;
}

// Interface for autocomplete events
export interface AutocompleteEvent {
	command_id: string;
	command_name: string;
	channel_id: string;
	guild_id?: string;
	user_id: string;
	options: {
		name: string;
		value: unknown;
	}[];
	focused_option?: string;
	choices?: CommandChoice[];
}

// Pending command executions
const pendingExecutions = new Map<string, {
	resolve: (value: CommandResponseEvent) => void;
	reject: (error: Error) => void;
	timeout: ReturnType<typeof setTimeout>;
}>();

// WebSocket command handler
let wsHandler: ((event: MessageEvent) => void) | null = null;

/**
 * Initialize WebSocket handling for slash commands.
 * This should be called when the WebSocket connection is established.
 */
export function initSlashCommandWebSocket(ws: WebSocket | null) {
	if (!browser || !ws) return;
	
	// Remove existing handler if any
	if (wsHandler) {
		ws.removeEventListener('message', wsHandler);
	}
	
	wsHandler = (event: MessageEvent) => {
		try {
			const message = JSON.parse(event.data);
			
			// Handle command response events
			if (message.t === WS_EVENT_COMMAND_RESPONSE) {
				handleCommandResponse(message.d);
			}
			
			// Handle autocomplete suggestion events
			if (message.t === WS_EVENT_AUTOCOMPLETE) {
				handleAutocompleteEvent(message.d);
			}
			
			// Handle interaction create events (for real-time command feedback)
			if (message.t === WS_EVENT_INTERACTION_CREATE) {
				handleInteractionCreate(message.d);
			}
		} catch (e) {
			console.error('[SlashCommands WS] Failed to parse message:', e);
		}
	};
	
	ws.addEventListener('message', wsHandler);
}

/**
 * Clean up WebSocket handlers
 */
export function cleanupSlashCommandWebSocket(ws: WebSocket | null) {
	if (!browser) return;
	
	if (wsHandler && ws) {
		ws.removeEventListener('message', wsHandler);
		wsHandler = null;
	}
	
	// Clear pending executions
	pendingExecutions.forEach(({ timeout }) => clearTimeout(timeout));
	pendingExecutions.clear();
}

/**
 * Handle incoming command response events
 */
function handleCommandResponse(data: CommandResponseEvent) {
	// Resolve any pending execution
	const pending = pendingExecutions.get(data.command_id);
	if (pending) {
		pending.resolve(data);
		clearTimeout(pending.timeout);
		pendingExecutions.delete(data.command_id);
	}
	
	// Dispatch store update
	slashCommandUI.setSubmitting(false);
	
	// You could dispatch a custom event here for UI notifications
	if (typeof window !== 'undefined') {
		window.dispatchEvent(new CustomEvent('slashcommand:response', { detail: data }));
	}
}

/**
 * Handle autocomplete events from the server
 */
function handleAutocompleteEvent(data: AutocompleteEvent) {
	// Update autocomplete results in the store
	const result: AutocompleteResult = {
		command: {
			id: data.command_id,
			name: data.command_name,
			type: 1,
			application_id: '',
			description: '',
			version: '',
			default_permission: true,
			created_at: '',
			updated_at: ''
		},
		focusedOption: data.focused_option ? {
			type: 3,
			name: data.focused_option,
			description: ''
		} : undefined,
		choices: data.choices || []
	};
	
	slashCommandAutocomplete.update(results => {
		// Update or add this command's autocomplete
		const filtered = results.filter(r => r.command.id !== data.command_id);
		if (data.choices && data.choices.length > 0) {
			return [...filtered, result];
		}
		return filtered;
	});
}

/**
 * Handle interaction create events
 */
function handleInteractionCreate(data: {
	interaction_id: string;
	command_name: string;
	user_id: string;
	channel_id: string;
	guild_id?: string;
}) {
	// Could show typing indicator or command is being processed
	if (typeof window !== 'undefined') {
		window.dispatchEvent(new CustomEvent('slashcommand:executing', { detail: data }));
	}
}

/**
 * Send a slash command execution via WebSocket
 * Falls back to HTTP if WebSocket is not available
 */
export async function executeSlashCommand(
	command: SlashCommand,
	options: Record<string, unknown>,
	channelId: string,
	guildId?: string,
	ws: WebSocket | null = null
): Promise<CommandResponseEvent | null> {
	const executionId = crypto.randomUUID();
	
	return new Promise((resolve, reject) => {
		// Set up timeout
		const timeout = setTimeout(() => {
			pendingExecutions.delete(executionId);
			reject(new Error('Command execution timed out'));
		}, 30000); // 30 second timeout
		
		// Store pending execution
		pendingExecutions.set(executionId, { resolve, reject, timeout });
		
		// Update store
		slashCommandUI.setSubmitting(true);
		
		// Try WebSocket first
		if (ws && ws.readyState === WebSocket.OPEN) {
			const payload = {
				op: 0, // Dispatch
				t: WS_EVENT_COMMAND_EXECUTE,
				d: {
					id: executionId,
					command_id: command.id,
					command_name: command.name,
					channel_id: channelId,
					guild_id: guildId,
					options,
					timestamp: new Date().toISOString()
				}
			};
			ws.send(JSON.stringify(payload));
		} else {
			// Fall back to HTTP
			fetch(`${import.meta.env.VITE_API_URL || '/api/v1'}/interactions`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Authorization': `Bearer ${browser ? localStorage.getItem('hearth_token') : ''}`
				},
				body: JSON.stringify({
					id: executionId,
					type: 2, // Application command
					data: {
						id: command.id,
						name: command.name,
						type: command.type,
						options: Object.entries(options).map(([name, value]) => ({
							name,
							type: 3, // Assume string type
							value
						}))
					},
					channel_id: channelId,
					guild_id: guildId,
					token: executionId,
					application_id: command.application_id
				})
			}).then(async res => {
				if (!res.ok) {
					throw new Error(`HTTP error: ${res.status}`);
				}
				return res.json();
			}).then(data => {
				const response: CommandResponseEvent = {
					command_id: command.id,
					command_name: command.name,
					channel_id: channelId,
					guild_id: guildId,
					user_id: '',
					status: 'success',
					response: data,
					timestamp: new Date().toISOString()
				};
				resolve(response);
			}).catch(err => {
				pendingExecutions.delete(executionId);
				clearTimeout(timeout);
				slashCommandUI.setSubmitting(false);
				reject(err);
			});
		}
	});
}

/**
 * Send autocomplete request to the server
 */
export async function requestAutocomplete(
	command: SlashCommand,
	focusedOption: string,
	currentValue: string,
	channelId: string,
	guildId?: string
): Promise<CommandChoice[]> {
	try {
		const response = await api.post<{ choices: CommandChoice[] }>('/interactions', {
			id: crypto.randomUUID(),
			type: 4, // Autocomplete
			data: {
				id: command.id,
				name: command.name,
				type: command.type,
				options: [{
					name: focusedOption,
					value: currentValue
				}]
			},
			channel_id: channelId,
			guild_id: guildId,
			token: crypto.randomUUID(),
			application_id: command.application_id
		});
		return response.choices || [];
	} catch (e) {
		console.error('[SlashCommands] Autocomplete request failed:', e);
		return [];
	}
}
