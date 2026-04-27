import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { X } from 'lucide-react';

import type { VideoSharedPayload } from '../types';

type Props = {
  toastId: string | number;
  payload: VideoSharedPayload;
};

/**
 * NotificationToast renders a rich toast for a "video_shared" event:
 * thumbnail on the left, sharer + title in the middle, a Watch CTA on
 * the right, and a dismiss control. Designed to match the VideoCard
 * styling so it reads as a small preview of the shared video.
 */
export function NotificationToast({ toastId, payload }: Props) {
  const { t } = useTranslation('videos');
  const initial = payload.sharedByName?.[0]?.toUpperCase() ?? 'U';

  return (
    <div
      role="status"
      className="pointer-events-auto relative flex w-full max-w-sm gap-3 rounded-xl border border-(--color-border) bg-card p-3 pr-8 shadow-lg"
    >
      <a
        href={payload.youtubeId ? `https://www.youtube.com/watch?v=${payload.youtubeId}` : '#'}
        target="_blank"
        rel="noopener noreferrer"
        className="relative block aspect-video w-32 shrink-0 overflow-hidden rounded-lg bg-(--color-surface-2)"
        title={payload.title}
        onClick={() => toast.dismiss(toastId)}
      >
        <img
          src={payload.thumbnailUrl}
          alt={payload.title}
          className="h-full w-full object-cover"
          loading="lazy"
        />
      </a>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center gap-2">
          <div
            className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-surface-3 text-[11px] font-semibold text-(--color-foreground)"
            title={payload.sharedByName}
          >
            {initial}
          </div>
          <p className="truncate text-xs text-muted-foreground">
            {t('toast.sharedBy', {
              defaultValue: '{{name}} shared a video',
              name: payload.sharedByName,
            })}
          </p>
        </div>

        <h4 className="mt-1 line-clamp-2 text-sm font-semibold leading-snug text-(--color-foreground)">
          {payload.title}
        </h4>
      </div>

      <button
        type="button"
        onClick={() => toast.dismiss(toastId)}
        aria-label={t('toast.dismiss', { defaultValue: 'Dismiss' })}
        className="absolute top-2 right-2 inline-flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground hover:bg-(--color-surface-2) hover:text-(--color-foreground)"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
