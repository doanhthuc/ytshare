import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { useAuthStore } from './auth.store';

describe('useAuthStore', () => {
  beforeEach(() => {
    window.localStorage.clear();
    useAuthStore.setState({ session: null, isAuthenticated: false });
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it('persists session to localStorage on signIn', () => {
    useAuthStore.getState().signIn({
      user: { id: 'u1', email: 'a@b.c', name: 'Alice' },
      accessToken: 'a',
      refreshToken: 'r',
      expiresAt: '2099-01-01T00:00:00.000Z',
    });

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.session?.user.email).toBe('a@b.c');
    expect(window.localStorage.getItem('youtube-share.auth.session')).not.toBeNull();
  });

  it('clears localStorage on signOut', () => {
    useAuthStore.getState().signIn({
      user: { id: 'u1', email: 'a@b.c', name: 'Alice' },
      accessToken: 'a',
      refreshToken: 'r',
      expiresAt: '2099-01-01T00:00:00.000Z',
    });
    useAuthStore.getState().signOut();

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.session).toBeNull();
    expect(window.localStorage.getItem('youtube-share.auth.session')).toBeNull();
  });

  it('updateTokens swaps fresh credentials but keeps the user', () => {
    useAuthStore.getState().signIn({
      user: { id: 'u1', email: 'a@b.c', name: 'Alice' },
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      expiresAt: '2030-01-01T00:00:00.000Z',
    });
    useAuthStore.getState().updateTokens('new-access', 'new-refresh', '2099-01-01T00:00:00.000Z');

    const session = useAuthStore.getState().session;
    expect(session?.accessToken).toBe('new-access');
    expect(session?.refreshToken).toBe('new-refresh');
    expect(session?.user.id).toBe('u1');
  });
});
