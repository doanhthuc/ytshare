import { createFileRoute } from '@tanstack/react-router';

import { SignUpPage } from '@/modules/auth/pages';

export const Route = createFileRoute('/_public/signup')({
  component: SignUpPage,
});
