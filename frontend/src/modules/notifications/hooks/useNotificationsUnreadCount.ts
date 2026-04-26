import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { useAuthStore } from '@/modules/auth/stores';

import { getUnreadCount, markNotificationsSeen } from '../services';

const UNREAD_QUERY_KEY = ['notifications', 'unread-count'] as const;

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
