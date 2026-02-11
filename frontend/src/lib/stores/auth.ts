import { writable, derived } from 'svelte/store';
import type { User, LoginRequest, RegisterRequest, AuthResponse } from '$lib/types';

const TOKEN_KEY = 'hearth_token';

function createAuthStore() {
	const { subscribe, set, update } = writable<{
		user: User | null;
		token: string | null;
		loading: boolean;
		error: string | null;
	}>({
		user: null,
		token: null,
		loading: true,
		error: null
	});

	return {
		subscribe,

		init: async () => {
			const token = localStorage.getItem(TOKEN_KEY);
			if (token) {
				try {
					const res = await fetch('/api/auth/me', {
						headers: { Authorization: `Bearer ${token}` }
					});
					if (res.ok) {
						const user = await res.json();
						set({ user, token, loading: false, error: null });
						return;
					}
				} catch {
					localStorage.removeItem(TOKEN_KEY);
				}
			}
			set({ user: null, token: null, loading: false, error: null });
		},

		login: async (data: LoginRequest) => {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const res = await fetch('/api/auth/login', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(data)
				});
				if (!res.ok) {
					const err = await res.json();
					throw new Error(err.message || 'Login failed');
				}
				const { user, token }: AuthResponse = await res.json();
				localStorage.setItem(TOKEN_KEY, token);
				set({ user, token, loading: false, error: null });
				return true;
			} catch (e) {
				const error = e instanceof Error ? e.message : 'Login failed';
				update((s) => ({ ...s, loading: false, error }));
				return false;
			}
		},

		register: async (data: RegisterRequest) => {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const res = await fetch('/api/auth/register', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(data)
				});
				if (!res.ok) {
					const err = await res.json();
					throw new Error(err.message || 'Registration failed');
				}
				const { user, token }: AuthResponse = await res.json();
				localStorage.setItem(TOKEN_KEY, token);
				set({ user, token, loading: false, error: null });
				return true;
			} catch (e) {
				const error = e instanceof Error ? e.message : 'Registration failed';
				update((s) => ({ ...s, loading: false, error }));
				return false;
			}
		},

		logout: () => {
			localStorage.removeItem(TOKEN_KEY);
			set({ user: null, token: null, loading: false, error: null });
		},

		clearError: () => {
			update((s) => ({ ...s, error: null }));
		}
	};
}

export const auth = createAuthStore();
export const isAuthenticated = derived(auth, ($auth) => !!$auth.user);
export const currentUser = derived(auth, ($auth) => $auth.user);
