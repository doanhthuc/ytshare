import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { API_ENDPOINTS, WS_BASE_URL } from '@/shared/constants';
import { useAuthStore } from '@/modules/auth/stores';
import { videoKeys } from '@/modules/videos/constants';

import type { NotificationEvent } from '../types';

const RECONNECT_DELAY_MS = 2_000;

/**
 * useNotificationsSocket subscribes to the backend WebSocket and pops a
 * toast for every "video_shared" event it receives.
 *
 * Implementation notes:
 * - Reconnects with a fixed back-off when the socket drops.
 * - Skips notifications the current user generated themselves.
 * - Invalidates the videos list so the new share appears without a refresh.
 */
export function useNotificationsSocket() {
  const { t } = useTranslation('videos');
  const session = useAuthStore((s) => s.session);
  const queryClient = useQueryClient();

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (!session) return;

    let cancelled = false;
    const url = `${WS_BASE_URL}${API_ENDPOINTS.notificationsWS}?access_token=${encodeURIComponent(session.accessToken)}`;

    function connect() {
      if (cancelled) return;
      const socket = new WebSocket(url);
      socketRef.current = socket;

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as NotificationEvent;
          if (data.type !== 'video_shared') return;
          if (data.payload.sharedById === session?.user.id) return;

          toast.message(
            t('toast.newShare', {
              name: data.payload.sharedByName,
              title: data.payload.title,
            })
          );
          void queryClient.invalidateQueries({ queryKey: videoKeys.all });
          void queryClient.invalidateQueries({ queryKey: ['notifications'] });
        } catch (err) {
          console.warn('[notifications] malformed event', err);
        }
      };

      socket.onclose = () => {
        if (cancelled) return;
        reconnectTimerRef.current = window.setTimeout(connect, RECONNECT_DELAY_MS);
      };
    }

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimerRef.current) window.clearTimeout(reconnectTimerRef.current);
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [session, queryClient, t]);
}
