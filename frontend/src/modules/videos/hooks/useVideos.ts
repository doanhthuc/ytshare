import { useQuery } from '@tanstack/react-query';

import { videoKeys } from '../constants';
import { listVideos } from '../services';

export function useVideos(params: { limit?: number; offset?: number } = {}) {
  const limit = params.limit ?? 20;
  const offset = params.offset ?? 0;
  return useQuery({
    queryKey: videoKeys.list({ limit, offset }),
    queryFn: () => listVideos({ limit, offset }),
  });
}
