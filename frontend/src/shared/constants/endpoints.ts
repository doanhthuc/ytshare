const _API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string | undefined;
const _WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL as string | undefined;

export const API_BASE_URL = (_API_BASE_URL as string | undefined) ?? 'http://localhost:8080/api/v1';
export const WS_BASE_URL = (_WS_BASE_URL as string | undefined) ?? 'ws://localhost:8080/api/v1';

export const API_ENDPOINTS = {
  authSignUp: '/auth/signup',
  authSignIn: '/auth/signin',
  authRefresh: '/auth/refresh',
  videos: '/videos',
  notificationsWS: '/notifications/ws',
  notificationsUnreadCount: '/notifications/unread-count',
  notificationsMarkSeen: '/notifications/mark-seen',
  notificationsSince: '/notifications/since',
} as const;
