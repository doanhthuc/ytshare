import { API_ENDPOINTS, httpClient } from '@/shared/constants';

import type { NotificationsSinceResponse } from '../types';

export type UnreadCountResponse = { count: number };
export type MarkSeenResponse = { seenAt: string };

export async function getUnreadCount(): Promise<UnreadCountResponse> {
  const { data } = await httpClient.get<UnreadCountResponse>(
    API_ENDPOINTS.notificationsUnreadCount
  );
  return data;
}

export async function markNotificationsSeen(): Promise<MarkSeenResponse> {
  const { data } = await httpClient.post<MarkSeenResponse>(API_ENDPOINTS.notificationsMarkSeen);
  return data;
}

// Fallback recovery when WebSocket `?since=` replay is unavailable.
export async function getNotificationsSince(
  sinceId: string,
  limit = 100
): Promise<NotificationsSinceResponse> {
  const { data } = await httpClient.get<NotificationsSinceResponse>(
    API_ENDPOINTS.notificationsSince,
    { params: { id: sinceId, limit } }
  );
  return data;
}
