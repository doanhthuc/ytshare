import { API_ENDPOINTS, httpClient } from '@/shared/constants';

import type { ShareVideoPayload, Video, VideoListResponse } from '../types';

export async function listVideos(params: {
  limit: number;
  offset: number;
}): Promise<VideoListResponse> {
  const { data } = await httpClient.get<VideoListResponse>(API_ENDPOINTS.videos, { params });
  return data;
}

export async function shareVideo(payload: ShareVideoPayload): Promise<Video> {
  const { data } = await httpClient.post<Video>(API_ENDPOINTS.videos, payload);
  return data;
}
