'use client';

import { Post } from '@/lib/types';
import { postsApi } from '@/lib/api';
import { useState } from 'react';

interface PostCardProps {
  post: Post;
  onUpdate?: () => void;
  showActions?: boolean;
}

const channelEmoji: Record<string, string> = {
  twitter: '🐦',
  linkedin: '💼',
  facebook: '📘',
};

const statusColors: Record<string, string> = {
  scheduled: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
  published: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  failed: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
};

export function PostCard({ post, onUpdate, showActions = true }: PostCardProps) {
  const [deleting, setDeleting] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(post.content);
  const [editTitle, setEditTitle] = useState(post.title || '');

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this post?')) return;
    
    setDeleting(true);
    try {
      await postsApi.delete(post.id);
      onUpdate?.();
    } catch (error) {
      console.error('Failed to delete post:', error);
    } finally {
      setDeleting(false);
    }
  };

  const handleSaveEdit = async () => {
    try {
      await postsApi.update(post.id, {
        title: editTitle || undefined,
        content: editContent,
      });
      setEditing(false);
      onUpdate?.();
    } catch (error) {
      console.error('Failed to update post:', error);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      dateStyle: 'medium',
      timeStyle: 'short',
    });
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md p-4 border border-gray-200 dark:border-gray-700">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="text-2xl">{channelEmoji[post.channel] || '📱'}</span>
          <span className={`px-2 py-1 rounded-full text-xs font-medium ${statusColors[post.status]}`}>
            {post.status}
          </span>
        </div>
        
        {showActions && post.status === 'scheduled' && (
          <div className="flex gap-2">
            <button
              onClick={() => setEditing(!editing)}
              className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 text-sm"
            >
              {editing ? 'Cancel' : 'Edit'}
            </button>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 text-sm disabled:opacity-50"
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </button>
          </div>
        )}
      </div>

      {editing ? (
        <div className="space-y-3">
          <input
            type="text"
            value={editTitle}
            onChange={(e) => setEditTitle(e.target.value)}
            placeholder="Title (optional)"
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white"
          />
          <textarea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            rows={3}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white resize-none"
          />
          <button
            onClick={handleSaveEdit}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm"
          >
            Save Changes
          </button>
        </div>
      ) : (
        <>
          {post.title && (
            <h3 className="font-semibold text-gray-900 dark:text-white mb-2">
              {post.title}
            </h3>
          )}
          
          <p className="text-gray-700 dark:text-gray-300 mb-3 whitespace-pre-wrap">
            {post.content}
          </p>
        </>
      )}

      <div className="text-sm text-gray-500 dark:text-gray-400 space-y-1">
        <div className="flex items-center gap-2">
          <span>📅</span>
          <span>
            {post.status === 'published' 
              ? `Published: ${formatDate(post.published_at!)}`
              : `Scheduled: ${formatDate(post.scheduled_at)}`
            }
          </span>
        </div>
      </div>
    </div>
  );
}
