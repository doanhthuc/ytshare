import { API_ENDPOINTS, httpClient } from '@/shared/constants';

export type UnreadCountResponse = { count: number };
export type MarkSeenResponse = { seenAt: string };

export async function getUnreadCount(): Promise<UnreadCountResponse> {
  const { data } = await httpClient.get<UnreadCountResponse>(
    API_ENDPOINTS.notificationsUnreadCount
  );
  return data;
}

export async function markNotificationsSeen(): Promise<MarkSeenResponse> {
  const { data } = await httpClient.post<MarkSeenResponse>(
    API_ENDPOINTS.notificationsMarkSeen
  );
  return data;
}
