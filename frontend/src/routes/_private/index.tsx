import { createFileRoute } from '@tanstack/react-router';

import { VideosPage } from '@/modules/videos/pages';

export const Route = createFileRoute('/_private/')({
  component: VideosPage,
});
