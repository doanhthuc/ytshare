import { Navigate, Outlet, createFileRoute } from '@tanstack/react-router';

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

  if (!isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
