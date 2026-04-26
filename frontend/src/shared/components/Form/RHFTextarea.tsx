import { Controller, type FieldPath, type FieldValues, useFormContext } from 'react-hook-form';

import { Label, Textarea, type TextareaProps } from '@/components/ui';
import { cn } from '@/shared/utils';

type RHFTextareaProps<TFieldValues extends FieldValues = FieldValues> = Omit<
  TextareaProps,
  'name'
> & {
  name: FieldPath<TFieldValues>;
  label?: string;
};

export function RHFTextarea<TFieldValues extends FieldValues = FieldValues>({
  name,
  label,
  className,
  ...props
}: RHFTextareaProps<TFieldValues>) {
  const { control } = useFormContext<TFieldValues>();
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <div className={cn('flex w-full flex-col gap-2', className)}>
          {label ? (
            <Label htmlFor={name} className={cn(fieldState.invalid && 'text-danger')}>
              {label}
            </Label>
          ) : null}
          <Textarea
            id={name}
            aria-invalid={fieldState.invalid || undefined}
            {...field}
            {...props}
          />
          {fieldState.error?.message ? (
            <p className="text-xs text-danger">{fieldState.error.message}</p>
          ) : null}
        </div>
      )}
    />
  );
}
