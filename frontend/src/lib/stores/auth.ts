import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { api, setAuthToken, clearAuthToken } from '$lib/api';
import { setCurrentUserId } from './typing';

export interface User {
	id: string;
	username: string;
	discriminator?: string;
	email?: string;
	avatar_url?: string | null;
	banner_url?: string | null;
	bio?: string | null;
	custom_status?: string | null;
	flags?: number;
	created_at: string;
}

export interface AuthState {
	user: User | null;
	token: string | null;
	loading: boolean;
	initialized: boolean;
}

const initialState: AuthState = {
	user: null,
	token: null,
	loading: true,
	initialized: false
};

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(initialState);
	
	return {
		subscribe,
		
		async init() {
			if (!browser) return;
			
			const token = localStorage.getItem('hearth_token');
			if (!token) {
				update(s => ({ ...s, loading: false, initialized: true }));
				return;
			}
			
			setAuthToken(token);
			
			try {
				const user = await api.get<User>('/users/@me');
				setCurrentUserId(user.id);
				update(s => ({
					...s,
					user,
					token,
					loading: false,
					initialized: true
				}));
			} catch {
				// Token invalid
				localStorage.removeItem('hearth_token');
				localStorage.removeItem('hearth_refresh_token');
				clearAuthToken();
				update(s => ({ ...s, loading: false, initialized: true }));
			}
		},
		
		async login(email: string, password: string) {
			update(s => ({ ...s, loading: true }));
			
			try {
				const { access_token, refresh_token } = await api.post<{ access_token: string; refresh_token: string }>('/auth/login', {
					email,
					password
				});
				
				localStorage.setItem('hearth_token', access_token);
				localStorage.setItem('hearth_refresh_token', refresh_token);
				setAuthToken(access_token);
				
				const user = await api.get<User>('/users/@me');
				setCurrentUserId(user.id);
				
				update(s => ({
					...s,
					user,
					token: access_token,
					loading: false
				}));
				
				goto('/channels/@me');
			} catch (error: any) {
				update(s => ({ ...s, loading: false }));
				throw error;
			}
		},
		
		async register(email: string, username: string, password: string) {
			update(s => ({ ...s, loading: true }));
			
			try {
				const { access_token, refresh_token } = await api.post<{ access_token: string; refresh_token: string }>('/auth/register', {
					email,
					username,
					password
				});
				
				localStorage.setItem('hearth_token', access_token);
				localStorage.setItem('hearth_refresh_token', refresh_token);
				setAuthToken(access_token);
				
				const user = await api.get<User>('/users/@me');
				setCurrentUserId(user.id);
				
				update(s => ({
					...s,
					user,
					token: access_token,
					loading: false
				}));
				
				goto('/channels/@me');
			} catch (error: any) {
				update(s => ({ ...s, loading: false }));
				throw error;
			}
		},
		
		async logout() {
			try {
				await api.post('/auth/logout');
			} catch {
				// Ignore logout errors
			}
			
			localStorage.removeItem('hearth_token');
			localStorage.removeItem('hearth_refresh_token');
			clearAuthToken();
			setCurrentUserId(null);
			
			set({ ...initialState, loading: false, initialized: true });
			goto('/login');
		},
		
		async refreshToken() {
			const refreshToken = localStorage.getItem('hearth_refresh_token');
			if (!refreshToken) {
				throw new Error('No refresh token');
			}
			
			try {
				const { access_token, refresh_token } = await api.post<{ access_token: string; refresh_token: string }>('/auth/refresh', {
					refresh_token: refreshToken
				});
				
				localStorage.setItem('hearth_token', access_token);
				localStorage.setItem('hearth_refresh_token', refresh_token);
				setAuthToken(access_token);
				
				update(s => ({ ...s, token: access_token }));
			} catch (error) {
				// Token refresh failed - clear auth and redirect to login
				localStorage.removeItem('hearth_token');
				localStorage.removeItem('hearth_refresh_token');
				clearAuthToken();
				set({ ...initialState, loading: false, initialized: true });
				goto('/login');
				throw error;
			}
		},
		
		async updateProfile(updates: Partial<User>) {
			const user = await api.patch<User>('/users/@me', updates);
			update(s => ({ ...s, user }));
			return user;
		}
	};
}

export const auth = createAuthStore();

// Derived stores for convenience
export const user = derived(auth, $auth => $auth.user);
export const isAuthenticated = derived(auth, $auth => !!$auth.user);
export const isLoading = derived(auth, $auth => $auth.loading);
