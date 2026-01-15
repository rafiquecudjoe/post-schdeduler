import { AuthResponse, Post, CreatePostRequest, UpdatePostRequest, ErrorResponse } from './types';

// Use NEXT_PUBLIC_API_URL for browser (client-side) requests
// Use API_URL for server-side (SSR) requests
const API_URL = typeof window === 'undefined'
    ? (process.env.API_URL || 'http://localhost:8080')
    : (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080');

class ApiError extends Error {
    status: number;
    data: ErrorResponse;

    constructor(status: number, data: ErrorResponse) {
        super(data.message || data.error);
        this.status = status;
        this.data = data;
    }
}

async function fetchApi<T>(
    endpoint: string,
    options: RequestInit = {}
): Promise<T> {
    const url = `${API_URL}${endpoint}`;

    const response = await fetch(url, {
        ...options,
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
            ...options.headers,
        },
    });

    if (!response.ok) {
        let errorData: ErrorResponse;
        try {
            errorData = await response.json();
        } catch {
            errorData = { error: 'Unknown Error', message: response.statusText };
        }
        throw new ApiError(response.status, errorData);
    }

    if (response.status === 204) {
        return {} as T;
    }

    return response.json();
}

// Auth API
export const authApi = {
    register: (email: string, password: string) =>
        fetchApi<AuthResponse>('/api/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        }),

    login: (email: string, password: string) =>
        fetchApi<AuthResponse>('/api/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        }),

    logout: () =>
        fetchApi<{ message: string }>('/api/auth/logout', {
            method: 'POST',
        }),

    refresh: () =>
        fetchApi<AuthResponse>('/api/auth/refresh', {
            method: 'POST',
        }),

    me: () => fetchApi<AuthResponse>('/api/auth/me'),
};

// Posts API
export const postsApi = {
    create: (data: CreatePostRequest) =>
        fetchApi<Post>('/api/posts', {
            method: 'POST',
            body: JSON.stringify(data),
        }),

    getUpcoming: () => fetchApi<Post[]>('/api/posts/upcoming'),

    getHistory: () => fetchApi<Post[]>('/api/posts/history'),

    getById: (id: string) => fetchApi<Post>(`/api/posts/${id}`),

    update: (id: string, data: UpdatePostRequest) =>
        fetchApi<Post>(`/api/posts/${id}`, {
            method: 'PUT',
            body: JSON.stringify(data),
        }),

    delete: (id: string) =>
        fetchApi<void>(`/api/posts/${id}`, {
            method: 'DELETE',
        }),
};

export { ApiError };
