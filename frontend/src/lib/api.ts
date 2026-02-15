import { browser, dev } from '$app/environment';

const BASE_URL = browser 
	? (import.meta.env.VITE_API_URL || '/api/v1')
	: 'http://localhost:8080/api/v1';

let authToken: string | null = null;

// Debug helper for API request tracing
function apiDebug(message: string, data?: unknown) {
	if (dev && browser) {
		console.log(`[API] ${message}`, data !== undefined ? data : '');
	}
}

export function setAuthToken(token: string) {
	authToken = token;
}

export function clearAuthToken() {
	authToken = null;
}

export function getAuthToken(): string | null {
	return authToken;
}

interface ApiErrorData {
	message?: string;
	error?: string;
	code?: string;
	details?: Record<string, unknown>;
}

class ApiError extends Error {
	status: number;
	data: ApiErrorData | undefined;
	
	constructor(message: string, status: number, data?: ApiErrorData) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.data = data;
	}
}

const DEFAULT_TIMEOUT = 30000; // 30 seconds

async function request<T, B = unknown>(
	method: string,
	path: string,
	body?: B | FormData,
	options: RequestInit = {}
): Promise<T> {
	const headers: Record<string, string> = {
		...(options.headers as Record<string, string> || {})
	};
	
	if (authToken) {
		headers['Authorization'] = `Bearer ${authToken}`;
		apiDebug(`${method} ${path}`, { 
			hasAuthHeader: true, 
			tokenLength: authToken.length,
			tokenPrefix: authToken.substring(0, 20) + '...'
		});
	} else {
		apiDebug(`${method} ${path}`, { hasAuthHeader: false });
	}
	
	let requestBody: BodyInit | undefined;
	
	if (body) {
		if (body instanceof FormData) {
			requestBody = body;
		} else {
			headers['Content-Type'] = 'application/json';
			requestBody = JSON.stringify(body);
		}
	}
	
	// Create abort controller for timeout
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT);
	
	try {
		const response = await fetch(`${BASE_URL}${path}`, {
			method,
			headers,
			body: requestBody,
			signal: controller.signal,
			...options
		});
		
		clearTimeout(timeoutId);
		
		if (!response.ok) {
			let errorData;
			try {
				errorData = await response.json();
			} catch {
				errorData = { error: response.statusText };
			}
			
			throw new ApiError(
				errorData.message || errorData.error || 'Request failed',
				response.status,
				errorData
			);
		}
		
		// Handle empty responses
		if (response.status === 204) {
			return undefined as T;
		}
		
		return response.json();
	} catch (error) {
		clearTimeout(timeoutId);
		
		if (error instanceof Error && error.name === 'AbortError') {
			throw new ApiError('Request timeout', 408, { error: 'Request timeout' });
		}
		
		throw error;
	}
}

export const api = {
	get: <T = unknown>(path: string) => request<T>('GET', path),
	post: <T = unknown, B = unknown>(path: string, body?: B | FormData) => request<T, B>('POST', path, body),
	put: <T = unknown, B = unknown>(path: string, body?: B | FormData) => request<T, B>('PUT', path, body),
	patch: <T = unknown, B = unknown>(path: string, body?: B | FormData) => request<T, B>('PATCH', path, body),
	delete: <T = unknown>(path: string) => request<T>('DELETE', path),
};

export { ApiError };
