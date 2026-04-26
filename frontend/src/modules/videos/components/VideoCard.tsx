import { useTranslation } from 'react-i18next';

import type { Video } from '../types';
import { formatRelativeTime } from '@/shared/utils/formatRelativeTime';

type VideoCardProps = { video: Video };

export function VideoCard({ video }: VideoCardProps) {
  const { t } = useTranslation('videos');
  const initial = video.sharedBy.name?.[0]?.toUpperCase() ?? 'U';

  return (
    <a
      href={video.url}
      target="_blank"
      rel="noopener noreferrer"
      className="group block"
      title={video.title}
    >
      <div className="relative aspect-video w-full overflow-hidden rounded-xl bg-(--color-surface-2)">
        <img
          src={video.thumbnailUrl}
          alt={video.title}
          className="h-full w-full object-cover transition group-hover:scale-[1.01]"
          loading="lazy"
        />
      </div>
      <div className="mt-3 flex gap-3">
        <div
          className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-surface-3 text-sm font-semibold text-(--color-foreground)"
          title={video.sharedBy.name}
        >
          {initial}
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="line-clamp-2 text-sm font-semibold leading-snug text-(--color-foreground)">
            {video.title}
          </h3>
          <p className="mt-1 truncate text-xs text-muted-foreground">{video.sharedBy.name}</p>
          <p className="truncate text-xs text-muted-foreground">
            {t('card.sharedAgo', {
              defaultValue: '{{time}} ago',
              time: formatRelativeTime(video.sharedAt),
            })}
          </p>
        </div>
      </div>
    </a>
  );
}
