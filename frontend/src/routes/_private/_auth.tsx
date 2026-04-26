import { Outlet, createFileRoute } from '@tanstack/react-router';

import { requireAuth } from '@/modules/auth/utils';

export const Route = createFileRoute('/_private/_auth')({
  beforeLoad: () => {
    requireAuth();
  },
  component: AuthLayout,
});

function AuthLayout() {
  return <Outlet />;
}
