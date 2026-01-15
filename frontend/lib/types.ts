export interface User {
    id: string;
    email: string;
    created_at: string;
}

export interface AuthResponse {
    user: User;
}

export interface Post {
    id: string;
    user_id: string;
    title?: string;
    content: string;
    channel: 'twitter' | 'linkedin' | 'facebook';
    status: 'scheduled' | 'published' | 'failed';
    scheduled_at: string;
    published_at?: string;
    created_at: string;
    updated_at: string;
}

export interface CreatePostRequest {
    title?: string;
    content: string;
    channel: string;
    scheduled_at: string;
}

export interface UpdatePostRequest {
    title?: string;
    content?: string;
    channel?: string;
    scheduled_at?: string;
}

export interface ErrorResponse {
    error: string;
    message: string;
}
