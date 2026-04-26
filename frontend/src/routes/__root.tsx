import { Outlet, createRootRoute } from '@tanstack/react-router';

import { NotFound } from './-components/NotFound';
import { RouteErrorFallback } from './-components/RouteErrorFallback';

export const Route = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: NotFound,
  errorComponent: RouteErrorFallback,
});
