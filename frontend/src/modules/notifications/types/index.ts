export type VideoSharedPayload = {
  videoId: string;
  youtubeId: string;
  title: string;
  thumbnailUrl: string;
  sharedById: string;
  sharedByName: string;
};

export type NotificationEvent = {
  type: 'video_shared';
  timestamp: string;
  payload: VideoSharedPayload;
};
