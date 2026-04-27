import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { RouterProvider, createRouter } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { ErrorBoundary } from 'react-error-boundary';
import { Toaster } from 'sonner';

import { queryClient } from '@/shared/constants';
import { ErrorFallback } from '@/shared/components';
import { bootstrapAuth } from '@/modules/auth/utils';
import { routeTree } from '@/routeTree.gen';

// Wire the axios interceptor to the auth store before any component
// has a chance to issue a request.
bootstrapAuth();

const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export function App() {
  return (
    <ErrorBoundary FallbackComponent={ErrorFallback}>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
        <Toaster richColors position="top-right" closeButton />
        {import.meta.env.DEV ? (
          <TanStackRouterDevtools router={router} position="bottom-right" />
        ) : null}
        {import.meta.env.DEV ? <ReactQueryDevtools initialIsOpen={false} /> : null}
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
