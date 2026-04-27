import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { useAuthStore } from '@/modules/auth/stores';

import { getUnreadCount, markNotificationsSeen } from '../services';

// Exported so other hooks (e.g. useNotificationsSocket) can poke the
// cached value directly for optimistic UI updates.
export const UNREAD_QUERY_KEY = ['notifications', 'unread-count'] as const;
export const UNREAD_MAX = 99;

export function useNotificationsUnreadCount() {
  const isAuthed = useAuthStore((s) => Boolean(s.session));

  const { data } = useQuery({
    queryKey: UNREAD_QUERY_KEY,
    queryFn: getUnreadCount,
    enabled: isAuthed,
    refetchOnWindowFocus: true,
    staleTime: 30_000,
  });

  return data?.count ?? 0;
}

export function useMarkNotificationsSeen() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: markNotificationsSeen,
    onSuccess: () => {
      queryClient.setQueryData(UNREAD_QUERY_KEY, { count: 0 });
      void queryClient.invalidateQueries({ queryKey: UNREAD_QUERY_KEY });
    },
  });
}
