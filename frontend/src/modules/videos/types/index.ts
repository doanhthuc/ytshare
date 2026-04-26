export type SharedBy = {
  id: string;
  name: string;
  email: string;
};

export type Video = {
  id: string;
  youtubeId: string;
  url: string;
  title: string;
  description: string;
  thumbnailUrl: string;
  sharedAt: string;
  sharedBy: SharedBy;
};

export type ShareVideoPayload = {
  url: string;
  title?: string;
  description?: string;
};

export type VideoListResponse = {
  items: Video[];
  total: number;
};
