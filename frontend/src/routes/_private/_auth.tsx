import { useEffect } from 'react';
import { Outlet, createFileRoute, useNavigate } from '@tanstack/react-router';

import { requireAuth } from '@/modules/auth/utils';
import { useAuthStore } from '@/modules/auth/stores';

export const Route = createFileRoute('/_private/_auth')({
  beforeLoad: () => {
    requireAuth();
  },
  component: AuthLayout,
});

function AuthLayout() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      void navigate({ to: '/' });
    }
  }, [isAuthenticated, navigate]);

  if (!isAuthenticated) return null;

  return <Outlet />;
}
