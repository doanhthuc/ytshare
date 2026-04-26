import { useTranslation } from 'react-i18next';

import type { Video } from '../types';

import { VideoCard } from './VideoCard';
import { VideoCardSkeleton } from './VideoCardSkeleton';

type VideoListProps = {
  videos: Video[];
  isLoading?: boolean;
};

export function VideoList({ videos, isLoading }: VideoListProps) {
  const { t } = useTranslation('videos');
  if (isLoading) {
    return (
      <div
        className="grid gap-x-4 gap-y-10 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4"
        aria-label={t('list.loading')}
        aria-busy="true"
      >
        {Array.from({ length: 8 }).map((_, i) => (
          <VideoCardSkeleton key={i} />
        ))}
      </div>
    );
  }
  if (videos.length === 0) {
    return <p className="text-sm text-muted-foreground">{t('list.empty')}</p>;
  }
  return (
    <div className="grid gap-x-4 gap-y-10 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
      {videos.map((video) => (
        <VideoCard key={video.id} video={video} />
      ))}
    </div>
  );
}
