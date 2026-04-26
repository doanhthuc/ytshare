import { VideoList } from '../components';
import { useVideos } from '../hooks';

export function VideosPage() {
  const { data, isLoading } = useVideos();

  return (
    <div className="py-4">
      <VideoList videos={data?.items ?? []} isLoading={isLoading} />
    </div>
  );
}
