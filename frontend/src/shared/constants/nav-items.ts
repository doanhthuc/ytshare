import { Home, PlaySquare, Clapperboard, User, Share2 } from 'lucide-react';

export type NavItem = {
  id: string;
  labelKey: string;
  to: string;
  icon: React.ComponentType<{ className?: string }>;
};

export const navItems: NavItem[] = [
  { id: 'home', labelKey: 'nav.videos', to: '/', icon: Home },
  { id: 'shorts', labelKey: 'nav.shorts', to: '/', icon: Clapperboard },
  { id: 'subs', labelKey: 'nav.subscriptions', to: '/', icon: PlaySquare },
  { id: 'you', labelKey: 'nav.you', to: '/', icon: User },
  { id: 'share', labelKey: 'nav.share', to: '/share', icon: Share2 },
];
