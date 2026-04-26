import { useNavigate } from '@tanstack/react-router';
import axios from 'axios';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui';

import { SignUpForm } from '../components';
import { useSignUp } from '../hooks';
import { useAuthStore } from '../stores';

export function SignUpPage() {
  const { t } = useTranslation('auth');
  const navigate = useNavigate();
  const setSession = useAuthStore((s) => s.signIn);
  const mutation = useSignUp();

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl">{t('signUp.title')}</CardTitle>
          <CardDescription>{t('signUp.subtitle')}</CardDescription>
        </CardHeader>
        <CardContent>
          <SignUpForm
            isLoading={mutation.isPending}
            onSubmit={async (values) => {
              try {
                const session = await mutation.mutateAsync(values);
                setSession(session);
                toast.success(t('toast.signedUp'));
                await navigate({ to: '/' });
              } catch (err) {
                if (axios.isAxiosError(err) && err.response?.status === 409) {
                  toast.error(t('toast.emailTaken'));
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
