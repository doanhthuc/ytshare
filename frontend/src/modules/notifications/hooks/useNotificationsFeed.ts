import { useInfiniteQuery } from '@tanstack/react-query';

import { listVideos } from '@/modules/videos/services';

const PAGE_SIZE = 10;

export function useNotificationsFeed(enabled: boolean) {
  return useInfiniteQuery({
    queryKey: ['notifications', 'feed', { limit: PAGE_SIZE }] as const,
    enabled,
    initialPageParam: 0,
    queryFn: ({ pageParam }) => listVideos({ limit: PAGE_SIZE, offset: pageParam }),
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((n, p) => n + p.items.length, 0);
      return loaded < lastPage.total ? loaded : undefined;
    },
  });
}
