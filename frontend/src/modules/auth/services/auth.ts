import { API_ENDPOINTS, httpClient } from '@/shared/constants';

import type { AuthSession, SignInPayload, SignUpPayload } from '../types';

type ServerAuthResponse = {
  user: { id: string; email: string; name: string };
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
};

function toSession(data: ServerAuthResponse): AuthSession {
  return {
    user: data.user,
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
    expiresAt: data.expiresAt,
  };
}

export async function signIn(payload: SignInPayload): Promise<AuthSession> {
  const { data } = await httpClient.post<ServerAuthResponse>(API_ENDPOINTS.authSignIn, payload);
  return toSession(data);
}

export async function signUp(payload: SignUpPayload): Promise<AuthSession> {
  const { data } = await httpClient.post<ServerAuthResponse>(API_ENDPOINTS.authSignUp, payload);
  return toSession(data);
}
