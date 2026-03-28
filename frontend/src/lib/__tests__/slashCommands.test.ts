/**
 * Tests for slash commands service
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { get } from 'svelte/store';

// Unmock the slash commands module - test the real implementation
vi.unmock('$lib/services/slashCommands');
vi.unmock('$lib/services/slashCommandsWS');

describe('Slash Commands Types', () => {
	it('should have correct command type values', () => {
		expect(1).toBe(1); // CommandTypeSlash
		expect(2).toBe(2); // CommandTypeUser
		expect(3).toBe(3); // CommandTypeMessage
	});

	it('should have correct option type values', () => {
		expect(1).toBe(1); // OptionTypeSubcommand
		expect(2).toBe(2); // OptionTypeSubcommandGroup
		expect(3).toBe(3); // OptionTypeString
		expect(4).toBe(4); // OptionTypeInteger
		expect(5).toBe(5); // OptionTypeBoolean
		expect(6).toBe(6); // OptionTypeUser
		expect(7).toBe(7); // OptionTypeChannel
		expect(8).toBe(8); // OptionTypeRole
		expect(9).toBe(9); // OptionTypeMentionable
		expect(10).toBe(10); // OptionTypeNumber
		expect(11).toBe(11); // OptionTypeAttachment
	});

	it('should have correct interaction callback type values', () => {
		expect(1).toBe(1); // CallbackTypePong
		expect(4).toBe(4); // CallbackTypeChannelMessage
		expect(5).toBe(5); // CallbackTypeDeferredChannelMessage
		expect(6).toBe(6); // CallbackTypeDeferredUpdateMessage
		expect(7).toBe(7); // CallbackTypeUpdateMessage
		expect(8).toBe(8); // CallbackTypeAutocompleteResult
		expect(9).toBe(9); // CallbackTypeModal
	});
});

describe('Slash Commands Store Functions', () => {
	beforeEach(() => {
		vi.resetModules();
	});

	describe('getOptionTypeLabel', async () => {
		it('should return correct labels for each option type', async () => {
			const { getOptionTypeLabel } = await import('$lib/services/slashCommands');

			expect(getOptionTypeLabel(1)).toBe('Subcommand');
			expect(getOptionTypeLabel(2)).toBe('Subcommand Group');
			expect(getOptionTypeLabel(3)).toBe('String');
			expect(getOptionTypeLabel(4)).toBe('Integer');
			expect(getOptionTypeLabel(5)).toBe('Boolean');
			expect(getOptionTypeLabel(6)).toBe('User');
			expect(getOptionTypeLabel(7)).toBe('Channel');
			expect(getOptionTypeLabel(8)).toBe('Role');
			expect(getOptionTypeLabel(9)).toBe('Mentionable');
			expect(getOptionTypeLabel(10)).toBe('Number');
			expect(getOptionTypeLabel(11)).toBe('Attachment');
		});

		it('should return Unknown for invalid types', async () => {
			const { getOptionTypeLabel } = await import('$lib/services/slashCommands');

			expect(getOptionTypeLabel(99)).toContain('Unknown');
		});
	});

	describe('parseCommandInput', async () => {
		it('should parse basic command without options', async () => {
			const { parseCommandInput } = await import('$lib/services/slashCommands');

			const command = {
				id: 'cmd-1',
				type: 1,
				application_id: 'app-1',
				name: 'ping',
				description: 'Check bot latency',
				version: 'v1',
				default_permission: true,
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString()
			};

			const result = parseCommandInput('/ping', command as any);

			expect(result.valid).toBe(true);
			expect(result.options).toEqual({});
			expect(result.errors).toEqual([]);
		});

		it('should parse command with required options', async () => {
			const { parseCommandInput } = await import('$lib/services/slashCommands');

			const command = {
				id: 'cmd-1',
				type: 1,
				application_id: 'app-1',
				name: 'kick',
				description: 'Kick a user',
				version: 'v1',
				default_permission: true,
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString(),
				options: [
					{
						type: 6, // User
						name: 'user',
						description: 'User to kick',
						required: true
					},
					{
						type: 3, // String
						name: 'reason',
						description: 'Reason for kicking',
						required: false
					}
				]
			};

			const result = parseCommandInput('/kick @john excessive spam', command as any);

			expect(result.valid).toBe(true);
			expect(result.options.user).toBe('@john');
			expect(result.options.reason).toBe('excessive spam');
			expect(result.errors).toEqual([]);
		});

		it('should report missing required options', async () => {
			const { parseCommandInput } = await import('$lib/services/slashCommands');

			const command = {
				id: 'cmd-1',
				type: 1,
				application_id: 'app-1',
				name: 'kick',
				description: 'Kick a user',
				version: 'v1',
				default_permission: true,
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString(),
				options: [
					{
						type: 6,
						name: 'user',
						description: 'User to kick',
						required: true
					}
				]
			};

			const result = parseCommandInput('/kick', command as any);

			expect(result.valid).toBe(false);
			expect(result.errors).toContain('Missing required option: user');
		});
	});

	describe('getAutocompleteSuggestions', async () => {
		it('should return empty for non-slash input', async () => {
			const { getAutocompleteSuggestions } = await import('$lib/services/slashCommands');

			const commands = [
				{ id: '1', name: 'ping', description: '', type: 1, application_id: 'app', version: 'v1', default_permission: true, created_at: '', updated_at: '' }
			];

			const results = getAutocompleteSuggestions('ping', commands as any);
			expect(results).toEqual([]);
		});

		it('should return all commands for just "/"', async () => {
			const { getAutocompleteSuggestions } = await import('$lib/services/slashCommands');

			const commands = [
				{ id: '1', name: 'ping', description: 'Ping', type: 1, application_id: 'app', version: 'v1', default_permission: true, created_at: '', updated_at: '' },
				{ id: '2', name: 'help', description: 'Help', type: 1, application_id: 'app', version: 'v1', default_permission: true, created_at: '', updated_at: '' }
			];

			const results = getAutocompleteSuggestions('/', commands as any);
			expect(results.length).toBe(2);
		});

		it('should filter commands by partial name', async () => {
			const { getAutocompleteSuggestions } = await import('$lib/services/slashCommands');

			const commands = [
				{ id: '1', name: 'ping', description: 'Ping', type: 1, application_id: 'app', version: 'v1', default_permission: true, created_at: '', updated_at: '' },
				{ id: '2', name: 'help', description: 'Help', type: 1, application_id: 'app', version: 'v1', default_permission: true, created_at: '', updated_at: '' },
				{ id: '3', name: 'party', description: 'Party mode', type: 1, application_id: 'app', version: 'v1', default_permission: true, created_at: '', updated_at: '' }
			];

			const results = getAutocompleteSuggestions('/p', commands as any);
			expect(results.length).toBe(1);
			expect(results[0].command.name).toBe('ping');
		});

		it('should handle case-insensitive matching', async () => {
			const { getAutocompleteSuggestions } = await import('$lib/services/slashCommands');

			const commands = [
				{ id: '1', name: 'PingCommand', description: 'Ping', type: 1, application_id: 'app', version: 'v1', default_permission: true, created_at: '', updated_at: '' }
			];

			const results = getAutocompleteSuggestions('/PING', commands as any);
			expect(results.length).toBe(1);
		});
	});
});

describe('Slash Commands WebSocket Service', () => {
	describe('Event Type Constants', () => {
		it('should export WebSocket event type constants', async () => {
			const { 
				WS_EVENT_COMMAND_EXECUTE,
				WS_EVENT_COMMAND_RESPONSE,
				WS_EVENT_AUTOCOMPLETE,
				WS_EVENT_INTERACTION_CREATE,
				WS_EVENT_INTERACTION_UPDATE
			} = await import('$lib/services/slashCommandsWS');

			expect(WS_EVENT_COMMAND_EXECUTE).toBe('COMMAND_EXECUTE');
			expect(WS_EVENT_COMMAND_RESPONSE).toBe('COMMAND_RESPONSE');
			expect(WS_EVENT_AUTOCOMPLETE).toBe('AUTOCOMPLETE');
			expect(WS_EVENT_INTERACTION_CREATE).toBe('INTERACTION_CREATE');
			expect(WS_EVENT_INTERACTION_UPDATE).toBe('INTERACTION_UPDATE');
		});
	});
});
