import { redirect } from '@tanstack/react-router';

import { getAuthState } from '../stores';

export function requireAuth() {
  if (!getAuthState().isAuthenticated) {
    throw redirect({ to: '/signin' });
  }
}

export function redirectIfAuthenticated() {
  if (getAuthState().isAuthenticated) {
    throw redirect({ to: '/' });
  }
}
