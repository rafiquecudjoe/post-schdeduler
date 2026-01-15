'use client';

import { useState } from 'react';
import { CreatePostRequest } from '@/lib/types';
import { postsApi, ApiError } from '@/lib/api';

interface PostFormProps {
  onSuccess?: () => void;
}

export function PostForm({ onSuccess }: PostFormProps) {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [channel, setChannel] = useState('twitter');
  const [scheduledAt, setScheduledAt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      // Frontend validation
      const trimmedContent = content.trim();
      if (trimmedContent.length < 3) {
        setError('Content must be at least 3 characters');
        setLoading(false);
        return;
      }
      if (trimmedContent.length > 5000) {
        setError('Content must not exceed 5000 characters');
        setLoading(false);
        return;
      }

      const trimmedTitle = title.trim();
      if (trimmedTitle.length > 200) {
        setError('Title must not exceed 200 characters');
        setLoading(false);
        return;
      }

      // Convert datetime-local to RFC3339 format
      // datetime-local is in local timezone, so we need to format it correctly
      const date = new Date(scheduledAt);
      // Add timezone offset to get UTC time
      const offset = date.getTimezoneOffset();
      const utcDate = new Date(date.getTime() - offset * 60 * 1000);
      const rfc3339 = utcDate.toISOString();

      // Validate date is not too far in future (1 year max)
      const maxDate = new Date();
      maxDate.setFullYear(maxDate.getFullYear() + 1);
      if (date > maxDate) {
        setError('Scheduled date cannot be more than 1 year in the future');
        setLoading(false);
        return;
      }

      const data: CreatePostRequest = {
        content: trimmedContent,
        channel,
        scheduled_at: rfc3339,
      };

      if (trimmedTitle) {
        data.title = trimmedTitle;
      }

      await postsApi.create(data);
      
      // Reset form
      setTitle('');
      setContent('');
      setChannel('twitter');
      setScheduledAt('');
      
      onSuccess?.();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('Failed to create post');
      }
    } finally {
      setLoading(false);
    }
  };

  // Get minimum datetime (now + 1 minute)
  const getMinDateTime = () => {
    const now = new Date();
    now.setMinutes(now.getMinutes() + 1);
    return now.toISOString().slice(0, 16);
  };

  return (
    <form onSubmit={handleSubmit} className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 mb-8">
      <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
        Schedule a New Post
      </h2>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 p-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      <div className="space-y-4">
        <div>
          <label htmlFor="title" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Title (optional) <span className="text-xs text-gray-500">({title.length}/200)</span>
          </label>
          <input
            type="text"
            id="title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={200}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
            placeholder="Post title..."
          />
        </div>

        <div>
          <label htmlFor="content" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Content * <span className="text-xs text-gray-500">({content.length}/5000)</span>
          </label>
          <textarea
            id="content"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            required
            rows={4}
            maxLength={5000}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white resize-none"
            placeholder="What do you want to share?"
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label htmlFor="channel" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Channel *
            </label>
            <select
              id="channel"
              value={channel}
              onChange={(e) => setChannel(e.target.value)}
              required
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
            >
              <option value="twitter">🐦 Twitter</option>
              <option value="linkedin">💼 LinkedIn</option>
              <option value="facebook">📘 Facebook</option>
            </select>
          </div>

          <div>
            <label htmlFor="scheduledAt" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Schedule For *
            </label>
            <input
              type="datetime-local"
              id="scheduledAt"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
              required
              min={getMinDateTime()}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold py-3 px-6 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? 'Scheduling...' : 'Schedule Post'}
        </button>
      </div>
    </form>
  );
}
