import axios, { type AxiosError, type AxiosRequestConfig } from 'axios';

import { API_BASE_URL, API_ENDPOINTS } from './endpoints';

type AuthHooks = {
  getAccessToken: () => string | null;
  getRefreshToken: () => string | null;
  onTokensRefreshed: (accessToken: string, refreshToken: string, expiresAt: string) => void;
  onAuthFailed: () => void;
};

let hooks: AuthHooks | null = null;

export function configureAuth(impl: AuthHooks) {
  hooks = impl;
}

export const httpClient = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

httpClient.interceptors.request.use((config) => {
  const token = hooks?.getAccessToken();
  if (token) {
    config.headers = config.headers ?? {};
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

let refreshPromise: Promise<string> | null = null;

httpClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as (AxiosRequestConfig & { _retried?: boolean }) | undefined;
    if (!original || error.response?.status !== 401 || original._retried) {
      return Promise.reject(error);
    }
    const refreshToken = hooks?.getRefreshToken();
    if (!refreshToken) {
      hooks?.onAuthFailed();
      return Promise.reject(error);
    }

    try {
      original._retried = true;
      const newAccess = await (refreshPromise ??= (async () => {
        const res = await axios.post<{
          accessToken: string;
          refreshToken: string;
          expiresAt: string;
        }>(`${API_BASE_URL}${API_ENDPOINTS.authRefresh}`, { refreshToken });
        hooks?.onTokensRefreshed(res.data.accessToken, res.data.refreshToken, res.data.expiresAt);
        return res.data.accessToken;
      })());
      refreshPromise = null;

      original.headers = original.headers ?? {};
      (original.headers as Record<string, string>).Authorization = `Bearer ${newAccess}`;
      return httpClient.request(original);
    } catch (refreshErr) {
      refreshPromise = null;
      hooks?.onAuthFailed();
      return Promise.reject(refreshErr);
    }
  }
);
