import { Outlet, createFileRoute } from '@tanstack/react-router';

import { redirectIfAuthenticated } from '@/modules/auth/utils';

export const Route = createFileRoute('/_public')({
  beforeLoad: () => {
    redirectIfAuthenticated();
  },
  component: PublicLayout,
});

function PublicLayout() {
  return (
    <div className="min-h-screen bg-background text-(--color-foreground)">
      <Outlet />
    </div>
  );
}
