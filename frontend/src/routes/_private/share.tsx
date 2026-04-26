import { createFileRoute } from '@tanstack/react-router';

import { ShareVideoPage } from '@/modules/videos/pages';

export const Route = createFileRoute('/_private/share')({
  component: ShareVideoPage,
});
