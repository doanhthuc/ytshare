import { createFileRoute } from '@tanstack/react-router';

import { SignInPage } from '@/modules/auth/pages';

export const Route = createFileRoute('/_public/signin')({
  component: SignInPage,
});
