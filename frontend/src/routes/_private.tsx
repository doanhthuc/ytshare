import { Outlet, createFileRoute } from '@tanstack/react-router';

import { HorizontalNav, Sidebar } from '@/shared/components';
import { useNotificationsSocket } from '@/modules/notifications';

export const Route = createFileRoute('/_private')({
  component: PrivateLayout,
});

function PrivateLayout() {
  useNotificationsSocket();

  return (
    <div className="min-h-screen bg-background text-(--color-foreground)">
      <HorizontalNav />
      <div className="flex">
        <Sidebar />
        <main className="flex-1 min-w-0 px-6 py-2">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
