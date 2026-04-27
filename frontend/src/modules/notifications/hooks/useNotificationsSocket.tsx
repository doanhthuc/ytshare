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
 * useNotificationsSocket subscribes to the backend WebSocket and pops a
 * toast for every relevant event it receives.
 *
 * Reliability guarantees layered on top of the raw socket:
 * - Persists the last-processed event ID in localStorage.
 * - Reconnects with `?since=<id>` so the backend replays missed events
 *   inline as the first WebSocket frames after upgrade.
 * - HTTP fallback to GET /notifications/since when the WebSocket itself
 *   fails to open (e.g. transient proxy error during reconnect).
 * - Client-side dedup by event ID covers the small overlap window
 *   between the replay backlog and live broadcasts.
 */
export function useNotificationsSocket() {
  const session = useAuthStore((s) => s.session);
  const queryClient = useQueryClient();

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | undefined>(undefined);
  // Recently processed event IDs. Bounded by SEEN_EVENT_CAP so it
  // cannot grow without bound across a long session.
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
      if (!data.id || !rememberSeen(data.id)) return;

      lastEventIdRef.current = data.id;
      try {
        window.localStorage.setItem(LAST_EVENT_ID_KEY, data.id);
      } catch {
        // Storage may be disabled (private mode, quota); the socket
        // still works — we just lose cross-reload replay.
      }

      if (data.type !== 'video_shared') return;
      if (data.payload.sharedById === session?.user.id) return;

      toast.custom(
        (toastId) => <NotificationToast toastId={toastId} payload={data.payload} />,
        { duration: 6_000 }
      );

      // Optimistically bump the bell badge so it updates instantly,
      // without waiting for the backend roundtrip that
      // invalidateQueries would trigger. The invalidations below
      // reconcile with the authoritative server count in the
      // background — safer than relying solely on optimistic state in
      // case the user has the bell popover open or marks-as-seen races
      // with this event.
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

      const socket = new WebSocket(url);
      socketRef.current = socket;
      let opened = false;

      socket.onopen = () => {
        opened = true;
      };

      socket.onmessage = (event) => {
        try {
          handleEvent(JSON.parse(event.data) as NotificationEvent);
        } catch (err) {
          console.warn('[notifications] malformed event', err);
        }
      };

      socket.onclose = () => {
        if (cancelled) return;
        // If the socket never opened, the inline `?since=` replay also
        // never ran — recover the missed window over HTTP before the
        // next attempt.
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
