'use client';

import { useState, useEffect } from 'react';
import { useAuth } from '@/lib/auth';
import { usePostStream } from '@/lib/usePostStream';
import { PostForm } from '@/components/PostForm';
import { PostList } from '@/components/PostList';
import { Tabs } from '@/components/Tabs';
import { useRouter } from 'next/navigation';

export default function DashboardPage() {
  const { user, loading: authLoading, logout } = useAuth();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('upcoming');
  
  // Use SSE for real-time updates
  const { 
    upcoming: upcomingPosts, 
    history: publishedPosts, 
    isConnected,
    error: sseError,
    refresh 
  } = usePostStream({ enabled: !!user });

  useEffect(() => {
    if (!authLoading && !user) {
      router.push('/login');
      return;
    }
  }, [user, authLoading, router]);

  if (authLoading || (!user && !authLoading)) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-pulse text-xl text-gray-600 dark:text-gray-400">
          Loading...
        </div>
      </div>
    );
  }

  const tabs = [
    { id: 'upcoming', label: 'Upcoming', count: upcomingPosts.length },
    { id: 'history', label: 'History', count: publishedPosts.length },
  ];

  const connectionStatus = isConnected 
    ? '🟢 Live' 
    : sseError 
    ? '🔴 Reconnecting...' 
    : '🟡 Connecting...';

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 shadow-sm">
        <div className="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">
              📅 Post Scheduler
            </h1>
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {connectionStatus}
            </span>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {user?.email}
            </span>
            <button
              onClick={logout}
              className="text-sm text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
            >
              Logout
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-4xl mx-auto px-4 py-8">
        <PostForm onSuccess={refresh} />

        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6">
          <Tabs tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} />

          {activeTab === 'upcoming' ? (
            <PostList
              posts={upcomingPosts}
              onUpdate={refresh}
              showActions={true}
              emptyMessage="No upcoming posts. Schedule one above!"
            />
          ) : (
            <PostList
              posts={publishedPosts}
              onUpdate={refresh}
              showActions={false}
              emptyMessage="No published posts yet."
            />
          )}
        </div>
      </main>
    </div>
  );
}
