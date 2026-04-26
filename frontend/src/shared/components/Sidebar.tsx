import { Link, useRouterState } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';
import { Home, Share2 } from 'lucide-react';

import { cn } from '@/shared/utils';
import { useUIStore } from '@/shared/stores/ui.store';

type Item = {
  id: string;
  labelKey: string;
  to: string;
  icon: React.ComponentType<{ className?: string }>;
};

const items: Item[] = [
  { id: 'home', labelKey: 'nav.videos', to: '/', icon: Home },
  { id: 'share', labelKey: 'nav.share', to: '/share', icon: Share2 },
];

export function Sidebar() {
  const { t } = useTranslation('common');
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const expanded = useUIStore((s) => s.sidebarExpanded);

  return (
    <aside
      className={cn(
        'sticky top-14 hidden md:flex h-[calc(100vh-3.5rem)] shrink-0 flex-col gap-1 bg-[var(--color-background)] py-2 transition-[width] duration-200',
        expanded ? 'w-60 px-3' : 'w-20 items-stretch'
      )}
    >
      {items.map((item) => {
        const Icon = item.icon;
        const isActive =
          item.to === '/'
            ? pathname === '/'
            : pathname.startsWith(item.to);

        return (
          <Link
            key={item.id}
            to={item.to}
            className={cn(
              'rounded-xl text-sm font-medium transition',
              expanded
                ? 'flex items-center gap-6 px-3 py-2'
                : 'mx-1 flex flex-col items-center gap-1 px-1 py-3 text-[10px]',
              isActive
                ? 'bg-[var(--color-surface-2)] text-(--color-foreground)'
                : 'text-(--color-foreground) hover:bg-[var(--color-surface-2)]'
            )}
          >
            <Icon className={cn(expanded ? 'h-6 w-6' : 'h-6 w-6')} />
            <span className={cn(expanded ? '' : 'leading-none')}>{t(item.labelKey)}</span>
          </Link>
        );
      })}
    </aside>
  );
}
