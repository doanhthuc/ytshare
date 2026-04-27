const _API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string | undefined;
const _WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL as string | undefined;

// Trim and strip any trailing slashes so that a stray space, newline, or
// "/" in the env var (e.g. `VITE_WS_BASE_URL=wss://host/api/v1 `) does
// not produce malformed URLs like `/api/v1 /notifications/ws` — chi
// treats the literal space as part of the path and returns 404.
function clean(url: string | undefined, fallback: string): string {
  return (url ?? fallback).trim().replace(/\/+$/, '');
}

export const API_BASE_URL = clean(_API_BASE_URL, 'http://localhost:8080/api/v1');
export const WS_BASE_URL = clean(_WS_BASE_URL, 'ws://localhost:8080/api/v1');

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
