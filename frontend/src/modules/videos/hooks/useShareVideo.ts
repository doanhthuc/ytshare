import { useMutation, useQueryClient } from '@tanstack/react-query';

import { videoKeys } from '../constants';
import { shareVideo } from '../services';

export function useShareVideo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: shareVideo,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: videoKeys.all });
    },
  });
}
