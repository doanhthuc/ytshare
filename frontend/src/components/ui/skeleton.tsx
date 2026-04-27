import { cn } from '@/shared/utils';

type SkeletonProps = React.HTMLAttributes<HTMLDivElement>;

export function Skeleton({ className, ...props }: SkeletonProps) {
  return (
    <div className={cn('animate-pulse rounded-md bg-(--color-surface-2)', className)} {...props} />
  );
}
