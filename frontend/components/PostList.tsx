'use client';

import { Post } from '@/lib/types';
import { PostCard } from './PostCard';

interface PostListProps {
  posts: Post[];
  onUpdate?: () => void;
  showActions?: boolean;
  emptyMessage?: string;
}

export function PostList({ posts, onUpdate, showActions = true, emptyMessage = 'No posts yet' }: PostListProps) {
  if (posts.length === 0) {
    return (
      <div className="text-center py-12 text-gray-500 dark:text-gray-400">
        <p className="text-lg">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {posts.map((post) => (
        <PostCard 
          key={post.id} 
          post={post} 
          onUpdate={onUpdate}
          showActions={showActions}
        />
      ))}
    </div>
  );
}
