import { writable, derived, get } from 'svelte/store';
import type { SlashCommand, AutocompleteResult, CommandOption } from '$lib/services/slashCommands';

// Store for slash command state
export interface SlashCommandUIState {
	// Input state
	inputValue: string;
	showAutocomplete: boolean;
	selectedIndex: number;
	
	// Command state
	activeCommand: SlashCommand | null;
	parsedOptions: Record<string, unknown>;
	optionErrors: string[];
	
	// Autocomplete
	autocompleteResults: AutocompleteResult[];
	
	// Modal/Dashboard state
	showCommandBuilder: boolean;
	editingCommand: SlashCommand | null;
	
	// Loading states
	submitting: boolean;
	loadingCommands: boolean;
}

const initialState: SlashCommandUIState = {
	inputValue: '',
	showAutocomplete: false,
	selectedIndex: 0,
	activeCommand: null,
	parsedOptions: {},
	optionErrors: [],
	autocompleteResults: [],
	showCommandBuilder: false,
	editingCommand: null,
	submitting: false,
	loadingCommands: false
};

function createSlashCommandStore() {
	const { subscribe, set, update } = writable<SlashCommandUIState>(initialState);
	
	return {
		subscribe,
		
		// Reset to initial state
		reset() {
			set(initialState);
		},
		
		// Input handling
		setInput(value: string) {
			update(state => ({ ...state, inputValue: value }));
		},
		
		// Autocomplete visibility
		showAutocomplete() {
			update(state => ({ ...state, showAutocomplete: true }));
		},
		
		hideAutocomplete() {
			update(state => ({ ...state, showAutocomplete: false, selectedIndex: 0 }));
		},
		
		// Selection navigation
		selectNext(maxIndex: number) {
			update(state => ({
				...state,
				selectedIndex: Math.min(state.selectedIndex + 1, maxIndex - 1)
			}));
		},
		
		selectPrev() {
			update(state => ({
				...state,
				selectedIndex: Math.max(state.selectedIndex - 1, 0)
			}));
		},
		
		setSelectedIndex(index: number) {
			update(state => ({ ...state, selectedIndex: index }));
		},
		
		// Command selection from autocomplete
		selectCommand(command: SlashCommand) {
			update(state => ({
				...state,
				activeCommand: command,
				inputValue: `/${command.name} `,
				showAutocomplete: false,
				selectedIndex: 0,
				parsedOptions: {},
				optionErrors: []
			}));
		},
		
		// Option management
		setOption(name: string, value: unknown) {
			update(state => ({
				...state,
				parsedOptions: { ...state.parsedOptions, [name]: value }
			}));
		},
		
		removeOption(name: string) {
			update(state => {
				const newOptions = { ...state.parsedOptions };
				delete newOptions[name];
				return { ...state, parsedOptions: newOptions };
			});
		},
		
		setOptionErrors(errors: string[]) {
			update(state => ({ ...state, optionErrors: errors }));
		},
		
		// Command builder modal
		openCommandBuilder(command?: SlashCommand) {
			update(state => ({
				...state,
				showCommandBuilder: true,
				editingCommand: command || null
			}));
		},
		
		closeCommandBuilder() {
			update(state => ({
				...state,
				showCommandBuilder: false,
				editingCommand: null
			}));
		},
		
		// Loading states
		setSubmitting(value: boolean) {
			update(state => ({ ...state, submitting: value }));
		},
		
		setLoadingCommands(value: boolean) {
			update(state => ({ ...state, loadingCommands: value }));
		},
		
		// Autocomplete results
		setAutocompleteResults(results: AutocompleteResult[]) {
			update(state => ({
				...state,
				autocompleteResults: results,
				selectedIndex: 0
			}));
		},
		
		// Clear command but keep input
		clearActiveCommand() {
			update(state => ({
				...state,
				activeCommand: null,
				parsedOptions: {},
				optionErrors: [],
				inputValue: ''
			}));
		}
	};
}

export const slashCommandUI = createSlashCommandStore();

// Derived store for whether autocomplete should be shown
export const shouldShowAutocomplete = derived(
	slashCommandUI,
	$state => $state.showAutocomplete && $state.inputValue.startsWith('/')
);

// Derived store for the currently selected autocomplete item
export const selectedAutocompleteItem = derived(
	slashCommandUI,
	$state => $state.autocompleteResults[$state.selectedIndex] || null
);

// Derived store for whether we're in option input mode
export const isInputtingOptions = derived(
	slashCommandUI,
	$state => $state.activeCommand !== null
);

// Derived store for validation state
export const canSubmitCommand = derived(
	slashCommandUI,
	$state => {
		if (!$state.activeCommand) return false;
		if ($state.submitting) return false;
		if ($state.optionErrors.length > 0) return false;
		
		// Check required options
		const requiredOptions = $state.activeCommand.options?.filter(o => o.required) || [];
		for (const opt of requiredOptions) {
			if (!(opt.name in $state.parsedOptions)) {
				return false;
			}
		}
		
		return true;
	}
);
