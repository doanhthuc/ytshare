import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui';

type RouteErrorFallbackProps = { error: unknown; reset: () => void };

export function RouteErrorFallback({ error, reset }: RouteErrorFallbackProps) {
  const { t } = useTranslation('common');
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
      <h1 className="text-2xl font-semibold">{t('error.title')}</h1>
      <p className="max-w-md text-sm text-muted-foreground">
        {error instanceof Error ? error.message : 'Unexpected error'}
      </p>
      <Button onClick={reset}>{t('error.tryAgain')}</Button>
    </div>
  );
}
