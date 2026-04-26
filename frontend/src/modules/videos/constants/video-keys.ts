export const videoKeys = {
  all: ['videos'] as const,
  list: (params: { limit: number; offset: number }) => ['videos', 'list', params] as const,
};
