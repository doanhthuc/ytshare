import { create } from 'zustand';

import type { AuthSession } from '../types';
import { clearSession, readSession, writeSession } from '../utils/auth-storage.utils';

type AuthState = {
  session: AuthSession | null;
  isAuthenticated: boolean;
  signIn: (session: AuthSession) => void;
  signOut: () => void;
  updateTokens: (accessToken: string, refreshToken: string, expiresAt: string) => void;
};

export const useAuthStore = create<AuthState>((set, get) => ({
  session: readSession(),
  isAuthenticated: Boolean(readSession()),
  signIn: (session) => {
    writeSession(session);
    set({ session, isAuthenticated: true });
  },
  signOut: () => {
    clearSession();
    set({ session: null, isAuthenticated: false });
  },
  updateTokens: (accessToken, refreshToken, expiresAt) => {
    const current = get().session;
    if (!current) return;
    const next: AuthSession = { ...current, accessToken, refreshToken, expiresAt };
    writeSession(next);
    set({ session: next });
  },
}));

export function getAuthState() {
  return useAuthStore.getState();
}
