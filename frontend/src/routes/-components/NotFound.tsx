import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { buttonVariants } from '@/components/ui';
import { cn } from '@/shared/utils';

export function NotFound() {
  const { t } = useTranslation('common');
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
      <h1 className="text-3xl font-semibold">404</h1>
      <p className="text-sm text-muted-foreground">Page not found.</p>
      <Link to="/" className={cn(buttonVariants({ variant: 'default' }))}>
        {t('actions.back')}
      </Link>
    </div>
  );
}
