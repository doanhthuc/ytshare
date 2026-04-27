import { useCallback, useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { API_ENDPOINTS, WS_BASE_URL } from '@/shared/constants';
import { useAuthStore } from '@/modules/auth/stores';
import { videoKeys } from '@/modules/videos/constants';

import { NotificationToast } from '../components';
import { getNotificationsSince } from '../services';
import { UNREAD_MAX, UNREAD_QUERY_KEY } from './useNotificationsUnreadCount';
import type { NotificationEvent } from '../types';

const RECONNECT_DELAY_MS = 2_000;
const LAST_EVENT_ID_KEY = 'notifications.lastEventId';
const SEEN_EVENT_CAP = 256;

/**
 * Reliability layered over the raw socket:
 * - Persists last-processed event ID; reconnects with `?since=<id>` for inline replay.
 * - HTTP fallback to GET /notifications/since when the socket fails to open.
 * - Client-side dedup by event ID covers replay/live overlap.
 */
export function useNotificationsSocket() {
  const session = useAuthStore((s) => s.session);
  const queryClient = useQueryClient();

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | undefined>(undefined);
  const seenIdsRef = useRef<Set<string>>(new Set());
  const seenOrderRef = useRef<string[]>([]);
  const lastEventIdRef = useRef<string>(
    (typeof window !== 'undefined' && window.localStorage.getItem(LAST_EVENT_ID_KEY)) || ''
  );

  const rememberSeen = useCallback((id: string) => {
    if (seenIdsRef.current.has(id)) return false;
    seenIdsRef.current.add(id);
    seenOrderRef.current.push(id);
    if (seenOrderRef.current.length > SEEN_EVENT_CAP) {
      const evicted = seenOrderRef.current.shift();
      if (evicted) seenIdsRef.current.delete(evicted);
    }
    return true;
  }, []);

  const handleEvent = useCallback(
    (data: NotificationEvent) => {
      if (!data.id || !rememberSeen(data.id)) {
        console.debug('[notifications] skip duplicate or unidentified', data.id);
        return;
      }

      lastEventIdRef.current = data.id;
      try {
        window.localStorage.setItem(LAST_EVENT_ID_KEY, data.id);
      } catch {
        // Storage disabled (private mode/quota): lose cross-reload replay only.
      }

      if (data.type !== 'video_shared') {
        console.debug('[notifications] skip non-video_shared', data.type);
        return;
      }
      if (data.payload.sharedById === session?.user.id) {
        console.debug('[notifications] skip own share');
        return;
      }

      toast.custom((toastId) => <NotificationToast toastId={toastId} payload={data.payload} />, {
        duration: 6_000,
      });

      // Optimistic bump; invalidations below reconcile with server count.
      queryClient.setQueryData<{ count: number }>(UNREAD_QUERY_KEY, (prev) => ({
        count: Math.min((prev?.count ?? 0) + 1, UNREAD_MAX),
      }));
      void queryClient.invalidateQueries({ queryKey: videoKeys.all });
      void queryClient.invalidateQueries({ queryKey: ['notifications', 'feed'] });
    },
    [queryClient, rememberSeen, session?.user.id]
  );

  useEffect(() => {
    if (!session) return;

    let cancelled = false;

    function buildUrl() {
      if (!session) return null;
      const params = new URLSearchParams({ access_token: session.accessToken });
      if (lastEventIdRef.current) params.set('since', lastEventIdRef.current);
      return `${WS_BASE_URL}${API_ENDPOINTS.notificationsWS}?${params.toString()}`;
    }

    async function recoverViaHttp() {
      const sinceId = lastEventIdRef.current;
      if (!sinceId) return;
      try {
        const { events } = await getNotificationsSince(sinceId);
        for (const evt of events) handleEvent(evt);
      } catch (err) {
        console.warn('[notifications] http recovery failed', err);
      }
    }

    function connect() {
      if (cancelled) return;
      const url = buildUrl();
      if (!url) return;

      const debugUrl = url.replace(/access_token=[^&]+/, 'access_token=***');
      console.info('[notifications] ws connecting', debugUrl);

      const socket = new WebSocket(url);
      socketRef.current = socket;
      let opened = false;

      socket.onopen = () => {
        opened = true;
        console.info('[notifications] ws open');
      };

      socket.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data) as NotificationEvent;
          console.debug('[notifications] ws event', parsed.id, parsed.type);
          handleEvent(parsed);
        } catch (err) {
          console.warn('[notifications] malformed event', err);
        }
      };

      socket.onerror = (event) => {
        console.warn('[notifications] ws error', event);
      };

      socket.onclose = (event) => {
        if (cancelled) return;
        console.info(
          '[notifications] ws close',
          'code=' + event.code,
          'reason=' + (event.reason || '(empty)'),
          'wasOpened=' + opened
        );
        // Socket never opened: inline `?since=` replay didn't run, recover via HTTP.
        if (!opened) void recoverViaHttp();
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
  }, [session, handleEvent]);
}
