import { useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import type { TFunction } from 'i18next';
import { z } from 'zod';
import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui';
import { RHFFormProvider, RHFInput } from '@/shared/components';

import type { SignUpPayload } from '../types';

type SignUpFormProps = {
  onSubmit: (values: SignUpPayload) => void | Promise<void>;
  isLoading?: boolean;
};

const buildSchema = (t: TFunction) =>
  z.object({
    email: z.string().min(1, t('validation.emailRequired')).email(t('validation.emailInvalid')),
    name: z.string().min(1, t('validation.nameRequired')),
    password: z
      .string()
      .min(1, t('validation.passwordRequired'))
      .min(6, t('validation.passwordMin')),
  });

export function SignUpForm({ onSubmit, isLoading }: SignUpFormProps) {
  const { t } = useTranslation('auth');
  const schema = useMemo(() => buildSchema(t), [t]);

  const form = useForm<SignUpPayload>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', name: '', password: '' },
  });

  return (
    <RHFFormProvider form={form} onSubmit={onSubmit} className="space-y-4">
      <RHFInput
        name="name"
        autoComplete="name"
        label={t('form.name')}
        placeholder={t('form.namePlaceholder')}
      />
      <RHFInput
        name="email"
        type="email"
        autoComplete="email"
        label={t('form.email')}
        placeholder={t('form.emailPlaceholder')}
      />
      <RHFInput
        name="password"
        type="password"
        autoComplete="new-password"
        label={t('form.password')}
        placeholder={t('form.passwordPlaceholder')}
      />
      <Button type="submit" className="w-full" disabled={isLoading}>
        {isLoading ? t('signUp.submitting') : t('signUp.submit')}
      </Button>
      <p className="text-center text-sm text-muted-foreground">
        {t('signUp.switchLabel')}{' '}
        <Link to="/signin" className="text-primary underline-offset-4 hover:underline">
          {t('signUp.switchAction')}
        </Link>
      </p>
    </RHFFormProvider>
  );
}
