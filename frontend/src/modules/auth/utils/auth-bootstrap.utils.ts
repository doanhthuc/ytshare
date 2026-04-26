import { configureAuth } from '@/shared/constants';

import { getAuthState } from '../stores';

export function bootstrapAuth() {
  configureAuth({
    getAccessToken: () => getAuthState().session?.accessToken ?? null,
    getRefreshToken: () => getAuthState().session?.refreshToken ?? null,
    onTokensRefreshed: (accessToken, refreshToken, expiresAt) =>
      getAuthState().updateTokens(accessToken, refreshToken, expiresAt),
    onAuthFailed: () => getAuthState().signOut(),
  });
}
