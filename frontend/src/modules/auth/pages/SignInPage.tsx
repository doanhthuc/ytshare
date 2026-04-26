import { useNavigate } from '@tanstack/react-router';
import axios from 'axios';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui';

import { SignInForm } from '../components';
import { useSignIn } from '../hooks';
import { useAuthStore } from '../stores';

export function SignInPage() {
  const { t } = useTranslation('auth');
  const navigate = useNavigate();
  const setSession = useAuthStore((s) => s.signIn);
  const mutation = useSignIn();

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl">{t('signIn.title')}</CardTitle>
          <CardDescription>{t('signIn.subtitle')}</CardDescription>
        </CardHeader>
        <CardContent>
          <SignInForm
            isLoading={mutation.isPending}
            onSubmit={async (values) => {
              try {
                const session = await mutation.mutateAsync(values);
                setSession(session);
                toast.success(t('toast.signedIn', { name: session.user.name }));
                await navigate({ to: '/' });
              } catch (err) {
                if (axios.isAxiosError(err) && err.response?.status === 401) {
                  toast.error(t('toast.invalidCredentials'));
                } else {
                  toast.error(t('toast.genericError'));
                }
              }
            }}
          />
        </CardContent>
      </Card>
    </div>
  );
}
