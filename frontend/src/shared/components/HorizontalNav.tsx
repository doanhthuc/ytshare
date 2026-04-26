import { useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';
import { Bell, LogIn, LogOut, Menu, Mic, Plus, Search, Youtube } from 'lucide-react';

import { Button } from '@/components/ui';
import { useAuthStore } from '@/modules/auth/stores';
import { useUIStore } from '@/shared/stores/ui.store';
import {
  NotificationsPopover,
  useMarkNotificationsSeen,
  useNotificationsUnreadCount,
} from '@/modules/notifications';

export function HorizontalNav() {
  const { t } = useTranslation('common');
  const session = useAuthStore((s) => s.session);
  const signOut = useAuthStore((s) => s.signOut);
  const toggleSidebar = useUIStore((s) => s.toggleSidebar);

  const [notifOpen, setNotifOpen] = useState(false);
  const bellRef = useRef<HTMLButtonElement>(null);
  const unreadCount = useNotificationsUnreadCount();
  const { mutate: markAllSeen } = useMarkNotificationsSeen();

  const handleBellClick = () => {
    setNotifOpen((v) => {
      const next = !v;
      if (next && unreadCount > 0) markAllSeen();
      return next;
    });
  };

  const initial = session?.user.name?.[0]?.toUpperCase() ?? 'U';

  return (
    <header className="sticky top-0 z-30 w-full bg-[var(--color-background)]">
      <div className="flex h-14 items-center justify-between gap-4 px-4">
        <div className="flex items-center gap-4 min-w-0">
          <button
            type="button"
            onClick={toggleSidebar}
            className="grid h-10 w-10 place-items-center rounded-full text-(--color-foreground) hover:bg-[var(--color-surface-2)]"
            aria-label="Menu"
          >
            <Menu className="h-5 w-5" />
          </button>
          <Link to="/" className="flex items-center gap-1 font-semibold text-(--color-foreground)">
            <Youtube className="h-7 w-7 fill-[var(--color-primary)] text-[var(--color-background)]" />
            <span className="text-lg tracking-tight">{t('app.name')}</span>
          </Link>
        </div>

        <div className="flex flex-1 items-center justify-center gap-2 max-w-2xl">
          <form
            onSubmit={(e) => e.preventDefault()}
            className="flex flex-1 items-stretch h-10 rounded-full border border-[var(--color-border)] bg-[var(--color-surface-2)] overflow-hidden focus-within:border-[var(--color-primary)]"
          >
            <input
              type="search"
              placeholder={t('nav.search')}
              className="flex-1 bg-transparent px-4 text-sm outline-none placeholder:text-[var(--color-muted-foreground)]"
            />
            <button
              type="submit"
              aria-label={t('nav.search')}
              className="grid w-16 place-items-center bg-[var(--color-surface-3)] border-l border-[var(--color-border)] hover:opacity-90"
            >
              <Search className="h-5 w-5" />
            </button>
          </form>
          <button
            type="button"
            aria-label="Voice search"
            className="grid h-10 w-10 place-items-center rounded-full bg-[var(--color-surface-2)] hover:bg-[var(--color-surface-3)]"
          >
            <Mic className="h-5 w-5" />
          </button>
        </div>

        <div className="flex items-center gap-2">
          {session ? (
            <>
              <Link
                to="/share"
                className="hidden sm:inline-flex items-center gap-2 rounded-full bg-[var(--color-surface-2)] px-4 py-2 text-sm font-medium hover:bg-[var(--color-surface-3)]"
              >
                <Plus className="h-4 w-4" />
                {t('nav.shareVideo')}
              </Link>
              <div className="relative">
                <button
                  ref={bellRef}
                  type="button"
                  aria-label="Notifications"
                  aria-expanded={notifOpen}
                  onClick={handleBellClick}
                  className="relative grid h-10 w-10 place-items-center rounded-full hover:bg-[var(--color-surface-2)]"
                >
                  <Bell className="h-5 w-5" />
                  {unreadCount > 0 ? (
                    <span
                      aria-label={`${unreadCount} unread notifications`}
                      className="absolute right-1 top-1 grid min-w-[18px] h-[18px] place-items-center rounded-full bg-[var(--color-primary)] px-1 text-[10px] font-semibold leading-none text-[var(--color-primary-foreground)] ring-2 ring-[var(--color-background)]"
                    >
                      {unreadCount > 9 ? '9+' : unreadCount}
                    </span>
                  ) : null}
                </button>
                <NotificationsPopover
                  open={notifOpen}
                  onClose={() => setNotifOpen(false)}
                  anchorRef={bellRef}
                />
              </div>
              <div className="flex items-center gap-2">
                <div
                  className="grid h-8 w-8 place-items-center rounded-full bg-[var(--color-primary)] text-sm font-semibold text-[var(--color-primary-foreground)]"
                  title={session.user.name}
                >
                  {initial}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => signOut()}
                  className="text-(--color-foreground) hover:bg-[var(--color-surface-2)]"
                >
                  <LogOut className="h-4 w-4" />
                  <span className="sr-only">{t('actions.signOut')}</span>
                </Button>
              </div>
            </>
          ) : (
            <Link
              to="/signin"
              className="inline-flex items-center gap-2 rounded-full border border-[var(--color-border)] bg-[var(--color-surface-2)] px-4 py-2 text-sm font-medium text-[var(--color-primary)] hover:bg-[var(--color-surface-3)]"
            >
              <LogIn className="h-4 w-4" />
              {t('actions.signIn')}
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
