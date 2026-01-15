'use client';

import { useState, useEffect, useCallback } from 'react';
import { Post } from './types';
import { postsApi } from './api';

interface SSEData {
    upcoming: Post[];
    history: Post[];
}

interface UsePostStreamOptions {
    enabled?: boolean;
}

export function usePostStream(options: UsePostStreamOptions = {}) {
    const { enabled = true } = options;
    const [upcoming, setUpcoming] = useState<Post[]>([]);
    const [history, setHistory] = useState<Post[]>([]);
    const [isConnected, setIsConnected] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Fallback: fetch via REST API
    const fetchViaREST = useCallback(async () => {
        try {
            const [upcomingData, historyData] = await Promise.all([
                postsApi.getUpcoming(),
                postsApi.getHistory(),
            ]);
            setUpcoming(upcomingData);
            setHistory(historyData);
        } catch (err) {
            console.error('REST: Failed to fetch posts', err);
        }
    }, []);

    useEffect(() => {
        if (!enabled) return;

        let eventSource: EventSource | null = null;
        let restFallbackTimer: NodeJS.Timeout | null = null;
        let consecutiveErrors = 0;
        const MAX_CONSECUTIVE_ERRORS = 5;

        const connect = () => {
            // Use NEXT_PUBLIC_API_URL for client-side connections (browser)
            // Falls back to localhost:8080 for local development
            const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
            const sseUrl = `${apiUrl}/api/posts/stream`;
            
            eventSource = new EventSource(sseUrl, {
                withCredentials: true,
            });

            eventSource.addEventListener('connected', () => {
                setIsConnected(true);
                setError(null);
                consecutiveErrors = 0; // Reset on successful connection
                
                // Clear REST fallback timer if connection succeeds
                if (restFallbackTimer) {
                    clearInterval(restFallbackTimer);
                    restFallbackTimer = null;
                }
            });

            eventSource.addEventListener('update', (event) => {
                try {
                    const data: SSEData = JSON.parse(event.data);
                    setUpcoming(data.upcoming || []);
                    setHistory(data.history || []);
                    consecutiveErrors = 0; // Reset on successful update
                } catch (err) {
                    console.error('SSE: Failed to parse update', err, 'Raw data:', event.data);
                }
            });

            eventSource.onerror = (e) => {
                consecutiveErrors++;
                console.error('SSE: Connection error (attempt ' + consecutiveErrors + ')', e);

                
                // Only fallback after many consecutive errors
                if (consecutiveErrors >= MAX_CONSECUTIVE_ERRORS) {
                    setIsConnected(false);
                    setError('SSE unstable. Using REST fallback.');
                    console.warn('SSE: Too many errors, falling back to REST polling');
                    
                    // Close EventSource and use REST polling
                    if (eventSource) {
                        eventSource.close();
                        eventSource = null;
                    }
                    
                    // Initial fetch
                    fetchViaREST();
                    
                    // Poll every 10 seconds (faster than before)
                    restFallbackTimer = setInterval(fetchViaREST, 10000);
                } else {
                    setError(`Connection interrupted. Reconnecting...`);
                    // EventSource will auto-reconnect
                }
            };
        };

        // Initial connection attempt
        connect();
        
        // Fetch initial data via REST as immediate fallback
        fetchViaREST();

        return () => {
            if (eventSource) {
                eventSource.close();
                setIsConnected(false);
            }
            if (restFallbackTimer) {
                clearInterval(restFallbackTimer);
            }
        };
    }, [enabled, fetchViaREST]);

    const refresh = useCallback(() => {
        fetchViaREST();
    }, [fetchViaREST]);

    return {
        upcoming,
        history,
        isConnected,
        error,
        refresh,
    };
}
