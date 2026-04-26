import { Skeleton } from '@/components/ui';

export function VideoCardSkeleton() {
  return (
    <div className="block">
      <Skeleton className="aspect-video w-full rounded-xl" />
      <div className="mt-3 flex gap-3">
        <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
        <div className="min-w-0 flex-1 space-y-2">
          <Skeleton className="h-4 w-11/12" />
          <Skeleton className="h-3 w-2/3" />
          <Skeleton className="h-3 w-1/3" />
        </div>
      </div>
    </div>
  );
}
