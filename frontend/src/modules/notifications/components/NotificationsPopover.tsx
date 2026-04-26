import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { formatRelativeTime } from '@/shared/utils/formatRelativeTime';

import { useNotificationsFeed } from '../hooks';

type Props = {
  open: boolean;
  onClose: () => void;
  anchorRef: React.RefObject<HTMLElement | null>;
};

export function NotificationsPopover({ open, onClose, anchorRef }: Props) {
  const { t } = useTranslation('common');
  const popoverRef = useRef<HTMLDivElement | null>(null);
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage } =
    useNotificationsFeed(open);

  // Close on outside click / Escape
  useEffect(() => {
    if (!open) return;
    function onDocClick(e: MouseEvent) {
      const target = e.target as Node;
      if (popoverRef.current?.contains(target)) return;
      if (anchorRef.current?.contains(target)) return;
      onClose();
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    document.addEventListener('mousedown', onDocClick);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDocClick);
      document.removeEventListener('keydown', onKey);
    };
  }, [open, onClose, anchorRef]);

  // Infinite scroll via IntersectionObserver
  useEffect(() => {
    if (!open) return;
    const sentinel = sentinelRef.current;
    const root = scrollRef.current;
    if (!sentinel || !root) return;

    const io = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting && hasNextPage && !isFetchingNextPage) {
            void fetchNextPage();
          }
        }
      },
      { root, rootMargin: '120px', threshold: 0 }
    );
    io.observe(sentinel);
    return () => io.disconnect();
  }, [open, hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (!open) return null;

  const items = data?.pages.flatMap((p) => p.items) ?? [];

  return (
    <div
      ref={popoverRef}
      role="dialog"
      aria-label={t('notifications.title')}
      className="absolute right-2 top-12 z-40 w-[420px] max-w-[92vw] rounded-xl border border-border bg-card shadow-2xl"
    >
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h2 className="text-base font-semibold">{t('notifications.title')}</h2>
      </div>

      <div ref={scrollRef} className="max-h-[70vh] overflow-y-auto py-2">
        {isLoading ? (
          <p className="px-4 py-6 text-sm text-muted-foreground">{t('notifications.loading')}</p>
        ) : items.length === 0 ? (
          <p className="px-4 py-6 text-sm text-muted-foreground">{t('notifications.empty')}</p>
        ) : (
          <ul>
            {items.map((v) => {
              const initial = v.sharedBy.name?.[0]?.toUpperCase() ?? 'U';
              return (
                <li key={v.id}>
                  <a
                    href={v.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex gap-3 px-4 py-3 hover:bg-(--color-surface-2)"
                  >
                    <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-surface-3 text-sm font-semibold">
                      {initial}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="line-clamp-2 text-sm leading-snug text-(--color-foreground)">
                        <span className="font-medium">{v.sharedBy.name}</span>
                        {': '}
                        {v.title}
                      </p>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {t('notifications.sharedAgo', { time: formatRelativeTime(v.sharedAt) })}
                      </p>
                    </div>
                    <img
                      src={v.thumbnailUrl}
                      alt=""
                      loading="lazy"
                      className="h-12 w-20 shrink-0 rounded object-cover"
                    />
                  </a>
                </li>
              );
            })}
          </ul>
        )}

        <div ref={sentinelRef} className="h-4" />

        {isFetchingNextPage ? (
          <p className="px-4 py-3 text-center text-xs text-muted-foreground">
            {t('notifications.loading')}
          </p>
        ) : null}
      </div>
    </div>
  );
}
