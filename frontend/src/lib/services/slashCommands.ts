import { writable } from 'svelte/store';
import { api, ApiError } from '$lib/api';

// Types
export type CommandType = 1 | 2 | 3; // SLASH=1, USER=2, MESSAGE=3
export type OptionType = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11;

export interface CommandOption {
	type: OptionType;
	name: string;
	description: string;
	required?: boolean;
	choices?: CommandChoice[];
	options?: CommandOption[];
	channel_types?: number[];
	min_value?: number;
	max_value?: number;
	min_length?: number;
	max_length?: number;
	autocomplete?: boolean;
}

export interface CommandChoice {
	name: string;
	value: string | number;
}

export interface CommandPermissions {
	overrides?: PermissionOverride[];
}

export interface PermissionOverride {
	id: string;
	type: number; // 1=role, 2=user
	allow?: boolean;
	deny?: boolean;
}

export interface SlashCommand {
	id: string;
	type: CommandType;
	application_id: string;
	guild_id?: string;
	name: string;
	description: string;
	options?: CommandOption[];
	permissions?: CommandPermissions;
	version: string;
	creator_id?: string;
	default_permission: boolean;
	created_at: string;
	updated_at: string;
}

export interface RegisterCommandRequest {
	type?: CommandType;
	guild_id?: string;
	name: string;
	description: string;
	options?: CommandOption[];
	permissions?: CommandPermissions;
	default_permission?: boolean;
}

export interface InteractionCallbackType {
	type: number;
	data?: InteractionCallbackData;
}

export interface InteractionCallbackData {
	content?: string;
	embeds?: unknown[];
	components?: unknown[];
	flags?: number;
	choices?: CommandChoice[];
	title?: string;
	custom_id?: string;
	rows?: unknown[];
}

// Store types
export interface SlashCommandState {
	commands: SlashCommand[];
	loading: boolean;
	error: string | null;
	autocompleteResults: AutocompleteResult[];
	activeCommand: SlashCommand | null;
	inputMode: 'autocomplete' | 'options' | 'submitting';
}

export interface AutocompleteResult {
	command: SlashCommand;
	focusedOption?: CommandOption;
	choices: CommandChoice[];
}

// Stores
export const slashCommands = writable<SlashCommand[]>([]);
export const slashCommandsLoading = writable(false);
export const slashCommandsError = writable<string | null>(null);
export const slashCommandAutocomplete = writable<AutocompleteResult[]>([]);
export const activeSlashCommand = writable<SlashCommand | null>(null);
export const slashCommandInputMode = writable<'autocomplete' | 'options' | 'submitting'>('autocomplete');

// Load commands for an application
export async function loadApplicationCommands(appId: string): Promise<SlashCommand[]> {
	slashCommandsLoading.set(true);
	slashCommandsError.set(null);
	
	try {
		const response = await api.get<{ commands: SlashCommand[] }>(`/applications/${appId}/commands`);
		const commands = response.commands || [];
		slashCommands.set(commands);
		return commands;
	} catch (error) {
		console.error('Failed to load application commands:', error);
		if (error instanceof ApiError) {
			slashCommandsError.set(error.message);
		}
		throw error;
	} finally {
		slashCommandsLoading.set(false);
	}
}

// Load commands for a server (all available commands)
export async function loadServerCommands(serverId: string): Promise<SlashCommand[]> {
	slashCommandsLoading.set(true);
	slashCommandsError.set(null);
	
	try {
		// Get commands from all apps in the server
		const response = await api.get<{ commands: SlashCommand[] }>(`/applications/${serverId}/commands`);
		const commands = response.commands || [];
		slashCommands.set(commands);
		return commands;
	} catch (error) {
		console.error('Failed to load server commands:', error);
		if (error instanceof ApiError) {
			slashCommandsError.set(error.message);
		}
		throw error;
	} finally {
		slashCommandsLoading.set(false);
	}
}

// Register a new command
export async function registerCommand(
	appId: string,
	command: RegisterCommandRequest
): Promise<SlashCommand> {
	const response = await api.post<SlashCommand>(`/applications/${appId}/commands`, command);
	slashCommands.update(cmds => [...cmds, response]);
	return response;
}

// Bulk register commands
export async function bulkRegisterCommands(
	appId: string,
	commands: RegisterCommandRequest[]
): Promise<SlashCommand[]> {
	const response = await api.post<{ commands: SlashCommand[] }>(
		`/applications/${appId}/commands/bulk`,
		commands
	);
	slashCommands.set(response.commands || []);
	return response.commands || [];
}

// Update a command
export async function updateCommand(
	appId: string,
	commandId: string,
	command: Partial<RegisterCommandRequest>
): Promise<SlashCommand> {
	const response = await api.put<SlashCommand>(
		`/applications/${appId}/commands/${commandId}`,
		command
	);
	slashCommands.update(cmds => cmds.map(c => c.id === commandId ? response : c));
	return response;
}

// Delete a command
export async function deleteCommand(appId: string, commandId: string): Promise<void> {
	await api.delete(`/applications/${appId}/commands/${commandId}`);
	slashCommands.update(cmds => cmds.filter(c => c.id !== commandId));
}

// Get autocomplete suggestions for a partial command input
export function getAutocompleteSuggestions(input: string, commands: SlashCommand[]): AutocompleteResult[] {
	if (!input.startsWith('/')) return [];
	
	const partial = input.slice(1).toLowerCase();
	const parts = partial.split(/\s+/);
	const commandName = parts[0];
	
	if (!commandName) {
		// Show all commands
		return commands.map(cmd => ({
			command: cmd,
			choices: []
		}));
	}
	
	// Filter commands by name
	const matchingCommands = commands.filter(cmd => 
		cmd.name.toLowerCase().startsWith(commandName)
	);
	
	if (parts.length === 1) {
		// Just command name - return matching commands
		return matchingCommands.map(cmd => ({
			command: cmd,
			choices: []
		}));
	}
	
	// We're in options - find the focused command and option
	if (matchingCommands.length === 1) {
		const cmd = matchingCommands[0];
		const optionPath = parts.slice(1);
		
		// Navigate nested options
		let currentOptions = cmd.options || [];
		let focusedOption: CommandOption | undefined;
		let currentPath: CommandOption[] = [];
		
		for (let i = 0; i < optionPath.length; i++) {
			const optName = optionPath[i];
			const found = currentOptions.find(o => o.name.toLowerCase() === optName.toLowerCase());
			if (!found) {
				// This might be a value for the previous option
				focusedOption = currentPath[currentPath.length - 1];
				break;
			}
			focusedOption = found;
			currentPath.push(found);
			if (found.options) {
				currentOptions = found.options;
			} else {
				break;
			}
		}
		
		// Generate suggestions
		let choices: CommandChoice[] = [];
		if (focusedOption?.choices && focusedOption.choices.length > 0) {
			// Filter choices by partial input
			const partialValue = optionPath[optionPath.length - 1]?.toLowerCase() || '';
			choices = focusedOption.choices.filter(c => 
				c.name.toLowerCase().includes(partialValue)
			);
		}
		
		return [{
			command: cmd,
			focusedOption,
			choices
		}];
	}
	
	return [];
}

// Parse command input into structured data
export function parseCommandInput(input: string, command: SlashCommand): {
	valid: boolean;
	options: Record<string, unknown>;
	errors: string[];
} {
	const parts = input.slice(1).split(/\s+/);
	const options: Record<string, unknown> = {};
	const errors: string[] = [];
	
	// parts[0] is command name, rest are options
	let currentOption: CommandOption | undefined;
	let currentSubcommand: CommandOption | undefined;
	let optionIndex = 0;
	
	const cmdOptions = command.options || [];
	
	for (let i = 1; i < parts.length; i++) {
		const part = parts[i];
		
		// Check if this is a subcommand
		if (!currentSubcommand && cmdOptions.length > 0) {
			const subOpt = cmdOptions.find(o => o.name.toLowerCase() === part.toLowerCase());
			if (subOpt && (subOpt.type === 1 || subOpt.type === 2)) {
				currentSubcommand = subOpt;
				currentOption = undefined;
				optionIndex = 0;
				continue;
			}
		}
		
		// Get current options context
		const contextOptions = currentSubcommand?.options || currentOption?.options || cmdOptions;
		
		// If we have a current option expecting a value
		if (currentOption) {
			// Parse value based on option type
			const parsed = parseOptionValue(currentOption.type, part);
			options[currentOption.name] = parsed.value;
			if (!parsed.valid) {
				errors.push(`Invalid value for ${currentOption.name}: ${part}`);
			}
			currentOption = undefined;
			continue;
		}
		
		// Find next required option or match by name
		if (part.startsWith('--')) {
			// Named option
			const optName = part.slice(2);
			const opt = contextOptions.find(o => o.name.toLowerCase() === optName.toLowerCase());
			if (!opt) {
				errors.push(`Unknown option: ${optName}`);
				continue;
			}
			currentOption = opt;
			if (!opt.required && opt.choices && opt.choices.length === 0) {
				// Boolean or flag - set to true
				options[opt.name] = true;
				currentOption = undefined;
			}
		} else {
			// Positional
			const requiredOpts = contextOptions.filter(o => o.required && !options[o.name]);
			const positionalOpt = requiredOpts[optionIndex] || contextOptions[optionIndex];
			if (!positionalOpt) {
				errors.push(`Unexpected value: ${part}`);
				continue;
			}
			const parsed = parseOptionValue(positionalOpt.type, part);
			options[positionalOpt.name] = parsed.value;
			if (!parsed.valid) {
				errors.push(`Invalid value for ${positionalOpt.name}: ${part}`);
			}
			optionIndex++;
		}
	}
	
	// Check required options
	for (const opt of cmdOptions) {
		if (opt.required && !(opt.name in options)) {
			errors.push(`Missing required option: ${opt.name}`);
		}
	}
	
	return {
		valid: errors.length === 0,
		options,
		errors
	};
}

function parseOptionValue(type_: OptionType, value: string): { value: unknown; valid: boolean } {
	switch (type_) {
		case 3: // String
			return { value, valid: true };
		case 4: // Integer
			const intVal = parseInt(value, 10);
			return { value: intVal, valid: !isNaN(intVal) };
		case 5: // Boolean
			const lower = value.toLowerCase();
			if (lower === 'true' || lower === 'yes' || lower === '1') return { value: true, valid: true };
			if (lower === 'false' || lower === 'no' || lower === '0') return { value: false, valid: true };
			return { value: Boolean(value), valid: true };
		case 10: // Number
			const numVal = parseFloat(value);
			return { value: numVal, valid: !isNaN(numVal) };
		default:
			return { value, valid: true };
	}
}

// Get option type label
export function getOptionTypeLabel(type_: number): string {
	const labels: Record<number, string> = {
		1: 'Subcommand',
		2: 'Subcommand Group',
		3: 'String',
		4: 'Integer',
		5: 'Boolean',
		6: 'User',
		7: 'Channel',
		8: 'Role',
		9: 'Mentionable',
		10: 'Number',
		11: 'Attachment'
	};
	return labels[type_] || `Unknown (${type_})`;
}

// Clear active command state
export function clearActiveCommand(): void {
	activeSlashCommand.set(null);
	slashCommandInputMode.set('autocomplete');
	slashCommandAutocomplete.set([]);
}
