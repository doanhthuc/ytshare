import { useNavigate } from '@tanstack/react-router';
import axios from 'axios';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui';

import { ShareVideoForm } from '../components';
import { useShareVideo } from '../hooks';

export function ShareVideoPage() {
  const { t } = useTranslation('videos');
  const navigate = useNavigate();
  const mutation = useShareVideo();

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-2xl">{t('share.title')}</CardTitle>
          <CardDescription>{t('share.subtitle')}</CardDescription>
        </CardHeader>
        <CardContent>
          <ShareVideoForm
            isLoading={mutation.isPending}
            onSubmit={async (values) => {
              try {
                await mutation.mutateAsync(values);
                toast.success(t('toast.shared'));
                await navigate({ to: '/' });
              } catch (err) {
                if (axios.isAxiosError(err)) {
                  if (err.response?.status === 400) {
                    toast.error(t('toast.invalidUrl'));
                    return;
                  }
                  if (err.response?.status === 409) {
                    toast.error(t('toast.alreadyShared'));
                    return;
                  }
                }
                toast.error(t('toast.genericError'));
              }
            }}
          />
        </CardContent>
      </Card>
    </div>
  );
}
