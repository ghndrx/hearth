import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

export type ThemeChoice = 'system' | 'dark' | 'light' | 'midnight';
export type ResolvedTheme = 'dark' | 'light' | 'midnight';

const STORAGE_KEY = 'hearth_theme_preference';

function getOsPreference(): ResolvedTheme {
	if (!browser) return 'dark';
	return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function loadPreference(): ThemeChoice {
	if (!browser) return 'system';
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored && ['system', 'dark', 'light', 'midnight'].includes(stored)) {
			return stored as ThemeChoice;
		}
	} catch {
		// ignore
	}
	return 'system';
}

function savePreference(choice: ThemeChoice) {
	if (!browser) return;
	try {
		localStorage.setItem(STORAGE_KEY, choice);
	} catch {
		// ignore
	}
}

function applyTheme(resolved: ResolvedTheme) {
	if (!browser) return;
	document.documentElement.setAttribute('data-theme', resolved === 'dark' ? '' : resolved);
}

const osPreference = writable<ResolvedTheme>(getOsPreference());

function createThemeStore() {
	const choice = writable<ThemeChoice>(loadPreference());

	if (browser) {
		const mql = window.matchMedia('(prefers-color-scheme: light)');
		mql.addEventListener('change', (e) => {
			osPreference.set(e.matches ? 'light' : 'dark');
		});
	}

	const resolved = derived([choice, osPreference], ([$choice, $os]) => {
		return $choice === 'system' ? $os : $choice;
	});

	// Apply theme reactively
	if (browser) {
		resolved.subscribe((theme) => {
			applyTheme(theme);
		});
	}

	return {
		choice,
		resolved,
		set(newChoice: ThemeChoice) {
			savePreference(newChoice);
			choice.set(newChoice);
		}
	};
}

export const theme = createThemeStore();
export const themeChoice = theme.choice;
export const resolvedTheme = theme.resolved;
