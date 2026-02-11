import { browser } from '$app/environment';

const BASE_URL = browser 
	? (import.meta.env.VITE_API_URL || '/api/v1')
	: 'http://localhost:8080/api/v1';

let authToken: string | null = null;

export function setAuthToken(token: string) {
	authToken = token;
}

export function clearAuthToken() {
	authToken = null;
}

class ApiError extends Error {
	status: number;
	data: any;
	
	constructor(message: string, status: number, data?: any) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.data = data;
	}
}

async function request<T>(
	method: string,
	path: string,
	body?: any,
	options: RequestInit = {}
): Promise<T> {
	const headers: Record<string, string> = {
		...(options.headers as Record<string, string> || {})
	};
	
	if (authToken) {
		headers['Authorization'] = `Bearer ${authToken}`;
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
	
	const response = await fetch(`${BASE_URL}${path}`, {
		method,
		headers,
		body: requestBody,
		...options
	});
	
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
}

export const api = {
	get: <T = any>(path: string) => request<T>('GET', path),
	post: <T = any>(path: string, body?: any) => request<T>('POST', path, body),
	put: <T = any>(path: string, body?: any) => request<T>('PUT', path, body),
	patch: <T = any>(path: string, body?: any) => request<T>('PATCH', path, body),
	delete: <T = any>(path: string) => request<T>('DELETE', path),
};

export { ApiError };
