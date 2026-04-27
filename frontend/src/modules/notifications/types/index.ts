export type VideoSharedPayload = {
  videoId: string;
  youtubeId: string;
  title: string;
  thumbnailUrl: string;
  sharedById: string;
  sharedByName: string;
};

export type NotificationEvent = {
  id: string;
  type: 'video_shared';
  timestamp: string;
  recipientId?: string;
  payload: VideoSharedPayload;
};

export type NotificationsSinceResponse = {
  events: NotificationEvent[];
};
