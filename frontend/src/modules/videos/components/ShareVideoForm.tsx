import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import type { TFunction } from 'i18next';
import { z } from 'zod';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui';
import { RHFFormProvider, RHFInput, RHFTextarea } from '@/shared/components';

import type { ShareVideoPayload } from '../types';

type ShareVideoFormProps = {
  onSubmit: (values: ShareVideoPayload) => void | Promise<void>;
  isLoading?: boolean;
};

const buildSchema = (t: TFunction) =>
  z.object({
    url: z.string().min(1, t('validation.urlRequired')).url(t('validation.urlInvalid')),
    title: z.string().max(255).optional(),
    description: z.string().max(4096).optional(),
  });

export function ShareVideoForm({ onSubmit, isLoading }: ShareVideoFormProps) {
  const { t } = useTranslation('videos');
  const schema = useMemo(() => buildSchema(t), [t]);

  const form = useForm<ShareVideoPayload>({
    resolver: zodResolver(schema),
    defaultValues: { url: '', title: '', description: '' },
  });

  return (
    <RHFFormProvider form={form} onSubmit={onSubmit} className="space-y-4">
      <RHFInput
        name="url"
        type="url"
        label={t('share.url')}
        placeholder={t('share.urlPlaceholder')}
      />
      <RHFInput
        name="title"
        label={t('share.titleField')}
        placeholder={t('share.titlePlaceholder')}
      />
      <RHFTextarea
        name="description"
        label={t('share.description')}
        placeholder={t('share.descriptionPlaceholder')}
        rows={4}
      />
      <Button type="submit" disabled={isLoading} className="w-full sm:w-auto">
        {isLoading ? t('share.submitting') : t('share.submit')}
      </Button>
    </RHFFormProvider>
  );
}
